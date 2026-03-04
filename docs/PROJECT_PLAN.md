# EPO CLI Project Plan (Agent-Ready)

Date: 2026-03-03  
Status: Draft v1 (implementation-ready)

## 1. Objective

Build a Go-based `epo` CLI for EPO OPS v3.2 that:

- Exposes all meaningful OPS data query methods through stable commands.
- Is optimized for agent use (structured output, deterministic errors, command discovery).
- Ships as a single static binary with zero runtime dependencies.
- Includes a `skills/epo` package to guide agents through EPO workflows.

## 2. Why Go

- Single static binary distribution (`Windows`, `macOS`, `Linux`).
- Strong concurrency/control for throttled APIs.
- Fast startup and low memory for repeated agent invocations.
- Easy CI cross-compilation and reproducible releases.

## 3. Scope

### In Scope

- OPS authentication (OAuth client credentials).
- Full command coverage for:
  - published-data
  - family
  - number-service
  - register
  - legal
  - classification/cpc
  - usage API
- Unified machine-readable output envelope.
- Agent ergonomics (`--dry-run`, `--all`, `--pick`, `--stdin`, `methods --json`).
- Skill package (`skills/epo`) with references and workflows.

### Out of Scope (v1)

- GUI/TUI interfaces.
- Local full-text indexing or search engine.
- Long-running server daemon mode.

## 4. Product Requirements (Agent-First)

### 4.1 Command Discoverability

Provide:

- `epo methods --json`

Returns command catalog with:

- command path
- service mapping
- required/optional flags
- argument schemas
- output shape hints
- examples

### 4.2 Output Contract

All JSON responses follow a stable envelope:

```json
{
  "ok": true,
  "command": "pub search",
  "service": "published-data",
  "request": { "method": "GET", "path": "...", "query": {}, "range": "1-25" },
  "pagination": { "offset": 1, "limit": 25, "total": 781, "hasMore": true },
  "throttle": { "system": "busy", "services": { "search": { "color": "yellow", "limit": 20 } } },
  "quota": { "hourUsed": 3006, "weekUsed": 900006 },
  "results": [],
  "warnings": [],
  "version": "0.1.0"
}
```

Error envelope:

```json
{
  "ok": false,
  "error": {
    "code": 429,
    "type": "RATE_LIMITED",
    "message": "OPS rate limit exceeded",
    "hint": "Retry after backoff"
  },
  "version": "0.1.0"
}
```

### 4.3 Exit Codes

- `0` success
- `1` generic error
- `2` usage/validation error
- `3` auth error
- `4` not found
- `5` rate limited
- `6` server error

### 4.4 Essential Agent Flags

- `-f, --format` (`json`, `ndjson`, `csv`, `table`)
- `-q, --quiet`
- `--minify`
- `--dry-run`
- `--timeout`
- `--all` (auto pagination where valid)
- `--pick` (output-field projection)
- `--stdin` (batch IDs/queries from stdin)

## 5. Command Surface (v1)

## 5.1 Top-Level

- `epo auth`
- `epo usage`
- `epo pub`
- `epo family`
- `epo legal`
- `epo register`
- `epo number`
- `epo cpc`
- `epo raw`
- `epo methods`
- `epo config`
- `epo update` (optional v1.1 if release flow is ready)

### 5.2 Service Coverage Matrix

1. `auth`
- `configure`
- `token`
- `check`

2. `usage`
- `stats --date dd/mm/yyyy`
- `stats --from dd/mm/yyyy --to dd/mm/yyyy`

3. `pub`
- `biblio`
- `abstract`
- `fulltext` (inquiry)
- `claims`
- `description`
- `images inquiry`
- `images fetch`
- `equivalents`
- `search`

4. `family`
- `get` (constituents: none, biblio, legal, biblio+legal)

5. `legal`
- `get`

6. `register`
- `get`
- `events`
- `procedural-steps`
- `upp`
- `search`

7. `number`
- `convert` (reference type + from format + to format)

8. `cpc`
- `get`
- `search`
- `media`
- `map`

9. `raw`
- `get`
- `post`

10. `methods`
- `--json`

## 6. Technical Architecture

Proposed layout:

```text
cmd/epo/main.go
internal/
  api/
    client.go
    request.go
    response.go
  auth/
    token_provider.go
  config/
    config.go
    env.go
  ops/
    published.go
    family.go
    register.go
    legal.go
    number.go
    cpc.go
    usage.go
  throttle/
    parser.go
    backoff.go
  output/
    envelope.go
    json.go
    ndjson.go
    csv.go
    table.go
  errors/
    errors.go
    exit_codes.go
  catalog/
    methods.go
  cli/
    root.go
    *.go (subcommands)
tests/
  integration/
  golden/
skills/
  epo/
```

## 7. Fair Use, Throttling, and Reliability

Per response:

- Parse `X-Throttling-Control`
- Parse `Retry-After`
- Parse quota headers

Behavior:

