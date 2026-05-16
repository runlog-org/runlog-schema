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
    """Mirrors Go's TestSchemaVersionShape. The current source is the
    bundled VERSION file (bare semver, e.g. ``"0.4.1"``); the regex
    also accepts the tag-prefixed form (``"v0.4.1"``) so the test
    survives any future change to source from `git describe` instead.
    """
    v = schema_version()
    assert v, "schema_version returned empty string"
    assert re.match(r"^v?\d+\.\d+\.\d+(-[A-Za-z0-9.-]+)?$", v), (
        f"schema_version={v!r} doesn't look like a semver-ish version"
    )


# ── Cross-schema / structural invariants ──────────────────────────
#
# The three guards below close a parity gap: schema_test.go enforces
# them on the Go side, and this file's module docstring claims "same
# invariants", but they were previously Go-only. A schema edit that
# broke any of them would have been caught by `generators-smoke` (Go)
# yet passed `python-generator` (Python) — an asymmetry that defeats
# the point of dual-language coverage. They are pure validator-level
# assertions against the canonical JSON; no wire-contract change.


def _entry_validator() -> Draft202012Validator:
    """Compile the entry schema once for the structural tests below.

    Python's ``jsonschema`` resolves local ``$ref`` against the root
    document, so validating a crafted ``branch``/``mutation`` instance
    is done by wrapping it under the relevant top-level property rather
    than compiling a sub-schema by ``$ref`` (the Go side's
    ``compiler.Compile(url + "#/$defs/...")`` has no direct analogue).
    """
    return Draft202012Validator(json.loads(entry_schema_json()))


def _is_valid(validator: Draft202012Validator, instance) -> bool:
    return next(iter(validator.iter_errors(instance)), None) is None


def test_kb_id_pattern_matches_unit_id_pattern() -> None:
    """Mirrors Go's TestKbIdPatternMatchesUnitIdPattern. ``kb_id`` in
    the manifest is the reference to an entry's ``unit_id``; the two
    patterns MUST stay byte-identical (the manifest schema documents
    this explicitly) because cross-file ``$ref`` isn't used — each
    schema is published as an independent artifact under its own
    ``$id``. A drift would let the manifest accept identifiers the
    entry schema rejects, or vice versa."""
    entry_doc = json.loads(entry_schema_json())
    manifest_doc = json.loads(manifest_schema_json())

    entry_pattern = entry_doc["properties"]["unit_id"]["pattern"]
    kb_id_pattern = (
        manifest_doc["properties"]["entries"]["items"]["properties"][
            "kb_id"
        ]["pattern"]
    )
    assert entry_pattern == kb_id_pattern, (
        "kb_id/unit_id pattern drift\n"
        f"  entry.unit_id.pattern  = {entry_pattern!r}\n"
        f"  manifest.kb_id.pattern = {kb_id_pattern!r}\n"
        "these MUST stay byte-identical "
        "(see manifest.schema.yaml comment)"
    )


@pytest.mark.parametrize(
    "name,action,valid",
    [
        (
            "arm1 single step_fragment",
            {"type": "code", "body": "print('x')"},
            True,
        ),
        (
            "arm2 array of step_fragments",
            [
                {"type": "code", "body": "print('x')"},
                {"type": "shell", "body": "ls"},
            ],
            True,
        ),
        (
            "arm3 action_steps_block",
            {"steps": [{"id": "main", "type": "code", "body": "print('x')"}]},
            True,
        ),
        (
            "ambiguous shape (type + steps) rejected",
            {
                "type": "code",
                "body": "print('x')",
                "steps": [{"id": "main", "type": "code"}],
            },
            False,
        ),
        (
            "ambiguous shape inside arm2 array rejected",
            [
                {
                    "type": "code",
                    "body": "print('x')",
                    "steps": [{"id": "main", "type": "code"}],
                }
            ],
            False,
        ),
    ],
)
def test_branch_action_oneof_disambiguation(name, action, valid) -> None:
    """Mirrors Go's TestBranchActionOneOfDisambiguation. Arm 1 (single
    step_fragment) inherits ``additionalProperties: true``, so without
    the ``not: {required: [steps]}`` guard a document carrying both a
    ``type`` and a ``steps`` array would silently pass arm 1 by
    treating ``steps`` as an unrecognized extra. Fails loudly if a
    future refactor drops the guard."""
    validator = _entry_validator()
    branch = {
        "description": "x" * 40,
        "action": action,
        "assertion": {"type": "status", "expect": "success"},
    }
    instance = {
        "unit_id": "kb-fixture-entry",
        "domain": ["example"],
        "version_constraints": {"runtime": {"name": "python", "version": ">=3.10"}},
        "failed_approach": branch,
        "working_approach": branch,
        "verification": {
            "type": "assertion_only",
            "primitives_required": ["equal"],
            "differential": {},
            "mutations": [
                {"strategy": "drop_flag", "target": "x", "expected_result": "fail"},
                {"strategy": "drop_flag", "target": "y", "expected_result": "unchanged"},
            ],
            "timeout_seconds": 30,
        },
    }
    assert _is_valid(validator, instance) is valid, (
        f"{name}: expected validity={valid}"
    )


@pytest.mark.parametrize(
    "name,extra,valid",
    [
        ("field selector (new shape)", {"field": "body"}, True),
        ("action selector (legacy back-compat)", {"action": "body"}, True),
        (
            "header.<NAME> via field",
            {"field": "header.X-RateLimit-Remaining"},
            True,
        ),
        (
            "header.<NAME> via legacy action",
            {"action": "header.X-RateLimit-Remaining"},
            True,
        ),
        (
            "both field and action set (consumer prefers field)",
            {"field": "body", "action": "status"},
            True,
        ),
    ],
)
def test_mutation_cassette_response_field_or_action(name, extra, valid) -> None:
    """Mirrors Go's TestMutationCassetteResponseFieldOrAction. The F76
    split: ``mutate_cassette_response`` accepts the response-field
    selector under either ``field`` (preferred, schema 0.4.1+) or
    ``action`` (legacy). Both must validate so in-flight seeds keep
    working while new seeds migrate to ``field``."""
    validator = Draft202012Validator(
        json.loads(entry_schema_json())["$defs"]["mutation"]
    )
    mutation = {
        "strategy": "mutate_cassette_response",
        "target": "step-1",
        "new_value": "200",
        "expected_result": "fail",
        **extra,
    }
    assert _is_valid(validator, mutation) is valid, (
        f"{name}: expected validity={valid}"
    )
