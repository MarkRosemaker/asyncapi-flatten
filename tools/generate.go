// Command generate is the entry point for this repository's own dev tooling,
// run via `go generate ./...` (see generate.go).
package main

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/MarkRosemaker/asyncapi"
	flatten "github.com/MarkRosemaker/asyncapi-flatten"
	"github.com/MarkRosemaker/fsutil/osutil"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := copyPreviousStep(); err != nil {
		return err
	}

	entries, err := os.ReadDir("testdata")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		doc, err := asyncapi.LoadFromFile(filepath.Join("testdata", entry.Name(), "api", "asyncapi.json"))
		if err != nil {
			return err
		}

		if err := flatten.Document(doc); err != nil {
			return err
		}

		if err := doc.WriteToFile(filepath.Join("testdata", entry.Name(), "api", "golden.json")); err != nil {
			return err
		}
	}

	return nil
}

// copyPreviousStep copies each fixture's recorded, still-unflattened
// asyncapi-enrich output in as this repository's own input — the same
// pairing asyncapi-codegen's tools/generate.go uses today, kept only until
// asyncapi-codegen starts reading from this repository's own golden.json
// instead (see docs/roadmap.md).
func copyPreviousStep() error {
	const srcDir = "../asyncapi-enrich/testdata"

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}

		return err
	}

	for _, e := range entries {
		if err := osutil.Copy(
			filepath.Join(srcDir, e.Name(), "api", "golden.json"),
			filepath.Join("testdata", e.Name(), "api", "asyncapi.json"),
		); err != nil {
			return err
		}
	}

	return nil
}
