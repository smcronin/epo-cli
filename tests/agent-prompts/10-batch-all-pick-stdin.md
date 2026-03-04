# Prompt 10: Agent Ergonomics and Batch Paths

## The Prompt

> I want to stress-test agent-scale workflows in EPO CLI:
>
> 1) Run a published-data search with auto-pagination behavior where applicable.
> 2) Use field projection so output is concise and agent-friendly.
> 3) Use stdin-driven batching for multiple inputs (queries or references).
> 4) If you hit an endpoint edge case, use `epo raw` as fallback and explain why.
>
> Keep this as a tooling UAT run: show what helped and what hurt agent productivity.

## What This Tests

- `pub search --all`
- `--pick`
- `--stdin` batch behavior
- `register search --all` (optional extension)
- `raw get/post` escape-hatch usage

## Pass Criteria

- Agent exercises at least one `--all` flow
- Agent uses `--pick` and stdin batching
- Agent explains fallback usage if `raw` is used
- Final output focuses on CLI UX feedback, not business conclusions

