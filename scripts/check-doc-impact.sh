#!/bin/sh

set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_root"

usage() {
	echo "usage: $0 --staged | --range <git-revision>" >&2
	exit 2
}

case "${1:-}" in
	--staged)
		if git rev-parse --verify HEAD >/dev/null 2>&1; then
			changed=$(git diff --cached --name-only --diff-filter=ACMR)
		else
			changed=$(git ls-files --cached)
		fi
		;;
	--range)
		[ "$#" -eq 2 ] || usage
		changed=$(git diff --name-only --diff-filter=ACMR "$2"...HEAD)
		;;
	*)
		usage
		;;
esac

[ -n "$changed" ] || exit 0

contains_path() {
	printf '%s\n' "$changed" | grep -Eq "$1"
}

failed=0

if contains_path '^docs/decisions/[0-9]{4}-.+\.md$' \
	&& ! contains_path '^docs/decisions/README\.md$'; then
	echo "ADR changed without updating docs/decisions/README.md" >&2
	failed=1
fi

if contains_path '^(internal|cmd)/.*\.go$|^go\.mod$' \
	&& ! contains_path '^(docs/|\.agents/memory/)'; then
	echo "documentation impact: Go behavior changed but no project document is staged" >&2
	echo "review docs/README.md change matrix; no update may be the correct result" >&2
	if [ "${DOCS_IMPACT_STRICT:-0}" = "1" ]; then
		failed=1
	fi
fi

if contains_path '\.(asyncapi|openapi)\.(yaml|yml|json)$|\.proto$' \
	&& ! contains_path '^(docs/|generated/)'; then
	echo "external contract changed without staged documentation/generated output" >&2
	failed=1
fi

exit "$failed"
