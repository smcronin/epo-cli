# EPO CLI Command Map

This file maps task intent to command families.

Current implementation in this repo:
- `auth configure`
- `auth token`
- `auth check`

Target command surface (from `docs/PROJECT_PLAN.md`):
- `usage stats`
- `pub biblio|abstract|fulltext|claims|description|images|equivalents|search`
- `family get`
- `legal get`
- `register get|events|procedural-steps|upp|search`
- `number convert`
- `cpc get|search|media|map`
- `raw get|post`
- `methods --json`

## Intent Routing

1. "Set up credentials"
- `epo auth configure --client-id ... --client-secret ...`

2. "Check auth works"
- `epo auth check -f json -q`

3. "Get a bearer token"
- `epo auth token --raw`

4. "Find how to call X command"
- `epo methods --json` (planned)

5. "Search publications by query"
- `epo pub search ...` (planned)

6. "Retrieve biblio/claims/description/images"
- `epo pub <endpoint> ...` (planned)

7. "Get INPADOC family/legal/register data"
- `epo family get ...`
- `epo legal get ...`
- `epo register ...` (all planned)

8. "Convert number formats"
- `epo number convert ...` (planned)

9. "Classifications and mapping"
- `epo cpc get|search|map ...` (planned)

10. "Unsupported edge request"
- `epo raw get|post ...` (planned fallback)
