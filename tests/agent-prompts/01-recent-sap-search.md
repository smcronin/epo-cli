# Prompt 1: Recent SAP Patent Search

## The Prompt

> I'm preparing a quick competitive update on SAP patenting activity.
> Please use the EPO data source to find recent SAP patent publications and
> give me a concise summary of the most recent results (publication reference,
> title if available, and jurisdiction spread). Keep it focused and structured.

## What This Tests

- Skill invocation and command discovery behavior (`/EPO`, `epo methods`)
- `pub search` with relevant applicant query construction
- Agent summarization quality from OPS-style payloads

## Pass Criteria

- Agent uses `epo` commands only
- Agent returns recent SAP results with structured fields
- Agent reports any query limitations clearly

