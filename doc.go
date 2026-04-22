// Package mcptool is the shared, transport-agnostic contract that every
// teslashibe MCP-exposing package depends on.
//
// It defines:
//
//   - [Tool] — a self-describing tool definition (name, description,
//     auto-derived JSON schema, opaque invoke function, source method
//     reference).
//   - [Provider] — the interface every package's mcp/ subpackage exposes
//     (Platform name + Tools slice).
//   - [Define] — the canonical generic constructor that derives the input
//     JSON schema from a typed Go struct, eliminating hand-maintained schemas
//     and the drift they cause.
//   - Response-shaping helpers ([Page], [PageOf], [TruncateString],
//     [CompactJSON], [Summary]) that every package can use uniformly so the
//     agent's context window stays small.
//
// The package intentionally has no MCP-server / MCP-protocol code in it. Its
// only dependency is github.com/invopop/jsonschema (for runtime schema
// reflection). Hosting code (e.g. an HTTP+SSE server, OAuth, per-user
// credentials, registry) lives in the consuming application —
// agent-setup/backend/internal/mcp in our reference deployment.
//
// # Why co-locate tool definitions in each package
//
// When linkedin-go ships a new method, the corresponding MCP tool ships in
// the same PR. Tests live next to the code they wrap. Other consumers
// (CLIs, sidecars, alternative agent frameworks) can reuse the same Tools()
// without re-implementing them.
//
// # Drift prevention
//
// Each package's mcp/ subpackage is expected to ship with a coverage test
// that fails if a new exported method on the underlying *Client is added
// without either being wrapped by a Tool (Tool.WrapsMethod) or appearing in
// an explicit excluded.go list with a reason comment. See
// CoverageReport for the helper that powers those tests.
package mcptool
