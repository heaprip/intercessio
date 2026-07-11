#!/bin/sh
# Runs the legal-engine spike cases and checks each against its expected
# outcome. Every case is one question the spike had to answer; the expected
# outcome is written here so a regression is visible, not remembered.

set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root"

scasp="$root/third_party/bin/scasp"
here="spike/legal-engine"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

[ -x "$scasp" ] || { echo "run scripts/setup-spike.sh first" >&2; exit 1; }

failed=0

# run CASE EXPECT [EXTRA-FLAG...]
#   EXPECT is "answer", "no-models" or "error"
run() {
	case_file=$1; expect=$2; shift 2
	name=$(basename "$case_file" .pl)
	cat "$here/core.pl" "$here/cases/$case_file" > "$work/run.pl"
	out=$(perl -e 'alarm 60; exec @ARGV' "$scasp" -s 1 "$@" "$work/run.pl" 2>&1 || true)

	case $out in
		*"ERROR:"*)    got=error ;;
		*"No models"*) got=no-models ;;
		*"Answer 1"*)  got=answer ;;
		*)             got=unknown ;;
	esac

	if [ "$got" = "$expect" ]; then
		printf 'ok    %-22s %s\n' "$name" "$expect"
	else
		printf 'FAIL  %-22s expected %s, got %s\n' "$name" "$expect" "$got" >&2
		failed=1
	fi
}

# цепочка intentio - exceptio - replicatio, порядок объявлен
run chain-replicatio.pl  answer
run chain-exceptio.pl    answer
# два применимых правила с противоположными выводами и без порядка: non liquet
run unordered-conflict.pl no-models
# защита приобретённого: A4 в силе - лишение не проходит
run vested-protection.pl no-models
# компетенция: правило по умолчанию вытесняется назначением
run competence.pl        answer
# выпавшее правило-исключение опирается на предикат, который нигде не выводится
run dropped-exception.pl answer                      # молча выдаёт вывод
run dropped-exception.pl error   --unknown=error     # и ловится флагом

exit "$failed"
