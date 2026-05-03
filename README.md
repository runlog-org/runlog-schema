# Runlog Schema — Contract Shared Between Verifier, Server, and Skills

> Part of the **[Runlog](https://github.com/runlog-org)** project — see the [project home](https://github.com/runlog-org) for the overview.

**Repo:** [`runlog-org/runlog-schema`](https://github.com/runlog-org/runlog-schema) — public, Apache-2.0
**Content:** YAML + JSON Schema (Draft 2020-12)
**Role:** the contract that pins what entries and session manifests look like. Pinned to a specific version by [`runlog-verifier`](https://github.com/runlog-org/runlog-verifier) and the server.

> **About this project:** Runlog is a hobby side project by [Volker Otto](https://volkerotto.net) — not a commercial product today. A paid model is not ruled out for a later stage. See [About this project](https://runlog.org/#about) for the canonical framing.

Single source of truth for:

- [`entry.schema.yaml`](./entry.schema.yaml) — submission YAML structure, including `cassette` (integration cassette shape) and `verification` blocks as nested `$defs`
- [`manifest.schema.yaml`](./manifest.schema.yaml) — session manifest the agent submits with `runlog_report`

Additional contract surfaces (signed-bundle output, registered `$PLACEHOLDER` vocabulary) are documented in the private architecture docs and consumed by the verifier directly; they are not yet split out as standalone schema files in this repo.

Semver-versioned. Downstream consumers ([`runlog-verifier`](https://github.com/runlog-org/runlog-verifier), [`runlog`](https://github.com/runlog-org/runlog) (server), [`runlog-skills`](https://github.com/runlog-org/runlog-skills)) pin to a specific schema version. Backward-incompatible changes require a major bump and coordinated release across consumers.

## Distribution

- **Go module** — the repo root is itself the Go module `github.com/runlog-org/runlog-schema`. Consumers do `import "github.com/runlog-org/runlog-schema"` and call `EntrySchemaYAML()` / `ManifestSchemaYAML()` (or the `…JSON` variants for libraries that don't accept YAML). See [`schema.go`](./schema.go) and [`DECISIONS.md`](./DECISIONS.md) §"Distribution: Go module layout" for the layout rationale.
- **Python package** — [`generators/python/`](./generators/python/) holds the publishable Python distribution `runlog-schema`, mirroring the Go API (`entry_schema_yaml()`, `manifest_schema_yaml()`, `entry_schema_json()`, `manifest_schema_json()`, `schema_version()`, `SCHEMA_VERSION`). Built with `hatchling`; bundles a CI-gated copy of the canonical YAML files under `runlog_schema/_data/`. PyPI publish is a separate slice. See [`generators/python/README.md`](./generators/python/README.md) and [`DECISIONS.md`](./DECISIONS.md) §"Distribution: Python package layout (F28c)".
