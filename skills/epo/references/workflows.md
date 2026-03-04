# EPO Workflow Patterns

## 1. Authentication Bootstrap

1. Configure credentials:
```bash
epo auth configure --client-id "$EPO_CLIENT_ID" --client-secret "$EPO_CLIENT_SECRET"
```

2. Validate:
```bash
epo auth check -f json -q
```

3. Optionally request raw token:
```bash
epo auth token --raw
```

## 2. Research Session Pattern (planned command surface)

1. Start with narrow retrieval/search request.
2. Confirm response schema and paging fields.
3. Expand with `--all` only if needed.
4. Watch throttle headers before high-volume fetch.

## 3. Family + Legal + Register Investigation (planned)

1. Resolve canonical identifier.
2. Pull family members.
3. Pull legal events for focal members.
4. Pull register events/procedural steps for EP coverage.
5. Produce timeline summary.

## 4. Classification-Led Search Expansion (planned)

1. Start from a CPC symbol.
2. Expand hierarchy by depth and ancestors.
3. Run published-data query constrained by CPC + dates.
4. De-duplicate results and summarize key applicants.

## 5. Agent Output Policy

- Always prefer JSON in automation:
```bash
epo <command> -f json -q --minify
```
- Preserve the exact command used in the report.
- Return concise summaries with evidence fields, not full payload dumps.
