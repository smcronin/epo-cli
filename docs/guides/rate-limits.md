# Rate Limits & Fair Use Policy

OPS is free but enforced. Violate fair use and you get throttled or blocked.

## Quota Tiers

| Tier | Limit | Notes |
|------|-------|-------|
| Anonymous | None | No access (must register) |
| Registered (free) | ~4 GB/week | Standard free tier |
| Registered (paid) | Negotiated | Contact EPO |
| Per-hour individual | ~450 MB/hour | ~1 Mbps global cap |

Weekly quota resets every Monday at midnight UTC/GMT.
Hourly quota rolls on a sliding 60-second window.

---

## Response Headers You Must Read

Every OPS response includes these headers:

```
X-IndividualQuotaPerHour-Used: 3006
X-RegisteredQuotaPerWeek-Used: 900006
X-RegisteredPayingQuotaPerWeek-Used: 0
X-Throttling-Control: idle (retrieval=green:200, search=yellow:20, inpadoc=red:30, images=green:200, other=green:1000)
```

### Throttle Services Map

| Throttle name | OPS endpoints |
|---------------|--------------|
| `search` | `/published-data/search/*` |
| `retrieval` | `/published-data/*/` |
| `inpadoc` | `/family/*`, `/legal/*` |
| `images` | `/published-data/images/*`, `/classification/cpc/media/*` |
| `other` | Everything else |

---

## X-Throttling-Control: Traffic Light System

Format:
```
X-Throttling-Control: <system-state> (service=<color>:<limit>, ...)
```

**System states:** `idle` | `busy` | `overloaded`

**Traffic light colors:**
| Color | Meaning |
|-------|---------|
| 🟢 green | < 50% of request limit used |
| 🟡 yellow | 50–75% used |
| 🔴 red | > 75% used — slow down |
| ⚫ black | Limit exceeded — service suspended |

When `black`, a `Retry-After` header appears:
```
Retry-After: 60000
```
(value in milliseconds)

---

## Request Limits by System State

| State | retrieval | search | inpadoc | images | other |
|-------|-----------|--------|---------|--------|-------|
| idle | 200 | 30 | 60 | 200 | 1000 |
| busy | 100 | 15 | 45 | 100 | 1000 |
| overloaded | 50 | 5 | 30 | 50 | 1000 |

These are requests per 60-second window, per user.

---

## Quota Exhausted Response

```
HTTP/1.1 403 Forbidden
X-Rejection-Reason: RegisteredQuotaPerWeek

<error>
  <code>403</code>
  <message>This request has been rejected due to the violation of Fair Use policy</message>
  <moreInfo>http://www.epo.org/searching/free/espacenet/fair-use.html</moreInfo>
</error>
```

---

## Best Practices

1. **Respect the traffic light.** When you see yellow or red, back off. When you see black, stop and wait for `Retry-After`.

2. **Spread requests evenly.** OPS penalizes burst patterns. Steady drip > intense bursts.

3. **Use POST for bulk.** Bulk biblio retrieval: up to 100 patent numbers per POST request.

4. **Use JSON.** `Accept: application/json` avoids XML parsing overhead.

5. **Cache aggressively.** Patent data is stable. Cache biblio results for at least 24h.

6. **Check throttle state on every response** — OPS runs on multiple instances and state varies per-instance.

---

## Self-Throttling Implementation (Go)

```go
import "time"

type ThrottleState struct {
    System  string // idle | busy | overloaded
    Service map[string]struct {
        Color string // green | yellow | red | black
        Limit int
    }
}

func HandleResponse(resp *http.Response) {
    throttle := resp.Header.Get("X-Throttling-Control")
    // parse throttle...

    retryAfter := resp.Header.Get("Retry-After")
    if retryAfter != "" {
        ms, _ := strconv.Atoi(retryAfter)
        time.Sleep(time.Duration(ms) * time.Millisecond)
    }

    // Back off if any service is red/black
    if strings.Contains(throttle, "red") {
        time.Sleep(2 * time.Second)
    }
    if strings.Contains(throttle, "black") {
        time.Sleep(10 * time.Second)
    }
}
```

---

## Data Usage API

Track your own usage (not counted against quota):

```bash
# Single day
GET https://ops.epo.org/3.2/developers/me/stats/usage?timeRange=01/03/2024
Authorization: Bearer <token>

# Date range
GET https://ops.epo.org/3.2/developers/me/stats/usage?timeRange=01/03/2024~07/03/2024
Authorization: Bearer <token>
```

**Response fields:**
- `total_response_size` — bytes consumed (includes free + paid)
- `message_count` — number of requests
- Timestamps in Unix time (ms)

Updates within 10 minutes of each hour.
