# Schema Format Decisions — Ratified for v0.1

Positions on the sixteen format decisions surfaced across four batches of seed-entry drafting (see [`../server/seeds/CRITIQUE.md`](https://github.com/runlog-org/runlog/blob/main/server/seeds/CRITIQUE.md)).

These are **ratified and encoded in [`entry.schema.yaml`](./entry.schema.yaml)**. Override requires a schema revision.

Numbering follows the historical order the decisions were surfaced. Amendments added after batch 1 are inline under their respective parent sections. New sections (§7–§10) were added from batches 2–4.

---

## §1. `literals:` block — top-level map with categorized entries

```yaml
literals:
  $LITERAL_1:
    value: 600
    reason: "tolerance seconds — public constant, no PII"
    category: public_constant
```

**Categories (registered enum):**

- `public_constant` — numeric/string constant from a published spec (HTTP status codes, RFC constants)
- `public_identifier` — named identifier from a public vocabulary (header names, operator-class names)
- `standard_name` — widely-recognized name (protocol names, algorithm names)

**Why categories matter:** the post-publication audit tracks which public vocabularies are referenced, so literals that fall out of currency can be flagged automatically.

---

## §2. `assertion.type` vocabulary + expression grammar + custom extension

### Registered `assertion.type` enum

| Type | Asserts |
|---|---|
| `status` | HTTP or subprocess exit-code matches regex |
| `raises` | Exception of type X with message matching regex |
| `returns` | Return value of type X with structural constraints |
| `plan_node` | Query plan contains/doesn't contain specified node types (database) |
| `http_shape` | HTTP response shape matches cassette expectation |
| `output_match` | Stdout/stderr matches regex |
| `value_equals` / `value_contains` / `value_not_equals` | Direct value comparison |
| `length_equals` / `collection_contains_type` | Collection structure |
| **`compile_error_contains`** (batch 4) | Compiler rejects code; diagnostic text matches regex |
| **`compile_succeeds`** (batch 4) | Compiler accepts code; optional warning assertions |
| `custom` | Plug-in module for niche cases (`module:` field required) |

### Expression grammar (load-bearing)

Assertion fields may reference input placeholders and action-local bindings via a tiny grammar:

```
expr         := literal | placeholder | local_binding | call | binop
literal      := number | string | boolean | null
placeholder  := $IDENT                    # references an input or literal
local_binding := identifier               # variable exported from action body
call         := (len | type | keys | values)(expr)
binop        := expr (+ | - | * | /) expr
```

Example: `result_length: "len($ITEMS) + 1"` evaluates to the length of the bound `$ITEMS` input plus one at verification time.

Not Turing-complete by design. Entries needing more use `type: custom`.

### `assertion_only` primitive library (batch 3)

Entries with `verification.type: "assertion_only"` declare `primitives_required` from a registered set. The verifier's pure-logic evaluator executes these without any language runtime:

**Registered initial primitive set:**

- **Hashes:** `sha1`, `sha256`, `sha512`, `md5` (md5 flagged unsafe)
- **Encodings:** `base64`, `base64url`, `hex`, `utf8`
- **Structural:** `json_canonical`, `yaml_load_1_1`, `yaml_load_1_2`
- **Arithmetic:** `add`, `sub`, `mul`, `div`, `mod`
- **Comparison:** `equal`, `not_equal`, `lt`, `gt`, `lte`, `gte`
- **Logic:** `logic.and`, `logic.or`, `logic.not`, `logic.any`, `logic.all`
- **String:** `string.matches_regex`, `string.matches_glob_arn`
- **Domain-registered:** `k8s.parse_quantity_to_milli` and similar narrow primitives per domain

If an entry needs anything outside the registered set, it is not `assertion_only` — it's `unit` with a specific runtime.

### Extension mechanism

`type: custom` + `module: "<domain>.<assertion_module>"`. Verifier loads from a pinned registry. Unknown types without `custom` + module are rejected at submission.

### `isolation` enum (batch 4 addition)

Verifier isolation level declares how the test is executed:

- `function` — single function call in the language runtime
- `subprocess` — spawns a separate OS process
- **`compiler` (batch 4)** — feeds `action` to the language compiler and asserts on diagnostics, does not execute
- `database` — runs against a provisioned DB instance
- `http_client` — runs against a cassette (replay) or the live URL (reexecute)
- `docker_daemon` — runs against a provisioned Docker daemon

---

## §3. `verification.cassette:` structure — mode, steps, and time model

### Mode distinguishes replay vs re-execute

```yaml
verification:
  type: "integration"
  cassette:
    mode: "replay" | "reexecute"
    artifact: "name-of-cassette-file"
    steps: { … }               # named request/response steps (batch 2)
    captures: [ … ]
    strips: [ … ]
    replay_targets: [ … ]
    time_model: { … }          # simulated clock (batch 3)
```

- **`mode: replay`** — HTTP/RPC case. Consumer replays the cassette against a mock. Staleness detected when mock diverges from live.
- **`mode: reexecute`** — database / filesystem / compiler case. Consumer runs against its own live resource; cassette documents expected shape, not literal output.

**Default by type:** HTTP/RPC → replay; database/filesystem/compiler → reexecute.

### `cassette.steps` — named sequence definitions (batch 2 amendment)

Cassettes define named steps; branches reference ordered sequences of step names:

```yaml
cassette:
  steps:
    primary-ok:     { request: ..., response: ... }
    secondary-hit:  { request: ..., response: ... }
    retry-success:  { request: ..., response: ... }
differential:
  failed_approach_replay_sequence:  [primary-ok, secondary-hit]
  working_approach_replay_sequence: [primary-ok, secondary-hit, retry-success]
```

Step IDs are addressable units — sequences name them; assertions may target their state.

### `cassette.time_model` — simulated clock (batch 3 amendment)

For eventual-consistency and throttling claims:

```yaml
cassette:
  time_model:
    replication_delay_seconds: 3
    bucket_refill_per_second: 50
```

The replay harness advances a simulated clock (not real time) between steps. Tests are deterministic and don't burn wall-clock time. Real-time integration opts in via `mode: reexecute` with `use_real_clock: true`.

---

## §4. Mutation strategy vocabulary + per-branch inputs + expected_result trichotomy

### Registered mutation strategy set

| Strategy | Mutates |
|---|---|
| `set_literal_value` | Overwrite a declared `$LITERAL_N` with a different value |
| `remove_kwarg` | Drop a named keyword argument from the working-branch action |
| `swap_identifier` | Replace one allow-list identifier token with another |
| `drop_flag` | Remove a boolean flag or CLI flag |
| `swap_function_call` | Replace `f(x)` with `g(x)` where both are allow-list members |
| **`mutate_fixture`** | Perturb verifier input bindings, cassette state, or fixture resources (not code) |
| `custom` | Plug-in module for niche cases |

### Per-branch inputs (batch 2 amendment)

`differential.inputs` accepts either a single mapping (shared across branches) or per-branch mapping:

```yaml
# Shared inputs (original §7.3 form)
differential:
  inputs:
    $ENTITIES: "range(1, 200)"

# Per-branch inputs (batch 2; YAML octal-style entries)
differential:
  inputs:
    failed_approach:  { $INPUT: "permissions: 022" }
    working_approach: { $INPUT: "permissions: \"022\"" }
```

Per-branch overrides handle claims where the load-bearing distinction IS an input difference.

### `expected_result` trichotomy (batch 2 amendment)

Each mutation declares one of:

- **`fail`** — outcome flips from pass to fail (proves the mutated element IS the claim)
- **`unchanged`** — outcome stays the same (proves the mutated element is NOT the claim)
- **`pass`** — outcome flips from fail to pass (rarely used; fixture-reparation-style mutations)

**Submission requirement:** every entry must include at least two mutations of different strategies, at minimum one `mutate_fixture` and one other. Entries must include at least one `fail` mutation AND one `unchanged` mutation to demonstrate both necessity and specificity of the claim.

### Per-branch mutation outcomes (batch 3 amendment)

Mutations may target one branch only; the effect on each branch is specified:

```yaml
mutations:
  - strategy: "mutate_fixture"
    target: "$INPUT.size"
    new_value: 1024
    expected_branch_outcome:
      failed_approach: "assertion_does_not_match"
      working_approach: "unchanged"
```

---

## §5. `version_constraints.packages` operators — normalize internally, preserve source

Keep source syntax in entries (PHP `~6.5`, Python `>=7.0,<12.0`, npm `^6.5.0`). On ingestion, platform computes and indexes a normalized form:

```yaml
# Source (in entry):
packages:
  - name: "shopware/core"
    version: "~6.5"

# Normalized (computed, indexed, not in entry):
packages_normalized:
  - name: "shopware/core"
    min_version: "6.5.0"
    max_version: "6.6.0"
    exclusions: []
```

Search queries match normalized. Display shows source.

---

## §6. `version_constraints.runtime` format — structured, not free-form string

```yaml
version_constraints:
  runtime:
    name: "python"
    version: ">=3.10"
```

Free-form strings don't index. Structured form enables queries like "all entries whose runtime is `python >= 3.12`."

---

## §7. Multi-context harness (batch 2 amendment — new section)

Some entries need multiple independent execution environments (multiple database connections, multiple browser tabs, multiple processes). Single-context tests are the default; entries requiring multiple declare them in a top-level `contexts:` block:

```yaml
contexts:
  reader:
    kind: sqlite_connection
    database: "$FIXTURE_SQLITE_PATH"
    pragmas: { journal_mode: WAL }
  writer:
    kind: sqlite_connection
    database: "$FIXTURE_SQLITE_PATH"
    pragmas: { journal_mode: WAL }

failed_approach:
  action:
    steps:
      - context: reader
        type: code
        body: "$CONNECTION.execute('BEGIN'); ..."
      - context: writer
        type: code
        body: "writer_conn.execute('INSERT ...'); ..."
      - context: reader
        type: code
        body: "reader_view_after = ..."
```

Context kinds: `sqlite_connection`, `postgres_connection`, `redis_connection`, `http_endpoint`, `browser_tab`, `subprocess`, `docker_container`.

---

## §8. `action.steps` with inter-step mutations (batch 3 amendment — new section)

Actions may be composed of ordered steps with state mutations between them:

```yaml
action:
  steps:
    - id: "initial-build"
      type: "shell"
      body: "docker buildx build ..."
    - id: "mutate-external-state"
      type: "mutation"
      change: { base_image: "$LITERAL_2" }
    - id: "rebuild"
      type: "shell"
      body: "docker buildx build ..."
assertion:
  after_step: "rebuild"
  type: "output_match"
  pattern_present: "CACHED .* COPY"
```

Single-fragment actions remain valid shorthand — a bare `action: { type, body }` is equivalent to `action: { steps: [ { id: main, ... } ] }` with assertion binding to `main`, **provided `type` is in the base set** (`code`, `shell`, `sql`, `dockerfile`, `data`, `logic`). Using `mutation` or `wait` requires the explicit `action.steps` form, because each carries multi-step-only semantics (mutating state between adjacent steps; advancing the cassette clock between them).

Step types in `action.steps`: the base set above plus `mutation` (state change between adjacent steps) and `wait` (advances the cassette clock in replay mode). The two enums live as `$defs.step_kind_base` and `$defs.step_kind_action` in `entry.schema.yaml`; `step_fragment.type` references the base, `action_step.type` references the extended set.

---

## §9. `claim_scope` — asymmetric version applicability (batch 3 amendment — new section)

Some claims only apply within a narrow version window; outside that window the claim is trivially inapplicable, not incorrect:

```yaml
version_constraints:
  packages:
    - name: "next"
      version: ">=15.0.0"

claim_scope:
  applies_when:
    packages:
      - name: "next"
        version: ">=15.0.0"
  outside_scope_disposition: "inapplicable"
```

When the verifier processes a mutation that moves inputs outside `claim_scope.applies_when`, it does not report success/failure against the claim — it marks the run `inapplicable`. Mutations targeting version pins to prove scope behavior use `expected_branch_outcome` with `inapplicable` as a valid outcome value.

---

## §10. `fixtures:` — top-level resource provisioning (batch 3 amendment — new section)

Entries declare resources the verifier must provision before execution and clean up after:

```yaml
fixtures:
  $FIXTURE_FILE_PATH:
    kind: sparse_file
    size: 2254857830
    content: zero
  $FIXTURE_SQLITE_PATH:
    kind: sqlite_database
    schema: "CREATE TABLE $TABLE (id INTEGER PRIMARY KEY, data TEXT)"
    pragmas: { journal_mode: WAL }
  $FIXTURE_DOCKER_CONTEXT:
    kind: docker_context
    files:
      "Dockerfile": "$INLINE_DOCKERFILE"
      "src/app.py": "print('hi')"
```

Registered fixture kinds:

- `sparse_file` — size + content pattern
- `regular_file` — explicit content
- `directory` — file tree
- `sqlite_database` — schema + initial data + pragmas
- `postgres_database` — connection config + schema + seed
- `redis_instance` — config + preloaded keys
- `docker_context` — file tree for a Docker build
- `http_endpoint` — local mock server serving a cassette

Fixtures bind to `$FIXTURE_*` placeholders visible throughout the entry.

---

## Implementation Order

Already encoded in [`entry.schema.yaml`](./entry.schema.yaml). This doc is the authoritative rationale; the schema file is the authoritative constraint.

## Post-Ratification Corrections

Two schema bugs were surfaced when validating the 21 migrated seed entries against the initial `entry.schema.yaml` and fixed in place:

1. **`verification.isolation` was unconditionally required.** The `assertion_only` tier has no runtime to isolate — `isolation` doesn't apply. Fixed by removing `isolation` from the root `required` list on `verification` and adding a conditional `allOf` branch: `if: type == unit` → `required: [isolation]`, `if: type == integration` → `required: [isolation, cassette]`. The `assertion_only` branch still requires `primitives_required`.
2. **`differential.inputs` oneOf was ambiguous.** The "shared inputs" branch had no constraint preventing it from matching an entry whose inputs are actually per-branch, causing JSON Schema `oneOf` to match both branches and fail. Fixed by making the per-branch form the explicit first option and constraining the shared form with `not: { required: [failed_approach, working_approach] }` so only one branch matches any given input.

Both fixes are non-breaking (they accept a strict superset of what the original schema accepted). No re-authoring of entries was required after the schema corrections.

## Distribution: Go module layout (F28a)

The Go wrapper around the canonical schemas lives at the **repo root** as module `github.com/runlog-org/runlog-schema` (Option A). Picked over a sub-module (`.../go`) because the public API is just `//go:embed` of the YAML files plus a tiny YAML→JSON helper — keeping `go.mod` at the root means a single `vX.Y.Z` git tag versions both the schemas and the wrapper as one unit, which matches how downstream consumers (verifier, server, skills) already pin "the schema" as a single thing. Future generators in other languages (Python, TypeScript) will live at `generators/<lang>/` since only Go has a strong opinion about `go.mod` placement. First publishable tag will be `v0.1.0`, cut once the verifier consumer migration (F28b) is staged behind it.

## Distribution: Python package layout (F28c)

The Python distribution lives at **`generators/python/`** with the package name `runlog_schema` (PyPI: `runlog-schema`). Unlike Go, Python has no "repo root must be the package root" constraint — keeping `pyproject.toml` out of the repo root avoids a namespace fight with the Go-module-at-root layout (the schemas already live at root and would awkwardly straddle two distribution roots).

Build backend is **`hatchling`** — modern, uv-friendly, first-class data-file support; setuptools' `MANIFEST.in` / `package_data` quirks aren't worth the legacy mass for a 1-module package. The public API mirrors the Go module byte-for-byte where Python permits: snake_case function names (`entry_schema_yaml()` ↔ `EntrySchemaYAML()`), `bytes` returns (Go `[]byte`), `lru_cache` for the JSON forms (Go `sync.Once`), and a module-level `SCHEMA_VERSION` constant matching Go's `SchemaVersionConst`.

Schema files are **committed copies** under `runlog_schema/_data/`, kept in sync with the canonical files at repo root via a CI gate (`python-generator` job) and a helper script (`generators/python/scripts/sync_schemas.sh`). PyPI wheels are zipfiles and don't follow symlinks, so a build-time copy hook would be fragile; committed copies + a `cmp` gate trade a few KB of duplicate-on-disk for guaranteed consistency with a single failure point if a contributor forgets to run the sync script.

Consumed via `importlib.resources.files()`, stable since Python 3.9 and load-bearing identically on the supported floor (3.10) and production target (3.12). Python ≥3.10 matches the server's pinned floor. PyPI publish is a separate slice — this lands the structure first.
