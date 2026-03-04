# Prompt 7: CPC Classification Workflow

## The Prompt

> Please run a CPC workflow for telecom/networking classes:
> 1) Search CPC for a networking keyword,
> 2) inspect one returned symbol in detail,
> 3) map that symbol to another scheme.
> Summarize what each step contributed and where the outputs were strongest/weakest.

## What This Tests

- `cpc search`
- `cpc get`
- `cpc map`
- Agent reasoning over XML-forward classification outputs

## Pass Criteria

- Agent executes all three CPC steps
- Output linkage between steps is coherent
- Limitations/ambiguities are documented

