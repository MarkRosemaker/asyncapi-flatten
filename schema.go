package flatten

import (
	"fmt"

	"github.com/MarkRosemaker/asyncapi"
	"github.com/MarkRosemaker/errpath"
)

type mode int

const (
	moveIfNecessary mode = iota
	alwaysMove
	neverMove
)

// primaryType returns the type a schema's union actually carries data as —
// see asyncapi-enrich's unionTypes doc comment for why the first entry is the
// one that matters: null, when present, is always kept last.
func primaryType(dt asyncapi.DataTypes) asyncapi.DataType {
	if len(dt) == 0 {
		return ""
	}

	return dt[0]
}

func schemaRef(d *asyncapi.Document, s *asyncapi.AnySchemaRef, name string, mode mode) error {
	if s.Ref != nil {
		return nil // already processed
	}

	if s.Value.Schema == nil {
		// A schema in a format other than AsyncAPI's own (Avro, Protobuf, ...)
		// travels as an opaque string — there is no structure here to decide
		// "simple enough to leave inline" about, so it always gets a name,
		// the same as alwaysMove, except when explicitly told never to (an
		// allOf/oneOf/anyOf member).
		if mode != neverMove {
			moveSchemaToComponents(d, name, s)
		}

		return nil
	}

	if mode == alwaysMove {
		moveSchemaToComponents(d, name, s)

		return schema(d, s.Value.Schema, name)
	}

	switch primaryType(s.Value.Schema.Type) {
	case asyncapi.TypeInteger, asyncapi.TypeNumber, asyncapi.TypeBoolean, asyncapi.TypeNull:
		// no need to move to components
	case asyncapi.TypeString:
		if s.Value.Schema.Enum != nil && mode != neverMove {
			moveSchemaToComponents(d, name, s)
		} // else just a string, no need to move to components
	case asyncapi.TypeArray:
		if s.Value.Schema.Items == nil {
			break
		}

		items := s.Value.Schema.Items.Value.Schema
		if items == nil {
			// items is itself a non-AsyncAPI-format schema; give the array a
			// name so the item schema below still gets one of its own.
			if mode != neverMove {
				moveSchemaToComponents(d, name, s)
			}

			break
		}

		switch primaryType(items.Type) {
		case asyncapi.TypeInteger, asyncapi.TypeNumber, asyncapi.TypeBoolean, asyncapi.TypeNull: // e.g. []int, []bool
		case asyncapi.TypeString:
			if items.Enum != nil && mode != neverMove {
				moveSchemaToComponents(d, name, s)
			} // else just []string, no need to move to components
		case asyncapi.TypeObject:
			if len(items.Properties) > 0 && mode != neverMove {
				moveSchemaToComponents(d, name, s)
			}
		case asyncapi.TypeArray: // TODO: later
		case "": // no explicit type — items uses anyOf/oneOf/allOf (e.g. a union)
		default:
			return fmt.Errorf("unimplemented item type %q", items.Type)
		}
	case asyncapi.TypeObject:
		if len(s.Value.Schema.Properties) > 0 && mode != neverMove {
			moveSchemaToComponents(d, name, s)
		}
	case "": // no explicit type — oneOf/anyOf/allOf composition, or bare properties
		v := s.Value.Schema
		hasComposition := len(v.OneOf) > 0 || len(v.AnyOf) > 0 || len(v.AllOf) > 0
		if (hasComposition || len(v.Properties) > 0) && mode != neverMove {
			moveSchemaToComponents(d, name, s)
		}
	default:
		return fmt.Errorf("unimplemented schema type %q", s.Value.Schema.Type)
	}

	return schema(d, s.Value.Schema, name)
}

func schema(d *asyncapi.Document, s *asyncapi.Schema, name string) error {
	// contentSchema describes what a string decodes to (per contentEncoding
	// and contentMediaType) and so applies orthogonally to Type — most often
	// attached to a plain base64 string, which the switch below would
	// otherwise return out of before ever seeing it.
	if s.ContentSchema != nil {
		if err := schemaRef(d, s.ContentSchema, name+"Content", moveIfNecessary); err != nil {
			return &errpath.ErrField{Field: "contentSchema", Err: err}
		}
	}

	switch primaryType(s.Type) {
	case asyncapi.TypeString, asyncapi.TypeInteger, asyncapi.TypeNumber,
		asyncapi.TypeBoolean, asyncapi.TypeNull: // no need to do anything else
		return nil
	case asyncapi.TypeArray, asyncapi.TypeObject: // do below
	case "": // valid if the schema is oneOf/anyOf/allOf composition, or bare properties
	default:
		return fmt.Errorf("unimplemented schema type %q", s.Type)
	}

	if err := schemaRefList(d, s.AllOf, name+"AllOf"); err != nil {
		return &errpath.ErrField{Field: "allOf", Err: err}
	}

	if err := schemaRefList(d, s.OneOf, name+"OneOf"); err != nil {
		return &errpath.ErrField{Field: "oneOf", Err: err}
	}

	if err := schemaRefList(d, s.AnyOf, name+"AnyOf"); err != nil {
		return &errpath.ErrField{Field: "anyOf", Err: err}
	}

	if s.Items != nil {
		if err := schemaRef(d, s.Items, name+"Item", moveIfNecessary); err != nil {
			return &errpath.ErrField{Field: "items", Err: err}
		}
	}

	if err := schemaRefs(d, s.Properties, name); err != nil {
		return &errpath.ErrField{Field: "properties", Err: err}
	}

	if s.AdditionalProperties != nil {
		if err := schemaRef(d, s.AdditionalProperties, name+"Value", moveIfNecessary); err != nil {
			return &errpath.ErrField{Field: "additionalProperties", Err: err}
		}
	}

	return nil
}

func moveSchemaToComponents(d *asyncapi.Document, name string, s *asyncapi.AnySchemaRef) {
	name = uniqueName(d.Components.Schemas, name)
	d.Components.Schemas.Set(name, &asyncapi.AnySchemaRef{Value: s.Value})
	s.Ref = newRef("schemas", name)
}

func schemaRefList(d *asyncapi.Document, ss asyncapi.AnySchemaRefList, prefix string) error {
	for i, s := range ss {
		if err := schemaRef(d, s, fmt.Sprintf("%s%d", prefix, i), neverMove); err != nil {
			return &errpath.ErrIndex{Index: i, Err: err}
		}
	}

	return nil
}
