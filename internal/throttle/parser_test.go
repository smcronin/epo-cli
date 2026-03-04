package throttle

import (
	"net/http"
	"testing"
	"time"
)

func TestParseHeaders(t *testing.T) {
	t.Parallel()

	header := http.Header{}
	header.Set("X-IndividualQuotaPerHour-Used", "3006")
	header.Set("X-RegisteredQuotaPerWeek-Used", "900006")
	header.Set("X-RegisteredPayingQuotaPerWeek-Used", "0")
	header.Set("X-Throttling-Control", "idle (retrieval=green:200, search=yellow:20)")
	header.Set("Retry-After", "60000")

	meta := ParseHeaders(header)
	if meta.Quota.IndividualPerHourUsed != 3006 {
		t.Fatalf("unexpected individual quota: %d", meta.Quota.IndividualPerHourUsed)
	}
	if meta.Quota.RegisteredPerWeekUsed != 900006 {
		t.Fatalf("unexpected registered quota: %d", meta.Quota.RegisteredPerWeekUsed)
	}
	if meta.Throttle.System != "idle" {
		t.Fatalf("unexpected system: %q", meta.Throttle.System)
	}
	if meta.Throttle.Services["search"].Color != "yellow" {
		t.Fatalf("unexpected search color: %q", meta.Throttle.Services["search"].Color)
	}
	if meta.Throttle.Services["retrieval"].Limit != 200 {
		t.Fatalf("unexpected retrieval limit: %d", meta.Throttle.Services["retrieval"].Limit)
	}
	if meta.RetryAfter != 60*time.Second {
		t.Fatalf("unexpected retry-after: %v", meta.RetryAfter)
	}
}

func TestParseThrottleControlInvalid(t *testing.T) {
	t.Parallel()

	state := ParseThrottleControl("garbled throttle text")
	if state.Raw == "" {
		t.Fatal("expected raw text to be preserved")
	}
	if state.System != "" {
		t.Fatalf("expected empty system for invalid value, got %q", state.System)
	}
}
