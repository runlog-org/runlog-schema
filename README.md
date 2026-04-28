# Runlog Schema — Contract Shared Between Verifier, Server, and Skills

> Part of the **[Runlog](https://github.com/runlog-org)** project — see the [project home](https://github.com/runlog-org) for the overview.

**Repo:** [`runlog-org/runlog-schema`](https://github.com/runlog-org/runlog-schema) — public, Apache-2.0
**Content:** YAML + JSON Schema
**Role:** the contract that pins what entries, cassettes, signed bundles, and session manifests look like. Pinned to a specific version by [`runlog-verifier`](https://github.com/runlog-org/runlog-verifier) and the server.

Single source of truth for:

- `entry.schema.yaml` — submission YAML structure (§7.3)
- `cassette.schema.yaml` — integration cassette shape (§7.5)
- `signed-bundle.schema.yaml` — what the verifier produces (§5.3)
- `session-manifest.schema.yaml` — dependency manifest an agent tags into its working context (§6.2)
- `placeholders.yaml` — registered `$PLACEHOLDER` vocabulary (§7.2)

Semver-versioned. Downstream consumers ([`runlog-verifier`](https://github.com/runlog-org/runlog-verifier), [`runlog`](https://github.com/runlog-org/runlog) (server), [`runlog-skills`](https://github.com/runlog-org/runlog-skills)) pin to a specific schema version. Backward-incompatible changes require a major bump and coordinated release across consumers.

## Distribution

- **Go module** — the repo root is itself the Go module `github.com/runlog-org/runlog-schema`. Consumers do `import "github.com/runlog-org/runlog-schema"` and call `EntrySchemaYAML()` / `ManifestSchemaYAML()` (or the `…JSON` variants for libraries that don't accept YAML). See [`schema.go`](./schema.go) and [`DECISIONS.md`](./DECISIONS.md) §"Distribution: Go module layout" for the layout rationale.
- **Python package** — [`generators/python/`](./generators/python/) holds the publishable Python distribution `runlog-schema`, mirroring the Go API (`entry_schema_yaml()`, `manifest_schema_yaml()`, `entry_schema_json()`, `manifest_schema_json()`, `schema_version()`, `SCHEMA_VERSION`). Built with `hatchling`; bundles a CI-gated copy of the canonical YAML files under `runlog_schema/_data/`. PyPI publish is a separate slice. See [`generators/python/README.md`](./generators/python/README.md) and [`DECISIONS.md`](./DECISIONS.md) §"Distribution: Python package layout (F28c)".
