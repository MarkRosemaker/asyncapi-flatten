package flatten

import (
	"fmt"

	"github.com/MarkRosemaker/asyncapi"
)

func newRef(tp, name string) *asyncapi.Reference {
	return &asyncapi.Reference{Identifier: fmt.Sprintf("#/components/%s/%s", tp, name)}
}
