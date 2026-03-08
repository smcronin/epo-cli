# Contributing to epo

Thanks for your interest. This project is intentionally small, agent-focused, and API-contract driven.

## Process

1. Open an issue first with the problem statement and proposed behavior.
2. If you used an AI coding agent, include the original prompt and constraints in the issue/PR.
3. Wait for maintainer confirmation before large implementation work.

## What Makes a Good PR

- Bug fixes with reproducible input/output examples
- OPS endpoint coverage improvements that preserve stable envelope output
- Deterministic tests for parsing/projection/format behavior
- Documentation updates that reflect actual CLI behavior and flags

## What Is Usually Rejected

- Large refactors without prior discussion
- New runtime dependencies without a clear need
- Behavior changes that break existing output contracts for automation
- Changes without tests for non-trivial command behavior

## Development

```bash
go test ./...

# Live OPS integration tests (requires credentials)
# PowerShell example:
$env:EPO_INTEGRATION="1"
$env:EPO_CLIENT_ID="..."
$env:EPO_CLIENT_SECRET="..."
go test ./tests/integration -v -count=1 -timeout 600s
```

## Release Notes

The release workflow is tag-driven (`v*`) and injects binary version metadata with `-ldflags`.
Use `powershell -ExecutionPolicy Bypass -File tools/release.ps1 -Bump patch|minor|major`
to run tests, bump version metadata, update `CHANGELOG.md`, sync repo-owned skills,
create the release commit/tag, and push GitHub.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
