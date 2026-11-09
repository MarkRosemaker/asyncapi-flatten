package flatten_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/MarkRosemaker/asyncapi"
	flatten "github.com/MarkRosemaker/asyncapi-flatten"
)

// minimalDoc wraps a schema JSON fragment into the smallest valid AsyncAPI
// document that exercises schemaRef via a receive operation's payload.
func minimalDoc(t *testing.T, schemaJSON string) *asyncapi.Document {
	t.Helper()

	raw := `{
		"asyncapi": "3.0.0",
		"info": {"title": "test", "version": "0.0.1"},
		"channels": {
			"c": {
				"address": "/test",
				"messages": {"m": {"payload": ` + schemaJSON + `}}
			}
		},
		"operations": {
			"op": {"action": "receive", "channel": {"$ref": "#/channels/c"}}
		}
	}`

	doc, err := asyncapi.LoadFromReader(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("LoadFromReader: %v", err)
	}

	return doc
}

func TestFlatten_TypeNull(t *testing.T) {
	doc := minimalDoc(t, `{"type": "null"}`)

	if err := flatten.Document(doc); err != nil {
		t.Fatalf("unexpected error for TypeNull schema: %v", err)
	}
}

func TestFlatten_ArrayOfBoolean(t *testing.T) {
	doc := minimalDoc(t, `{"type": "array", "items": {"type": "boolean"}}`)

	if err := flatten.Document(doc); err != nil {
		t.Fatalf("unexpected error for array-of-boolean schema: %v", err)
	}
}

func TestFlatten_ArrayOfNull(t *testing.T) {
	doc := minimalDoc(t, `{"type": "array", "items": {"type": "null"}}`)

	if err := flatten.Document(doc); err != nil {
		t.Fatalf("unexpected error for array-of-null schema: %v", err)
	}
}

func TestFlatten_ArrayNoItems(t *testing.T) {
	doc := minimalDoc(t, `{"type": "array"}`)

	if err := flatten.Document(doc); err != nil {
		t.Fatalf("unexpected error for array with no items schema: %v", err)
	}
}

func TestFlatten_ArrayOfAnyOf(t *testing.T) {
	// items with no explicit type, using anyOf (e.g. a nullable union)
	doc := minimalDoc(t, `{
		"type": "array",
		"items": {
			"anyOf": [
				{"type": "string", "format": "date-time"},
				{"type": "null"}
			]
		}
	}`)

	if err := flatten.Document(doc); err != nil {
		t.Fatalf("unexpected error for array-of-anyOf schema: %v", err)
	}
}

func TestFlatten_UnionTypeWithEnum(t *testing.T) {
	// AsyncAPI's type is a list, unlike OpenAPI 3.0's single value — a
	// ["string", "null"] union with an enum should still promote by its
	// primary (non-null) type.
	doc := minimalDoc(t, `{"type": ["string", "null"], "enum": ["a", "b"]}`)

	if err := flatten.Document(doc); err != nil {
		t.Fatalf("unexpected error for a nullable string enum: %v", err)
	}

	if len(doc.Components.Schemas) == 0 {
		t.Fatal("expected the enum schema to be promoted to components")
	}
}

func TestFlatten_AllOfComposition(t *testing.T) {
	// allOf members are never moved on their own (they exist solely to
	// compose the enclosing schema), but each is still recursed into.
	doc := minimalDoc(t, `{
		"allOf": [
			{"type": "object", "properties": {"a": {"type": "string", "enum": ["x", "y"]}}},
			{"type": "object", "properties": {"b": {"type": "integer"}}}
		]
	}`)

	if err := flatten.Document(doc); err != nil {
		t.Fatalf("unexpected error for allOf composition: %v", err)
	}

	if err := doc.Validate(); err != nil {
		t.Fatalf("produced invalid document: %v", err)
	}

	for name, s := range doc.Components.Schemas {
		if s.Ref != nil {
			t.Fatalf("schema %q inside allOf should never be moved to its own component", name)
		}
	}

	if _, ok := doc.Components.Schemas["MAllOf0A"]; !ok {
		t.Error("expected the enum nested inside the first allOf member to still be promoted")
	}
}

func TestFlatten_RawSchemaFormat(t *testing.T) {
	// A payload in a non-AsyncAPI schema format (Avro, Protobuf, ...) has no
	// native Schema to recurse into, but still gets a name of its own.
	doc := minimalDoc(t, `{
		"schemaFormat": "application/vnd.apache.avro+json;version=1.9.0",
		"schema": {"type": "record", "name": "Whatever"}
	}`)

	if err := flatten.Document(doc); err != nil {
		t.Fatalf("unexpected error for a raw schema format payload: %v", err)
	}

	if _, ok := doc.Components.Schemas["M"]; !ok {
		t.Error("expected the raw-format payload to be promoted to components/schemas as \"M\"")
	}
}

