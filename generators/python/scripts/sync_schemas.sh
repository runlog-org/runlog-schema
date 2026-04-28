#!/usr/bin/env bash
# Sync the canonical schemas at the repo root into the Python
# package's bundled _data/ directory. CI gates that the two are
# identical (see python-generator job in
# ../../.github/workflows/ci.yml); run this whenever you edit either
# canonical schema.
#
# Usage: from any directory:
#   bash generators/python/scripts/sync_schemas.sh
# Or from generators/python/:
#   ./scripts/sync_schemas.sh
set -euo pipefail

# Repo root is two levels up from this script's directory
# (generators/python/scripts/ → generators/python/ → repo root).
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
repo_root=$(cd -- "${script_dir}/../../.." &>/dev/null && pwd)
data_dir="${repo_root}/generators/python/runlog_schema/_data"

mkdir -p "${data_dir}"

for f in entry.schema.yaml manifest.schema.yaml; do
    src="${repo_root}/${f}"
    dst="${data_dir}/${f}"
    if [[ ! -f "${src}" ]]; then
        echo "::error::canonical schema missing at ${src}" >&2
        exit 1
    fi
    cp -f "${src}" "${dst}"
    echo "synced ${f}"
done

echo
echo "Bundled copies updated. Stage and commit:"
echo "  git add generators/python/runlog_schema/_data/"
