# Contributing

`runlog-schema` is the root contract pinned by `runlog-verifier`, the server, and the skills. Treat every change here as load-bearing.

## Local validation

Before opening a PR, run the same check CI runs:

```sh
pip install pyyaml jsonschema
python -c "
import glob, yaml
from jsonschema import Draft202012Validator
for f in sorted(glob.glob('*.schema.yaml')):
    Draft202012Validator.check_schema(yaml.safe_load(open(f)))
    print(f, 'OK')
"
```

## Making changes

1. Edit the relevant `*.schema.yaml`. Keep the `$schema: https://json-schema.org/draft/2020-12/schema` line — the validator is pinned to Draft 2020-12.
2. Updating `$id` (versioned URI) for breaking changes is handled in the F28 generator/release workflow; for now, document the change in `DECISIONS.md`.
3. Open a PR against `main`. The `validate-schemas` job in `.github/workflows/ci.yml` must pass before merge.

## Backward compatibility

Downstream consumers pin to a specific schema version. Backward-incompatible changes require a major bump and a coordinated release across `runlog-verifier`, `runlog` (server), and `runlog-skills`.
