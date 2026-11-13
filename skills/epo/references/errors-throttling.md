# Errors and Throttling

## Headers to Inspect

1. `X-Throttling-Control`
2. `Retry-After`
3. `X-IndividualQuotaPerHour-Used` (bytes; about 450 MB per rolling hour)
4. `X-RegisteredQuotaPerWeek-Used` (bytes; 4 GB per calendar week on the free tier)
5. `X-Rejection-Reason` (sent with a 403 when a quota is exhausted, for example `RegisteredQuotaPerWeek`; surfaced as `error.reason`)

## CLI Exit Codes

1. `0`: success
2. `1`: general error
3. `2`: usage/validation error
4. `3`: auth failure (401, a blocked developer account, or a 403 with no quota reason)
5. `4`: not found
6. `5`: rate limited (429, or a 403 quota / fair-use rejection)
7. `6`: server error

## Error Envelope

```json
{
  "ok": false,
  "error": {
    "code": 429,
    "type": "RATE_LIMITED",
    "message": "...",
    "hint": "..."
  },
  "version": "v0.1.0"
}
```

`error.reason` is added when OPS sent an `X-Rejection-Reason` header. It is the raw OPS value; treat it as an opaque string.

## HTTP 403 Is Usually Not a Credentials Problem

OPS answers 403 for quota and fair-use rejections, blocked accounts, and blacklisting. Bad credentials fail earlier, at the token request.

| 403 response | `error.type` | Exit | What to do |
|---|---|---|---|
| `X-Rejection-Reason` header present, or body mentions "Fair Use" | `RATE_LIMITED` | 5 | Stop. Credentials are fine. The hourly quota frees up on a rolling one-hour window; the weekly quota resets at the start of the calendar week. Read `error.reason` and the hint. |
| Body "Developer account is blocked" | `AUTH_FAILURE` | 3 | The account is blocked by EPO. Re-entering credentials will not help; the owner must contact EPO support. |
| Any other 403 (for example "This request has been rejected") | `AUTH_FAILURE` | 3 | Read the response body in `error.message`. Do not assume the credentials are wrong; do not loop on re-entering them. |

The CLI retries a 403 only when OPS sends a `Retry-After` that fits inside `--timeout`. A 401 still means the credentials or token were not accepted.

`epo eps` commands take no credentials, so an EPS 403 is never a credentials problem; EPS fair use is IP-based.

## Retry Policy

1. `AUTH_FAILURE`:
- read the hint first: a blocked account or an unexplained 403 is not fixed by re-entering credentials
- for a 401 or missing credentials: refresh credential source, then retry the original command once

2. `RATE_LIMITED` with `error.code` 403 (quota or fair use):
- do not retry in a loop and do not touch credentials
- report `error.reason` and stop until the quota resets

3. `RATE_LIMITED` with `error.code` 429:
- honor `Retry-After` when present
- otherwise exponential backoff with cap
- abort after bounded attempts

4. `SERVER_ERROR`:
- retry idempotent reads with jittered exponential backoff
- do not retry validation errors

5. `NOT_FOUND`:
- stop retrying
- report exact reference and endpoint
- **Disambiguation checklist:**
  - Missing kind code? Try A1, A2, B1, B2
  - Wrong format? Run `epo number normalize <ref>`
  - Endpoint mismatch? `register events` needs application epodoc, not publication
  - Patent genuinely doesn't exist in OPS? Check via `epo pub search --query 'pn=<number>'`

## Safe Agent Behavior

1. Keep retries bounded and explicit in output.
2. Log the exact command for each failed call.
3. For batch mode, preserve per-item errors in the final summary.
