AsyncAPI allows schemas, messages, and channel parameters to be defined inline
anywhere they are used. While convenient for small specs, deeply nested inline
definitions make large specs harder to read, harder to reuse, and harder to
generate consistent client code from. Flattening gives every meaningful type a
name and a single canonical location.

This matters most upstream of code generation: a generator can only emit a
named, reusable Go type for a schema that *has* a name.
[`asyncapi-codegen`](https://github.com/MarkRosemaker/asyncapi-codegen) has its
own, narrower version of exactly this promotion built in — see its
`FromComponentSchemas` doc comment — because this package did not exist yet
when that gap first needed closing. Running specs through `asyncapi-flatten`
first is meant to make that internal logic unnecessary, the same way
[`openapi-flatten`](https://github.com/MarkRosemaker/openapi-flatten) already
lets [`openapi-codegen`](https://github.com/MarkRosemaker/openapi-codegen)
assume every type it sees already has a name.