func TestFlatten_NameCollision(t *testing.T) {
	raw := `{
		"asyncapi": "3.0.0",
		"info": {"title": "test", "version": "0.0.1"},
		"channels": {
			"c": {
				"address": "/test",
				"messages": {
					"event": {"payload": {"type": "object", "properties": {"a": {"type": "string"}}}}
				}
			}
		},
		"operations": {
			"op": {"action": "receive", "channel": {"$ref": "#/channels/c"}}
		},
		"components": {
			"schemas": {
				"Event": {"type": "object", "properties": {"b": {"type": "string"}}}
			}
		}
	}`

	doc, err := asyncapi.LoadFromReader(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("LoadFromReader: %v", err)
	}

	if err := flatten.Document(doc); err != nil {
		t.Fatalf("Document: %v", err)
	}

	if err := doc.Validate(); err != nil {
		t.Fatalf("produced invalid document: %v", err)
	}

	// The pre-existing "Event" component must survive untouched, and the
	// channel's own inline "event" message's payload must land under a
	// different name rather than silently overwriting it.
	if _, ok := doc.Components.Schemas["Event2"]; !ok {
		t.Error(`expected the colliding payload schema to be named "Event2"`)
	}

	if props := doc.Components.Schemas["Event"].Value.Schema.Properties; len(props) != 1 {
		t.Errorf("expected the pre-existing Event component to be untouched, got %d properties", len(props))
	}
}

func TestFlatten_PromotionRules(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
		want    []string // component schema names after flattening, sorted
	}{
		{"integer stays inline", `{"type": "integer"}`, nil},
		{"number stays inline", `{"type": "number"}`, nil},
		{"boolean stays inline", `{"type": "boolean"}`, nil},
		{"plain string stays inline", `{"type": "string", "format": "uuid"}`, nil},
		{"string enum", `{"type": "string", "enum": ["a", "b"]}`, []string{"M"}},
		{"object without properties stays inline", `{"type": "object"}`, nil},
		{"object with properties", `{"type": "object", "properties": {"a": {"type": "string"}}}`, []string{"M"}},
		{"bare properties", `{"properties": {"a": {"type": "string"}}}`, []string{"M"}},
		{"oneOf", `{"oneOf": [{"type": "string"}, {"type": "integer"}]}`, []string{"M"}},
		{"anyOf", `{"anyOf": [{"type": "string"}, {"type": "null"}]}`, []string{"M"}},
		{"array of strings stays inline", `{"type": "array", "items": {"type": "string"}}`, nil},
		{"array of integers stays inline", `{"type": "array", "items": {"type": "integer"}}`, nil},
		{"array of string enums", `{"type": "array", "items": {"type": "string", "enum": ["a"]}}`, []string{"M", "MItem"}},
		{"array of empty objects stays inline", `{"type": "array", "items": {"type": "object"}}`, nil},
		{
			"array of objects",
			`{"type": "array", "items": {"type": "object", "properties": {"a": {"type": "string"}}}}`,
			[]string{"M", "MItem"},
		},
		{"array of arrays stays inline", `{"type": "array", "items": {"type": "array", "items": {"type": "integer"}}}`, nil},
		{
			"array of raw-format items",
			`{"type": "array", "items": {"schemaFormat": "application/vnd.apache.avro+json;version=1.9.0", "schema": {"type": "string"}}}`,
			[]string{"M", "MItem"},
		},
		{
			"nested objects named by path",
			`{"type": "object", "properties": {"result": {"type": "object", "properties": {"status": {"type": "string", "enum": ["x"]}}}}}`,
			[]string{"M", "MResult", "MResultStatus"},
		},
		{
			"property names with slashes",
			`{"type": "object", "properties": {"a/b": {"type": "string", "enum": ["x"]}}}`,
			[]string{"M", "MAB"},
		},
		{
			"additionalProperties",
			`{"type": "object", "additionalProperties": {"type": "object", "properties": {"a": {"type": "string"}}}}`,
			[]string{"MValue"},
		},
		{
			"contentSchema of a string",
			`{"type": "string", "contentEncoding": "base64", "contentSchema": {"type": "object", "properties": {"a": {"type": "string"}}}}`,
			[]string{"MContent"},
		},
		{
			"oneOf members stay inline but are recursed into",
			`{"oneOf": [{"type": "object", "properties": {"k": {"type": "string", "enum": ["x"]}}}]}`,
			[]string{"M", "MOneOf0K"},
		},
		{
			"anyOf members stay inline but are recursed into",
			`{"anyOf": [{"type": "string", "enum": ["x"]}, {"type": "array", "items": {"type": "object", "properties": {"a": {"type": "string"}}}}]}`,
			[]string{"M", "MAnyOf1Item"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := minimalDoc(t, tc.payload)

			if err := flatten.Document(doc); err != nil {
				t.Fatalf("Document: %v", err)
			}

			if err := doc.Validate(); err != nil {
				t.Fatalf("produced invalid document: %v", err)
			}

			got := slices.Sorted(maps.Keys(doc.Components.Schemas))
			if !slices.Equal(got, tc.want) {
				t.Errorf("component schemas: got %q, want %q", got, tc.want)
			}

			payload := doc.Channels["c"].Value.Messages["m"].Value.Payload
			if promoted := slices.Contains(tc.want, "M"); promoted != (payload.Ref != nil) {
				t.Errorf("payload promoted = %v, want %v", payload.Ref != nil, promoted)
			}
		})
	}
}
