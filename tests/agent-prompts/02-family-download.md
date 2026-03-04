# Prompt 2: Family Retrieval and Save

## The Prompt

> I need a downloadable family snapshot for publication EP1000000.A1.
> Pull the full INPADOC family details, include useful constituents, and save
> the raw structured output to a file so another analyst can inspect it later.
> Then give me a short family summary by country and family size.

## What This Tests

- `family get` with constituent options
- File-oriented workflow from CLI output
- Agent ability to move from raw payload to analyst summary

## Pass Criteria

- Agent retrieves family data with `epo`
- Agent creates a local artifact file from command output
- Agent summarizes country distribution/family breadth

