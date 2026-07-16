#!/bin/sh
# Kill test for the concept "law under pressure".
# Standard-library Python only. live reads LLM_URL and LLM_KEY from the
# environment or from .env at the repository root.

set -eu

here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
[ $# -eq 0 ] && set -- stub

exec python3 "$here/spike.py" "$@"
