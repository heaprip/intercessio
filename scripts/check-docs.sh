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

# --- design: component cards ------------------------------------------------
# Each card declares component, owns, requires and seam in its frontmatter.
# requires must point at existing cards and form no cycle, every owned entity
# has exactly one owner, and the graph drawn in the spine matches requires.

cards=docs/design/components
spine=docs/design/README.md

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
: > "$work/edges"
: > "$work/owners"

frontmatter() {
	awk 'NR == 1 && $0 == "---" { f = 1; next } f && $0 == "---" { exit } f' "$1"
}

# prints the items of a block list "key:\n  - a\n  - b"; "key: []" prints nothing
field_list() {
	awk -v key="$1:" '
		$1 == key { f = ($0 !~ /\[\]/); next }
		f && /^  - / { sub(/^  - /, ""); print; next }
		{ f = 0 }'
}

for path in "$cards"/*.md; do
	[ -e "$path" ] || continue
	slug=$(basename "$path" .md)
	fm=$(frontmatter "$path")

	[ "$(printf '%s\n' "$fm" | sed -n 's/^component: *//p')" = "$slug" ] ||
		fail "component card must declare 'component: $slug': $path"
	printf '%s\n' "$fm" | grep -q '^seam:' ||
		fail "component card must declare 'seam:': $path"
	printf '%s\n' "$fm" | grep -q '^requires:' ||
		fail "component card must declare 'requires:': $path"
	printf '%s\n' "$fm" | grep -q '^owns:' ||
		fail "component card must declare 'owns:': $path"

	for dep in $(printf '%s\n' "$fm" | field_list requires); do
		[ -f "$cards/$dep.md" ] || fail "component requires a missing card: $slug -> $dep"
		echo "$slug $dep" >> "$work/edges"
	done

	for entity in $(printf '%s\n' "$fm" | field_list owns); do
		echo "$entity $slug" >> "$work/owners"
	done

	grep -q "\[\[$cards/$slug[|\\]" "$spine" ||
		fail "component card missing from the design spine: $path"
done

if [ -s "$work/edges" ] && tsort "$work/edges" 2>&1 >/dev/null | grep -q .; then
	tsort "$work/edges" 2>&1 >/dev/null | head -5 >&2
	fail "component dependencies contain a cycle"
fi

for entity in $(cut -d' ' -f1 "$work/owners" | sort | uniq -d); do
	fail "entity owned by more than one component: $entity"
done

awk '
	/^[[:space:]]*%% components[[:space:]]*$/ { f = 1; next }
	f && /^[[:space:]]*```/ { exit }
	f && NF == 3 && $2 == "-->" { gsub(/_/, "-"); print $1, $3 }' "$spine" |
	sort > "$work/spine-edges"
sort "$work/edges" > "$work/card-edges"

if ! cmp -s "$work/spine-edges" "$work/card-edges"; then
	diff "$work/spine-edges" "$work/card-edges" >&2 || true
	fail "component graph in $spine does not match 'requires' in the cards (< spine, > cards)"
fi

# --- design: stages ---------------------------------------------------------

for path in docs/design/stages/*.md; do
	[ -e "$path" ] || continue
	slug=$(basename "$path" .md)
	grep -q '^Зависит от:' "$path" || fail "stage must state 'Зависит от:': $path"
	grep -q "\[\[docs/design/stages/$slug[|\\]" "$spine" ||
		fail "stage missing from the design spine: $path"
done

# --- drafts are marked ------------------------------------------------------

for path in docs/proposals/*.md; do
	[ -e "$path" ] || continue
	[ "$path" = "docs/proposals/README.md" ] && continue
	grep -q 'НЕ ПРИНЯТО' "$path" ||
		fail "proposal must be marked 'Статус: НЕ ПРИНЯТО': $path"
done

# --- no dangling wikilinks --------------------------------------------------

# A target may point at a heading: [[docs/concepts/law#Норма|норма]]. The file
# must exist, and so must a heading with exactly that text.

find . -name '*.md' -not -path './.git/*' -exec cat {} + |
	grep -oE '\[\[[^]|]+' | sed 's/^\[\[//; s/\\$//' | sort -u > "$work/targets"

while IFS= read -r target; do
	file=${target%%#*}
	if [ ! -f "$file.md" ]; then
		fail "dangling wikilink: [[$target]]"
		continue
	fi
	case $target in
		*'#'*)
			heading=${target#*#}
			sed -n 's/^#\{1,6\} //p' "$file.md" | grep -Fxq -- "$heading" ||
				fail "wikilink to a missing heading: [[$target]]"
			;;
	esac
done < "$work/targets"

# --- concepts: every term heading is indexed in the registry ----------------

for path in docs/concepts/*.md; do
	[ "$path" = "docs/concepts/README.md" ] && continue
	slug=$(basename "$path" .md)
	sed -n 's/^#\{2,3\} //p' "$path" > "$work/headings"
	while IFS= read -r heading; do
		grep -Fq "[[docs/concepts/$slug#$heading" docs/concepts/README.md ||
			fail "concept term missing from the registry: $slug#$heading"
	done < "$work/headings"
done

if [ "$failed" -ne 0 ]; then
	exit 1
fi

echo "documentation checks passed"
