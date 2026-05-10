package runlogschema

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// schemaAccessor names one of the two embedded schemas plus its YAML/JSON
// accessor pair. Used to table-drive the parallel YAML-/JSON-side guards
// below so a refactor that adds a third schema file (or drops one of the
// existing accessors) can't accidentally leave half the invariants
// unchecked.
type schemaAccessor struct {
	name         string // "entry" / "manifest" — the test subtest label
	yamlFn       func() []byte
	jsonFn       func() ([]byte, error)
	requireID    bool   // both schemas declare $id today; the flag stays so a future schema without one can opt out
	yamlAccessor string // "EntrySchemaYAML" — used in error messages so failures point at the right symbol
	jsonAccessor string // "EntrySchemaJSON" — same
}

var schemaAccessors = []schemaAccessor{
	{
		name:         "entry",
		yamlFn:       EntrySchemaYAML,
		jsonFn:       EntrySchemaJSON,
		requireID:    true,
		yamlAccessor: "EntrySchemaYAML",
		jsonAccessor: "EntrySchemaJSON",
	},
	{
		name:         "manifest",
		yamlFn:       ManifestSchemaYAML,
		jsonFn:       ManifestSchemaJSON,
		requireID:    true,
		yamlAccessor: "ManifestSchemaYAML",
		jsonAccessor: "ManifestSchemaJSON",
	},
}

// TestSchemaYAMLNonEmpty guards the //go:embed wiring: an empty slice
// means the file got renamed without updating the embed directive, or
// the build was run from a directory where the file is invisible.
func TestSchemaYAMLNonEmpty(t *testing.T) {
	for _, sa := range schemaAccessors {
		sa := sa
		t.Run(sa.name, func(t *testing.T) {
			got := sa.yamlFn()
			if len(got) == 0 {
				t.Fatalf("%s returned empty bytes; //go:embed not wired up?", sa.yamlAccessor)
			}
			if !strings.HasPrefix(string(got), "$schema:") {
				t.Errorf("%s schema YAML doesn't start with $schema: declaration; got %q...", sa.name, firstLine(got))
			}
		})
	}
}

// TestSchemaYAMLReturnsCopy verifies the documented mutation safety:
// callers can mutate the returned slice without corrupting the embedded
// data for subsequent calls. Run for every accessor so a future refactor
// that drops cloneBytes from one of them can't silently regress mutation
// safety on just that one.
func TestSchemaYAMLReturnsCopy(t *testing.T) {
	for _, sa := range schemaAccessors {
		sa := sa
		t.Run(sa.name, func(t *testing.T) {
			a := sa.yamlFn()
			if len(a) == 0 {
				t.Fatal("empty result")
			}
			a[0] = 0
			b := sa.yamlFn()
			if b[0] == 0 {
				t.Fatalf("%s returned a shared slice; mutation leaked into embedded data", sa.yamlAccessor)
			}
		})
	}
}

// TestSchemaJSONIsValidJSON checks that the JSON form parses cleanly
// with encoding/json — the basic invariant any downstream jsonschema
// library will assume.
func TestSchemaJSONIsValidJSON(t *testing.T) {
	for _, sa := range schemaAccessors {
		sa := sa
		t.Run(sa.name, func(t *testing.T) {
			data, err := sa.jsonFn()
			if err != nil {
				t.Fatalf("%s: %v", sa.jsonAccessor, err)
			}
			if len(data) == 0 {
				t.Fatalf("%s returned empty bytes", sa.jsonAccessor)
			}
			var doc map[string]any
			if err := json.Unmarshal(data, &doc); err != nil {
				t.Fatalf("%s: not valid JSON: %v", sa.jsonAccessor, err)
			}
			if doc["$schema"] == nil {
				t.Errorf("%s schema JSON missing $schema key", sa.name)
			}
			if sa.requireID && doc["$id"] == nil {
				t.Errorf("%s schema JSON missing $id key", sa.name)
			}
		})
	}
}

// schemaURL maps a schema accessor name to its canonical $id URL. Kept
// as a separate table from schemaAccessors because it's only used by the
// jsonschema-compiler tests below; folding it into the accessor struct
// would couple every accessor row to a URL even where it isn't relevant.
var schemaURL = map[string]string{
	"entry":    "https://runlog.org/schemas/entry/v1.json",
	"manifest": "https://runlog.org/schemas/manifest/v1.json",
}

