# Prompt 3: Register Prosecution Check

## The Prompt

> Please review EP register activity for EP99203729. I want a compact
> prosecution view: core dossier metadata, events, procedural steps, and whether
> any UPP data is present. Highlight what looks most recent and operationally important.

## What This Tests

- `register get`
- `register events`
- `register procedural-steps`
- `register upp`
- Multi-command synthesis into one prosecution summary

## Pass Criteria

- Agent chains the register commands correctly
- Agent identifies recent events/steps
- Agent states clearly if UPP is present/absent

