#!/bin/sh
# Runs the metrics spike and writes its report next to it.
# Standard-library Python only; deterministic by seed.

set -eu

here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
python3 "$here/sim.py" > "$here/report.md"
echo "wrote $here/report.md"
