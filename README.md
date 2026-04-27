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

## Generators

- `generators/go/` — emits a Go module for `runlog-verifier` to vendor
- `generators/python/` — emits a PyPI package for `runlog-server` to install
