```bash
go get -tool github.com/MarkRosemaker/asyncapi-flatten/cmd/asyncapi-flatten
```

or

```bash
go get github.com/MarkRosemaker/asyncapi-flatten
```


```go
import (
    "github.com/MarkRosemaker/asyncapi"
    flatten "github.com/MarkRosemaker/asyncapi-flatten"
)

// Load an AsyncAPI document (JSON or YAML)
doc, err := asyncapi.LoadFromDataJSON(jsonBytes)
if err != nil {
    log.Fatal(err)
}

// Flatten all inline definitions
if err := flatten.Document(doc); err != nil {
    log.Fatal(err)
}

// doc now has no nested inline objects — only $ref pointers
```

`Document` is the entire public API. It modifies the document in place.
