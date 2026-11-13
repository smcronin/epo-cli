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

## Number Format Quick Reference

| You have          | Format   | Example           | Notes                                    |
|-------------------|----------|-------------------|------------------------------------------|
| CC.number.kind    | docdb    | EP.1000000.A1     | Most reliable for pub/family/legal        |
| CCnumberKind      | epodoc   | EP1000000A1       | Compact; CLI auto-converts for most cmds  |
| CC number only    | epodoc   | EP1000000         | Works for biblio/family; not claims/desc  |
| App number        | epodoc   | EP99203729        | Required for register events/proc-steps   |
| Domestic format   | original | EP(99)203729      | Use number normalize first                |

**Rule of thumb:** paste whatever you have — the CLI auto-detects and converts for `pub`, `family`, and `legal` commands. If you get a 404, run `epo number normalize <ref>` to inspect.

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

## CQL Field Codes (EPO OPS)

| Field     | Code    | Example                                    |
|-----------|---------|-------------------------------------------|
| Title     | ti      | ti="machine learning"                      |
| Abstract  | ab      | ab="neural network"                        |
| Applicant | pa      | pa="Google LLC"                            |
| Inventor  | in      | in="Smith John"                            |
| Pub date  | pd      | pd within "20250101 20260307"              |
| CPC       | cpc     | cpc=G06N                                   |
| IPC       | ipc     | ipc=H04L                                   |
| Pub number| pn      | pn=EP1000000                               |

**CQL vs Minesoft differences:**
- CQL uses `=` for contains, Minesoft uses `:`
- CQL date ranges: `pd within "YYYYMMDD YYYYMMDD"` — Minesoft uses `[YYYY-MM-DD TO YYYY-MM-DD]`
- CQL applicant field: `pa` — Minesoft: `PA` or `APPLICANT`
- CQL has no AI semantic search — use Minesoft for concept-level queries
- CQL wildcards: `*` and `?` — same as Minesoft

## Pagination and Batch

1. Use `--all` only after validating shape and quota.
2. Use `--sort pub-date-desc` with `--all` for global chronological output. `--sort` is client-side: without `--all` it orders one page (25 rows), and `--summary` / `pub recent` add a warning plus `results.complete: false` when more results matched than were examined. Above 2000 matches even `--all` is partial; narrow the `pd` window.
3. If `--all` fails mid-run the error envelope has `partial: true`, the rows fetched so far and `pagination.resumeRange`; rerun with `--range <resumeRange> --all` instead of starting over (see `errors-throttling.md`).
4. Use stdin mode for repeatable batch jobs:
```bash
cat refs.txt | epo pub biblio --stdin --flat -f json -q
```

## Projection Patterns

1. Use `--pick` for compact answers. Bare names select fields from the result rows:
```bash
epo pub search --query 'pa="SAP SE"' --flat --pick reference,title,pubDate -f json -q
```
2. The output keeps its shape. A list command returns a `results` array even when there is exactly one hit, so always read `results[0]`; a single-record command (for example `number convert`, `auth check`) returns an object.
3. A pick made only of envelope paths (`results.x`, `pagination.x`, `quota.x`, `throttle.x`, `request.x`, `warnings`) returns one object keyed by those paths. Do not mix envelope paths with row fields; `ok`, `command`, `service`, `version`, `pagination` and `quota` are printed in the envelope anyway.
4. Array indexing is supported with bracket form:
```text
results.environments[0].dimensions[0].metrics
```
5. A pick field that matches nothing adds a `warnings` entry naming the field. Check `warnings` before concluding that a field is empty.
6. With `--summary`, row fields apply to `topResults` and the summary counts are kept.
7. In `--stdin` batches each input keeps `input`/`query`/`ok`, failed inputs keep their `error`, and the pick applies to each input's `results`.
