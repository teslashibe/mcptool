package mcptool_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/teslashibe/mcptool"
)

func TestCoverage_AllWrapped(t *testing.T) {
	t.Parallel()
	tools := []mcptool.Tool{
		mcptool.Define[*fakeClient, helloInput]("fake_hello", "x", "Hello",
			func(ctx context.Context, c *fakeClient, in helloInput) (any, error) { return nil, nil }),
		mcptool.Define[*fakeClient, helloInput]("fake_goodbye", "x", "GoodBye",
			func(ctx context.Context, c *fakeClient, in helloInput) (any, error) { return nil, nil }),
	}
	rep := mcptool.Coverage(reflect.TypeOf(&fakeClient{}), tools, nil)
	if len(rep.Missing) != 0 {
		t.Errorf("Missing = %v, want []", rep.Missing)
	}
	if len(rep.Wrapped) != 2 {
		t.Errorf("Wrapped = %v, want 2 entries", rep.Wrapped)
	}
}

func TestCoverage_DetectsMissing(t *testing.T) {
	t.Parallel()
	tools := []mcptool.Tool{
		mcptool.Define[*fakeClient, helloInput]("fake_hello", "x", "Hello",
			func(ctx context.Context, c *fakeClient, in helloInput) (any, error) { return nil, nil }),
	}
	rep := mcptool.Coverage(reflect.TypeOf(&fakeClient{}), tools, nil)
	if len(rep.Missing) != 1 || rep.Missing[0] != "GoodBye" {
		t.Errorf("Missing = %v, want [GoodBye]", rep.Missing)
	}
}

func TestCoverage_RespectsExclusions(t *testing.T) {
	t.Parallel()
	tools := []mcptool.Tool{
		mcptool.Define[*fakeClient, helloInput]("fake_hello", "x", "Hello",
			func(ctx context.Context, c *fakeClient, in helloInput) (any, error) { return nil, nil }),
	}
	excluded := map[string]string{"GoodBye": "deprecated, removal pending"}
	rep := mcptool.Coverage(reflect.TypeOf(&fakeClient{}), tools, excluded)
	if len(rep.Missing) != 0 {
		t.Errorf("Missing = %v, want []", rep.Missing)
	}
	if len(rep.Excluded) != 1 || rep.Excluded[0] != "GoodBye" {
		t.Errorf("Excluded = %v, want [GoodBye]", rep.Excluded)
	}
}

func TestCoverage_DetectsUnknownExclusions(t *testing.T) {
	t.Parallel()
	tools := []mcptool.Tool{
		mcptool.Define[*fakeClient, helloInput]("fake_hello", "x", "Hello",
			func(ctx context.Context, c *fakeClient, in helloInput) (any, error) { return nil, nil }),
		mcptool.Define[*fakeClient, helloInput]("fake_goodbye", "x", "GoodBye",
			func(ctx context.Context, c *fakeClient, in helloInput) (any, error) { return nil, nil }),
	}
	excluded := map[string]string{"NonExistentMethod": "renamed elsewhere"}
	rep := mcptool.Coverage(reflect.TypeOf(&fakeClient{}), tools, excluded)
	if len(rep.UnknownExclusions) != 1 || rep.UnknownExclusions[0] != "NonExistentMethod" {
		t.Errorf("UnknownExclusions = %v, want [NonExistentMethod]", rep.UnknownExclusions)
	}
}
