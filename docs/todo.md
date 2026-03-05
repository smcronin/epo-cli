# TODO: Agent Suggestions (Full Eval Run)

Source: `tests/agent-prompts/results/eval_summary.json`
Run timestamp: 2026-03-04T16:32:13.944324
Prompts: 10, Pass rate: 100%, Total command invocations: 132

## High-Priority Blockers

- [x] P04: register events and register procedural-steps reject publication-reference docdb format (EP.1000000.A1) with CLIENT.InvalidQuery — must use application epodoc (EP99203729) with no guidance from the error message
- [x] P05: epodoc format with kind code (EP1000000A1) returns 404 on claims/description endpoints while the same reference works for fulltext inquiry - forces agent to discover docdb dot-separated format through trial and error
- [x] P06: Link-path mismatch between inquiry @link output and fetch input causes 404 errors. The @link from inquiry includes 'published-data/images/' prefix which fetch also prepends, creating a doubled path. Agent must read source code or guess to strip the prefix. This is a significant agent-usability bug.

## Prompt-Level Suggestions

### P01

- [x] CRITICAL: Populate title and pubDate in --flat/--table output by auto-fetching biblio data or including it in the search response. Currently agents must make N additional API calls to get titles.
- [x] Make --input-format docdb the default, or auto-detect format — epodoc silently fails on valid patents.
- [x] Add a --enrich flag to pub search that automatically fetches biblio for each result and merges title/abstract/date into flat output.
- [x] Add CQL date syntax validation: detect pd>=YYYYMMDD and suggest pd within syntax before sending to API.
- [x] Consider adding a --pick option that works with flat mode (e.g., --pick title,pubDate,applicant) that triggers biblio fetches behind the scenes.
- [x] Add pub search examples showing the 'within' date syntax prominently in --help.
- [x] The batch --stdin mode should report per-item errors inline rather than silently skipping (EP.4565252.A1 returned ERROR with no detail).

### P02

- [x] Consider adding a --pick or --flatten option that normalizes family members into a consistent tabular structure (country, doc-number, kind, title, app-date, pub-date, legal-status) so agents and analysts don't need to handle structural variations.
- [x] A `--format table` output for family get would be extremely useful for quick human inspection without needing to parse 231KB of JSON.
- [x] Consider a `family summary` subcommand that returns a condensed country-count breakdown directly.
- [x] The help text could mention that some family members may lack exchange-document nodes (legal-only entries for national phase designations).

### P03

- [x] Add a `register get --constituents biblio` flag to return just biblio without the verbose term-of-grant change history, which ballooned the response to 53KB.
- [x] Consider a `--pick` shorthand or `--summary` mode that flattens the deeply nested XML-to-JSON structure into a compact prosecution summary (status, key dates, designated states, lapse list).
- [x] The register events output includes a mixed.layout array that mirrors the event count but adds no information - consider stripping it in JSON mode.
- [x] A combined `register get --constituents biblio,events,procedural-steps` vs separate calls: document whether the combined call saves quota hits (it should, per OPS docs).
- [x] The UPP command could accept application-format references and auto-convert to publication references internally, since users may not know the distinction.

### P04

- [x] Register commands should accept publication references (EP.1000000.A1) and auto-resolve to application number internally, or provide a clear error message suggesting the correct format.
- [x] Add an `epo legal get --flat` or `--summary` mode that returns a simplified array of {date, code, description, country, influence} objects instead of the deeply nested L00xEP structure — the current output is 110KB+ for a single patent.
- [x] Consider a `--pick` preset for legal events like `--pick=date,code,desc,country` that extracts the key fields.
- [x] The --help for `register events` should document that it expects an application reference in epodoc format, not a publication reference.
- [x] A combined `epo status EP.1000000.A1` command that merges legal events + register status + procedural steps into one timeline would be extremely useful for diligence workflows.

### P05

- [x] The CLI should auto-detect and handle kind-code-appended epodoc references (EP1000000A1) by parsing the kind code and routing to the correct API format, since the fulltext inquiry already returns results keyed by kind code.
- [x] When fulltext inquiry succeeds but a follow-up claims/description call 404s, the error message should suggest trying docdb format with the dot-separated syntax.
- [x] Consider adding a --kind flag (e.g., epo pub claims EP1000000 --kind B1) as an alternative to embedding kind codes in the reference string.
- [x] The fulltext inquiry output could include a 'suggested_retrieval_commands' field showing the exact CLI commands to run for each available fulltext instance.