- Yellow/red: add adaptive delay.
- Black: obey `Retry-After` strictly.
- 429/5xx: exponential backoff + jitter retries.
- Expose throttle and quota in JSON envelope for agent decision-making.

## 8. Testing Strategy

### 8.1 Unit Tests

- token management and refresh
- throttle header parsing
- request builders/path templating
- output envelope and error envelopes
- exit-code mapping

### 8.2 Integration Tests (live OPS, gated by creds)

- one test file per command domain
- include positive and negative cases
- verify machine envelope stability

### 8.3 Golden Output Tests

- freeze representative JSON outputs
- catch schema drift and breaking changes

### 8.4 Agent Prompt Tests

- create `tests/agent-prompts/` with real workflows:
  - prior art landscape
  - family tracing
  - register procedural analysis
  - legal event extraction
  - batch biblio retrieval

## 9. Skill Package Plan (`skills/epo`)

Required:

- `skills/epo/SKILL.md`

Recommended:

- `skills/epo/agents/openai.yaml`
- `skills/epo/references/commands.md`
- `skills/epo/references/workflows.md`
- `skills/epo/references/cql-published-data.md`
- `skills/epo/references/register-search.md`
- `skills/epo/references/errors-throttle.md`
- `skills/epo/references/output-envelope.md`
- `skills/epo/scripts/quick_validate.py` (optional)

Skill behavior standards:

- Default to `-f json -q`.
- Use `epo methods --json` first when uncertain.
- Prefer typed commands over `epo raw`.
- Use `epo raw` only for edge-paths not covered by typed commands.

## 10. Phased Delivery Plan

1. Phase 0: Bootstrap (2-3 days)
- Initialize Go module and command skeleton.
- Implement config/env loading.
- Implement OAuth token acquisition.
- Deliverable: `epo auth token` works.

2. Phase 1: Foundation (4-6 days)
- HTTP client, retry policy, throttle parsing.
- JSON envelope, error model, exit codes.
- Deliverable: one command group fully wired with stable output contract.

3. Phase 2: Core Retrieval/Search (7-10 days)
- `pub`, `family`, `number`.
- GET/POST variants and pagination controls.
- Deliverable: primary research workflows functioning.

4. Phase 3: Remaining OPS Services (7-10 days)
- `register`, `legal`, `cpc`, `usage`.
- Deliverable: full service parity for v1.

5. Phase 4: Agent Ergonomics (5-7 days)
- `methods --json`, `--pick`, `--stdin`, `--all`.
- Deliverable: agent can self-discover and chain commands.

6. Phase 5: Skills + Evaluation (4-5 days)
- Build `skills/epo` package.
- Add agent prompt/eval suite.
- Deliverable: skill-driven workflows validated.

7. Phase 6: Release Engineering (3-4 days)
- Cross-platform builds, checksums, release notes.
- Deliverable: reproducible binary releases.

## 11. Build Checklist

- [x] Initialize Go module and CLI root command.
- [x] Add config and credential loading (env + config file).
- [x] Implement OAuth token manager with refresh window.
- [x] Implement HTTP client with retry/backoff.
- [x] Parse and expose throttle/quota headers.
- [x] Implement standard JSON/NDJSON/CSV output.
- [x] Implement consistent error envelope and exit codes.
- [x] Implement all command groups (`pub`, `family`, `number`, `register`, `legal`, `cpc`, `usage`, `raw`).
- [x] Implement `methods --json`.
- [x] Add unit + integration + golden tests.
- [x] Add `skills/epo` (`SKILL.md` + references + openai.yaml).
- [ ] Add agent workflow prompts/tests.
- [x] Add CI build and release workflow.
- [ ] Publish v0.1.0 binary.

## 12. Milestones and Definition of Done

1. M1: Auth + Foundation
- DoD: token flow + stable envelope + exit codes + retries tested.

2. M2: Retrieval + Search
- DoD: `pub`, `family`, `number` production-usable with integration coverage.

3. M3: Full OPS Parity
- DoD: all documented service families represented by typed command or `raw` fallback.

4. M4: Agent-Ready
- DoD: `methods --json`, skill package, and agent prompt suite complete.

5. M5: Release-Ready
- DoD: CI builds signed artifacts for major OS/arch targets with versioned docs.

## 13. Risks and Mitigations

- OPS instance-level throttle variability.
  - Mitigation: adaptive backoff + per-response throttle parsing.

- API response shape inconsistency (XML/JSON conversion differences).
  - Mitigation: normalize into stable envelope `results`.

- Search syntax differences between published-data and register.
  - Mitigation: separate query validators/help for each service.

- Unknown endpoint edge-cases.
  - Mitigation: maintain `epo raw` escape hatch and document usage.

## 14. Immediate Next Actions

1. Add agent workflow prompts/tests for multi-step research tasks.
2. Publish first tagged pre-release artifact (v0.1.0-rc1) for smoke testing.
3. Add docs for release process and binary installation (`epo` on PATH).
4. Optional v1.1: add `epo update` command for release discovery/self-update.
