# Releasing runlog-schema

This repo ships two distributions from the same tag:

- The Go module `github.com/runlog-org/runlog-schema` (root `go.mod`).
  pkg.go.dev resolves any tag push automatically, so the tag itself is
  the publish — no workflow step needed.
- The Python package `runlog-schema` on PyPI (built from
  `generators/python/`). The [`release`](.github/workflows/release.yml)
  workflow re-validates the schemas on the tag commit (pre-flight),
  then creates a GitHub Release with auto-generated notes and uploads
  the sdist + wheel to PyPI via OIDC trusted publishing.

## Cut a release

1. Make sure CI is green on `main` and you're on it:

       git checkout main && git pull --ff-only

2. Pick a version. Schema is currently 0.x, so additive changes (new
   optional field, new enum value, new schema file) are MINOR or PATCH
   bumps; breaking shape changes (renamed/removed field, tightened
   required-set, narrowed enum) are MAJOR while pre-1.0 we bump MINOR.

   The next release accumulates the breaking T31 + T18 changes already
   on `main` (build_feature_pin → pin_strength rename; differential
   split into closed `consumed` + open `observations`), so the next
   tag is `schema/v0.2.0`.

3. Tag and push. The new convention is the path-scoped shape
   `schema/vX.Y.Z` (per the M02 release-train discipline — see
   [`runlog-docs/13-release-trains.md`](https://github.com/runlog-org/runlog-docs/blob/main/13-release-trains.md));
   the legacy plain `vX.Y.Z` is still accepted as a soft cut so
   existing consumers keep working:

       git tag -a schema/v0.2.0 -m "Release schema/v0.2.0"
       git push origin schema/v0.2.0

   Tags matching `*-rc*`, `*-beta*`, or `*-alpha*` (e.g.
   `schema/v0.2.0-rc1`) ship as **prereleases**; everything else ships
   as a normal release.

4. Watch the workflow on GitHub Actions. On success:
   - The tag appears on the Releases page with auto-generated notes.
   - The new version is live on
     [PyPI](https://pypi.org/project/runlog-schema/) within a minute.
   - pkg.go.dev publishes the Go module on first request (often
     triggered automatically by proxy.golang.org indexing).

`workflow_dispatch` is also wired up: triggering it manually from the
Actions tab runs the pre-flight validator and the Python build but
skips both the GitHub Release creation and the PyPI upload. Useful for
smoke-checking workflow changes without burning a version number.

## One-time PyPI trusted-publisher setup

Before the first tag-driven PyPI publish, a maintainer must register
this repo as a [trusted publisher](https://docs.pypi.org/trusted-publishers/)
on PyPI. This is human-side ops, not workflow code — the workflow
itself is correct as-shipped.

If `runlog-schema` does not yet exist on PyPI:

1. Sign in to https://pypi.org and visit
   *Your account → Publishing → Add a new pending publisher*.
2. Fill in:
   - **PyPI Project Name**: `runlog-schema`
   - **Owner**: `runlog-org`
   - **Repository name**: `runlog-schema`
   - **Workflow name**: `release.yml`
   - **Environment name**: *(leave blank — the workflow does not use
     a GitHub environment)*

If the project already exists on PyPI (claimed under a one-shot API
token, or migrated from an older publish path), do the same dance under
*Manage project → Publishing → Add a new trusted publisher* with the
same field values.

After that's saved, push a tag and the workflow's PyPI step will mint
its own short-lived upload token via OIDC — no API token lives in the
repo's GitHub Actions secrets.

## Pinning from a consumer

**Go module:**

    go get github.com/runlog-org/runlog-schema@schema/v0.2.0

The Go toolchain accepts the path-scoped tag shape directly; legacy
`@v0.1.0` pins also still resolve.

**Python package:**

    pip install runlog-schema==0.2.0

Or in `pyproject.toml`:

    dependencies = ["runlog-schema==0.2.0"]

**Schema YAMLs (server / verifier loading raw files):** consumers that
read `*.schema.yaml` directly should pin to a tag in their loader (the
exact mechanism is consumer-specific — e.g. a checkout of this repo at
`schema/v0.2.0`). Pinning to `main` works for development but exposes
consumers to mid-stream additions; tags are the supported contract.

## Versioning policy

While pre-1.0:

- Additive, backward-compatible schema changes → MINOR or PATCH bump.
  Examples: a new optional field, a new enum value (only if downstream
  validators tolerate unknown enum values — check before bumping
  PATCH), a new `*.schema.yaml` file.
- Breaking shape changes → MINOR bump (pre-1.0 convention). Examples:
  renaming or removing a field, tightening `required:`, narrowing a
  pattern, splitting a field into multiple.
- Restructuring beyond a single field — e.g. reorganising a top-level
  object — also warrants a MINOR bump and should be flagged
  explicitly in the release notes.

Post-1.0 the same matrix applies but breaking changes warrant MAJOR.

There is no `VERSION` file at the repo root: the git tag is the
authoritative version, and `pyproject.toml`'s `version` field is
bumped in the same commit that bumps the tag (or in a release-prep
commit immediately before tagging). For Go consumers, the tag is the
only version source. If a script needs the current version
programmatically, `git describe --tags --abbrev=0` reads it (and
strips the `schema/` prefix if you do `${tag##*/}`).
