# Changelog

All notable changes to this project will be documented in this file. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.1.1] — 2026-04-22

### Added
- Mirrored Cursor and Claude rules for forensic issue audits with user stories and acceptance criteria.

## [v0.1.0] — 2026-04-22

Initial release.

### Added
- `Tool` and `Provider` types — the transport-agnostic contract every teslashibe MCP-exposing package implements.
- `Define[C, I]` generic constructor — derives JSON schema from typed Go input structs, type-asserts the client at invoke time, propagates tool name into errors.
- `Error` structured error type for surfacing actionable failures (e.g. `credential_expired`) to the agent.
- Response helpers: `Page`, `PageOf`, `TruncateString`, `CompactJSON`, `Summary[T]`.
- `Coverage` helper + `CoverageReport` for per-package "every method exposed or excluded" tests.
- `ValidateToolName`, `ValidateDescription`, `ValidateTools` for naming/format checks.
- `SortedTools` for stable inventory generation across providers.

### Notes
- Pre-1.0; the `Tool` / `Provider` shapes are deliberately minimal and expected to be stable. Helper additions are non-breaking.
- Only runtime dependency: `github.com/invopop/jsonschema`.
