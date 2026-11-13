# EPO CLI Command Routing

Use `epo methods --json -f json -q` as the authoritative contract for command/flag support.

## Preflight

1. Show credential status:
```bash
epo config show -f json -q
```
The client secret is never shown: `secretMasked` is a fixed `********` when a secret is stored. Use `configured` / `secretConfigured` to check status, and `clientIDMasked` (first and last three characters) to tell which key is in use.
2. Check quota before broad jobs:
```bash
epo usage quota -f json -q
```

## Intent to Command

1. "Find recent patents / quick shortlist":
```bash
epo pub search --query 'applicant="SAP SE" and pd within "20250101 20260305"' --summary -f json -q
epo pub recent --applicant "SAP SE" --days 7 --all --summary -f json -q
```
`--summary` returns `{query,total,examined,sortedBy,complete,topResults}`. `topResults` are taken from the `examined` rows only (one page of 25 unless `--all`), and `--sort` is applied client-side to those rows. When `complete` is `false` a warning says so; add `--all` (up to 2000 results) or narrow the date window for a true most-recent list.

2. "Fetch publication detail":
```bash
epo pub biblio EP.1000000.A1 --flat -f json -q
epo pub fulltext EP1000000 --pick results.suggested_retrieval_commands -f json -q
epo pub claims EP1000000A1 -f json -q
```

3. "Family and legal timeline":
```bash
epo family summary EP.1000000.A1 -f json -q
epo legal get EP.1000000.A1 --ref-type publication --input-format docdb --flat -f json -q
epo register get EP99203729 --summary -f json -q
epo status EP.1000000.A1 --register-ref EP99203729 -f json -q
```

4. "Register events/procedural steps":
```bash
epo register events EP99203729 -f json -q
epo register procedural-steps EP99203729 -f json -q
```
`register events` and `register procedural-steps` expect an application epodoc reference.

5. "Classification exploration":
```bash
epo cpc search --q "network routing" --normalize -f json -q
epo cpc get H04L45/00 --depth 1 --normalize -f json -q
epo cpc map H04L45/00 --from cpc --to ipc --normalize -f json -q
```

6. "Number normalization / mixed input cleanup":
```bash
epo number normalize EP1000000A1 -f json -q
```

7. "Batch mode":
```bash
@'
EP1000000A1
EP4703890A1
'@ | epo pub biblio --stdin --flat -f json -q
```

8. "Raw fallback":
```bash
epo raw get "/published-data/search" --query q=applicant=IBM -f json -q
```

9. "EP publication files / bulk download" (European Publication Server, no OPS credentials needed):
```bash
epo eps dates --limit 5 -f json -q
epo eps patents 20260225 --limit 20 -f json -q
epo eps formats EP1004359NWB1 -f json -q
epo eps fetch EP1004359NWB1 --doc-format zip --out .tmp/eps-bulk/sample/EP1004359NWB1.zip -f json -q
epo eps bulk --max-dates 1 --max-patents 25 --doc-format zip --out-dir .tmp/eps-bulk -f json -q
```
`eps` patent numbers use the publication-server form (country + number + correction + kind, for example `EP1004359NWB1`). `eps bulk --dry-run` or `--index-only` builds the indexes without downloading documents.

10. "Check for or install a newer CLI":
```bash
epo update --check -f json -q
epo update -f json -q
```
`epo update` downloads the release for this OS/architecture from GitHub and verifies its checksum; on Windows the replacement is applied after the process exits.
