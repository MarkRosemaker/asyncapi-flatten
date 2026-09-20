package flatten_test

import (
	"bytes"
	"embed"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/MarkRosemaker/asyncapi"
	flatten "github.com/MarkRosemaker/asyncapi-flatten"
)

//go:embed testdata
var testdata embed.FS

func TestFlatten_TestData(t *testing.T) {
	entries, err := testdata.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range entries {
		t.Run(tc.Name(), func(t *testing.T) {
			f, err := testdata.Open(filepath.Join("testdata", tc.Name(), "api", "asyncapi.json"))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close() //nolint

			doc, err := asyncapi.LoadFromReader(f)
			if err != nil {
				t.Fatal(err)
			}

			for it := range 3 {
				t.Run(fmt.Sprintf("iteration %d", it+1), func(t *testing.T) {
					if err := flatten.Document(doc); err != nil {
						t.Fatal(err)
					}

					if err := doc.Validate(); err != nil {
						t.Fatal(err)
					}

					gotDoc, err := doc.ToJSON()
					if err != nil {
						t.Fatal(err)
					}

					wantDoc, err := testdata.ReadFile(filepath.Join("testdata", tc.Name(), "api", "golden.json"))
					if err != nil {
						t.Fatal(err)
					}

					compareBytes(t, wantDoc, gotDoc)
				})
			}
		})
	}
}

// compareBytes prints a compact diff of two byte slices.
func compareBytes(t *testing.T, expected, actual []byte) {
	t.Helper()

	if bytes.Equal(expected, actual) {
		return
	}

	i := 0
	for i < len(expected) && i < len(actual) && expected[i] == actual[i] {
		i++
	}

	t.Errorf("\n┌─ Diff at offset %d\n│ Expected: %q\n│ Actual:   %q\n└─ %s",
		i, expected[i:min(len(expected), i+20)], actual[i:min(len(actual), i+20)],
		func() string {
			if len(expected) != len(actual) {
				return fmt.Sprintf("length %d vs %d", len(expected), len(actual))
			}
			return fmt.Sprintf("0x%02x vs 0x%02x", expected[i], actual[i])
		}())
}
