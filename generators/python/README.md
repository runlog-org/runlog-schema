# runlog-schema (Python)

Canonical Runlog schemas (entry + session manifest) packaged for
Python. Mirrors the Go module
[`github.com/runlog-org/runlog-schema`](../../schema.go) so Python
consumers don't need to do filesystem-path arithmetic against a
sibling clone.

> Part of the **[Runlog](https://github.com/runlog-org)** project.
> The canonical YAML lives at the repo root
> ([`entry.schema.yaml`](../../entry.schema.yaml),
> [`manifest.schema.yaml`](../../manifest.schema.yaml)); this package
> bundles a CI-gated copy under `runlog_schema/_data/`.

## Install

Once published to PyPI (separate slice — see `[F28]` in the project
TODO):

```bash
pip install runlog-schema
```

For now, install from the source tree:

```bash
pip install ./generators/python   # from runlog-schema repo root
```

## Use

```python
from runlog_schema import (
    entry_schema_yaml,
    manifest_schema_yaml,
    entry_schema_json,
    manifest_schema_json,
    schema_version,
    SCHEMA_VERSION,
)

# Raw YAML bytes (the canonical source form):
entry_yaml = entry_schema_yaml()

# JSON bytes for libraries that don't accept YAML
# (e.g. `jsonschema.Draft202012Validator`):
import json
from jsonschema import Draft202012Validator

doc = json.loads(entry_schema_json())
Draft202012Validator.check_schema(doc)

assert schema_version() == SCHEMA_VERSION == "0.1.0"
```

## API parity with the Go module

| Go                            | Python                    |
| ----------------------------- | ------------------------- |
| `EntrySchemaYAML() []byte`    | `entry_schema_yaml()`     |
| `ManifestSchemaYAML() []byte` | `manifest_schema_yaml()`  |
| `EntrySchemaJSON()`           | `entry_schema_json()`     |
| `ManifestSchemaJSON()`        | `manifest_schema_json()`  |
| `SchemaVersion() string`      | `schema_version()`        |
| `SchemaVersionConst`          | `SCHEMA_VERSION`          |

Both modules compile from the same canonical YAML files at the repo
root. Commit-time CI ensures the bundled copies under
`runlog_schema/_data/` match those canonical files byte-for-byte; if
you edit either schema, run:

```bash
cd generators/python
./scripts/sync_schemas.sh
```

…then commit the updated `_data/` copies alongside your schema edit.

## Develop

```bash
cd generators/python

# Editable install with test extras:
uv pip install -e '.[test]'   # or: pip install -e '.[test]'

# Run the tests:
pytest -v
```

Python ≥3.10 is supported (matches the server's pinned floor; 3.12 is
what production runs). The package uses
`importlib.resources.files()`, stable since Python 3.9.

## Build backend

[`hatchling`](https://hatch.pypa.io/) — the modern uv-friendly
default. See `pyproject.toml`.

## License

Apache-2.0 — same as the canonical schemas. See [`../../LICENSE`](../../LICENSE).
