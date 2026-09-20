package flatten

import (
	"github.com/MarkRosemaker/asyncapi"
	"github.com/MarkRosemaker/errpath"
)

func channels(d *asyncapi.Document, cs asyncapi.Channels) error {
	for name, c := range cs.ByIndex() {
		if c.Value == nil {
			continue // a $ref into components/channels — already flattened there
		}

		if err := channel(d, c.Value); err != nil {
			return &errpath.ErrKey{Key: name, Err: err}
		}
	}

	return nil
}
