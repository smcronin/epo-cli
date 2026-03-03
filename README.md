# epo-cli

A Go CLI for the EPO Open Patent Services (OPS) API — the REST API that powers Espacenet.

## What is OPS?

OPS provides machine-to-machine access to the EPO's full patent data corpus (120M+ documents) via a RESTful API. It's free with registration (up to ~4GB/week).

- Free: register at https://developers.epo.org
- Base URL: `https://ops.epo.org/3.2/`
- Docs: [OPS Reference Guide v1.3.20 (June 2024)](docs/api-reference/ops-reference-guide-v1.3.20.md)

## Services

| Service | What it does |
|---------|-------------|
| `published-data` | Bibliographic data, fulltext, abstracts, images |
| `family` | INPADOC patent family trees (worldwide equivalents) |
| `number-service` | Convert between docdb / epodoc / original number formats |
| `register` | EP Register: prosecution history, procedural steps, events |
| `legal` | Legal status events for the full patent lifecycle |
| `classification/cpc` | CPC classification retrieval, search, and mapping |

## Installation

```bash
go install github.com/smcronin/epo-cli@latest
```

Or build from source:

```bash
git clone https://github.com/smcronin/epo-cli
cd epo-cli
go build -o epo-cli ./cmd/epo-cli
```

## Authentication

Get credentials at https://developers.epo.org → My Apps → Add App → select `OPS v3.2`

```bash
export EPO_CLIENT_ID=your_consumer_key
export EPO_CLIENT_SECRET=your_consumer_secret

# Or use a config file
epo-cli auth configure
```

See [Authentication Guide](docs/guides/authentication.md) for full details.

## Quick Examples

```bash
# Bibliographic data
epo-cli biblio EP1000000

# Patent family (INPADOC)
epo-cli family EP1000000

# Full text search
epo-cli search --applicant IBM --cpc H04W

# Claims
epo-cli claims EP1000000

# Legal status
epo-cli legal EP1000000

# Register history (EP only)
epo-cli register EP99203729

# Number format conversion
epo-cli number EP1000000 --to docdb
```

See [Examples](examples/) for real-world usage patterns.

## Docs

- [API Reference — Full OPS Guide](docs/api-reference/ops-reference-guide-v1.3.20.md)
- [Authentication](docs/guides/authentication.md)
- [Rate Limits & Throttling](docs/guides/rate-limits.md)
- [Number Formats](docs/guides/number-formats.md)
- [CQL Search Syntax](docs/guides/cql-search.md)
- [Services Reference](docs/api-reference/services.md)
