package flatten_test

import (
	"strings"
	"testing"

	"github.com/MarkRosemaker/asyncapi"
	flatten "github.com/MarkRosemaker/asyncapi-flatten"
)

func loadDoc(t *testing.T, raw string) *asyncapi.Document {
	t.Helper()

	doc, err := asyncapi.LoadFromReader(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("LoadFromReader: %v", err)
	}

	return doc
}

// componentsDoc wraps a components JSON fragment into a document with no channels.
func componentsDoc(t *testing.T, componentsJSON string) *asyncapi.Document {
	t.Helper()

	return loadDoc(t, `{
		"asyncapi": "3.0.0",
		"info": {"title": "test", "version": "0.0.1"},
		"components": `+componentsJSON+`
	}`)
}

func TestFlatten_ErrorPath(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  func(t *testing.T) *asyncapi.Document
		want string
	}{
		{
			name: "payload",
			doc:  func(t *testing.T) *asyncapi.Document { return minimalDoc(t, `{"type": "whatever"}`) },
			want: `channels["c"].messages["m"].payload: unimplemented schema type "whatever"`,
		},
		{
			name: "array item type",
			doc: func(t *testing.T) *asyncapi.Document {
				return minimalDoc(t, `{"type": "array", "items": {"type": "whatever"}}`)
			},
			want: `channels["c"].messages["m"].payload: unimplemented item type "whatever"`,
		},
		{
			name: "nested property",
			doc: func(t *testing.T) *asyncapi.Document {
				return minimalDoc(t, `{"type": "object", "properties": {"p": {"type": "whatever"}}}`)
			},
			want: `channels["c"].messages["m"].payload.properties["p"]: unimplemented schema type "whatever"`,
		},
		{
			name: "array items",
			doc: func(t *testing.T) *asyncapi.Document {
				return minimalDoc(t, `{"type": "object", "properties": {"p": {"type": "array", "items": {"anyOf": [{"type": "whatever"}]}}}}`)
			},
			want: `channels["c"].messages["m"].payload.properties["p"].items.anyOf[0]: unimplemented schema type "whatever"`,
		},
		{
			name: "allOf member",
			doc: func(t *testing.T) *asyncapi.Document {
				return minimalDoc(t, `{"allOf": [{"type": "string"}, {"type": "whatever"}]}`)
			},
			want: `channels["c"].messages["m"].payload.allOf[1]: unimplemented schema type "whatever"`,
		},
		{
			name: "oneOf member",
			doc: func(t *testing.T) *asyncapi.Document {
				return minimalDoc(t, `{"oneOf": [{"type": "whatever"}]}`)
			},
			want: `channels["c"].messages["m"].payload.oneOf[0]: unimplemented schema type "whatever"`,
		},
		{
			name: "additionalProperties",
			doc: func(t *testing.T) *asyncapi.Document {
				return minimalDoc(t, `{"type": "object", "additionalProperties": {"type": "whatever"}}`)
			},
			want: `channels["c"].messages["m"].payload.additionalProperties: unimplemented schema type "whatever"`,
		},
		{
			name: "contentSchema",
			doc: func(t *testing.T) *asyncapi.Document {
				return minimalDoc(t, `{"type": "string", "contentSchema": {"type": "whatever"}}`)
			},
			want: `channels["c"].messages["m"].payload.contentSchema: unimplemented schema type "whatever"`,
		},
		{
			name: "message headers",
			doc: func(t *testing.T) *asyncapi.Document {
				return componentsDoc(t, `{"messages": {"m": {"headers": {"type": "whatever"}}}}`)
			},
			want: `components.messages["m"].headers: unimplemented schema type "whatever"`,
		},
		{
			name: "component schema",
			doc: func(t *testing.T) *asyncapi.Document {
				return componentsDoc(t, `{"schemas": {"S": {"type": "object", "properties": {"p": {"type": "whatever"}}}}}`)
			},
			want: `components.schemas["S"].properties["p"]: unimplemented schema type "whatever"`,
		},
		{
			name: "component schema type",
			doc: func(t *testing.T) *asyncapi.Document {
				return componentsDoc(t, `{"schemas": {"S": {"type": "whatever"}}}`)
			},
			want: `components.schemas["S"]: unimplemented schema type "whatever"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := flatten.Document(tc.doc(t))
			if err == nil {
				t.Fatal("expected an error")
			}

			if got := err.Error(); got != tc.want {
				t.Errorf("error:\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

func TestFlatten_ComponentMessages(t *testing.T) {
	doc := componentsDoc(t, `{
		"messages": {
			"verifyProgress": {
				"payload": {"type": "object", "properties": {"status": {"type": "string", "enum": ["ok", "failed"]}}},
				"headers": {"type": "object", "properties": {"trace": {"type": "string"}}},
				"correlationId": {"location": "$message.header#/trace"}
			}
		}
	}`)

	if err := flatten.Document(doc); err != nil {
		t.Fatalf("Document: %v", err)
	}

	if err := doc.Validate(); err != nil {
		t.Fatalf("produced invalid document: %v", err)
	}

	if len(doc.Components.Messages) != 1 {
		t.Errorf("a message already in components must not be promoted again, got %d messages", len(doc.Components.Messages))
	}

	msg := doc.Components.Messages["verifyProgress"]
	if msg.Ref != nil {
		t.Errorf("a component message must stay inline in components, got $ref %q", msg.Ref.Identifier)
	}

	for field, ref := range map[string]*asyncapi.Reference{
		"#/components/schemas/VerifyProgress":                     msg.Value.Payload.Ref,
		"#/components/schemas/VerifyProgressHeaders":              msg.Value.Headers.Ref,
		"#/components/correlationIds/VerifyProgressCorrelationId": msg.Value.CorrelationID.Ref,
	} {
		if ref == nil || ref.Identifier != field {
			t.Errorf("expected $ref %q, got %+v", field, ref)
		}
	}

	if _, ok := doc.Components.Schemas["VerifyProgressStatus"]; !ok {
		t.Error("expected the nested enum to be promoted as VerifyProgressStatus")
	}
}

func TestFlatten_ExistingRefsAreKept(t *testing.T) {
	doc := loadDoc(t, `{
		"asyncapi": "3.0.0",
		"info": {"title": "test", "version": "0.0.1"},
		"channels": {
			"a": {
				"address": "a/{id}",
				"parameters": {"id": {"$ref": "#/components/parameters/id"}},
				"messages": {"m": {"$ref": "#/components/messages/m"}}
			},
			"b": {"$ref": "#/components/channels/b"}
		},
		"operations": {
			"op": {"action": "receive", "channel": {"$ref": "#/channels/a"}}
		},
		"components": {
			"channels": {
				"b": {"address": "b", "messages": {"n": {"payload": {"type": "object", "properties": {"x": {"type": "string"}}}}}}
			},
			"parameters": {"id": {"description": "the id"}},
			"messages": {
				"m": {
					"payload": {"$ref": "#/components/schemas/S"},
					"correlationId": {"$ref": "#/components/correlationIds/c"}
				}
			},
			"correlationIds": {"c": {"location": "$message.payload#/id"}},
			"schemas": {"S": {"type": "object", "properties": {"id": {"type": "string"}}}}
		}
	}`)

	if err := flatten.Document(doc); err != nil {
		t.Fatalf("Document: %v", err)
	}

	if err := doc.Validate(); err != nil {
		t.Fatalf("produced invalid document: %v", err)
	}

	// Only channel b's inline message n and its payload N are new.
	c := doc.Components
	if len(c.Parameters) != 1 || len(c.Messages) != 2 || len(c.CorrelationIDs) != 1 || len(c.Schemas) != 2 {
		t.Errorf("expected only n and N to be added, got %d parameters, %d messages, %d correlationIds, %d schemas",
			len(c.Parameters), len(c.Messages), len(c.CorrelationIDs), len(c.Schemas))
	}

	if ref := doc.Channels["a"].Value.Parameters["id"].Ref; ref == nil || ref.Identifier != "#/components/parameters/id" {
		t.Errorf("expected the parameter $ref to be kept, got %+v", ref)
	}

	if ref := c.Messages["n"].Value.Payload.Ref; ref == nil || ref.Identifier != "#/components/schemas/N" {
		t.Errorf("expected the $ref'd channel's message payload to be promoted as N, got %+v", ref)
	}
}

func TestFlatten_PromotedNamesAvoidCollisions(t *testing.T) {
	doc := loadDoc(t, `{
		"asyncapi": "3.0.0",
		"info": {"title": "test", "version": "0.0.1"},
		"channels": {
			"c": {
				"address": "c/{id}",
				"parameters": {"id": {"description": "inline"}},
				"messages": {"m": {"correlationId": {"location": "$message.payload#/inline"}}}
			}
		},
		"operations": {
			"op": {"action": "receive", "channel": {"$ref": "#/channels/c"}}
		},
		"components": {
			"parameters": {"id": {"description": "existing"}},
			"messages": {"m": {"payload": {"type": "string"}}},
			"correlationIds": {"M2CorrelationId": {"location": "$message.payload#/existing"}}
		}
	}`)

	if err := flatten.Document(doc); err != nil {
		t.Fatalf("Document: %v", err)
	}

	c := doc.Components

	if got := c.Parameters["id2"]; got == nil || got.Value.Description != "inline" {
		t.Errorf("expected the inline parameter as id2, got %+v", got)
	}

	if got := c.Parameters["id"].Value.Description; got != "existing" {
		t.Errorf("existing parameter overwritten: %q", got)
	}

	if _, ok := c.Messages["m2"]; !ok {
		t.Error("expected the inline message as m2")
	}

	if c.Messages["m"].Value.Payload == nil {
		t.Error("existing message overwritten")
	}

	ch := doc.Channels["c"].Value
	if ref := ch.Messages["m"].Ref; ref == nil || ref.Identifier != "#/components/messages/m2" {
		t.Errorf("expected the channel message to point to m2, got %+v", ref)
	}

	if ref := ch.Messages["m"].Value.CorrelationID.Ref; ref == nil || ref.Identifier != "#/components/correlationIds/M2CorrelationId2" {
		t.Errorf("expected the correlation ID named from the promoted message, clear of the existing one, got %+v", ref)
	}

	if got := c.CorrelationIDs["M2CorrelationId"].Value.Location; got != "$message.payload#/existing" {
		t.Errorf("existing correlation ID overwritten: %q", got)
	}
}
