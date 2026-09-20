package flatten

import (
	"github.com/MarkRosemaker/asyncapi"
	"github.com/MarkRosemaker/errpath"
)

func components(d *asyncapi.Document, c asyncapi.Components) error {
	if err := schemas(d, c.Schemas); err != nil {
		return &errpath.ErrField{Field: "schemas", Err: err}
	}

	if err := messages(d, c.Messages); err != nil {
		return &errpath.ErrField{Field: "messages", Err: err}
	}

	// Parameters and CorrelationIDs are leaf values (see [parameterRef] and
	// [correlationIDRef]) — there is nothing left to flatten in one already
	// sitting in components.

	return nil
}
