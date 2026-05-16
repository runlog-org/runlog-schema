"""Mirror of CI's schema-data byte-diff cmp job — fails locally if
runlog_schema/_data/ drifts from the canonical sibling at the repo
root. Run via `pytest` in generators/python/.

The set of gated files is derived the same way the canonical sync
helper derives it (`generators/python/scripts/sync_schemas.sh`): every
``*.schema.yaml`` at the repo root, plus the ``VERSION`` pin. Deriving
the list rather than hardcoding ``[entry, manifest]`` keeps this local
gate in lockstep with the helper and the ``python-generator`` CI job —
adding a third schema (or forgetting to sync VERSION) can no longer
slip past pytest while CI still catches it.
"""
from pathlib import Path

import pytest

_REPO_ROOT = Path(__file__).resolve().parents[3]  # generators/python/tests/ → repo root
_DATA_DIR = Path(__file__).resolve().parents[1] / "runlog_schema" / "_data"

# Glob *.schema.yaml at the repo root + VERSION, mirroring the loop in
# scripts/sync_schemas.sh. Sorted for deterministic parametrize IDs.
_GATED = sorted(p.name for p in _REPO_ROOT.glob("*.schema.yaml")) + ["VERSION"]


def test_gated_files_discovered() -> None:
    """Guard the glob itself: an empty schema set means the test is
    running from the wrong directory or every canonical schema was
    renamed — either way the byte-diff gate below would vacuously pass."""
    schema_files = [n for n in _GATED if n.endswith(".schema.yaml")]
    assert schema_files, (
        f"no *.schema.yaml discovered under {_REPO_ROOT}; "
        f"test_data_sync would vacuously pass"
    )


@pytest.mark.parametrize("name", _GATED)
def test_data_in_sync_with_canonical(name: str):
    canonical = _REPO_ROOT / name
    bundled = _DATA_DIR / name
    assert canonical.read_bytes() == bundled.read_bytes(), (
        f"{name} drifted between canonical ({canonical}) and bundled ({bundled}). "
        f"Run generators/python/scripts/sync_schemas.sh and `git add` the result."
    )
