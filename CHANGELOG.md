# Changelog

All notable changes to this project will be documented in this file.

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
