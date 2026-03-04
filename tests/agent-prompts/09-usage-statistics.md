# Prompt 9: Usage and Quota Monitoring

## The Prompt

> I need an OPS usage snapshot for operational monitoring. Pull usage statistics
> for a recent date range, summarize message volume and any notable patterns,
> and tell me what this implies for safe automation throughput.

## What This Tests

- `usage stats --from/--to` (or `--date`)
- Interpretation of usage payload for operational decisions
- Agent handling of account-specific/credential-specific responses

## Pass Criteria

- Agent runs usage stats command(s)
- Summary includes operational implications
- Any access limitations are clearly described

