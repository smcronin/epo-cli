# EPO CLI Command Routing

Use `epo methods --json -f json -q` as the authoritative contract for command/flag support.

## Preflight

1. Show credential status:
```bash
epo config show -f json -q
```
2. Check quota before broad jobs:
```bash
epo usage quota -f json -q
```

## Intent to Command

1. "Find recent patents / quick shortlist":
```bash
epo pub search --query 'applicant="SAP SE" and pd within "20250101 20260305"' --summary -f json -q
```

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
