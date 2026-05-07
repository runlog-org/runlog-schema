package runlogschema

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
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
			doc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(data)))
			if err != nil {
				t.Fatalf("UnmarshalJSON: %v", err)
			}
			if err := compiler.AddResource(tc.url, doc); err != nil {
				t.Fatalf("AddResource: %v", err)
			}
			if _, err := compiler.Compile(tc.url); err != nil {
				t.Fatalf("Compile: %v", err)
			}
		})
	}
}

// TestBranchActionOneOfDisambiguation pins the structural disambiguation
// of `branch.action`'s 3-way oneOf. Arm 1 (single step_fragment) inherits
// `additionalProperties: true`, so without the `not: {required: [steps]}`
// guard a document carrying both a `type` and a `steps` array would
// silently pass arm 1 by treating `steps` as an unrecognized extra.
// The schema enforces that arm 1 cannot claim a document that has
// `steps`; arm 3's `additionalProperties: false` already forbids `type`
// on its side. This test fails loudly if a future refactor drops the
// guard, restoring the silent-failure path.
func TestBranchActionOneOfDisambiguation(t *testing.T) {
	data, err := EntrySchemaJSON()
	if err != nil {
		t.Fatalf("EntrySchemaJSON: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	url := "https://runlog.org/schemas/entry/v1.json"
	if err := compiler.AddResource(url, doc); err != nil {
		t.Fatalf("AddResource: %v", err)
	}
	root, err := compiler.Compile(url)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	branch := root.DynamicRef
	_ = branch // appease unused-var if dynamic-ref path differs
	// Compile the branch sub-schema directly via $ref.
	branchURL := url + "#/$defs/branch"
	branchSchema, err := compiler.Compile(branchURL)
	if err != nil {
		t.Fatalf("Compile branch: %v", err)
	}

	mkBranch := func(action any) map[string]any {
		return map[string]any{
			"description": strings.Repeat("x", 40),
			"action":      action,
			"assertion": map[string]any{
				"type":   "status",
				"expect": "success",
			},
		}
	}

	cases := []struct {
		name     string
		action   any
		validate bool
	}{
		{
			name:     "arm1 single step_fragment",
			action:   map[string]any{"type": "code", "body": "print('x')"},
			validate: true,
		},
		{
			name: "arm2 array of step_fragments",
			action: []any{
				map[string]any{"type": "code", "body": "print('x')"},
				map[string]any{"type": "shell", "body": "ls"},
			},
			validate: true,
		},
		{
			name: "arm3 action_steps_block",
			action: map[string]any{
				"steps": []any{
					map[string]any{"id": "main", "type": "code", "body": "print('x')"},
				},
			},
			validate: true,
		},
		{
			name: "ambiguous shape (type + steps) rejected",
			action: map[string]any{
				"type": "code",
				"body": "print('x')",
				"steps": []any{
					map[string]any{"id": "main", "type": "code"},
				},
			},
			validate: false,
		},
		{
			name: "ambiguous shape inside arm2 array rejected",
			action: []any{
				map[string]any{
					"type": "code",
					"body": "print('x')",
					"steps": []any{
						map[string]any{"id": "main", "type": "code"},
					},
				},
			},
			validate: false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := branchSchema.Validate(mkBranch(tc.action))
			gotValid := err == nil
			if gotValid != tc.validate {
				t.Errorf("validate=%v want=%v err=%v", gotValid, tc.validate, err)
			}
		})
	}
}

// TestMutationCassetteResponseFieldOrAction pins the F76 split:
// `mutate_cassette_response` accepts the response-field selector under
// either `field` (preferred, schema 0.4.1+) or `action` (legacy,
// pre-0.4.1). Both must validate so in-flight seeds keep working while
// new seeds migrate to `field`. A future major bump may retire `action`
// for this strategy; until then both shapes round-trip.
func TestMutationCassetteResponseFieldOrAction(t *testing.T) {
	data, err := EntrySchemaJSON()
	if err != nil {
		t.Fatalf("EntrySchemaJSON: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	url := "https://runlog.org/schemas/entry/v1.json"
	if err := compiler.AddResource(url, doc); err != nil {
		t.Fatalf("AddResource: %v", err)
	}
	mutSchema, err := compiler.Compile(url + "#/$defs/mutation")
	if err != nil {
		t.Fatalf("Compile mutation: %v", err)
	}

	mk := func(extra map[string]any) map[string]any {
		m := map[string]any{
			"strategy":        "mutate_cassette_response",
			"target":          "step-1",
			"new_value":       "200",
			"expected_result": "fail",
		}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}

	cases := []struct {
		name     string
		extra    map[string]any
		validate bool
	}{
		{
			name:     "field selector (new shape, schema 0.4.1+)",
			extra:    map[string]any{"field": "body"},
			validate: true,
		},
		{
			name:     "action selector (legacy back-compat)",
			extra:    map[string]any{"action": "body"},
			validate: true,
		},
		{
			name:     "header.<NAME> via field",
			extra:    map[string]any{"field": "header.X-RateLimit-Remaining"},
			validate: true,
		},
		{
			name:     "header.<NAME> via legacy action",
			extra:    map[string]any{"action": "header.X-RateLimit-Remaining"},
			validate: true,
		},
		{
			name:     "both field and action set (no validation conflict — consumer prefers field)",
			extra:    map[string]any{"field": "body", "action": "status"},
			validate: true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := mutSchema.Validate(mk(tc.extra))
			gotValid := err == nil
			if gotValid != tc.validate {
				t.Errorf("validate=%v want=%v err=%v", gotValid, tc.validate, err)
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
