package mcptool

import (
	"reflect"
	"sort"
)

// CoverageReport summarises which exported methods on a *Client type are
// wrapped by Tools (via Tool.WrapsMethod) and which are declared as
// intentionally excluded.
type CoverageReport struct {
	// Wrapped is the set of exported methods that are exposed via at least
	// one Tool.
	Wrapped []string

	// Excluded is the set of exported methods explicitly listed in the
	// excluded set (these are intentionally not exposed via MCP).
	Excluded []string

	// Missing is the set of exported methods that are neither wrapped nor
	// excluded — i.e. drift. A non-empty Missing slice is the failure mode a
	// per-package coverage test should flag.
	Missing []string

	// UnknownExclusions is the set of names listed in excluded that don't
	// correspond to any actual exported method on clientType — typically
	// caused by renames. Per-package tests should flag these too.
	UnknownExclusions []string
}

// Coverage computes a CoverageReport for the given client type and tool set.
//
//   - clientType should be the reflect.Type of the *Client value used by the
//     package (e.g. reflect.TypeOf(&linkedin.Client{})).
//   - tools is the slice returned by your Provider's Tools() method.
//   - excluded is the set of method names you have intentionally chosen not
//     to expose, conventionally maintained in an excluded.go file with
//     reason comments next to each name.
//
// The returned CoverageReport contains stable, sorted slices.
func Coverage(clientType reflect.Type, tools []Tool, excluded map[string]string) CoverageReport {
	wrappedSet := map[string]bool{}
	for _, t := range tools {
		if t.WrapsMethod != "" {
			wrappedSet[t.WrapsMethod] = true
		}
	}

	exported := exportedMethods(clientType)
	exportedSet := map[string]bool{}
	for _, m := range exported {
		exportedSet[m] = true
	}

	var rep CoverageReport
	for _, m := range exported {
		switch {
		case wrappedSet[m]:
			rep.Wrapped = append(rep.Wrapped, m)
		case excluded[m] != "":
			rep.Excluded = append(rep.Excluded, m)
		default:
			rep.Missing = append(rep.Missing, m)
		}
	}
	for name := range excluded {
		if !exportedSet[name] {
			rep.UnknownExclusions = append(rep.UnknownExclusions, name)
		}
	}

	sort.Strings(rep.Wrapped)
	sort.Strings(rep.Excluded)
	sort.Strings(rep.Missing)
	sort.Strings(rep.UnknownExclusions)
	return rep
}

// exportedMethods returns the names of all exported methods on t (and its
// pointer receiver, if t is a non-pointer struct), sorted alphabetically.
func exportedMethods(t reflect.Type) []string {
	if t == nil {
		return nil
	}
	seen := map[string]bool{}

	collect := func(t reflect.Type) {
		for i := 0; i < t.NumMethod(); i++ {
			m := t.Method(i)
			if !m.IsExported() {
				continue
			}
			seen[m.Name] = true
		}
	}
	collect(t)
	if t.Kind() != reflect.Pointer {
		collect(reflect.PointerTo(t))
	}

	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
