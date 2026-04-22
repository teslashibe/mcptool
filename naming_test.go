package mcptool_test

import (
	"context"
	"strings"
	"testing"

	"github.com/teslashibe/mcptool"
)

func TestValidateToolName(t *testing.T) {
	t.Parallel()
	good := []string{"linkedin_search_people", "x_send_dm", "hn_top_story_ids", "tiktok_for_you_feed"}
	for _, n := range good {
		if err := mcptool.ValidateToolName(n); err != nil {
			t.Errorf("ValidateToolName(%q) returned %v", n, err)
		}
	}
	bad := []string{"", "Search", "linkedin", "linkedin-search", "linkedinSearch", "_leading", "trailing_", "9start"}
	for _, n := range bad {
		if err := mcptool.ValidateToolName(n); err == nil {
			t.Errorf("ValidateToolName(%q) should have errored", n)
		}
	}
}

func TestValidateDescription(t *testing.T) {
	t.Parallel()
	if err := mcptool.ValidateDescription("hello", 100); err != nil {
		t.Errorf("happy path returned %v", err)
	}
	if err := mcptool.ValidateDescription("", 100); err == nil {
		t.Error("empty description should error")
	}
	if err := mcptool.ValidateDescription(strings.Repeat("a", 200), 100); err == nil {
		t.Error("over-long description should error")
	}
	if err := mcptool.ValidateDescription(strings.Repeat("a", 200), 0); err != nil {
		t.Errorf("maxLen=0 should disable length check, got %v", err)
	}
}

func TestValidateTools(t *testing.T) {
	t.Parallel()
	good := []mcptool.Tool{
		mcptool.Define[*fakeClient, helloInput]("fake_hello", "Greet someone", "Hello",
			func(ctx context.Context, c *fakeClient, in helloInput) (any, error) { return nil, nil }),
		mcptool.Define[*fakeClient, helloInput]("fake_goodbye", "Say goodbye", "GoodBye",
			func(ctx context.Context, c *fakeClient, in helloInput) (any, error) { return nil, nil }),
	}
	if err := mcptool.ValidateTools(good); err != nil {
		t.Errorf("ValidateTools(good) returned %v", err)
	}

	dup := append([]mcptool.Tool{}, good...)
	dup = append(dup, good[0])
	if err := mcptool.ValidateTools(dup); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("expected duplicate error, got %v", err)
	}

	badName := []mcptool.Tool{{Name: "Bad-Name", Description: "x", InputSchema: map[string]any{}, Invoke: good[0].Invoke}}
	if err := mcptool.ValidateTools(badName); err == nil {
		t.Error("expected name validation error")
	}

	noInvoke := []mcptool.Tool{{Name: "fake_x", Description: "x", InputSchema: map[string]any{}}}
	if err := mcptool.ValidateTools(noInvoke); err == nil || !strings.Contains(err.Error(), "Invoke") {
		t.Errorf("expected Invoke nil error, got %v", err)
	}

	noSchema := []mcptool.Tool{{Name: "fake_x", Description: "x", Invoke: good[0].Invoke}}
	if err := mcptool.ValidateTools(noSchema); err == nil || !strings.Contains(err.Error(), "InputSchema") {
		t.Errorf("expected InputSchema nil error, got %v", err)
	}
}
