#!/bin/sh
# Прогоняет свод правил репозитория против этой сессии и против корпуса.
# Ожидаемый исход записан здесь: регрессия должна быть видна, а не вспоминаться.

set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root"

scasp="$root/third_party/bin/scasp"
[ -x "$scasp" ] || { echo "run scripts/setup-spike.sh first" >&2; exit 1; }

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
failed=0

./spike/agent-rules/extract.sh > spike/agent-rules/repo.pl

# ask EXPECT LABEL QUERY [FILE...]
ask() {
	expect=$1; label=$2; query=$3; shift 3
	cat spike/legal-engine/core.pl "$@" > "$work/r.pl"
	printf '?- %s\n' "$query" >> "$work/r.pl"
	out=$(perl -e 'alarm 90; exec @ARGV' "$scasp" -s 1 "$work/r.pl" 2>&1 || true)
	case $out in
		*"ERROR:"*)    got=error ;;
		*"No models"*) got=none ;;
		*"Answer 1"*)  got=holds ;;
		*)             got=unknown ;;
	esac
	if [ "$got" = "$expect" ]; then
		printf 'ok    %-44s %s\n' "$label" "$expect"
	else
		printf 'FAIL  %-44s expected %s, got %s\n' "$label" "$expect" "$got" >&2
		failed=1
	fi
}

C="spike/agent-rules/contract.pl spike/agent-rules/session.pl"
R="spike/agent-rules/corpus.pl spike/agent-rules/repo.pl"

echo "-- допустимость действий этой сессии --"
ask holds "поручение: scripts/"        "may_write('scripts/setup-spike.sh')."               $C
ask holds "поручение: spike/"          "may_write('spike/legal-engine/core.pl')."           $C
ask holds "общее правило: proposals/"  "may_write('docs/proposals/some-draft.md')." $C

echo "-- запреты --"
ask none  "Go-файл: не разрешено"      "may_write('internal/deduction/eval.go')."           $C
ask holds "Go-файл: запрещено"         "no_may_write('internal/deduction/eval.go')."        $C
ask none  "decisions/: не разрешено"   "may_write('docs/decisions/a-new-decision.md')."     $C
ask holds "decisions/: запрещено"      "no_may_write('docs/decisions/a-new-decision.md')."  $C
ask none  "Accepted: не разрешено"     "may_set_accepted('norms-are-inference-rules')."     $C
ask holds "Accepted: запрещено строго" "no_may_set_accepted('norms-are-inference-rules')."  $C

echo "-- пробел в самом контракте --"
# поручение снимает умолчание, но «в остальные каталоги пишет автор» — тоже
# правило, и порядок между ними не объявлен. Оба запроса обязаны молчать.
ask none  "decisions/ по поручению: не разрешено" \
          "may_write('docs/decisions/norms-are-inference-rules.md')."    $C
ask none  "decisions/ по поручению: не запрещено" \
          "no_may_write('docs/decisions/norms-are-inference-rules.md')." $C

echo "-- согласованность корпуса --"
# решения подписаны, глава 6 больше не утверждает обратного
ask none  "глава 6 против статусов решений" \
          "contradicts('docs/guide/06-where-we-are.md', _D)."            $R
ask none  "Accepted без подписавшего: нет"  "unsigned_acceptance(_D)."   $R

exit "$failed"
