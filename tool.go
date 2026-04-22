package mcptool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"

	"github.com/invopop/jsonschema"
)

// Tool is a transport-agnostic, self-describing MCP tool.
//
// Construct one via [Define] (recommended — derives the schema from a typed
// input struct) or by populating the fields directly. Once constructed, a
// Tool is immutable.
type Tool struct {
	// Name is the unique tool identifier the agent calls. Conventionally
	// "<platform>_<verb>_<noun>" in snake_case, e.g. "linkedin_search_people".
	Name string

	// Description is a single sentence summarising what the tool does. Keep
	// it under 120 characters so the agent's tool inventory stays compact.
	Description string

	// InputSchema is a JSON Schema (draft-7) describing the tool's input
	// payload. When constructed via [Define] this is auto-derived from the
	// input struct's tags.
	InputSchema map[string]any

	// WrapsMethod is the name of the *Client method this tool wraps, e.g.
	// "SearchPeople". Used by per-package coverage tests to detect drift
	// (methods added to the client but not exposed via MCP).
	//
	// Leave empty when a tool aggregates multiple methods or wraps a
	// package-level function.
	WrapsMethod string

	// Tags are optional free-form labels that the host application can use
	// for filtering, grouping in docs, or per-tool feature flags (e.g.
	// "write", "experimental", "destructive").
	Tags []string

	// Invoke executes the tool. The host application is responsible for
	// constructing client (typically a per-user, per-platform *Client) and
	// passing it in. The returned value is marshalled to JSON by the host;
	// returning an [Error] produces a structured tool error, returning any
	// other error produces a generic internal-error response.
	Invoke func(ctx context.Context, client any, rawInput json.RawMessage) (any, error)
}

// Provider is the interface every per-package mcp subpackage exposes.
//
// A typical implementation is a zero-sized struct returning a fixed slice of
// Tools built once via [Define]:
//
//	type Provider struct{}
//
//	func (Provider) Platform() string { return "linkedin" }
//	func (Provider) Tools() []mcptool.Tool { return tools }
type Provider interface {
	// Platform is the short, lowercase identifier for this package, e.g.
	// "linkedin", "x", "facebook". Used to group tools in inventories and
	// to match against the host application's per-platform credential store.
	Platform() string

	// Tools returns every MCP tool this package exposes.
	Tools() []Tool
}

// ClientFactory is an optional, platform-specific helper a host application
// can ask the package to provide. It is intentionally not part of [Provider]
// because building a *Client typically requires platform-specific credential
// shapes and dependencies the host injects.
//
// Each package may optionally export a top-level ClientFactory (or a NewClient
// function) so hosts can construct a client uniformly; see the per-package
// docs for the recommended signature.
type ClientFactory interface {
	NewClient(ctx context.Context, credential json.RawMessage) (any, error)
}

// Define is the canonical Tool constructor. It derives the input JSON schema
// from struct tags on I (using github.com/invopop/jsonschema), wires up
// JSON unmarshalling and client type-assertion, and returns a ready-to-use
// Tool.
//
// Type parameters:
//
//   - C is the concrete client type the handler expects (e.g. *linkedin.Client).
//     Invoke type-asserts the opaque client argument to C and returns
//     [ErrWrongClientType] on mismatch.
//   - I is the input struct type. Use json and jsonschema tags to document
//     fields and constrain the schema:
//
//	type SearchPeopleInput struct {
//	    Query string `json:"query" jsonschema:"description=keywords or name,required"`
//	    Limit int    `json:"limit,omitempty" jsonschema:"minimum=1,maximum=50,default=10"`
//	}
//
// The wrapsMethod argument should be the bare method name on C, e.g.
// "SearchPeople". Pass an empty string for tools that don't wrap a single
// method.
func Define[C any, I any](
	name, description, wrapsMethod string,
	handler func(ctx context.Context, client C, in I) (any, error),
) Tool {
	if name == "" {
		panic("mcptool.Define: name must not be empty")
	}
	if description == "" {
		panic("mcptool.Define: description must not be empty (tool " + name + ")")
	}
	if handler == nil {
		panic("mcptool.Define: handler must not be nil (tool " + name + ")")
	}

	schema := reflectSchema[I]()

	return Tool{
		Name:        name,
		Description: description,
		InputSchema: schema,
		WrapsMethod: wrapsMethod,
		Invoke: func(ctx context.Context, client any, raw json.RawMessage) (any, error) {
			var in I
			if len(raw) > 0 && string(raw) != "null" {
				if err := json.Unmarshal(raw, &in); err != nil {
					return nil, &Error{
						Code:    "invalid_input",
						Message: fmt.Sprintf("invalid input for tool %s: %v", name, err),
					}
				}
			}
			c, ok := client.(C)
			if !ok {
				return nil, fmt.Errorf("%w: tool %s expects %T, got %T",
					ErrWrongClientType, name, *new(C), client)
			}
			return handler(ctx, c, in)
		},
	}
}

// ErrWrongClientType is returned by Tool.Invoke when the host passes a
// client value that does not match the type the tool was defined with.
var ErrWrongClientType = errors.New("mcptool: wrong client type")

// Error is the structured error type returned by tool handlers. The host
// application surfaces these to the agent so it can react meaningfully
// (e.g. "credential_expired" → tell the user to reconnect).
//
// Returning a non-Error error from a handler is treated as an internal
// failure and surfaced as code "internal_error".
type Error struct {
	// Code is a short, machine-readable identifier (snake_case, lowercase),
	// e.g. "credential_expired", "rate_limited", "not_found", "invalid_input".
	Code string

	// Message is a human-readable explanation suitable for surfacing to the
	// end user.
	Message string

	// Retryable hints to the agent whether retrying with the same input is
	// likely to succeed (e.g. true for transient network errors, false for
	// invalid input).
	Retryable bool

	// Data is optional structured detail (e.g. {"reconnect_url": "/settings/..."}).
	Data map[string]any
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Code + ": " + e.Message
}

// reflectSchema returns the JSON Schema for I as a generic map suitable for
// embedding in MCP tool definitions. The reflection is cached per type to
// avoid the cost of recomputation across many tool registrations.
func reflectSchema[I any]() map[string]any {
	var zero I
	t := reflect.TypeOf(zero)

	schemaCacheMu.Lock()
	defer schemaCacheMu.Unlock()
	if cached, ok := schemaCache[t]; ok {
		return cached
	}

	r := jsonschema.Reflector{
		Anonymous:                  true,
		AllowAdditionalProperties:  false,
		DoNotReference:             true,
		ExpandedStruct:             true,
		RequiredFromJSONSchemaTags: true,
	}
	s := r.Reflect(zero)
	b, err := json.Marshal(s)
	if err != nil {
		panic(fmt.Sprintf("mcptool: marshal schema for %T: %v", zero, err))
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		panic(fmt.Sprintf("mcptool: unmarshal schema for %T: %v", zero, err))
	}
	delete(m, "$schema")
	delete(m, "$id")

	schemaCache[t] = m
	return m
}

// SortedTools returns the tools for every provider, sorted first by platform
// and then by tool name. Useful for stable inventory generation.
func SortedTools(providers []Provider) []Tool {
	var out []Tool
	for _, p := range providers {
		out = append(out, p.Tools()...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return false
		}
		return out[i].Name < out[j].Name
	})
	return out
}
