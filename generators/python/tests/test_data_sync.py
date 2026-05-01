"""Mirror of CI's schema-data byte-diff cmp job — fails locally if
runlog_schema/_data/*.yaml drifts from the canonical sibling at the
repo root. Run via `pytest` in generators/python/."""
from pathlib import Path
import pytest

_REPO_ROOT = Path(__file__).resolve().parents[3]  # generators/python/tests/ → repo root
_DATA_DIR = Path(__file__).resolve().parents[1] / "runlog_schema" / "_data"


@pytest.mark.parametrize("name", ["entry.schema.yaml", "manifest.schema.yaml"])
def test_data_in_sync_with_canonical(name: str):
    canonical = _REPO_ROOT / name
    bundled = _DATA_DIR / name
    assert canonical.read_bytes() == bundled.read_bytes(), (
        f"{name} drifted between canonical ({canonical}) and bundled ({bundled}). "
        f"Run generators/python/scripts/sync_schemas.sh and `git add` the result."
    )
