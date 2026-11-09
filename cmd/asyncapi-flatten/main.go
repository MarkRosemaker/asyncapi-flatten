package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/MarkRosemaker/asyncapi"
	flatten "github.com/MarkRosemaker/asyncapi-flatten"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stderr); err != nil {
		os.Exit(1)
	}
}

// run reports its own errors to stderr; the error it returns only sets the exit code.
func run(_ context.Context, args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("asyncapi-flatten", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var specPath string
	fs.StringVar(&specPath, "spec", "api/asyncapi.json", "path to AsyncAPI spec file")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return err // Parse has already printed the error and usage
	}

	if err := flattenFile(specPath); err != nil {
		_, _ = fmt.Fprintf(stderr, "asyncapi-flatten: %v\n", err)

		fs.Usage()

		return err
	}

	return nil
}

func flattenFile(specPath string) error {
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

	return doc.WriteToFile(specPath)
}
