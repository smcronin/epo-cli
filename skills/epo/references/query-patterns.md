# Query and Identifier Patterns

## OPS Number Formats

- `docdb`: `CC.number.KC[.date]` (example: `EP.1000000.A1`)
- `epodoc`: compact EPO format (example: `EP1000000A1` or `EP1000000`)
- `original`: domestic input format (used mainly for number conversion)

Use number conversion when source numbers are ambiguous.

## Published-Data CQL Notes

Use CQL in `published-data/search` with explicit indexes where possible:
- `pa=IBM`
- `cpc=H04W`
- `pd within "20240101 20241231"`

Prefer explicit ranges over open-ended search for reproducibility.

## Register Search Note

Register CQL identifiers differ from published-data CQL.
Do not assume published-data index aliases are valid for register.

## Pagination/Range Guidance

- Default range is usually 1-25.
- Typical max page size is 100.
- Use bounded loops with stop conditions (`hasMore` / returned count).

## URL Encoding

Encode query special characters:
- `=` -> `%3D`
- `:` -> `%3A`
- space -> `%20`
- `,` -> `%2C`
