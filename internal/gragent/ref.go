package gragent

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

// reference is a set of strings for something that can be referenced, such as
// []string{"discovery", "static", "localhost"}.
//
// While the reference type is free-form, Parse expects a specific structure
// that matches up to the real references.
type reference []string

// Equals returns true if other is equal to r.
func (r reference) Equals(other reference) bool {
	if len(r) != len(other) {
		return false
	}
	for i := 0; i < len(r); i++ {
		if r[i] != other[i] {
			return false
		}
	}
	return true
}

// String returns the dot-separated form of r.
func (r reference) String() string {
	return strings.Join([]string(r), ".")
}

// TODO(rfratto): parsing references really doesn't need to be like this, we
// can just parse by comparing to what exists.
//
// Note that references to elements in an array would still be more
// complicated, but we're not there yet.

// parseReference interprets the hcl.Traversal into a Reference. It supports parsing the
// following references:
//
//     discovery.<kind>.<name>
//     scrape.<name>
//     remote_write.<name>
//
// These align with the top-level blocks and labels known by root. The
// Traversal is only parsed up to these names; the remainder of the Traversal
// is ignored.
func parseReference(t hcl.Traversal) (reference, hcl.Diagnostics) {
	var (
		split = t.SimpleSplit()
		diags hcl.Diagnostics
	)

	switch split.RootName() {
	case "discovery":
		return parseDiscoveryRef(split.Rel, split.Abs.SourceRange())
	case "scrape":
		return parseScrapeRef(split.Rel, split.Abs.SourceRange())
	case "remote_write":
		return parseRemoteWriteRef(split.Rel, split.Abs.SourceRange())
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   fmt.Sprintf("%q is not a valid key name", split.RootName()),
			Subject:  split.Abs.SourceRange().Ptr(),
		})
		return nil, diags
	}
}

func parseDiscoveryRef(rel hcl.Traversal, startRange hcl.Range) (reference, hcl.Diagnostics) {
	var (
		kindAttr, nameAttr string
		diags              hcl.Diagnostics
	)

	if len(rel) < 2 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   `"discovery" must be followed by two attribute names: the discovery kind and name.`,
			Subject:  startRange.Ptr(),
		})
		return nil, diags
	}

	switch tt := rel[0].(type) {
	case hcl.TraverseAttr:
		kindAttr = tt.Name
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   `The "discovery" object does not support this operation.`,
			Subject:  rel[0].SourceRange().Ptr(),
		})
		return nil, diags
	}

	switch tt := rel[1].(type) {
	case hcl.TraverseAttr:
		nameAttr = tt.Name
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   `The "discovery" object does not support this operation.`,
			Subject:  rel[1].SourceRange().Ptr(),
		})
		return nil, diags
	}

	return reference{"discovery", kindAttr, nameAttr}, nil
}

func parseScrapeRef(rel hcl.Traversal, startRange hcl.Range) (reference, hcl.Diagnostics) {
	var (
		nameAttr string
		diags    hcl.Diagnostics
	)

	if len(rel) < 1 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   `"scrape" must be followed by the name attribute.`,
			Subject:  startRange.Ptr(),
		})
		return nil, diags
	}

	switch tt := rel[0].(type) {
	case hcl.TraverseAttr:
		nameAttr = tt.Name
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   `The "scrape" object does not support this operation.`,
			Subject:  rel[0].SourceRange().Ptr(),
		})
		return nil, diags
	}

	return reference{"scrape", nameAttr}, nil
}

func parseRemoteWriteRef(rel hcl.Traversal, startRange hcl.Range) (reference, hcl.Diagnostics) {
	var (
		nameAttr string
		diags    hcl.Diagnostics
	)

	if len(rel) < 1 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   `"remote_write" must be followed by the name attribute.`,
			Subject:  startRange.Ptr(),
		})
		return nil, diags
	}

	switch tt := rel[0].(type) {
	case hcl.TraverseAttr:
		nameAttr = tt.Name
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   `The "remote_write" object does not support this operation.`,
			Subject:  rel[0].SourceRange().Ptr(),
		})
		return nil, diags
	}

	return reference{"remote_write", nameAttr}, nil
}
