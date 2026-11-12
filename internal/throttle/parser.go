package throttle

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var throttlePattern = regexp.MustCompile(`^\s*([a-zA-Z]+)\s*\((.*)\)\s*$`)

type ServiceState struct {
	Color string `json:"color"`
	Limit int    `json:"limit"`
}

type State struct {
	System   string                  `json:"system,omitempty"`
	Raw      string                  `json:"raw,omitempty"`
	Services map[string]ServiceState `json:"services,omitempty"`
}

type Quota struct {
	IndividualPerHourUsed       int `json:"individualPerHourUsed,omitempty"`
	RegisteredPerWeekUsed       int `json:"registeredPerWeekUsed,omitempty"`
	RegisteredPayingPerWeekUsed int `json:"registeredPayingPerWeekUsed,omitempty"`
}

type Metadata struct {
	Throttle   State         `json:"throttle,omitempty"`
	Quota      Quota         `json:"quota,omitempty"`
	RetryAfter time.Duration `json:"-"`
}

func ParseHeaders(header http.Header) Metadata {
	if header == nil {
		return Metadata{}
	}

	meta := Metadata{
		Throttle: ParseThrottleControl(header.Get("X-Throttling-Control")),
		Quota: Quota{
			IndividualPerHourUsed:       parseInt(header.Get("X-IndividualQuotaPerHour-Used")),
			RegisteredPerWeekUsed:       parseInt(header.Get("X-RegisteredQuotaPerWeek-Used")),
			RegisteredPayingPerWeekUsed: parseInt(header.Get("X-RegisteredPayingQuotaPerWeek-Used")),
		},
		RetryAfter: ParseRetryAfter(header.Get("Retry-After")),
	}
	return meta
}

func ParseThrottleControl(raw string) State {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return State{}
	}

	matches := throttlePattern.FindStringSubmatch(raw)
	if len(matches) != 3 {
		return State{Raw: raw}
	}

	system := strings.TrimSpace(matches[1])
	servicesRaw := strings.TrimSpace(matches[2])

	services := map[string]ServiceState{}
	for _, part := range strings.Split(servicesRaw, ",") {
		entry := strings.TrimSpace(part)
		if entry == "" {
			continue
		}

		keyValue := strings.SplitN(entry, "=", 2)
		if len(keyValue) != 2 {
			continue
		}

		name := strings.TrimSpace(keyValue[0])
		colorLimit := strings.TrimSpace(keyValue[1])
		pieces := strings.SplitN(colorLimit, ":", 2)
		if len(pieces) != 2 {
			continue
		}

		color := strings.TrimSpace(pieces[0])
		limit := parseInt(pieces[1])
		services[name] = ServiceState{
			Color: color,
			Limit: limit,
		}
	}

	return State{
		System:   system,
		Raw:      raw,
		Services: services,
	}
}

// OPS Retry-After values are documented in milliseconds.
func ParseRetryAfter(raw string) time.Duration {
	ms := parseInt(raw)
	if ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

// OPS reports quota usage in bytes of response data, not request counts. The
// OPS reference guide enforces its 1 Mbps rule as roughly 450 MB per rolling
// hour, and the fair use charter allows registered non-paying users 4 GB per
// calendar week.
const (
	HourlyQuotaBytes int64 = 450_000_000
	WeeklyQuotaBytes int64 = 4_000_000_000

	nearLimitPercent = 90
)

func (q Quota) NearLimit() (bool, string) {
	if used := int64(q.IndividualPerHourUsed); used*100 >= HourlyQuotaBytes*nearLimitPercent {
		return true, fmt.Sprintf("quota: %s of ~%s hourly data quota used", formatBytes(used), formatBytes(HourlyQuotaBytes))
	}
	if used := int64(q.RegisteredPerWeekUsed); used*100 >= WeeklyQuotaBytes*nearLimitPercent {
		return true, fmt.Sprintf("quota: %s of ~%s weekly data quota used", formatBytes(used), formatBytes(WeeklyQuotaBytes))
	}
	return false, ""
}

func formatBytes(n int64) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.2f GB", float64(n)/1e9)
	}
	return fmt.Sprintf("%.0f MB", float64(n)/1e6)
}

func (s State) HasBlackService() (bool, string) {
	for name, svc := range s.Services {
		if strings.EqualFold(svc.Color, "black") {
			return true, fmt.Sprintf("throttle: %s service is overloaded (black)", name)
		}
	}
	return false, ""
}

func parseInt(raw string) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return v
}
