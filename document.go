package flatten

import (
	"github.com/MarkRosemaker/asyncapi"
	"github.com/MarkRosemaker/errpath"
)

// Document flattens an entire AsyncAPI document so it contains no nested
// inline schemas, messages, parameters or correlation IDs.
func Document(d *asyncapi.Document) error {
	if err := channels(d, d.Channels); err != nil {
		return &errpath.ErrField{Field: "channels", Err: err}
	}

	if err := components(d, d.Components); err != nil {
		return &errpath.ErrField{Field: "components", Err: err}
	}

	return nil
}
