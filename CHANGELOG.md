# Changelog

All notable changes to this project will be documented in this file.

## [0.3.0] - 2026-03-07

### Added
- New `epo pub recent` convenience command for date-windowed CPC/applicant/inventor searches
- Quota and throttle warnings on stderr when approaching OPS limits (suppressed by `--quiet`)
- Richer 404 error hints with kind code, docdb format, and `number normalize` guidance
- `NearLimit()` and `HasBlackService()` helpers in throttle package

### Changed
- Skill docs: added recipes section, CQL field codes table, number format quick reference, CQL vs Minesoft disambiguation, NOT_FOUND disambiguation checklist
- Release skill and `tools/release.ps1` workflow added

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
