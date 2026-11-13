# Changelog

All notable changes to this project will be documented in this file.

## [0.4.0] - 2026-11-13

### Added
- An `--all` run that fails after at least one page (`epo pub search --all`, `epo pub search --summary --all`, `epo pub recent --all`, `epo register search --all`) no longer throws away what it fetched. The error envelope keeps `ok:false`, the same `error` object and the same exit code, and adds `partial:true`, `command`, `service`, `request`, `pagination` (`pagesFetched`, `returned`, `failedRange`, `resumeRange`), the rows fetched so far in `results` and a warning with the resume command. Rerun with `--range <resumeRange> --all` to fetch the rest. A first-page failure is unchanged. With `-f ndjson|csv|table` the fetched rows go to stdout and the error to stderr. In a `--stdin` batch the failed query's row carries `partial`, `pagination` and `results` next to `ok:false` and `error`

### Fixed
- `epo pub search --summary` and `epo pub recent --summary` no longer exit 0 with a fake `ok:true, total:0` row when the query fails; the error is returned
- Flat, summary and `--all` modes of `pub`, `family`, `legal`, `number` and `register` return the original typed error, so exit codes 3 (auth), 4 (not found) and 5 (rate limited) are reachable instead of `GENERAL_ERROR` / exit 1
- `epo update` extracts the `epo_<os>_<arch>[.exe]` binary that release archives actually contain
- `epo pub search --all` stops at the OPS ceiling of 2000 results per query (with a warning) instead of requesting past it
- `epo register search --all` stops on the reported total, so a result count that is an exact multiple of the page size no longer fails; `epo register search` now reports `pagination.total` / `hasMore`
- A mistyped subcommand (`epo pub bogus`) is a usage error with the JSON error envelope instead of help text and exit 0; unknown flags and wrong argument counts exit 2 instead of 1
- An unsupported `--format` is rejected before any OPS call or `--out` write
- A 429 whose `Retry-After` exceeds the remaining `--timeout` returns `RATE_LIMITED` (exit 5) with the wait in the hint instead of `context deadline exceeded`
- Quota warnings treat the OPS quota headers as bytes (about 450 MB per hour, 4 GB per week) instead of request counts, so ordinary responses no longer warn
- Register table/csv/ndjson rows keep publication and status fields when OPS returns `reg:publication-reference` or `reg:ep-patent-status` as arrays
- `--kind` is attached unless the reference already ends with that kind code (`CA2000000 --kind C` used to drop it)
- `epo register upp` and `epo status` no longer treat a kindless 7-digit EP publication number as an application number
- An OPS 403 quota or fair-use rejection (an `X-Rejection-Reason` header, or a "Fair Use" body) is reported as `RATE_LIMITED` (exit 5) instead of `AUTH_FAILURE` (exit 3). The hint quotes the reason and says credentials are not the problem, and the raw header value is exposed as `error.reason`. A 403 "Developer account is blocked" and any other 403 stay `AUTH_FAILURE` but no longer tell you to verify credentials; 401 is unchanged
- A 403 is retried only when OPS sends a `Retry-After` that fits the remaining `--timeout`
- An EPS 403 no longer says "verify client credentials" (EPS takes none)
- `epo config show`, `epo config set-creds` and `epo auth configure` no longer print the first and last three characters (and the length) of the client secret. The secret is shown as a fixed `********`; `secretConfigured` says whether one is stored. The client ID keeps its 3+3 mask and the JSON key names are unchanged
- `epo pub search --summary --all` warns when it stops at the OPS ceiling of 2000 results (it stopped silently)
- A `--pick` field that matches nothing adds a `warnings` entry instead of silently returning empty rows
- `--pick` on a `--stdin` batch with a failed input keeps the successful inputs' rows and each failed input's `input`/`query`/`ok`/`error` (it used to return one empty object per input)
- `--summary` with `--pick` / `--flat-pick` projects the `topResults` rows and keeps the summary counts (it used to return an empty `results` object)

### Changed
- **Breaking:** `--pick` keeps the shape of its input. A list result stays a `results` array when exactly one row matches (it used to collapse to a single object), and the same holds for each input's `results` in a `--stdin` batch. Single-record commands such as `number convert` and `auth check` still return an object. Consumers that read a single-hit pick as an object must read `results[0]`
- **Breaking:** `--pick` projects the envelope only when every field is an explicit envelope path (`results.x`, `pagination.x`, `quota.x`, `throttle.x`, `request.x`, `warnings`). Bare names such as `command`, `service`, `ok` or `pagination` now resolve against the result rows, so `epo methods --pick command,summary` returns every row instead of one envelope object
- `--summary` output (`epo pub search --summary`, `epo pub recent --summary`) gains `examined` (rows fetched), `sortedBy` and `complete` (`examined >= total`), and its envelope gains `pagination`. `topResults` is unchanged
- A client-side sort that covered only part of the matches now says so: a sorted `--summary` and the flat sorted page (`epo pub recent`, `epo pub search --sort ...` without `--all`) add a warning naming the rows fetched and the total. An unsorted `--summary` adds no warning; `complete: false` is the signal there. No extra OPS calls are made. Help text and examples for `pub recent` no longer imply a true most-recent list without `--all`
- `--all` without `--range` requests 100 results per page (the OPS maximum) instead of 25; pass `--range 1-25 --all` to keep the smaller pages
- One OAuth token is requested per process and reused across OPS requests
- `epo methods` describes the default cpc output accurately; skill docs now cover the `eps` and `update` command groups
- Removed unused code and merged duplicated request, envelope and batch-output blocks into shared helpers

