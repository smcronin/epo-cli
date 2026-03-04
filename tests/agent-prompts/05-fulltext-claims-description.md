# Prompt 5: Fulltext Availability and Retrieval

## The Prompt

> For EP1000000, check what fulltext is available first, then retrieve claims and
> description content using the correct sequence. I need a short note on coverage
> and any retrieval caveats you encountered.

## What This Tests

- `pub fulltext` inquiry-first workflow
- `pub claims`
- `pub description`
- Agent handling of fulltext availability constraints

## Pass Criteria

- Agent performs inquiry before retrieval
- Claims/description retrieval attempts are explicit
- Caveats/errors are explained clearly

