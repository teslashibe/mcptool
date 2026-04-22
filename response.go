package mcptool

import (
	"bytes"
	"encoding/json"
	"unicode/utf8"
)

// Page is the canonical paginated-list shape every package should return
// from list-style tools. The host application's response middleware can
// uniformly cap the Items slice and surface NextCursor to the agent.
type Page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
	Truncated  bool   `json:"truncated,omitempty"`
}

// PageOf builds a Page[T], applying the requested cap. If maxItems is <= 0,
// the items are returned untouched. If len(items) > maxItems the slice is
// truncated and Truncated is set to true. nextCursor is passed through
// unchanged so the underlying API's cursor semantics are preserved.
func PageOf[T any](items []T, nextCursor string, maxItems int) Page[T] {
	p := Page[T]{Items: items, NextCursor: nextCursor}
	if maxItems > 0 && len(items) > maxItems {
		p.Items = items[:maxItems]
		p.Truncated = true
	}
	return p
}

// TruncateString shortens s to at most max runes, appending an ellipsis if
// truncation occurred. Safe for arbitrary UTF-8. Returns s unchanged if
// max <= 0 or s already fits.
func TruncateString(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

// CompactJSON marshals v to JSON with no indentation and HTML escaping
// disabled. Use this when you need a string-form payload (e.g. for logging
// or for embedding in an MCP text response). Returns "null" on marshal
// error to keep call sites simple; if you need real error handling, use
// encoding/json directly.
func CompactJSON(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "null"
	}
	out := buf.Bytes()
	if n := len(out); n > 0 && out[n-1] == '\n' {
		out = out[:n-1]
	}
	return string(out)
}

// Summary is a lightweight wrapper that documents a "compact, list-friendly"
// view of a domain object. It carries no behaviour — it exists purely as a
// convention so package authors and reviewers can spot at a glance which
// types are list summaries vs. full payloads.
//
//	type PostSummary mcptool.Summary[struct {
//	    ID        string `json:"id"`
//	    Author    string `json:"author"`
//	    Title     string `json:"title"`
//	    Score     int    `json:"score"`
//	    CreatedAt string `json:"created_at"`
//	}]
type Summary[T any] struct {
	V T `json:"-"`
}

// MarshalJSON implements json.Marshaler so Summary[T] serialises identically
// to T.
func (s Summary[T]) MarshalJSON() ([]byte, error) { return json.Marshal(s.V) }
