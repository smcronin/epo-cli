# EPO Workflow Patterns

## 1. Preflight Session

1. Confirm auth and config:
```bash
epo config show -f json -q
epo auth check -f json -q
```
2. Confirm quota/throttle baseline:
```bash
epo usage quota -f json -q
epo usage week -f table -q
```

## 2. Recent Prior-Art Sweep

1. Start with bounded CQL:
```bash
epo pub search --query 'applicant="SAP SE" and pd within "20250101 20260305"' --summary -f json -q
```
2. Expand with flat rows for export:
```bash
epo pub search --query 'applicant="SAP SE" and pd within "20250101 20260305"' --all --sort pub-date-desc --flat -f json -q
```
3. Pull detailed records only for selected refs:
```bash
epo pub biblio EP4703890A1 --flat -f json -q
```

## 3. EP Prosecution and Legal Timeline

1. Get compact legal events:
```bash
epo legal get EP.1000000.A1 --ref-type publication --input-format docdb --flat -f json -q
```
2. Pull register summary with clean lapse/designation fields:
```bash
epo register get EP99203729 --summary -f json -q
```
3. Merge legal/register/procedural in one response:
```bash
epo status EP.1000000.A1 --register-ref EP99203729 -f json -q
```

## 4. Fulltext and Images Retrieval

1. Discover fulltext instances:
```bash
epo pub fulltext EP1000000 --pick results.suggested_retrieval_commands -f json -q
```
2. Retrieve claims/description:
```bash
epo pub claims EP1000000A1 -f json -q
epo pub description EP1000000A1 -f json -q
```
3. Fetch image PDF from inquiry link:
```bash
epo pub images inquiry EP1000000A1 -f json -q
epo pub images fetch "published-data/images/EP/1000000/A1/fullimage" --link --range 1 --accept application/pdf --out out/EP1000000-page1.pdf -f json -q
```

## 5. CPC-Led Expansion

1. Search CPC keyword:
```bash
epo cpc search --q "network routing" --normalize -f json -q
```
2. Inspect hierarchy:
```bash
epo cpc get H04L45/00 --depth 1 --normalize -f json -q
```
3. Map to IPC/ECLA:
```bash
epo cpc map H04L45/00 --from cpc --to ipc --normalize -f json -q
```

## 6. Batch Agent Pattern

1. Keep one item per line on stdin.
2. Prefer `--flat` + `--pick`/`--summary` when available.
3. Report per-item failures directly from output rows (do not drop failures silently).
