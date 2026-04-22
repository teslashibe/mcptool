package mcptool_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/teslashibe/mcptool"
)

type fakeClient struct {
	greeting string
}

func (c *fakeClient) Hello(name string) string {
	return c.greeting + " " + name
}

func (c *fakeClient) GoodBye(name string) string {
	return "bye " + name
}

type helloInput struct {
	Name string `json:"name" jsonschema:"description=who to greet,required"`
}

type helloOutput struct {
	Message string `json:"message"`
}

func helloTool() mcptool.Tool {
	return mcptool.Define[*fakeClient, helloInput](
		"fake_hello",
		"Greet someone by name",
		"Hello",
		func(ctx context.Context, c *fakeClient, in helloInput) (any, error) {
			return helloOutput{Message: c.Hello(in.Name)}, nil
		},
	)
}

func TestDefine_HappyPath(t *testing.T) {
	t.Parallel()
	tool := helloTool()

	if tool.Name != "fake_hello" {
		t.Errorf("Name = %q, want fake_hello", tool.Name)
	}
	if tool.WrapsMethod != "Hello" {
		t.Errorf("WrapsMethod = %q, want Hello", tool.WrapsMethod)
	}
	if tool.InputSchema == nil {
		t.Fatalf("InputSchema is nil")
	}
	if got := tool.InputSchema["type"]; got != "object" {
		t.Errorf("schema type = %v, want object", got)
	}
	props, ok := tool.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema has no properties map: %#v", tool.InputSchema)
	}
	if _, ok := props["name"]; !ok {
		t.Errorf("schema missing name property: %v", props)
	}
	required, _ := tool.InputSchema["required"].([]any)
	if len(required) != 1 || required[0] != "name" {
		t.Errorf("required = %v, want [name]", required)
	}
}

func TestDefine_Invoke(t *testing.T) {
	t.Parallel()
	tool := helloTool()

	out, err := tool.Invoke(context.Background(), &fakeClient{greeting: "hi"}, json.RawMessage(`{"name":"world"}`))
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	got, ok := out.(helloOutput)
	if !ok {
		t.Fatalf("output type = %T, want helloOutput", out)
	}
	if got.Message != "hi world" {
		t.Errorf("Message = %q, want %q", got.Message, "hi world")
	}
}

func TestDefine_Invoke_WrongClientType(t *testing.T) {
	t.Parallel()
	tool := helloTool()
	_, err := tool.Invoke(context.Background(), "not a client", json.RawMessage(`{"name":"x"}`))
	if !errors.Is(err, mcptool.ErrWrongClientType) {
		t.Fatalf("err = %v, want ErrWrongClientType", err)
	}
}

func TestDefine_Invoke_BadJSON(t *testing.T) {
	t.Parallel()
	tool := helloTool()
	_, err := tool.Invoke(context.Background(), &fakeClient{}, json.RawMessage(`{not json`))
	var toolErr *mcptool.Error
	if !errors.As(err, &toolErr) {
		t.Fatalf("err = %v, want *mcptool.Error", err)
	}
	if toolErr.Code != "invalid_input" {
		t.Errorf("Code = %q, want invalid_input", toolErr.Code)
	}
}

func TestDefine_Invoke_EmptyInput(t *testing.T) {
	t.Parallel()
	tool := helloTool()
	out, err := tool.Invoke(context.Background(), &fakeClient{greeting: "hey"}, nil)
	if err != nil {
		t.Fatalf("Invoke nil input error: %v", err)
	}
	if out.(helloOutput).Message != "hey " {
		t.Errorf("Message = %q, want 'hey '", out.(helloOutput).Message)
	}
}

func TestDefine_PanicsOnEmptyName(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	mcptool.Define[*fakeClient, helloInput]("", "x", "Hello", nil)
}

func TestDefine_PanicsOnEmptyDescription(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	mcptool.Define[*fakeClient, helloInput]("fake_hello", "", "Hello", nil)
}

func TestDefine_PanicsOnNilHandler(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	mcptool.Define[*fakeClient, helloInput]("fake_hello", "x", "Hello", nil)
}

func TestError_Format(t *testing.T) {
	t.Parallel()
	e := &mcptool.Error{Code: "rate_limited", Message: "slow down"}
	if e.Error() != "rate_limited: slow down" {
		t.Errorf("Error() = %q", e.Error())
	}
	var nilErr *mcptool.Error
	if nilErr.Error() != "" {
		t.Errorf("nil Error() = %q, want empty", nilErr.Error())
	}
}

type fakeProvider struct {
	platform string
	tools    []mcptool.Tool
}

func (p fakeProvider) Platform() string       { return p.platform }
func (p fakeProvider) Tools() []mcptool.Tool { return p.tools }

func TestSortedTools(t *testing.T) {
	t.Parallel()
	a := mcptool.Define[*fakeClient, helloInput]("zeta_one", "z", "", func(ctx context.Context, c *fakeClient, in helloInput) (any, error) { return nil, nil })
	b := mcptool.Define[*fakeClient, helloInput]("alpha_two", "a", "", func(ctx context.Context, c *fakeClient, in helloInput) (any, error) { return nil, nil })
	out := mcptool.SortedTools([]mcptool.Provider{
		fakeProvider{platform: "z", tools: []mcptool.Tool{a}},
		fakeProvider{platform: "a", tools: []mcptool.Tool{b}},
	})
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if out[0].Name != "alpha_two" || out[1].Name != "zeta_one" {
		t.Errorf("order = [%s, %s]", out[0].Name, out[1].Name)
	}
}

func TestSchemaCachedAcrossDefinitions(t *testing.T) {
	t.Parallel()
	t1 := helloTool()
	t2 := helloTool()
	if !reflect.DeepEqual(t1.InputSchema, t2.InputSchema) {
		t.Error("schemas differ between Define calls for same input type")
	}
}

func TestDefine_ToolNamePropagatesIntoErrors(t *testing.T) {
	t.Parallel()
	tool := helloTool()
	_, err := tool.Invoke(context.Background(), 42, json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "fake_hello") {
		t.Errorf("err should mention tool name, got: %v", err)
	}
}