### P06

- [x] CRITICAL: Fix the link-path mismatch between inquiry @link output and fetch input. Either (a) have fetch accept the full @link as-is by not prepending the prefix, or (b) strip the prefix from @link values in inquiry output, or (c) add a prominent note in fetch --help explaining that the 'published-data/images/' prefix is auto-added and should be stripped from inquiry links.
- [x] Add an example in fetch --help showing the workflow: 'Run inquiry, take the @link, strip prefix, pass to fetch'.
- [x] Consider adding a --link flag to fetch that accepts the raw @link from inquiry without modification.
- [x] The inquiry JSON output could include a 'fetch_path' field alongside @link that is ready to pass directly to the fetch command.

### P07

- [x] Parse XML natively for CPC responses: The CLI already handles JSON parsing for other endpoints; adding an XML-to-JSON normalizer for CPC would eliminate the 'not valid JSON' warning and make results machine-readable without post-processing.
- [x] Add a --parsed or --normalize flag that extracts classification-symbol, class-title, and mapping pairs into a structured JSON array.
- [x] For cpc search, consider emitting a structured array like [{symbol: 'H04L45/00', title: 'Routing or path finding...', percentage: 15.23}] instead of raw XML.
- [x] For cpc map, return {from: 'H04L45/00', fromScheme: 'CPC', to: 'H04L45/00', toScheme: 'IPC'} as parsed fields.
- [x] The --additional flag on cpc map didn't produce visibly different output for H04L45/50; document what additional context it provides or note when it has no effect.
- [x] Consider adding a --format table option that actually works for CPC (currently table format may not render XML data cleanly).

### P08

- [x] Add a --flatten or --normalize flag that extracts just the converted output fields (country, doc-number, kind, date) from the deeply nested OPS XML structure into a flat object — the raw XML-to-JSON is verbose for agent consumption.
- [x] For table/csv formats in batch mode, flatten the nested results into readable columns (inputCC, inputNum, inputKind → outputCC, outputNum, outputKind, outputDate) rather than dumping raw JSON. The single-patent table mode already does this perfectly — extend it to batch.
- [x] Consider adding a --auto-detect or --guess-format flag that infers whether input is docdb/epodoc/original based on dot separators and structure, since users often have mixed formats.
- [x] Add input validation or warning when 'INVALID' or clearly non-patent-like strings are passed — currently silently attempts the API call.
- [x] The --pick flag should document supported paths for this command, or support the deeply nested OPS paths with colon escaping.
- [x] Consider adding a convenience alias like 'epo number normalize <ref>' that auto-detects format and converts to docdb (the most structured format) in one step.

### P09

- [x] Fix -f table for usage stats: render message_count and total_response_size as columns with date rows — this is the primary use case for table format on this command.
- [x] Support array indexing in --pick (e.g., results.environments[0].dimensions[0].metrics) or at minimum flatten the nested usage data so --pick can project metric values.
- [x] Add a human-readable date column alongside epoch timestamps in the output (or a --human-dates flag).
- [x] Consider a convenience alias like 'epo usage today' or 'epo usage week' that auto-computes the date range.
- [x] Document what quota counters mean and when they update — the zero values with 200+ messages is confusing without context.
- [x] Add a 'epo usage quota' subcommand that shows just the current quota/throttle status without the full stats payload — useful for pre-flight checks in automation scripts.

### P10

- [x] Auto-include --constituents biblio when --flat is used for search (or warn that title/pubDate will be empty).
- [x] Make --pick work inside stdin batch results, not just top-level results.
- [x] Add a --flat equivalent for non-search commands (pub biblio, family get) that extracts key fields from the deeply nested XML-to-JSON.
- [x] Add an agent-friendly summary mode: 'epo pub search ... --summary' that returns {total, query, topResults: [{ref, title, date}]}.
- [x] Consider --sort with --all to sort globally across all paginated pages.
- [x] Document MSYS_NO_PATHCONV=1 requirement for epo raw on Windows/Git Bash in help text.
- [x] For stdin, silently ignore empty lines rather than treating them as positional args.
- [x] Consider a --flat-pick shorthand that combines --flat --constituents biblio --pick for the most common agent pattern.

## Notes

- This list is a direct extraction of agent suggestions from the full 10-prompt evaluation run.
- Items are intentionally unedited to preserve original intent and wording.

