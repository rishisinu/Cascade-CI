package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
)

fmt.Print("hola")
// Tool def in MCP format. name, desc for model, json schema for input. Not dep on an LLM provider
type Spec struct {
	Name string
	Description string
	Properties map[string]any //Json input schema
	Required []string

}
//THis defines what the agent can call, we pass in spec so the agent knows what it needs, and
// so when run is called, we pass in context and the input will be tools args schema, differs from provider to provider so we take from spec and build appropirate mapping
type Tool interface {
	Spec() Spec //Tool needs to have a spec
	Run(ctx context.Context, input json.RawMessage) (string, error)
}

type Tools struct {
	tools map[string]Tool
}

func newTools(tools...Tool) *Tools {
	r := &Tools{tools: map[string]Tool{}} // R is new tools struct type *Tools

	for _, t := range tools { //Loop thru tools and append to r
		r.tools[t.Spec().Name] = t
	}
	return r
}

func (r *Tools) Specs() []Spec { //The signature before the func just defines what you're allowed to call this on, so in this case only on our tool registry
	specs := make([]Spec, 0, len((*r).tools)			// Quick note : Even if you pass in a raw Tool oobj , go will auto
		for _, t := range r.tools {
			specs = append(specs, t.Spec())
		}
	sort.Slice(specs, func(i,j int) bool {return specs[i].Name < specs[j].Name })  // sort by name,
	return spec
}
