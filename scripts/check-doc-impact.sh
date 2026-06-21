#!/bin/sh
# Looks at staged changes and separates hard mechanical errors from reminders.
# A file path never proves semantic impact, so most of this is advisory.

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

# hard: a decision record changed without touching the index
if contains_path '^docs/decisions/[a-z0-9-]+\.md$' &&
	! contains_path '^docs/decisions/template\.md$' &&
	! contains_path '^docs/decisions/README\.md$'; then
	echo "decision changed without updating docs/decisions/README.md" >&2
	failed=1
fi

# hard: a concept added or renamed without touching the registry
if contains_path '^docs/concepts/[a-z0-9-]+\.md$' &&
	! contains_path '^docs/concepts/README\.md$'; then
	echo "concept changed without updating docs/concepts/README.md" >&2
	failed=1
fi

# hard: an external contract changed without documentation or generated output
if contains_path '\.(asyncapi|openapi)\.(yaml|yml|json)$|\.proto$' &&
	! contains_path '^(docs/|generated/)'; then
	echo "external contract changed without staged documentation or generated output" >&2
	failed=1
fi

# soft: behaviour changed and no project document is staged
if contains_path '^(internal|cmd)/.*\.go$|^go\.mod$' && ! contains_path '^docs/'; then
	echo "documentation impact: Go behaviour changed but no project document is staged" >&2
	echo "review the matrix in docs/README.md; no update may be the correct result" >&2
	if [ "${DOCS_IMPACT_STRICT:-0}" = "1" ]; then
		failed=1
	fi
fi

exit "$failed"