// TestSchemasAreValidDraft2020Schemas pulls in the same compiler the
// verifier uses and asserts both schemas are well-formed JSON Schema
// Draft 2020-12. Mirrors the Python `Draft202012Validator.check_schema`
// gate that runs in CI on every PR.
func TestSchemasAreValidDraft2020Schemas(t *testing.T) {
	for _, sa := range schemaAccessors {
		sa := sa
		t.Run(sa.name, func(t *testing.T) {
			data, err := sa.jsonFn()
			if err != nil {
				t.Fatalf("get JSON: %v", err)
			}
			compiler := jsonschema.NewCompiler()
			doc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(data)))
			if err != nil {
				t.Fatalf("UnmarshalJSON: %v", err)
			}
			url := schemaURL[sa.name]
			if err := compiler.AddResource(url, doc); err != nil {
				t.Fatalf("AddResource: %v", err)
			}
			if _, err := compiler.Compile(url); err != nil {
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
	if _, err := compiler.Compile(url); err != nil {
		t.Fatalf("Compile: %v", err)
	}
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

// TestKbIdPatternMatchesUnitIdPattern pins the byte-identical invariant
// that manifest.schema.yaml#/properties/entries/items/properties/kb_id/pattern
// promises against entry.schema.yaml#/properties/unit_id/pattern. The
// kb_id is the manifest's reference to a Runlog entry's unit_id, so a
// drift here would let the manifest accept identifiers the entry schema
// rejects (or vice versa). Cross-file $ref isn't used because each
// schema is published as an independent artifact under its own $id —
// this test substitutes for the $ref-based equivalence enforcement.
func TestKbIdPatternMatchesUnitIdPattern(t *testing.T) {
	entryJSON, err := EntrySchemaJSON()
	if err != nil {
		t.Fatalf("EntrySchemaJSON: %v", err)
	}
	manifestJSON, err := ManifestSchemaJSON()
	if err != nil {
		t.Fatalf("ManifestSchemaJSON: %v", err)
	}

	var entryDoc, manifestDoc map[string]any
	if err := json.Unmarshal(entryJSON, &entryDoc); err != nil {
		t.Fatalf("unmarshal entry: %v", err)
	}
	if err := json.Unmarshal(manifestJSON, &manifestDoc); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	entryPattern, err := dig[string](entryDoc, "properties", "unit_id", "pattern")
	if err != nil {
		t.Fatalf("entry.unit_id.pattern: %v", err)
	}
	kbIdPattern, err := dig[string](manifestDoc, "properties", "entries", "items", "properties", "kb_id", "pattern")
	if err != nil {
		t.Fatalf("manifest.entries.items.kb_id.pattern: %v", err)
	}
	if entryPattern != kbIdPattern {
		t.Errorf("kb_id/unit_id pattern drift\n  entry.unit_id.pattern    = %q\n  manifest.kb_id.pattern   = %q\nthese MUST stay byte-identical (see manifest.schema.yaml comment)", entryPattern, kbIdPattern)
	}
}

// dig walks a string-keyed nested map and returns the typed leaf value.
// Returns an error describing the missing/wrong-typed segment so test
// failures point at the exact path that drifted, not a generic nil-deref.
func dig[T any](m map[string]any, path ...string) (T, error) {
	var zero T
	cur := any(m)
	for i, key := range path {
		obj, ok := cur.(map[string]any)
		if !ok {
			return zero, fmt.Errorf("at %v: expected object, got %T", path[:i], cur)
		}
		next, ok := obj[key]
		if !ok {
			return zero, fmt.Errorf("missing key %q at %v", key, path[:i])
		}
		cur = next
	}
	v, ok := cur.(T)
	if !ok {
		return zero, fmt.Errorf("at %v: expected %T, got %T", path, zero, cur)
	}
	return v, nil
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
// something that looks like a version string. The current source is
// the embedded VERSION file (bare semver, e.g. "0.4.1"); the regex
// also accepts the tag-prefixed form ("v0.4.1") so the test survives
// any future change to embed `git describe` output instead.
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
