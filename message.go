package flatten

import (
	"github.com/MarkRosemaker/asyncapi"
	"github.com/MarkRosemaker/errpath"
	"github.com/ettle/strcase"
)

// messageRef promotes m to components/messages if it is not already a
// reference, then flattens the message itself either way.
func messageRef(d *asyncapi.Document, m *asyncapi.MessageRef, name string) error {
	if m.Ref == nil {
		name = uniqueName(d.Components.Messages, name)
		d.Components.Messages.Set(name, &asyncapi.MessageRef{Value: m.Value})
		m.Ref = newRef("messages", name)
	}

	return message(d, m.Value, name)
}

// message flattens a message's payload, headers and correlation ID. name is
// the message's own component name (or, for one not yet promoted, what it
// will become) — schemas and the correlation ID promoted from within it are
// named from it, the same way [FromComponentSchemas]-adjacent naming works
// throughout this family: a message named "verifyProgress" gives its payload
// schema the name "VerifyProgress", not something derived from the payload's
// own (often absent) title.
func message(d *asyncapi.Document, m *asyncapi.Message, name string) error {
	typeName := strcase.ToGoPascal(name)

	if m.Payload != nil {
		if err := schemaRef(d, m.Payload, typeName, moveIfNecessary); err != nil {
			return &errpath.ErrField{Field: "payload", Err: err}
		}
	}

	if m.Headers != nil {
		if err := schemaRef(d, m.Headers, typeName+"Headers", moveIfNecessary); err != nil {
			return &errpath.ErrField{Field: "headers", Err: err}
		}
	}

	if m.CorrelationID != nil {
		correlationIDRef(d, m.CorrelationID, typeName+"CorrelationId")
	}

	return nil
}
