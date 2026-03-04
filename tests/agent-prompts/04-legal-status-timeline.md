# Prompt 4: Legal Status Timeline

## The Prompt

> I need a legal-status timeline for EP1000000.A1. Pull legal events from EPO,
> sort them into a chronological narrative, and call out any status transitions
> that would matter for diligence (e.g., lapses, grants, oppositions, fee events).

## What This Tests

- `legal get`
- Timeline extraction from raw legal-event structures
- Agent interpretation quality for legal lifecycle signals

## Pass Criteria

- Agent retrieves legal events via `epo`
- Timeline is chronological and readable
- Important legal transitions are explicitly flagged

