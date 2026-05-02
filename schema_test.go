package runlogschema

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// TestEntrySchemaYAMLNonEmpty guards the //go:embed wiring: an empty
// slice means the file got renamed without updating the embed
// directive, or the build was run from a directory where the file is
// invisible.
func TestEntrySchemaYAMLNonEmpty(t *testing.T) {
	got := EntrySchemaYAML()
	if len(got) == 0 {
		t.Fatal("EntrySchemaYAML returned empty bytes; //go:embed not wired up?")
	}
	if !strings.HasPrefix(string(got), "$schema:") {
		t.Errorf("entry schema YAML doesn't start with $schema: declaration; got %q...", firstLine(got))
	}
}

func TestManifestSchemaYAMLNonEmpty(t *testing.T) {
	got := ManifestSchemaYAML()
	if len(got) == 0 {
		t.Fatal("ManifestSchemaYAML returned empty bytes; //go:embed not wired up?")
	}
	if !strings.HasPrefix(string(got), "$schema:") {
		t.Errorf("manifest schema YAML doesn't start with $schema: declaration; got %q...", firstLine(got))
	}
}

// TestEntrySchemaYAMLReturnsCopy verifies the documented mutation
// safety: callers can mutate the returned slice without corrupting the
// embedded data for subsequent calls.
func TestEntrySchemaYAMLReturnsCopy(t *testing.T) {
	a := EntrySchemaYAML()
	if len(a) == 0 {
		t.Fatal("empty result")
	}
	a[0] = 0
	b := EntrySchemaYAML()
	if b[0] == 0 {
		t.Fatal("EntrySchemaYAML returned a shared slice; mutation leaked into embedded data")
	}
}

// TestManifestSchemaYAMLReturnsCopy mirrors the entry-side guard so a
// future refactor that drops cloneBytes from ManifestSchemaYAML can't
// silently regress mutation safety on just one of the two accessors.
func TestManifestSchemaYAMLReturnsCopy(t *testing.T) {
	a := ManifestSchemaYAML()
	if len(a) == 0 {
		t.Fatal("empty result")
	}
	a[0] = 0
	b := ManifestSchemaYAML()
	if b[0] == 0 {
		t.Fatal("ManifestSchemaYAML returned a shared slice; mutation leaked into embedded data")
	}
}

// TestEntrySchemaJSONIsValidJSON checks that the JSON form parses
// cleanly with encoding/json — the basic invariant any downstream
// jsonschema library will assume.
func TestEntrySchemaJSONIsValidJSON(t *testing.T) {
	data, err := EntrySchemaJSON()
	if err != nil {
		t.Fatalf("EntrySchemaJSON: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("EntrySchemaJSON returned empty bytes")
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("EntrySchemaJSON: not valid JSON: %v", err)
	}
	if doc["$schema"] == nil {
		t.Error("entry schema JSON missing $schema key")
	}
	if doc["$id"] == nil {
		t.Error("entry schema JSON missing $id key")
	}
}

func TestManifestSchemaJSONIsValidJSON(t *testing.T) {
	data, err := ManifestSchemaJSON()
	if err != nil {
		t.Fatalf("ManifestSchemaJSON: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("ManifestSchemaJSON returned empty bytes")
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("ManifestSchemaJSON: not valid JSON: %v", err)
	}
	if doc["$schema"] == nil {
		t.Error("manifest schema JSON missing $schema key")
	}
}

// TestSchemasAreValidDraft2020Schemas pulls in the same compiler the
// verifier uses and asserts both schemas are well-formed JSON Schema
// Draft 2020-12. Mirrors the Python `Draft202012Validator.check_schema`
// gate that runs in CI on every PR.
func TestSchemasAreValidDraft2020Schemas(t *testing.T) {
	cases := []struct {
		name string
		url  string
		fn   func() ([]byte, error)
	}{
		{"entry", "https://runlog.org/schemas/entry/v1.json", EntrySchemaJSON},
		{"manifest", "https://runlog.org/schemas/manifest/v1.json", ManifestSchemaJSON},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.fn()
			if err != nil {
				t.Fatalf("get JSON: %v", err)
			}
			compiler := jsonschema.NewCompiler()
			if err := compiler.AddResource(tc.url, strings.NewReader(string(data))); err != nil {
				t.Fatalf("AddResource: %v", err)
			}
			if _, err := compiler.Compile(tc.url); err != nil {
				t.Fatalf("Compile: %v", err)
			}
		})
	}
}

// TestSchemaVersionMatchesConstant pins the accessor to the constant,
// which itself sources from the embedded VERSION file. Mirrors
// test_schema_version_matches_constant on the Python side.
func TestSchemaVersionMatchesConstant(t *testing.T) {
	if got, want := SchemaVersion(), SchemaVersionConst; got != want {
		t.Errorf("SchemaVersion()=%q, SchemaVersionConst=%q; the two must agree", got, want)
	}
}

// TestSchemaVersionShape sanity-checks that SchemaVersion returns
// something that looks like a version string. We deliberately accept
// both bare semver ("1.2.3") and tag-prefixed ("v1.2.3") forms so a
// future move to a VERSION file doesn't have to fight this test.
func TestSchemaVersionShape(t *testing.T) {
	v := SchemaVersion()
	if v == "" {
		t.Fatal("SchemaVersion returned empty string")
	}
	semverish := regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[A-Za-z0-9.-]+)?$`)
	if !semverish.MatchString(v) {
		t.Errorf("SchemaVersion=%q doesn't look like a semver-ish version", v)
	}
}

func firstLine(b []byte) string {
	if i := strings.IndexByte(string(b), '\n'); i >= 0 {
		return string(b[:i])
	}
	return string(b)
}
