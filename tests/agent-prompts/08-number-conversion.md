# Prompt 8: Number Conversion Robustness

## The Prompt

> I have mixed patent-number formats and need normalization.
> Use EP1000000.A1 as a seed and demonstrate conversions across supported formats.
> Show me where conversion works cleanly and where format constraints show up.

## What This Tests

- `number convert` across format combinations
- Agent understanding of ref-type/from-format/to-format constraints
- Error handling on unsupported conversion paths

## Pass Criteria

- Agent performs multiple conversion attempts
- Agent distinguishes valid vs invalid combinations
- Final output provides practical normalization guidance

