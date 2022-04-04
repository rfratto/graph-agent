// Package config holds the raw runtime representation of component types. They
// are intended to interact with HCL and go-cty, while the raw types are used
// for interacting with various subsystems.
package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// Root is the top-level settings.
type Root struct {
	ScrapeInterval string `hcl:"scrape_interval,optional" cty:"scrape_interval"`
	ScrapeTimeout  string `hcl:"scrape_timeout,optional" cty:"scrape_timeout"`

	// NOTE(rfratto): Root only holds direct fields, and not embedded components.
}

// DiscoveryStatic is a config block for static Prometheus SD.
type DiscoveryStatic struct {
	Hosts  []string          `hcl:"hosts" cty:"hosts"`
	Labels map[string]string `hcl:"labels,optional" cty:"labels"`
}

// DiscoveryChain configures chain Prometheus SD.
type DiscoveryChain struct {
	Input []TargetGroup `hcl:"input" cty:"input"`
}

// TargetGroup is a set of targets that share a common set of labels.
type TargetGroup struct {
	Targets []LabelSet `hcl:"targets" cty:"targets"`
	Labels  LabelSet   `hcl:"labels,optional" cty:"labels"`
}

// LabelSet is a map of label names to values.
type LabelSet map[string]string

// MetricsScrape configures scraping a set of metrics from targets.
type MetricsScrape struct {
	Targets []TargetGroup `hcl:"targets" cty:"targets"`
}

// RemoteWrite configures where to send metrics to.
type RemoteWrite struct {
	URL string `hcl:"url" cty:"url"`
}

// DecodeHCL decodes the provided hcl.Body into v. v should be a pointer type
// to a struct in this package. If not, it must be a struct with a hcl and cty
// tag for every field in the struct.
func DecodeHCL(ectx *hcl.EvalContext, b hcl.Body, v interface{}) hcl.Diagnostics {
	return gohcl.DecodeBody(b, ectx, v)
}

// DecodeCty decodes the provided cty.Value into v. v should be a pointer type
// to a struct in this package. If not, it must be a struct with a hcl and cty
// tag for every field in the struct.
func DecodeCty(val cty.Value, v interface{}) error {
	return gocty.FromCtyValue(val, v)
}

// EncodeCty encodes v into a cty.Value. v should be a type of struct in this
// package. If not, it must be a struct with a hcl and cty tag for every field
// in the struct.
//
// Encoding capsule types is not supported.
func EncodeCty(v interface{}) (cty.Value, error) {
	ty, err := gocty.ImpliedType(v)
	if err != nil {
		return cty.NilVal, err
	}
	return gocty.ToCtyValue(v, ty)
}
