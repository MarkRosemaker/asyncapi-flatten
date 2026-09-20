package flatten

import (
	"github.com/MarkRosemaker/asyncapi"
	"github.com/MarkRosemaker/errpath"
)

func schemas(d *asyncapi.Document, ss asyncapi.Schemas) error {
	for name, s := range ss.ByIndex() {
		if s.Value.Schema == nil {
			continue // a non-AsyncAPI-format schema (Avro, Protobuf, ...) has nothing to recurse into
		}

		if err := schema(d, s.Value.Schema, name); err != nil {
			return &errpath.ErrKey{Key: name, Err: err}
		}
	}

	return nil
}
