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

## Partial `--all` Failure

When `pub search --all`, `pub search --summary --all`, `pub recent --all` or `register search --all` fails after at least one page, the error envelope also carries what was fetched. `ok` stays `false` and the exit code is the error's own, so a check on `ok` or the exit code behaves as before.

```json
{
  "ok": false,
  "partial": true,
  "error": { "code": 429, "type": "RATE_LIMITED", "message": "...", "hint": "..." },
  "command": "epo pub search",
  "service": "published-data",
  "request": { "method": "GET", "path": "/published-data/search/biblio", "query": { "q": "applicant=IBM" }, "all": true },
  "pagination": { "offset": 1, "limit": 100, "total": 5000, "pagesFetched": 7, "returned": 700, "failedRange": "701-800", "resumeRange": "701-800" },
  "results": ["...700 rows..."],
  "warnings": ["--all stopped after 7 pages (700 of 2000 retrievable results); the rows fetched so far are included. Resume with --range 701-800 --all."],
  "version": "..."
}
```

What to do:
- keep the rows in `results`; do not rerun from the start
- decide from `error.type` (and `error.reason` for a 403 quota rejection) whether to wait or stop
- resume with the same command plus `--range <pagination.resumeRange> --all`, then append the new rows
- with `--summary` the partial `results` is a summary of the rows examined so far (`complete: false`)
- with `-f ndjson|csv|table` the fetched rows are on stdout and the error, hint and resume warning on stderr
- a failure on the first page has no `partial` key and no rows
- in a `--stdin` batch the envelope is `ok: true` as usual and the failed query's row has `ok: false`, `error`, `partial: true`, `pagination` and its `results`

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
