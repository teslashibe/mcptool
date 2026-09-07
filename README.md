# mcptool

The shared, transport-agnostic contract every teslashibe MCP-exposing package depends on.

`mcptool` is intentionally tiny (~400 LOC, stdlib + [`invopop/jsonschema`](https://github.com/invopop/jsonschema)). It defines the `Tool` and `Provider` types, a generic `Define[C, I]` constructor that derives JSON schemas from typed Go input structs, response-shaping helpers (`Page`, `PageOf`, `TruncateString`, `CompactJSON`, `Summary`), and a `Coverage` helper that powers per-package "every method is exposed or excluded" tests.

It does **not** contain MCP server / protocol code. Hosting (HTTP+SSE transport, OAuth, per-user credentials, registry) lives in the consuming application — see [`teslashibe/agent-setup`](https://github.com/teslashibe/agent-setup) for the reference deployment.

## Why

Each scraper / service package (`linkedin-go`, `x-go`, `facebook-go`, …) ships its own `mcp/` subpackage exposing a `Provider`. When the package adds a new client method, the corresponding MCP tool ships in the same PR — no drift, no follow-up wiring elsewhere. Other consumers (CLIs, sidecars, alternative agent frameworks) reuse the same tools by importing the package's `mcp` subpackage.

`mcptool` is the narrow contract that makes this work uniformly across packages.

## Install

```sh
go get github.com/teslashibe/mcptool@latest
```

Requires Go 1.25 or newer on a currently supported patched release.

## Defining a tool

```go
package mcp

import (
    "context"

    "github.com/teslashibe/mcptool"
    linkedin "github.com/teslashibe/linkedin-go"
)

type SearchPeopleInput struct {
    Query string `json:"query" jsonschema:"description=keywords or name to search,required"`
    Limit int    `json:"limit,omitempty" jsonschema:"minimum=1,maximum=50,default=10"`
}

func searchPeople(ctx context.Context, c *linkedin.Client, in SearchPeopleInput) (any, error) {
    res, err := c.SearchPeople(ctx, linkedin.SearchParams{Query: in.Query, Limit: in.Limit})
    if err != nil {
        return nil, err
    }
    return mcptool.PageOf(res, "", in.Limit), nil
}

var searchPeopleTool = mcptool.Define[*linkedin.Client, SearchPeopleInput](
    "linkedin_search_people",
    "Search LinkedIn people by keyword or name",
    "SearchPeople",
    searchPeople,
)
```

The JSON schema is reflected from `SearchPeopleInput` at construction time, so changing the struct automatically updates the schema. The handler signature is type-safe — no `map[string]interface{}` fishing.

## Defining a provider

```go
type Provider struct{}

func (Provider) Platform() string         { return "linkedin" }
func (Provider) Tools() []mcptool.Tool {
    return []mcptool.Tool{
        searchPeopleTool,
        getProfileTool,
        // ...
    }
}
```

## Coverage tests (drift detection)

Every package's `mcp/` subpackage should ship a coverage test that fails when a client method is added without exposure or exclusion:

```go
package mcp_test

import (
    "reflect"
    "testing"

    linkedin "github.com/teslashibe/linkedin-go"
    linkmcp "github.com/teslashibe/linkedin-go/mcp"
    "github.com/teslashibe/mcptool"
)

func TestEveryClientMethodIsWrappedOrExcluded(t *testing.T) {
    rep := mcptool.Coverage(
        reflect.TypeOf(&linkedin.Client{}),
        linkmcp.Provider{}.Tools(),
        linkmcp.Excluded,
    )
    if len(rep.Missing) > 0 {
        t.Fatalf("methods missing MCP exposure (add a tool or list in excluded.go): %v", rep.Missing)
    }
    if len(rep.UnknownExclusions) > 0 {
        t.Fatalf("excluded.go references methods that don't exist on *Client: %v", rep.UnknownExclusions)
    }
}

func TestToolsValidate(t *testing.T) {
    if err := mcptool.ValidateTools(linkmcp.Provider{}.Tools()); err != nil {
        t.Fatal(err)
    }
}
```

`linkmcp.Excluded` is a `map[string]string` mapping method name to exclusion reason; conventionally it lives in `excluded.go` next to the tool definitions.

## Response shaping

Use the helpers to keep tool results compact:

```go
mcptool.PageOf(items, nextCursor, in.Limit)         // caps + flags truncation
mcptool.TruncateString(post.Body, 800)              // safe UTF-8 truncation
mcptool.CompactJSON(v)                              // unindented, HTML-safe-off
mcptool.Summary[PostSummary]{V: PostSummary{...}}   // documents the shape
```

The host application's response middleware can apply uniform caps on top of these (e.g. cap any list at 50 items regardless of what a tool returns).

## Structured tool errors

Return `*mcptool.Error` from a handler to surface a structured error to the agent:

```go
return nil, &mcptool.Error{
    Code:    "credential_expired",
    Message: "LinkedIn cookies have expired; please reconnect.",
    Data:    map[string]any{"reconnect_url": "/settings/platforms/linkedin"},
}
```

Returning any other error is surfaced as `internal_error`.

## Stability

`mcptool` is pre-1.0 but the contract is intentionally minimal. Breaking changes (Tool/Provider shape) require coordinated bumps across all consuming packages and so will be infrequent. Helper additions are non-breaking.

## License

MIT — see [LICENSE](./LICENSE).
