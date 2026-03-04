# Prompt 6: Images Inquiry and Fetch

## The Prompt

> I need drawing/first-page image artifacts for EP1000000.A1.
> Run an images inquiry first, choose one useful link from the response, fetch it,
> and save the binary output to disk. Tell me what format and size you got.

## What This Tests

- `pub images inquiry`
- `pub images fetch`
- Binary output handling (`--out`, format selection)

## Pass Criteria

- Agent performs inquiry first and uses returned link path
- Agent fetches at least one image/document artifact
- Output metadata (format/size) is reported

