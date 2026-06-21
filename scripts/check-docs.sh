#!/bin/sh
# Structural checks for the documentation. Checks form, never meaning.

set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_root"

failed=0
fail() { echo "$1" >&2; failed=1; }

# --- required files ---------------------------------------------------------

required='AGENTS.md
README.md
docs/README.md
docs/open-questions.md
docs/guide/01-what-it-is.md
docs/guide/02-the-world.md
docs/guide/03-how-law-works.md
docs/guide/04-what-goes-wrong.md
docs/guide/05-how-it-is-built.md
docs/guide/06-where-we-are.md
docs/concepts/README.md
docs/decisions/README.md
docs/decisions/template.md
docs/design/README.md
docs/impl/README.md
docs/research/README.md
docs/proposals/README.md'

for path in $required; do
	[ -s "$path" ] || fail "missing or empty required document: $path"
done

# --- guide chapters are numbered consecutively ------------------------------

n=0
for path in docs/guide/[0-9][0-9]-*.md; do
	n=$((n + 1))
	expected=$(printf 'docs/guide/%02d-' "$n")
	case "$path" in
		"$expected"*) ;;
		*) fail "guide chapters must be numbered consecutively from 01: $path" ;;
	esac
done

# --- layer invariant: guide never links into impl ---------------------------

if grep -rn 'docs/impl' docs/guide >/dev/null 2>&1; then
	grep -rn 'docs/impl' docs/guide >&2
	fail "guide/ must not link into impl/: intent may not depend on the stack"
fi

# --- decision records -------------------------------------------------------

for path in docs/decisions/*.md; do
	case "$path" in
		docs/decisions/README.md|docs/decisions/template.md) continue ;;
	esac

	status=$(sed -n 's/^Status: \([A-Za-z]*\).*/\1/p' "$path")
	case "$status" in
		Proposed|Accepted|Rejected|Superseded|Deprecated) ;;
		*) fail "invalid or missing decision status: $path" ;;
	esac

	grep -q '^Proposed by:' "$path" || fail "decision must state 'Proposed by:': $path"
	grep -q '^Accepted by:' "$path" || fail "decision must state 'Accepted by:': $path"

	if [ "$status" = "Superseded" ] && ! grep -q '^Superseded by:' "$path"; then
		fail "superseded decision must contain 'Superseded by:': $path"
	fi

	slug=$(basename "$path" .md)
	grep -q "\[\[docs/decisions/$slug" docs/decisions/README.md ||
		fail "decision missing from the index: $path"
done

# --- concepts are listed in the registry ------------------------------------

for path in docs/concepts/*.md; do
	[ "$path" = "docs/concepts/README.md" ] && continue
	slug=$(basename "$path" .md)
	grep -q "\[\[docs/concepts/$slug" docs/concepts/README.md ||
		fail "concept missing from the registry: $path"
done

# --- drafts are marked ------------------------------------------------------

for path in docs/proposals/*.md; do
	[ -e "$path" ] || continue
	[ "$path" = "docs/proposals/README.md" ] && continue
	grep -q 'НЕ ПРИНЯТО' "$path" ||
		fail "proposal must be marked 'Статус: НЕ ПРИНЯТО': $path"
done

# --- no dangling wikilinks --------------------------------------------------

targets=$(find . -name '*.md' -not -path './.git/*' -exec cat {} + |
	grep -oE '\[\[[^]|]+' | sed 's/^\[\[//; s/\\$//' | sort -u)

for target in $targets; do
	[ -f "$target.md" ] || fail "dangling wikilink: [[$target]]"
done

if [ "$failed" -ne 0 ]; then
	exit 1
fi

echo "documentation checks passed"
