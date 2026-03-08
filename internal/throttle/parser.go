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

func (q Quota) NearLimit() (bool, string) {
	if q.IndividualPerHourUsed >= 2500 {
		return true, fmt.Sprintf("quota: %d/~3000 hourly requests used", q.IndividualPerHourUsed)
	}
	if q.RegisteredPerWeekUsed >= 45000 {
		return true, fmt.Sprintf("quota: %d/~50000 weekly requests used", q.RegisteredPerWeekUsed)
	}
	return false, ""
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
