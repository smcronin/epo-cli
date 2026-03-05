---
name: epo
description: "Use this skill when working with EPO Open Patent Services (OPS) data through the local `epo` CLI: authentication, published-data retrieval/search, family analysis, register/legal lookups, CPC classification queries, number-format conversion, usage/quota checks, and agent-oriented JSON extraction workflows. Use it for patent research tasks that require reproducible terminal commands and structured outputs."
---

# EPO OPS CLI

Use this skill to run patent research and CLI QA against the local `epo` binary with reproducible commands and structured output.

## Quick Start

1. Configure credentials:
- `epo config set-creds --from-env`
- `epo config set-creds --from-dotenv .env`
- or `epo auth configure --client-id ... --client-secret ...`

2. Validate auth and inspect config:
```bash
epo config show -f json -q
epo auth check -f json -q
```

3. Use machine-readable output for all agent flows:
```bash
epo <command> -f json -q
```

4. Use the command catalog as source of truth:
```bash
epo methods --json -f json -q
```

## Workflow

1. Classify task:
- publication search/retrieval
- family/legal/register timeline
- CPC/number conversion
- usage/quota diagnostics

2. Pick the smallest command that answers the question.

3. Start narrow, then fan out:
- search with bounded range or precise date window
- confirm schema with one item
- use `--all` only after confirming throttle headroom

4. Use agent-friendly projections:
- `--summary`, `--flat`, `--pick`, and stdin batch mode where useful

5. Handle throttle/errors with bounded retries.

6. Return concise evidence:
- key fields
- exact command used
- explicit unresolved gaps if endpoint fails

## Research Defaults

- Prefer `-f json -q`.
- Prefer `pub search --summary` for quick answerability.
- Prefer `family summary` before `family get` for triage.
- Prefer `legal get --flat` and `register get --summary` for diligence timelines.
- Use `status` to merge legal + register + procedural data in one response.
- Use `usage quota` before large fan-out runs.

## Reference Files

- Command routing: `references/commands.md`
- End-to-end workflows: `references/workflows.md`
- Query/identifier patterns: `references/query-patterns.md`
- Error and throttle handling: `references/errors-throttling.md`
