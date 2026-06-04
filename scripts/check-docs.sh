#!/bin/sh

set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_root"

required_files='AGENTS.md
.agents/memory/context.md
.agents/memory/current.md
.agents/memory/open-questions.md
docs/README.md
docs/product/vision.md
docs/domain/glossary.md
docs/domain/context-map.md
docs/architecture/principles.md
docs/architecture/testing.md
docs/architecture/llm-boundary.md
docs/architecture/api-boundary.md
docs/architecture/durable-execution.md
docs/research/README.md
docs/tooling/automation.md
docs/decisions/README.md
docs/decisions/0000-template.md'

failed=0

for path in $required_files; do
	if [ ! -s "$path" ]; then
		echo "missing or empty required document: $path" >&2
		failed=1
	fi
done

for path in docs/decisions/[0-9][0-9][0-9][0-9]-*.md; do
	[ -e "$path" ] || continue
	[ "$path" = "docs/decisions/0000-template.md" ] && continue

	status=$(sed -n 's/^Status: \([^ ]*\).*/\1/p' "$path")
	case "$status" in
		Proposed|Accepted|Rejected|Superseded|Deprecated) ;;
		*)
			echo "invalid or missing ADR status: $path" >&2
			failed=1
			;;
	esac

	if [ "$status" = "Superseded" ] && ! grep -q '^Superseded by:' "$path"; then
		echo "superseded ADR must contain 'Superseded by:': $path" >&2
		failed=1
	fi
done

for path in docs/proposals/*.md; do
	[ -e "$path" ] || continue

	if ! grep -q 'НЕ ПРИНЯТО' "$path"; then
		echo "proposal must be marked 'Статус: НЕ ПРИНЯТО': $path" >&2
		failed=1
	fi
done

if [ "$failed" -ne 0 ]; then
	exit 1
fi

echo "documentation checks passed"
