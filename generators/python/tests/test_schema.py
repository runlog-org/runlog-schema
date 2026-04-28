"""Mirrors ../../schema_test.go — same invariants, idiomatic Python.

The Go test guards //go:embed wiring; the Python test guards
importlib.resources wiring. Both ultimately fail loudly if the data
file got renamed, the package was built without the data dir, or the
canonical schemas drifted from what's bundled in _data/.
"""

from __future__ import annotations

import json
import re

import pytest
from jsonschema import Draft202012Validator

from runlog_schema import (
    SCHEMA_VERSION,
    entry_schema_json,
    entry_schema_yaml,
    manifest_schema_json,
    manifest_schema_yaml,
    schema_version,
)


# ── YAML accessors ────────────────────────────────────────────────


def test_entry_schema_yaml_non_empty() -> None:
    data = entry_schema_yaml()
    assert isinstance(data, bytes)
    assert len(data) > 0, "entry_schema_yaml returned empty bytes; data dir not packaged?"
    assert data.startswith(b"$schema:"), (
        f"entry schema YAML doesn't start with $schema: declaration; "
        f"got {data[:80]!r}"
    )


def test_manifest_schema_yaml_non_empty() -> None:
    data = manifest_schema_yaml()
    assert isinstance(data, bytes)
    assert len(data) > 0, "manifest_schema_yaml returned empty bytes; data dir not packaged?"
    assert data.startswith(b"$schema:"), (
        f"manifest schema YAML doesn't start with $schema: declaration; "
        f"got {data[:80]!r}"
    )


# ── JSON accessors ────────────────────────────────────────────────


def test_entry_schema_json_is_valid_json() -> None:
    data = entry_schema_json()
    assert isinstance(data, bytes)
    assert len(data) > 0
    doc = json.loads(data)
    assert isinstance(doc, dict)
    assert doc.get("$schema"), "entry schema JSON missing $schema key"
    assert doc.get("$id"), "entry schema JSON missing $id key"


def test_manifest_schema_json_is_valid_json() -> None:
    data = manifest_schema_json()
    assert isinstance(data, bytes)
    assert len(data) > 0
    doc = json.loads(data)
    assert isinstance(doc, dict)
    assert doc.get("$schema"), "manifest schema JSON missing $schema key"


def test_entry_schema_json_caches() -> None:
    """The JSON form is lru_cache'd; identity equality on consecutive
    calls is the cheapest signal that the cache is wired up. We don't
    care which exact identity (bytes are immutable), only that the
    parse-and-encode happens at most once."""
    a = entry_schema_json()
    b = entry_schema_json()
    assert a is b, "entry_schema_json should return the cached bytes object"


# ── Draft 2020-12 well-formedness ─────────────────────────────────


@pytest.mark.parametrize(
    "name,fn",
    [
        ("entry", entry_schema_json),
        ("manifest", manifest_schema_json),
    ],
)
def test_schemas_are_valid_draft_2020_12(name: str, fn) -> None:
    """Same gate as the validate-schemas CI job and the Go test of the
    same name. Catches regressions where a schema edit accidentally
    introduces a non-Draft-2020-12 construct."""
    doc = json.loads(fn())
    Draft202012Validator.check_schema(doc)


# ── Version ───────────────────────────────────────────────────────


def test_schema_version_matches_constant() -> None:
    assert schema_version() == SCHEMA_VERSION


def test_schema_version_shape() -> None:
    """Mirrors Go's TestSchemaVersionShape: accept bare semver and
    tag-prefixed forms so a future move to a VERSION file doesn't have
    to fight this test."""
    v = schema_version()
    assert v, "schema_version returned empty string"
    assert re.match(r"^v?\d+\.\d+\.\d+(-[A-Za-z0-9.-]+)?$", v), (
        f"schema_version={v!r} doesn't look like a semver-ish version"
    )
