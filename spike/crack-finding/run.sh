#!/bin/sh
# Kill test for the concept "law under pressure".
# stub and replay need only python3; live needs the Anthropic SDK and runs through uv.

set -eu

here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
[ $# -eq 0 ] && set -- stub

case "$1" in
	live) exec uv run --no-project --python 3.12 --with anthropic python "$here/spike.py" "$@" ;;
	*) exec python3 "$here/spike.py" "$@" ;;
esac