### Removed
- The four EPO PDFs (OPS reference guide, OPS swagger quick help, OPS terms and conditions, EPS REST services; 9.7 MB) are no longer tracked. The README lists the official EPO pages under "Upstream references"; the reference guide full text and the EPS Markdown conversion stay in `docs/api-reference/`. Git history and the `v0.2.0` / `v0.3.0` tags still contain the files
- The unused `examples/` package (`auth.go`, `throttle.go`, `queries.sh`). It was not imported or built into the CLI, and its auth example retried a 403 as an expired token

### Testing
- Added offline tests that run command paths through a stub HTTP transport (errors, pagination, usage errors, token reuse) and a synthetic-archive test for `epo update`

## [0.3.0] - 2026-03-07

### Added
- New `epo pub recent` convenience command for date-windowed CPC/applicant/inventor searches
- Quota and throttle warnings on stderr when approaching OPS limits (suppressed by `--quiet`)
- Richer 404 error hints with kind code, docdb format, and `number normalize` guidance
- `NearLimit()` and `HasBlackService()` helpers in throttle package

### Changed
- Skill docs: added recipes section, CQL field codes table, number format quick reference, CQL vs Minesoft disambiguation, NOT_FOUND disambiguation checklist
- Maintainer-only release tooling added (kept local, not part of the repository)

## [0.2.0] - 2026-03-05

### Added
- New EPS command group for European Publication Server workflows:
  - `epo eps dates`
  - `epo eps patents`
  - `epo eps formats`
  - `epo eps fetch`
  - `epo eps bulk`
- New `internal/eps` client package for unauthenticated EPS REST calls.
- Bulk indexing and download workflow that writes:
  - `indexes/publication-dates.txt`
  - `indexes/patents/<date>.txt`
  - `documents/<date>/<patent>.<format>`
  - `manifest.json` summary
- EPS bulk download guide: `docs/guides/eps-bulk-download.md`.
- EPS reference conversion doc integrated into docs navigation.

### Changed
- Root CLI help text now documents both OPS and EPS support.
- Methods catalog (`epo methods`) now includes full EPS command contracts and examples.

### Fixed
- Avoided output-format flag collision by using `--doc-format` for EPS fetch/bulk payload format selection.

### Testing
- Added parser tests for EPS date/patent/format extraction.
- Added CLI tests for EPS date filtering and format/path validation.
- Ran full test suite (`go test ./...`) successfully.
- Verified live EPS requests and a real bulk pull into `.tmp/eps-bulk-live`.

## [0.1.0] - 2026-03-05

### Added
- Agent-focused search ergonomics: `epo pub search --enrich`, `--summary`, and `--flat-pick`.
- New flattened/summary modes:
  - `epo pub biblio --flat|--summary`
  - `epo family get --flat|--table`
  - `epo family summary`
  - `epo legal get --flat|--summary`
  - `epo register get --summary`
- New convenience commands:
  - `epo status <reference>` (combined legal + register timeline)
  - `epo number normalize <reference>`
  - `epo usage today`, `epo usage week`, `epo usage quota`
- CPC normalization flags: `--normalize` / `--parsed` for `cpc get/search/map`.
- Image workflow improvements:
  - `epo pub images fetch --link`
  - inquiry responses include `fetch_path` values.
- MIT `LICENSE` and contributor guidance (`CONTRIBUTING.md`).

### Changed
- Publication commands now support `--input-format auto` defaults where applicable.
- Claims/description retrieval now better handles epodoc references with embedded kind codes.
- `--pick` now supports array indexing and better envelope-level projection patterns.
- Usage table extraction now includes flattened metric/date rows when present.
- Raw command docs/examples now explicitly include Windows Git Bash `MSYS_NO_PATHCONV=1` guidance.
- Methods catalog updated for new commands/flags and examples.

### Fixed
- Register UX now gives clearer guidance when publication-style references are used for application-only endpoints.
- `register events` JSON output strips redundant `mixed.layout` artifacts.
- Claims/description 404 handling now includes a docdb-format retry hint.

### Testing
- Added deterministic unit coverage for new projection, parsing, routing, and normalization behavior.
- Ran full unit/integration test suite (`go test ./...`) successfully.
- Ran local CLI evaluator (`tools/eval`) successfully.
