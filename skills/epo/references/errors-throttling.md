# Errors and Throttling

## OPS Headers to Watch

- `X-Throttling-Control`
- `Retry-After`
- `X-IndividualQuotaPerHour-Used`
- `X-RegisteredQuotaPerWeek-Used`

## Throttle Interpretation

- Green: normal pace.
- Yellow: reduce request rate.
- Red: aggressive backoff.
- Black: pause immediately and honor `Retry-After`.

## CLI Exit Codes

- `0`: success
- `1`: general error
- `2`: usage/validation
- `3`: auth failure
- `4`: not found
- `5`: rate limited
- `6`: server error

## Agent Handling Pattern

1. On `3`: refresh credential setup and re-run auth check.
2. On `5`: wait/backoff, then retry bounded times.
3. On `6`: retry with exponential backoff and cap attempts.
4. On repeated `4`: stop retrying and report missing record.

## JSON Error Envelope Shape

```json
{
  "ok": false,
  "error": {
    "code": 429,
    "type": "RATE_LIMITED",
    "message": "..."
  },
  "version": "dev"
}
```
