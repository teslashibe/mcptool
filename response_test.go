package mcptool_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/teslashibe/mcptool"
)

func TestPageOf_NoCap(t *testing.T) {
	t.Parallel()
	items := []int{1, 2, 3, 4, 5}
	p := mcptool.PageOf(items, "cur", 0)
	if len(p.Items) != 5 || p.Truncated || p.NextCursor != "cur" {
		t.Errorf("got %+v", p)
	}
}

func TestPageOf_Truncates(t *testing.T) {
	t.Parallel()
	items := []int{1, 2, 3, 4, 5}
	p := mcptool.PageOf(items, "cur", 3)
	if len(p.Items) != 3 || !p.Truncated || p.NextCursor != "cur" {
		t.Errorf("got %+v", p)
	}
	if p.Items[2] != 3 {
		t.Errorf("Items[2] = %d, want 3", p.Items[2])
	}
}

func TestPageOf_AtCap(t *testing.T) {
	t.Parallel()
	p := mcptool.PageOf([]int{1, 2, 3}, "", 3)
	if p.Truncated {
		t.Error("Truncated should be false when len == max")
	}
}

func TestTruncateString(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 5, "hell…"},
		{"", 5, ""},
		{"abc", 0, "abc"},
		{"abc", -1, "abc"},
		{"日本語テスト", 3, "日本…"},
		{"abcd", 1, "a"},
	}
	for _, c := range cases {
		if got := mcptool.TruncateString(c.in, c.max); got != c.want {
			t.Errorf("TruncateString(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
		}
	}
}

func TestCompactJSON(t *testing.T) {
	t.Parallel()
	got := mcptool.CompactJSON(map[string]any{"a": 1, "b": "two"})
	if !strings.Contains(got, `"a":1`) || !strings.Contains(got, `"b":"two"`) {
		t.Errorf("got %q", got)
	}
	if strings.Contains(got, "\n") || strings.Contains(got, "  ") {
		t.Errorf("compact output should have no whitespace, got %q", got)
	}
}

func TestCompactJSON_NoHTMLEscape(t *testing.T) {
	t.Parallel()
	got := mcptool.CompactJSON("a<b>c&d")
	if strings.Contains(got, "\\u003c") || strings.Contains(got, "\\u0026") {
		t.Errorf("HTML chars should not be escaped, got %q", got)
	}
}

func TestCompactJSON_BadValue(t *testing.T) {
	t.Parallel()
	if mcptool.CompactJSON(make(chan int)) != "null" {
		t.Error("expected fallback to 'null' on marshal error")
	}
}

func TestSummary_MarshalsAsInner(t *testing.T) {
	t.Parallel()
	type inner struct {
		ID string `json:"id"`
	}
	s := mcptool.Summary[inner]{V: inner{ID: "abc"}}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"id":"abc"}` {
		t.Errorf("got %s", string(b))
	}
}
