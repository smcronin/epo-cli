# Query and Identifier Patterns

## Identifier Formats

1. `docdb`: `CC.number.kind` (example: `EP.1000000.A1`)
2. `epodoc`: compact (example: `EP1000000A1` or `EP1000000`)
3. `original`: domestic form for number conversion inputs

Use:
```bash
epo number normalize <reference> -f json -q
```
when mixed sources produce ambiguous IDs.

## Service-Specific Reference Rules

1. `pub` commands accept `--input-format auto`; docdb works best for stable retrieval.
2. `pub claims`/`pub description` accept kind-appended epodoc and auto-route to docdb when needed.
3. `register events` and `register procedural-steps` require application epodoc (example: `EP99203729`).
4. `legal get` for publication timelines is most reliable with:
```bash
--ref-type publication --input-format docdb
```

## CQL Patterns

Prefer explicit date windows:
```cql
pd within "YYYYMMDD YYYYMMDD"
```
Example:
```bash
epo pub search --query 'applicant="SAP SE" and pd within "20250101 20260305"' -f json -q
```

Avoid `pd>=YYYYMMDD`; the CLI validates and rejects this pattern.

## Pagination and Batch

1. Use `--all` only after validating shape and quota.
2. Use `--sort pub-date-desc` with `--all` for global chronological output.
3. Use stdin mode for repeatable batch jobs:
```bash
cat refs.txt | epo pub biblio --stdin --flat -f json -q
```

## Projection Patterns

1. Use `--pick` for compact answers.
2. Array indexing is supported with bracket form:
```text
results.environments[0].dimensions[0].metrics
```
