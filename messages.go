package flatten

import (
	"github.com/MarkRosemaker/asyncapi"
	"github.com/MarkRosemaker/errpath"
)

func messages(d *asyncapi.Document, ms asyncapi.Messages) error {
	for name, m := range ms.ByIndex() {
		// Not messageRef: called from Components, where the message should
		// already be — see [schemas] for the same reasoning.
		if err := message(d, m.Value, name); err != nil {
			return &errpath.ErrKey{Key: name, Err: err}
		}
	}

	return nil
}
