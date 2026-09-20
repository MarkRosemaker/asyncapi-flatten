package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/MarkRosemaker/asyncapi"
	flatten "github.com/MarkRosemaker/asyncapi-flatten"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "asyncapi-flatten: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	var specPath string
	flag.StringVar(&specPath, "spec", "api/asyncapi.json", "path to AsyncAPI spec file")
	flag.Parse()

	doc, err := asyncapi.LoadFromFile(specPath)
	if err != nil {
		return err
	}

	wasValid := doc.Validate() == nil

	if err := flatten.Document(doc); err != nil {
		return err
	}

	// Sort components alphabetically so the promoted names read as a stable,
	// tidy list rather than whatever order traversal happened to find them
	// in — flatten.Document itself leaves that choice to the caller.
	doc.Components.SortMaps()

	if wasValid {
		if err := doc.Validate(); err != nil {
			return fmt.Errorf("produced invalid doc: %w", err)
		}
	}

	if err := doc.WriteToFile(specPath); err != nil {
		return err
	}

	return nil
}
