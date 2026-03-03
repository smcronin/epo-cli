// Example: Parsing and respecting X-Throttling-Control headers from OPS.
// Critical for avoiding temporary suspension and fair use violations.

package examples

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ThrottleStatus represents parsed X-Throttling-Control
type ThrottleStatus struct {
	SystemState string // idle | busy | overloaded
	Services    map[string]ServiceThrottle
}

type ServiceThrottle struct {
	Color string // green | yellow | red | black
	Limit int
}

// ParseThrottleHeader parses X-Throttling-Control header.
// Format: idle (retrieval=green:200, search=yellow:20, inpadoc=red:30, images=green:200, other=green:1000)
func ParseThrottleHeader(header string) ThrottleStatus {
	ts := ThrottleStatus{
		Services: make(map[string]ServiceThrottle),
	}
	if header == "" {
		return ts
	}

	// Split "idle (....)"
	parts := strings.SplitN(header, " (", 2)
	if len(parts) < 1 {
		return ts
	}
	ts.SystemState = strings.TrimSpace(parts[0])

	if len(parts) < 2 {
		return ts
	}

	// Parse service=color:limit pairs
	serviceStr := strings.TrimSuffix(parts[1], ")")
	for _, svc := range strings.Split(serviceStr, ", ") {
		kv := strings.SplitN(svc, "=", 2)
		if len(kv) != 2 {
			continue
		}
		name := strings.TrimSpace(kv[0])
		colorLimit := strings.SplitN(kv[1], ":", 2)
		if len(colorLimit) != 2 {
			continue
		}
		limit, _ := strconv.Atoi(colorLimit[1])
		ts.Services[name] = ServiceThrottle{
			Color: colorLimit[0],
			Limit: limit,
		}
	}
	return ts
}

// ShouldBackOff returns whether we should pause before the next request.
func (ts ThrottleStatus) ShouldBackOff() bool {
	for _, svc := range ts.Services {
		if svc.Color == "red" || svc.Color == "black" {
			return true
		}
	}
	return ts.SystemState == "overloaded"
}

// BackOffDuration returns how long to sleep based on throttle state.
func (ts ThrottleStatus) BackOffDuration() time.Duration {
	hasBlack := false
	hasRed := false
	for _, svc := range ts.Services {
		switch svc.Color {
		case "black":
			hasBlack = true
		case "red":
			hasRed = true
		}
	}
	switch {
	case hasBlack:
		return 10 * time.Second
	case hasRed:
		return 2 * time.Second
	case ts.SystemState == "overloaded":
		return 1 * time.Second
	case ts.SystemState == "busy":
		return 250 * time.Millisecond
	}
	return 0
}

// HandleThrottle inspects OPS response headers and sleeps if needed.
// Call after every response. Returns the parsed throttle status.
func HandleThrottle(resp *http.Response) ThrottleStatus {
	// Parse Retry-After first (hard pause when black)
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		ms, err := strconv.Atoi(retryAfter)
		if err == nil && ms > 0 {
			fmt.Printf("[throttle] Retry-After: sleeping %dms\n", ms)
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}
	}

	ts := ParseThrottleHeader(resp.Header.Get("X-Throttling-Control"))

	if dur := ts.BackOffDuration(); dur > 0 {
		fmt.Printf("[throttle] state=%s backing off %s\n", ts.SystemState, dur)
		time.Sleep(dur)
	}

	// Log quota headers
	if quota := resp.Header.Get("X-IndividualQuotaPerHour-Used"); quota != "" {
		fmt.Printf("[quota] hour=%s week=%s\n",
			quota,
			resp.Header.Get("X-RegisteredQuotaPerWeek-Used"))
	}

	// Quota exhausted
	if reason := resp.Header.Get("X-Rejection-Reason"); reason != "" {
		fmt.Printf("[quota] REJECTED: %s\n", reason)
	}

	return ts
}
