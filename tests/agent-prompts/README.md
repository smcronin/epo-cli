# EPO CLI Agent UAT Prompts

10 prompts written as realistic analyst requests. The objective is **not** to complete
an end-user task perfectly; it is to stress-test CLI discoverability, coverage, and agent UX.

Use this with `tests/agent-prompts/eval_runner.py`, which runs each prompt through
`frix-headless` and captures structured agent feedback.

## Coverage Matrix

| Prompt | Primary Commands Exercised | Focus |
|--------|----------------------------|-------|
| 1 | `methods`, `pub search` | Command discovery + recent SAP results |
| 2 | `family get`, `pub biblio` | Family retrieval and cross-jurisdiction output |
| 3 | `register get`, `register events`, `register procedural-steps`, `register upp` | Register prosecution workflow |
| 4 | `legal get` | Legal status timeline extraction |
| 5 | `pub fulltext`, `pub claims`, `pub description` | Fulltext inquiry + retrieval flow |
| 6 | `pub images inquiry`, `pub images fetch` | Images inquiry/retrieval and binary handling |
| 7 | `cpc search`, `cpc get`, `cpc map` | CPC exploration and mapping |
| 8 | `number convert` | Number format conversion paths |
| 9 | `usage stats` | Account usage statistics and interpretation |
| 10 | `pub search --all`, `--pick`, `--stdin`, `register search --all`, `raw get` | Agent-scale/batch ergonomics |

## Ground Rules

- Use `/EPO` skill in every prompt run.
- Use **only** the `epo` CLI for patent data retrieval.
- Do **not** use MineSoft tools or any other patent API/tool.
- Treat this as CLI UAT: evaluate command choices, errors, and UX quality.

## Running

```bash
export FRIX_ROOT=/path/to/frix-agent
python tests/agent-prompts/eval_runner.py
python tests/agent-prompts/eval_runner.py --prompts 1,3,10
python tests/agent-prompts/eval_runner.py --dry-run
# or explicitly:
python tests/agent-prompts/eval_runner.py --frix-root /path/to/frix-agent
```

Output:

- `tests/agent-prompts/results/prompt01.json` ... `prompt10.json`
- `tests/agent-prompts/results/eval_summary.json`
- `tests/agent-prompts/results/logs/prompt01.log` ... `prompt10.log`

Notes:

- `results/` and `workspaces/` are intentionally git-ignored local artifacts.
