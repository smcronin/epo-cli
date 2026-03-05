# EPS Bulk Download Guide

EPS base URL: `https://data.epo.org/publication-server/rest/v1.2`

Unlike OPS endpoints, EPS publication-server endpoints are publicly accessible and do not require OAuth credentials.

## Commands

### 1. List available publication dates

```bash
epo eps dates --limit 10 -f json -q
```

Optional filters:

- `--from-date YYYYMMDD`
- `--to-date YYYYMMDD`
- `--order asc|desc`

### 2. List patents for a publication date

```bash
epo eps patents 20260225 --limit 50 -f json -q
```

### 3. Inspect formats for a patent

```bash
epo eps formats EP1004359NWB1 -f json -q
```

Typical formats: `xml`, `html`, `pdf`, `zip`.

### 4. Download one document

```bash
epo eps fetch EP1004359NWB1 --format zip --out .tmp/eps-bulk/sample/EP1004359NWB1.zip -f json -q
```

## Bulk Workflow

`epo eps bulk` creates indexes plus optional document downloads.

```bash
epo eps bulk \
  --max-dates 2 \
  --max-patents 200 \
  --format zip \
  --out-dir .tmp/eps-bulk \
  -f json -q
```

Useful flags:

- `--date YYYYMMDD` (single date shortcut)
- `--from-date YYYYMMDD`, `--to-date YYYYMMDD`
- `--max-dates N` and `--max-patents N` for controlled batches
- `--concurrency N` for parallel downloads
- `--skip-existing` to resume safely
- `--index-only` to build date/patent indexes without downloading
- `--dry-run` to preview queue sizes

## Output Layout

Default output root: `.tmp/eps-bulk` (gitignored in this repository).

Generated files:

- `indexes/publication-dates.txt`
- `indexes/patents/<date>.txt`
- `documents/<date>/<patent>.<format>`
- `manifest.json` (run summary and parameters)

## Fair-Use Constraint

EPS fair use is IP-based: **10GB in any sliding 7-day window**.

Practical recommendation:

1. Run bounded batches (`--max-dates`, `--max-patents`).
2. Track `bytesDownloaded` in `manifest.json`.
3. Pause when near 10GB and resume after the window rolls forward.
