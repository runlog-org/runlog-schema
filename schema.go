// Package runlogschema exposes the canonical Runlog schema files (entry +
// session manifest) as embedded byte slices, with optional JSON conversion
// for jsonschema libraries that don't accept YAML directly.
//
// The schemas themselves are authored as YAML — see entry.schema.yaml and
// manifest.schema.yaml at the module root. This package wraps them so Go
// consumers (today: runlog-verifier; tomorrow: any other tool that pins
// the contract) can `import "github.com/runlog-org/runlog-schema"`
// instead of doing filesystem-path arithmetic against a sibling clone.
//
// The wrapper is intentionally thin: a one-line accessor for the raw
// YAML, a one-line accessor for the JSON-normalized form, and the
// schema version. Callers do their own jsonschema compilation.
package runlogschema

import (
	_ "embed"
	"encoding/json"
	"errors"
	"sync"

	"gopkg.in/yaml.v3"
)

// SchemaVersionConst is the current schema version exposed by SchemaVersion.
//
// Source of truth today: this constant. Both schema files carry a
// versioned $id (".../v1.json"), so v1 is the honest current value.
//
// TODO(F31): once the release-train work lands, derive this from a
// VERSION file at the repo root rather than a hardcoded constant so the
// release-train can bump it atomically alongside the schema $id changes.
const SchemaVersionConst = "0.1.0"

//go:embed entry.schema.yaml
var entrySchemaYAML []byte

//go:embed manifest.schema.yaml
var manifestSchemaYAML []byte

// EntrySchemaYAML returns the canonical entry schema as raw YAML bytes,
// embedded at build time. The returned slice is a fresh copy so callers
// can mutate it without affecting the embedded data; the underlying
// content is stable across calls and safe to treat as a constant.
func EntrySchemaYAML() []byte {
	return cloneBytes(entrySchemaYAML)
}

// ManifestSchemaYAML returns the canonical session-manifest schema as
// raw YAML bytes, embedded at build time. See EntrySchemaYAML for
// mutation semantics.
func ManifestSchemaYAML() []byte {
	return cloneBytes(manifestSchemaYAML)
}

// cloneBytes returns a fresh copy of src. Used by every accessor that
// hands embedded schema bytes back to callers, so a caller mutating the
// returned slice can never poison the embedded data for subsequent
// callers in the same process.
func cloneBytes(src []byte) []byte {
	out := make([]byte, len(src))
	copy(out, src)
	return out
}

// SchemaVersion returns the current schema version string. Today this
// is a hardcoded constant matching the versioned $id in the YAML files
// (v1 → "0.1.0" as the first publishable Go-module tag). See
// SchemaVersionConst for the F31 follow-up to source this from a
// VERSION file.
func SchemaVersion() string {
	return SchemaVersionConst
}

// jsonCache lazily holds the JSON-normalized form of one embedded
// schema. The conversion is deterministic, so the cached value is
// stable for the lifetime of the process; we pay the YAML→JSON
// round-trip once per schema.
type jsonCache struct {
	once sync.Once
	data []byte
	err  error
}

// get returns a fresh copy of the cached JSON bytes, computing them on
// first call from the supplied YAML source. The returned slice is a
// fresh copy on every successful call so callers can mutate it without
// poisoning the cache.
func (c *jsonCache) get(src []byte) ([]byte, error) {
	c.once.Do(func() {
		c.data, c.err = yamlBytesToJSON(src)
	})
	if c.err != nil {
		return nil, c.err
	}
	return cloneBytes(c.data), nil
}

var (
	entryJSONCache    jsonCache
	manifestJSONCache jsonCache
)

// EntrySchemaJSON returns the entry schema normalized to JSON bytes.
// Useful for jsonschema libraries (e.g. santhosh-tekuri/jsonschema/v5)
// that compile from JSON rather than YAML. The result is cached after
// the first successful call; the returned slice is a fresh copy.
func EntrySchemaJSON() ([]byte, error) {
	return entryJSONCache.get(entrySchemaYAML)
}

// ManifestSchemaJSON returns the manifest schema normalized to JSON
// bytes. See EntrySchemaJSON for caching and copy semantics.
func ManifestSchemaJSON() ([]byte, error) {
	return manifestJSONCache.get(manifestSchemaYAML)
}

// yamlBytesToJSON parses YAML and re-encodes as canonical JSON. The
// jsonschema/v5 compiler and json.Decoder both expect JSON-native
// values (string-keyed maps, slices, JSON primitives); routing through
// json.Marshal guarantees that shape regardless of yaml.v3 quirks.
//
// Mirrors the same helper in runlog-verifier's schema_oracle_test.go so
// migrating that consumer to this package later (F28b) is a drop-in.
func yamlBytesToJSON(data []byte) ([]byte, error) {
	var node any
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, err
	}
	normalized, err := normalizeYAMLForJSON(node)
	if err != nil {
		return nil, err
	}
	return json.Marshal(normalized)
}

// normalizeYAMLForJSON converts a yaml.v3-decoded value into something
// json.Marshal accepts: map[any]any → map[string]any with stringified
// keys, slices recurse, scalars pass through.
//
// yaml.v3 normally produces map[string]any for object decode, but
// document-level edge cases (anchors, tagged !!map nodes, integer keys)
// can leak any-keyed maps; this is a defensive guard.
func normalizeYAMLForJSON(v any) (any, error) {
	switch m := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			nv, err := normalizeYAMLForJSON(val)
			if err != nil {
				return nil, err
			}
			out[k] = nv
		}
		return out, nil
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			ks, ok := k.(string)
			if !ok {
				return nil, errors.New("non-string map key in YAML; not representable as JSON object")
			}
			nv, err := normalizeYAMLForJSON(val)
			if err != nil {
				return nil, err
			}
			out[ks] = nv
		}
		return out, nil
	case []any:
		out := make([]any, len(m))
		for i, val := range m {
			nv, err := normalizeYAMLForJSON(val)
			if err != nil {
				return nil, err
			}
			out[i] = nv
		}
		return out, nil
	default:
		return v, nil
	}
}
