package flatten

import "github.com/MarkRosemaker/asyncapi"

// parameterRef promotes p to components/parameters if it is not already a
// reference. A Parameter has no schema or content of its own to recurse
// into — enum, default, description, examples and location are all leaf
// values — so there is nothing further to flatten once it has a name.
func parameterRef(d *asyncapi.Document, p *asyncapi.ParameterRef, name string) {
	if p.Ref != nil {
		return
	}

	name = uniqueName(d.Components.Parameters, name)
	d.Components.Parameters.Set(name, &asyncapi.ParameterRef{Value: p.Value})
	p.Ref = newRef("parameters", name)
}
