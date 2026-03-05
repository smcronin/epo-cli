# Errors and Throttling

## Headers to Inspect

1. `X-Throttling-Control`
2. `Retry-After`
3. `X-IndividualQuotaPerHour-Used`
4. `X-RegisteredQuotaPerWeek-Used`

## CLI Exit Codes

1. `0`: success
2. `1`: general error
3. `2`: usage/validation error
4. `3`: auth failure
5. `4`: not found
6. `5`: rate limited
7. `6`: server error

## Error Envelope

```json
{
  "ok": false,
  "error": {
    "code": 429,
    "type": "RATE_LIMITED",
    "message": "..."
  },
  "version": "v0.1.0"
}
```

## Retry Policy

1. `AUTH_FAILURE`:
- run `epo auth check -f json -q`
- refresh credential source
- retry once

2. `RATE_LIMITED`:
- honor `Retry-After` when present
- otherwise exponential backoff with cap
- abort after bounded attempts

3. `SERVER_ERROR`:
- retry idempotent reads with jittered exponential backoff
- do not retry validation errors

4. `NOT_FOUND`:
- stop retrying
- report exact reference and endpoint

## Safe Agent Behavior

1. Keep retries bounded and explicit in output.
2. Log the exact command for each failed call.
3. For batch mode, preserve per-item errors in the final summary.
