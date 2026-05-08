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

# Glob *.schema.yaml at the repo root rather than hardcode the filename
# list — adding a third schema (or renaming one) would otherwise leave
# the bundled copy out of sync silently, and the python-generator CI
# job would only catch it on the file the loop still mentions. VERSION
# is included explicitly because it isn't a *.schema.yaml file but is
# bundled alongside as the single-source version pin (T37).
shopt -s nullglob
schema_files=( "${repo_root}"/*.schema.yaml )
shopt -u nullglob

if (( ${#schema_files[@]} == 0 )); then
    echo "::error::no *.schema.yaml files found at ${repo_root}" >&2
    exit 1
fi

for src in "${schema_files[@]}" "${repo_root}/VERSION"; do
    f=$(basename -- "${src}")
    dst="${data_dir}/${f}"
    if [[ ! -f "${src}" ]]; then
        echo "::error::canonical file missing at ${src}" >&2
        exit 1
    fi
    cp -f "${src}" "${dst}"
    echo "synced ${f}"
done

echo
echo "Bundled copies updated. Stage and commit:"
echo "  git add generators/python/runlog_schema/_data/"
