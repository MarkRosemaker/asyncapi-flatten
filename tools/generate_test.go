package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

const inputSpec = `{
	"asyncapi": "3.0.0",
	"info": {"title": "test", "version": "0.0.1"},
	"channels": {
		"c": {
			"address": "/test",
			"messages": {"event": {"payload": {"type": "object", "properties": {"a": {"type": "string"}}}}}
		}
	},
	"operations": {
		"op": {"action": "receive", "channel": {"$ref": "#/channels/c"}}
	}
}`

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return b
}

func TestRun_CopiesEnrichOutputAndWritesGolden(t *testing.T) {
	root := t.TempDir()
	testdata := filepath.Join(root, "testdata")
	enrich := filepath.Join(root, "enrich")

	writeFile(t, filepath.Join(enrich, "fixture", "api", "golden.json"), inputSpec)
	writeFile(t, filepath.Join(testdata, "fixture", "api", "asyncapi.json"), "stale")
	writeFile(t, filepath.Join(testdata, "README"), "not a fixture")

	if err := run(testdata, enrich); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := readFile(t, filepath.Join(testdata, "fixture", "api", "asyncapi.json")); string(got) != inputSpec {
		t.Errorf("expected asyncapi-enrich's golden.json copied in as the input, got:\n%s", got)
	}

	golden := readFile(t, filepath.Join(testdata, "fixture", "api", "golden.json"))
	if !bytes.Contains(golden, []byte(`"$ref":"#/components/messages/event"`)) &&
		!bytes.Contains(golden, []byte(`"$ref": "#/components/messages/event"`)) {
		t.Errorf("expected a flattened golden.json, got:\n%s", golden)
	}
}

func TestRun_WithoutEnrichCheckout(t *testing.T) {
	testdata := filepath.Join(t.TempDir(), "testdata")
	writeFile(t, filepath.Join(testdata, "fixture", "api", "asyncapi.json"), inputSpec)

	if err := run(testdata, filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("a missing asyncapi-enrich checkout should be skipped, got %v", err)
	}

	if got := readFile(t, filepath.Join(testdata, "fixture", "api", "asyncapi.json")); string(got) != inputSpec {
		t.Errorf("input changed without an asyncapi-enrich checkout:\n%s", got)
	}

	if _, err := os.Stat(filepath.Join(testdata, "fixture", "api", "golden.json")); err != nil {
		t.Errorf("expected golden.json to be written: %v", err)
	}
}

func TestRun_Errors(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, testdata, enrich string)
	}{
		{"no testdata", func(*testing.T, string, string) {}},
		{"unloadable fixture", func(t *testing.T, testdata, _ string) {
			writeFile(t, filepath.Join(testdata, "fixture", "api", "asyncapi.json"), "{")
		}},
		{"unflattenable fixture", func(t *testing.T, testdata, _ string) {
			writeFile(t, filepath.Join(testdata, "fixture", "api", "asyncapi.json"),
				string(bytes.Replace([]byte(inputSpec), []byte(`"type": "object"`), []byte(`"type": "whatever"`), 1)))
		}},
		{"enrich fixture without golden.json", func(t *testing.T, testdata, enrich string) {
			writeFile(t, filepath.Join(testdata, "fixture", "api", "asyncapi.json"), inputSpec)
			writeFile(t, filepath.Join(enrich, "fixture", "README"), "")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			testdata, enrich := filepath.Join(root, "testdata"), filepath.Join(root, "enrich")
			tc.setup(t, testdata, enrich)

			if err := run(testdata, enrich); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
