package flatten_test

import (
	"strings"
	"testing"

	"github.com/MarkRosemaker/asyncapi"
	flatten "github.com/MarkRosemaker/asyncapi-flatten"
)

// TestFlatten_ChannelPromotesEverything exercises the shapes the golden
// fixtures don't happen to use: an inline channel parameter, an inline
// message, and that message's own inline headers and correlationId.
func TestFlatten_ChannelPromotesEverything(t *testing.T) {
	raw := `{
		"asyncapi": "3.0.0",
		"info": {"title": "test", "version": "0.0.1"},
		"channels": {
			"userEvents": {
				"address": "user/{userId}/events",
				"parameters": {"userId": {"description": "the user id"}},
				"messages": {
					"event": {
						"headers": {
							"type": "object",
							"properties": {"priority": {"type": "string", "enum": ["low", "high"]}}
						},
						"correlationId": {"location": "$message.payload#/id"},
						"payload": {
							"type": "object",
							"required": ["id"],
							"properties": {"id": {"type": "string"}}
						}
					}
				}
			}
		},
		"operations": {
			"receiveEvent": {
				"action": "receive",
				"channel": {"$ref": "#/channels/userEvents"},
				"messages": [{"$ref": "#/channels/userEvents/messages/event"}]
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

	ch := doc.Channels["userEvents"].Value

	if ref := ch.Parameters["userId"].Ref; ref == nil {
		t.Error("expected the inline parameter to be replaced with a $ref")
	} else if _, ok := doc.Components.Parameters["userId"]; !ok {
		t.Error("expected the parameter to be promoted to components/parameters as \"userId\"")
	}

	msgRef := ch.Messages["event"]
	if msgRef.Ref == nil {
		t.Error("expected the inline message to be replaced with a $ref")
	}

	if _, ok := doc.Components.Messages["event"]; !ok {
		t.Fatal("expected the message to be promoted to components/messages as \"event\"")
	}

	msg := msgRef.Value

	if ref := msg.Payload.Ref; ref == nil || ref.Identifier != "#/components/schemas/Event" {
		t.Errorf("expected payload to be promoted as Event, got %+v", ref)
	}

	if ref := msg.Headers.Ref; ref == nil || ref.Identifier != "#/components/schemas/EventHeaders" {
		t.Errorf("expected headers to be promoted as EventHeaders, got %+v", ref)
	}

	if ref := msg.CorrelationID.Ref; ref == nil || ref.Identifier != "#/components/correlationIds/EventCorrelationId" {
		t.Errorf("expected correlationId to be promoted as EventCorrelationId, got %+v", ref)
	}

	if _, ok := doc.Components.Schemas["EventHeadersPriority"]; !ok {
		t.Error("expected the nested enum in headers to be promoted as EventHeadersPriority")
	}
}
