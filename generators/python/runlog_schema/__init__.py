"""runlog_schema — canonical Runlog schemas as a Python package.

Mirrors the Go module ``github.com/runlog-org/runlog-schema`` (see
``../../schema.go``) so Python consumers — today the runlog server,
tomorrow any third-party tool that wants to pin the contract — can
``pip install runlog-schema`` and import the canonical YAML/JSON
instead of doing filesystem-path arithmetic against a sibling clone.

The wrapper is intentionally thin: accessors for raw YAML, accessors
for JSON-normalized form (cached), and the schema version. Callers do
their own jsonschema compilation.

Public API:

- ``entry_schema_yaml() -> bytes``
- ``manifest_schema_yaml() -> bytes``
- ``entry_schema_json() -> bytes``
- ``manifest_schema_json() -> bytes``
- ``schema_version() -> str``
- ``SCHEMA_VERSION`` (module-level constant, mirrors Go's
  ``SchemaVersionConst``)
"""

from __future__ import annotations

import json
from functools import lru_cache
from importlib.resources import files

import yaml

__all__ = [
    "SCHEMA_VERSION",
    "entry_schema_yaml",
    "manifest_schema_yaml",
    "entry_schema_json",
    "manifest_schema_json",
    "schema_version",
]

# Source of truth today: this constant. Both schema files carry a
# versioned $id (".../v1.json"), so v1 is the honest current value.
# Mirrors Go's SchemaVersionConst. F31 will derive this from a VERSION
# file at the repo root once the release-train work lands.
SCHEMA_VERSION: str = "0.1.0"


def _read_data(name: str) -> bytes:
    """Read a packaged data file from ``runlog_schema/_data/``.

    Uses ``importlib.resources.files()`` which is stable from Python
    3.9 forward — unlike the older ``importlib.resources.path()`` /
    ``read_text()`` shims that were deprecated in 3.11. Works
    identically on 3.10 and 3.12.
    """
    return (files(__package__) / "_data" / name).read_bytes()


def entry_schema_yaml() -> bytes:
    """Return the canonical entry schema as raw YAML bytes.

    The returned bytes are a fresh read each call; ``bytes`` is
    immutable so there's no shared-mutation hazard the way the Go
    version's ``[]byte`` has.
    """
    return _read_data("entry.schema.yaml")


def manifest_schema_yaml() -> bytes:
    """Return the canonical session-manifest schema as raw YAML bytes."""
    return _read_data("manifest.schema.yaml")


@lru_cache(maxsize=1)
def _entry_schema_json_cached() -> bytes:
    return _yaml_bytes_to_json(entry_schema_yaml())


@lru_cache(maxsize=1)
def _manifest_schema_json_cached() -> bytes:
    return _yaml_bytes_to_json(manifest_schema_yaml())


def entry_schema_json() -> bytes:
    """Return the entry schema normalized to JSON bytes.

    Useful for jsonschema libraries that compile from JSON rather
    than YAML. The result is cached after the first call. Mirrors
    Go's ``EntrySchemaJSON``.
    """
    return _entry_schema_json_cached()


def manifest_schema_json() -> bytes:
    """Return the manifest schema normalized to JSON bytes."""
    return _manifest_schema_json_cached()


def schema_version() -> str:
    """Return the schema version string. Mirrors Go's ``SchemaVersion()``."""
    return SCHEMA_VERSION


def _yaml_bytes_to_json(data: bytes) -> bytes:
    """Parse YAML and re-encode as canonical JSON.

    PyYAML's ``safe_load`` already produces JSON-compatible types
    (``dict``/``list``/scalars with string keys for normal YAML
    documents), so unlike the Go side we don't need a post-parse
    normalization pass — ``json.dumps`` handles it directly. If a
    schema file ever introduces non-string mapping keys this will
    raise ``TypeError`` from ``json.dumps``, which is the right
    failure mode (caller sees an explicit error, not silent data loss).
    """
    parsed = yaml.safe_load(data)
    return json.dumps(parsed).encode("utf-8")
