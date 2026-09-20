## What gets flattened

### Schemas

Inline schemas are moved to `components/schemas` when they contain meaningful structure. Simple scalar types (`integer`, `number`, `boolean`, plain `string`) stay inline to keep the spec readable. A schema is moved when it is:

- an **object** with properties
- a **string** (or array of strings) with `enum` values
- an **array of objects**
- in a schema format other than AsyncAPI's own (Avro, Protobuf, ...) — there is no structure to inspect, so it always gets a name

Schemas inside `allOf`, `oneOf`, and `anyOf` are never moved on their own, because they exist solely to compose a larger type. Each is still recursed into, so an object or enum nested inside one of them is still promoted.

Promotion recurses: a nested object's own properties are built the same way, so an enum or object any number of levels deep still gets a name, not just one directly on a message payload.

**Before** (adapted from a real fixture — see `testdata/private-file-vault`):

```json
{
  "components": {
    "messages": {
      "verifyProgress": {
        "payload": {
          "type": "object",
          "required": ["job_id", "status"],
          "properties": {
            "job_id": { "type": "string", "format": "uuid" },
            "status": { "type": "string", "enum": ["running", "complete", "failed"] },
            "result": {
              "type": "object",
              "required": ["status"],
              "properties": {
                "status": { "type": "string", "enum": ["match", "mismatch", "unknown"] }
              }
            }
          }
        }
      }
    }
  }
}
```

**After:**

```json
{
  "components": {
    "messages": {
      "verifyProgress": {
        "payload": { "$ref": "#/components/schemas/VerifyProgress" }
      }
    },
    "schemas": {
      "VerifyProgress": {
        "type": "object",
        "required": ["job_id", "status"],
        "properties": {
          "job_id": { "type": "string", "format": "uuid" },
          "status": { "$ref": "#/components/schemas/VerifyProgressStatus" },
          "result": { "$ref": "#/components/schemas/VerifyProgressResult" }
        }
      },
      "VerifyProgressStatus": { "type": "string", "enum": ["running", "complete", "failed"] },
      "VerifyProgressResult": {
        "type": "object",
        "required": ["status"],
        "properties": {
          "status": { "$ref": "#/components/schemas/VerifyProgressResultStatus" }
        }
      },
      "VerifyProgressResultStatus": { "type": "string", "enum": ["match", "mismatch", "unknown"] }
    }
  }
}
```

### Messages

Inline messages — defined directly in a channel's `messages` map rather than `$ref`'d to `components/messages` — are moved there, keyed by the same name they had in the channel. A message's own `payload`, `headers`, and `correlationId` are then flattened the same way, named from it: a message promoted as `verifyProgress` gives its payload the name `VerifyProgress`, its headers (if it has any worth naming) `VerifyProgressHeaders`, and its correlation ID `VerifyProgressCorrelationId`.

### Channel parameters

Inline parameters in a channel's `address` — the `{userId}` in `user/{userId}/events`, for example — are moved to `components/parameters`, keyed by the parameter's own name. A Parameter Object has no schema or content of its own to recurse into, so this is the entire step for one.

### Correlation IDs

A message's inline `correlationId` is moved to `components/correlationIds`, named `{Message}CorrelationId`. Like Parameter, a Correlation ID Object is a leaf value with nothing further to flatten.

## Name generation

All names are converted to Go-style PascalCase (e.g., `verify progress result status` → `VerifyProgressResultStatus`) — matching the convention the rest of this family already uses for a message's own name. If the generated name is already taken, a numeric suffix is appended (`Name2`, `Name3`, …) to avoid collisions.

## Error reporting

Errors include the full JSON path to the offending field, powered by [`errpath`](https://github.com/MarkRosemaker/errpath):

```
channels["userEvents"].messages["event"].payload: unimplemented schema type "whatever"
```

## The asyncapi family

| Module | Purpose |
|---|---|
| [asyncapi](https://github.com/MarkRosemaker/asyncapi) | Parse, validate, and write AsyncAPI 3.x specifications |
| **asyncapi-flatten** (this module) | Promote inline definitions into named `components` entries |
| [asyncapi-enrich](https://github.com/MarkRosemaker/asyncapi-enrich) | Infer specification content from observed WebSocket traffic |
| [asyncapi-codegen](https://github.com/MarkRosemaker/asyncapi-codegen) | Generate Go types, clients, and servers from a specification |

There is no `asyncapi-compress` in this family yet to collapse the near-duplicate
components flattening naturally produces (the same shape promoted from two
places gets two names) — see
[`openapi-compress`](https://github.com/MarkRosemaker/openapi-compress), which
plays that role for [`openapi-flatten`](https://github.com/MarkRosemaker/openapi-flatten)
in the OpenAPI family this one mirrors.
