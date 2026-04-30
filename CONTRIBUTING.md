# Contributing

`runlog-schema` is the root contract pinned by `runlog-verifier`, the server, and the skills. Treat every change here as load-bearing.

## Local validation

Before opening a PR, run the same check CI runs (from the repo root):

```sh
pip install pyyaml jsonschema
python -c "
import glob, sys, yaml
from jsonschema import Draft202012Validator
files = sorted(glob.glob('*.schema.yaml'))
if not files:
    sys.exit('no *.schema.yaml found — run from the repo root')
for f in files:
    Draft202012Validator.check_schema(yaml.safe_load(open(f)))
    print(f, 'OK')
"
```

If you also touched the Python distribution, run its tests too:

```sh
pip install ./generators/python[test]
pytest -v generators/python/tests/
```

After editing either canonical schema, sync the Python package's bundled copies (the `python-generator` CI job gates byte-for-byte equality):

```sh
bash generators/python/scripts/sync_schemas.sh
git add generators/python/runlog_schema/_data/
```

## Making changes

1. Edit the relevant `*.schema.yaml`. Keep the `$schema: https://json-schema.org/draft/2020-12/schema` line — the validator is pinned to Draft 2020-12.
2. Bumping the versioned `$id` URI (`.../v1.json` → `.../v2.json`) is reserved for backward-incompatible revisions; document the rationale in `DECISIONS.md` and coordinate the release across consumers.
3. Open a PR against `main`. All three CI jobs (`validate-schemas`, `generators-smoke`, `python-generator`) in `.github/workflows/ci.yml` must pass before merge.

## Backward compatibility

Downstream consumers pin to a specific schema version. Backward-incompatible changes require a major bump and a coordinated release across `runlog-verifier`, `runlog` (server), and `runlog-skills`.
