package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MarkRosemaker/asyncapi"
)

const spec = `{
	"asyncapi": "3.0.0",
	"info": {"title": "test", "version": "0.0.1"},
	"channels": {
		"c": {
			"address": "/test",
			"messages": {
				"event": {
					"payload": {
						"type": "object",
						"properties": {
							"zeta": {"type": "string", "enum": ["z"]},
							"alpha": {"type": "string", "enum": ["a"]}
						}
					}
				}
			}
		}
	},
	"operations": {
		"op": {"action": "receive", "channel": {"$ref": "#/channels/c"}}
	}
}`

func writeSpec(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestRun_FlattensInPlace(t *testing.T) {
	path := writeSpec(t, "asyncapi.json", spec)

	var stderr bytes.Buffer
	if err := run(t.Context(), []string{"-spec", path}, &stderr); err != nil {
		t.Fatalf("run: %v\n%s", err, stderr.String())
	}

	if stderr.Len() > 0 {
		t.Errorf("unexpected stderr: %s", stderr.String())
	}

	doc, err := asyncapi.LoadFromFile(path)
	if err != nil {
		t.Fatalf("written file does not load: %v", err)
	}

	if ref := doc.Components.Messages["event"]; ref == nil {
		t.Fatal("expected the inline message to be promoted in the written file")
	}

	// Traversal finds EventZeta before EventAlpha; the file lists components alphabetically.
	var names []string
	for name := range doc.Components.Schemas.ByIndex() {
		names = append(names, name)
	}

	if want := []string{"Event", "EventAlpha", "EventZeta"}; strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("component schemas in file order: got %q, want %q", names, want)
	}
}

func TestRun_YAML(t *testing.T) {
	path := writeSpec(t, "asyncapi.yaml", spec) // JSON is valid YAML

	var stderr bytes.Buffer
	if err := run(t.Context(), []string{"-spec", path}, &stderr); err != nil {
		t.Fatalf("run: %v\n%s", err, stderr.String())
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.HasPrefix(bytes.TrimSpace(out), []byte("{")) {
		t.Errorf("expected a .yaml spec to be written back as YAML, got:\n%s", out)
	}

	if !bytes.Contains(out, []byte("$ref: '#/components/schemas/Event'")) &&
		!bytes.Contains(out, []byte(`$ref: "#/components/schemas/Event"`)) {
		t.Errorf("expected the payload $ref in the YAML output, got:\n%s", out)
	}
}

func TestRun_DefaultSpecPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "api"), 0o755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "api", "asyncapi.json")
	if err := os.WriteFile(path, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Chdir(dir)

	var stderr bytes.Buffer
	if err := run(t.Context(), nil, &stderr); err != nil {
		t.Fatalf("run: %v\n%s", err, stderr.String())
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(out, []byte(`"#/components/messages/event"`)) {
		t.Errorf("expected api/asyncapi.json to be flattened, got:\n%s", out)
	}
}

func TestRun_InvalidInputIsStillFlattened(t *testing.T) {
	// No info.title: invalid before and after, so the output is not validated.
	path := writeSpec(t, "asyncapi.json", strings.Replace(spec, `"title": "test", `, "", 1))

	var stderr bytes.Buffer
	if err := run(t.Context(), []string{"-spec", path}, &stderr); err != nil {
		t.Fatalf("run: %v\n%s", err, stderr.String())
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(out, []byte(`"EventAlpha"`)) {
		t.Errorf("expected the invalid spec to be flattened anyway, got:\n%s", out)
	}
}

func TestRun_Errors(t *testing.T) {
	for _, tc := range []struct {
		name       string
		args       func(t *testing.T) []string
		wantStderr string
	}{
		{
			name:       "missing file",
			args:       func(t *testing.T) []string { return []string{"-spec", filepath.Join(t.TempDir(), "nope.json")} },
			wantStderr: "asyncapi-flatten: open ",
		},
		{
			name:       "unsupported extension",
			args:       func(t *testing.T) []string { return []string{"-spec", writeSpec(t, "asyncapi.txt", spec)} },
			wantStderr: "asyncapi-flatten: unsupported file extension: .txt",
		},
		{
			name: "flatten error",
			args: func(t *testing.T) []string {
				return []string{"-spec", writeSpec(t, "asyncapi.json", strings.Replace(spec, `"type": "object"`, `"type": "whatever"`, 1))}
			},
			wantStderr: `asyncapi-flatten: channels["c"].messages["event"].payload: unimplemented schema type "whatever"`,
		},
		{
			name:       "unknown flag",
			args:       func(*testing.T) []string { return []string{"-nope"} },
			wantStderr: "flag provided but not defined: -nope",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stderr bytes.Buffer
			if err := run(t.Context(), tc.args(t), &stderr); err == nil {
				t.Fatal("expected an error")
			}

			if !strings.Contains(stderr.String(), tc.wantStderr) {
				t.Errorf("stderr does not contain %q:\n%s", tc.wantStderr, stderr.String())
			}

			if !strings.Contains(stderr.String(), "-spec string") {
				t.Errorf("expected usage on stderr:\n%s", stderr.String())
			}
		})
	}
}

func TestRun_FlattenErrorLeavesFileUntouched(t *testing.T) {
	content := strings.Replace(spec, `"type": "object"`, `"type": "whatever"`, 1)
	path := writeSpec(t, "asyncapi.json", content)

	if err := run(t.Context(), []string{"-spec", path}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected an error")
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(out) != content {
		t.Errorf("file changed despite the error:\n%s", out)
	}
}

func TestRun_Help(t *testing.T) {
	var stderr bytes.Buffer
	if err := run(t.Context(), []string{"-h"}, &stderr); err != nil {
		t.Fatalf("-h should succeed, got %v", err)
	}

	if !strings.Contains(stderr.String(), "path to AsyncAPI spec file") {
		t.Errorf("expected usage on stderr:\n%s", stderr.String())
	}
}
