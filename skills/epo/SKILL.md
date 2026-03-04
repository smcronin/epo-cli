---
name: epo
description: "Use this skill when working with EPO Open Patent Services (OPS) data through the local `epo` CLI: authentication, published-data retrieval/search, family analysis, register/legal lookups, CPC classification queries, number-format conversion, usage/quota checks, and agent-oriented JSON extraction workflows. Use it for patent research tasks that require reproducible terminal commands and structured outputs."
---

# EPO OPS CLI

This skill helps an agent use and evolve the `epo` CLI in this repository for OPS v3.2 data access.

Current implementation note:
- The repository currently implements Phase 0 auth commands (`epo auth configure|token|check`).
- References in this skill describe both current commands and the target command surface from `docs/PROJECT_PLAN.md`.

## Quick Start

1. Ensure credentials are available via either:
- env vars: `EPO_CLIENT_ID` + `EPO_CLIENT_SECRET`
- env vars: `CONSUMER_KEY` + `CONSUMER_SECRET_KEY`
- stored config: `epo auth configure --client-id ... --client-secret ...`

2. Validate auth:
```bash
epo auth check -f json -q
```

3. Fetch a token when needed:
```bash
epo auth token --raw
```

4. Prefer machine output for agent workflows:
```bash
epo <command> -f json -q --minify
```

## Workflow

1. Identify task type
- Auth/setup
- Search/retrieval
- Family/legal/register analysis
- Classification/number conversion
- Usage/quota diagnostics

2. Choose command family
- See `references/commands.md`.

3. Build minimal query first
- Validate with a narrow request before broad pagination/bulk retrieval.
- Use deterministic identifiers (`docdb`/`epodoc`) and explicit date bounds.

4. Execute with JSON output
- Use `-f json -q`.
- For token-sensitive flows, avoid printing full secrets.

5. Handle throttling and errors
- Respect `Retry-After`.
- Parse `X-Throttling-Control`.
- See `references/errors-throttling.md`.

6. Summarize and cite
- Return key fields only, with the exact command used.

## Command Strategy

- Start from typed commands when available.
- If a planned command is not implemented yet, use this order:
1. Report the missing command plainly.
2. Use closest implemented command.
3. If needed, implement missing command in the CLI instead of inventing an external script.

## Reference Files

- Command map: `references/commands.md`
- Workflow patterns: `references/workflows.md`
- Query and number format patterns: `references/query-patterns.md`
- Error/throttle handling: `references/errors-throttling.md`

## Defaults

- Use `-f json -q` for agent calls.
- Use `--timeout` for long calls.
- Avoid broad fan-out requests until throttle state is known.
