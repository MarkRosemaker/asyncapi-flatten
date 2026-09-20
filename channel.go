package flatten

import (
	"github.com/MarkRosemaker/asyncapi"
	"github.com/MarkRosemaker/errpath"
)

func channel(d *asyncapi.Document, c *asyncapi.Channel) error {
	for name, p := range c.Parameters.ByIndex() {
		parameterRef(d, p, name)
	}

	for name, m := range c.Messages.ByIndex() {
		if err := messageRef(d, m, name); err != nil {
			return &errpath.ErrKey{Key: name, Err: err}
		}
	}

	return nil
}
