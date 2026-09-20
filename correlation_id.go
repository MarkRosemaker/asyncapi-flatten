package flatten

import "github.com/MarkRosemaker/asyncapi"

// correlationIDRef promotes c to components/correlationIds if it is not
// already a reference. Like Parameter, CorrelationID is a leaf value —
// description and location, nothing to recurse into.
func correlationIDRef(d *asyncapi.Document, c *asyncapi.CorrelationIDRef, name string) {
	if c.Ref != nil {
		return
	}

	name = uniqueName(d.Components.CorrelationIDs, name)
	d.Components.CorrelationIDs.Set(name, &asyncapi.CorrelationIDRef{Value: c.Value})
	c.Ref = newRef("correlationIds", name)
}
