package flatten

import (
	"fmt"
	"strings"

	"github.com/MarkRosemaker/asyncapi"
	"github.com/MarkRosemaker/errpath"
	"github.com/ettle/strcase"
)

func schemaRefs(d *asyncapi.Document, ss asyncapi.Schemas, prefix string) error {
	for name, s := range ss.ByIndex() {
		if err := schemaRef(d, s,
			strcase.ToGoPascal(fmt.Sprintf("%s %s", prefix, strings.ReplaceAll(name, "/", " "))), moveIfNecessary); err != nil {
			return &errpath.ErrKey{Key: name, Err: err}
		}
	}

	return nil
}
