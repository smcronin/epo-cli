# epo-cli

Agent-ready CLI for EPO Open Patent Services (OPS) and European Publication Server (EPS) APIs.

## Install

```bash
# From source
go install github.com/smcronin/epo-cli/cmd/epo@latest

# Or download release binaries:
# https://github.com/smcronin/epo-cli/releases
```

## Authentication

Get credentials at https://developers.epo.org and configure once:

```bash
export EPO_CLIENT_ID=your_consumer_key
export EPO_CLIENT_SECRET=your_consumer_secret

epo config set-creds --from-env
epo auth check -f json -q
```

## Quick Start

```bash
# Search with biblio-enriched flat output
epo pub search --query "applicant=\"SAP SE\" and pd within \"20250101 20260305\"" --all --sort pub-date-desc --flat --enrich -f json -q

# Agent summary output
epo pub search --query "applicant=IBM" --summary --flat-pick "reference,title,pubDate" -f json -q

# Biblio + family + legal
epo pub biblio EP.1000000.A1 --input-format auto --flat -f json -q
epo family summary EP.1000000.A1 -f json -q
epo legal get EP.1000000.A1 --flat -f json -q

# Register and combined status timeline
epo register get EP99203729 --summary -f json -q
epo status EP.1000000.A1 --register-ref EP99203729 -f json -q

# Number normalization
epo number normalize EP1000000A1 -f json -q

# Usage shortcuts
epo usage today -f json -q
epo usage week -f json -q
epo usage quota -f json -q

# EPS publication feed + bulk download
epo eps dates --limit 5 -f json -q
epo eps patents 20260225 --limit 20 -f json -q
epo eps formats EP1004359NWB1 -f json -q
epo eps fetch EP1004359NWB1 --doc-format zip --out .tmp/eps-bulk/sample/EP1004359NWB1.zip -f json -q
epo eps bulk --max-dates 1 --max-patents 25 --doc-format zip --out-dir .tmp/eps-bulk -f json -q

# CPC structured parsing
epo cpc search --q "network routing" --normalize -f json -q

# Images inquiry -> fetch using raw @link
epo pub images inquiry EP1000000.A1 -f json -q
epo pub images fetch "published-data/images/EP/1000000/A1/fullimage" --link --range 1 --accept application/pdf --out page1.pdf -f json -q

# Windows Git Bash/MSYS raw path usage
MSYS_NO_PATHCONV=1 epo raw get "/published-data/publication/docdb/EP.1000000.A1/claims" -f json -q
```

## Command Groups

- `epo pub` - published-data operations (biblio/abstract/fulltext/claims/description/search/images/equivalents)
- `epo family` - INPADOC family retrieval and summary
- `epo number` - number format conversion/normalization
- `epo register` - EP register dossier/events/procedural/UPP/search
- `epo legal` - legal status events
- `epo status` - combined timeline helper
- `epo cpc` - CPC retrieval/search/map/media
- `epo usage` - usage stats and quota shortcuts
- `epo eps` - EPS publication dates/patents/formats/raw document and bulk download workflows
- `epo raw` - direct OPS fallback requests
- `epo methods` - machine-readable command contract catalog

## Output and Agent Features

Global flags designed for automation:

- `-f, --format` (`json`, `ndjson`, `csv`, `table`)
- `--pick` projection with dot paths and array indexing
- `--stdin` batch mode for newline-delimited inputs
- `--all` auto-pagination where supported
- `--minify` compact JSON output
- `--timeout` request timeout in seconds

Stable JSON envelope:

```json
{
  "ok": true,
  "command": "epo pub search",
  "service": "published-data",
  "request": {},
  "pagination": {},
  "throttle": {},
  "quota": {},
  "results": [],
  "warnings": [],
  "version": "..."
}
```

## Release and Versioning

- Tag format: `v*` (for example `v0.1.0`)
- GitHub Actions release workflow cross-builds linux/darwin/windows archives
- Binary version is injected at build time via ldflags:
  `-X github.com/smcronin/epo-cli/internal/cli.version=<tag>`

## Development

```bash
go test ./...

# Optional live integration tests
$env:EPO_INTEGRATION="1"
$env:EPO_CLIENT_ID="..."
$env:EPO_CLIENT_SECRET="..."
go test ./tests/integration -v -count=1 -timeout 600s

# Local smoke evaluator
go run ./tools/eval --json-out .tmp/eval/report.json
```

## Docs

- [Changelog](CHANGELOG.md)
- [Contributing](CONTRIBUTING.md)
- [Authentication guide](docs/guides/authentication.md)
- [CQL search syntax](docs/guides/cql-search.md)
- [Number formats](docs/guides/number-formats.md)
- [Rate limits](docs/guides/rate-limits.md)
- [OPS service reference](docs/api-reference/services.md)
- [EPS bulk download guide](docs/guides/eps-bulk-download.md)
- [EPS REST services (official PDF converted)](docs/api-reference/eps-rest-services.md)

## License

MIT
