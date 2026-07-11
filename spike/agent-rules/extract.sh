#!/bin/sh
# Вытаскивает из репозитория факты, о которых можно спрашивать движок.
# Ничего не интерпретирует: только переводит написанное в факты.

set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root"

echo "% Сгенерировано spike/agent-rules/extract.sh. Не править руками."
echo

for path in docs/decisions/*.md; do
	case $path in
		docs/decisions/README.md|docs/decisions/template.md) continue ;;
	esac
	slug=$(basename "$path" .md)
	status=$(sed -n 's/^Status: \([A-Za-z]*\).*/\1/p' "$path" | tr 'A-Z' 'a-z')
	accepted_by=$(sed -n 's/^Accepted by: *//p' "$path" | head -1 | sed 's/[[:space:]]*$//')
	printf "decision_status('%s', %s).\n" "$slug" "$status"
	case $accepted_by in
		''|'—'|'-') printf "no_accepted_by('%s').\n" "$slug" ;;
	esac
done

echo
if grep -q 'Непринятых решений нет' docs/guide/06-where-we-are.md; then
	echo "claims_no_unaccepted('docs/guide/06-where-we-are.md')."
fi
