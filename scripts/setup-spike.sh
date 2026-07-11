#!/bin/sh
# Installs the two external reasoners used by the legal-engine spike and
# verifies they work. Nothing here is a project dependency: the spike answers
# whether the legal layer fits into existing engines at all.
#
#   clingo  - grounder and solver; used for its load-time diagnostics and for
#             questions about the whole world at once
#   s(CASP) - top-down solver; used for one question at a time and for the
#             justification tree

set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_root"

bin="$repo_root/third_party/bin"
pack_home="${HOME}/.local/share/swi-prolog/pack/scasp"

say() { printf '%s\n' "$1"; }

# --- clingo and SWI-Prolog --------------------------------------------------

if ! command -v brew >/dev/null 2>&1; then
	say "Homebrew not found. Install clingo and swi-prolog by hand, then rerun." >&2
	exit 1
fi

command -v clingo >/dev/null 2>&1 || brew install clingo
command -v swipl  >/dev/null 2>&1 || brew install swi-prolog

# --- the s(CASP) pack -------------------------------------------------------

if [ ! -d "$pack_home" ]; then
	swipl -q -g "pack_install(scasp,[interactive(false)])" -t halt
fi

# --- the scasp executable ---------------------------------------------------
# The pack ships a library, not a command. The command is what an exporter
# would call, so it is built here rather than driven through the Prolog API.

if [ ! -x "$bin/scasp" ]; then
	mkdir -p "$bin"
	swipl --no-pce --undefined=error -O \
	      -o "$bin/scasp" -c "$pack_home/prolog/scasp/main.pl"
fi

# --- verification -----------------------------------------------------------

say "clingo:  $(clingo --version | head -1)"
say "swipl:   $(swipl --version)"
say "s(CASP): $("$bin/scasp" --version 2>&1 | head -1)"

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

cat > "$work/absence.pl" <<'PROG'
p(a).
q(X) :- p(X), not resolved(X).
?- q(X).
PROG

if "$bin/scasp" --unknown=error "$work/absence.pl" >/dev/null 2>&1; then
	say "FAILED: s(CASP) accepted a conclusion resting on an undefined predicate" >&2
	exit 1
fi
say "ok: s(CASP) --unknown=error rejects derivation from an undefined predicate"

cat > "$work/unsafe.lp" <<'PROG'
nocomp(Off,Kind).
PROG

if clingo "$work/unsafe.lp" >/dev/null 2>&1; then
	say "FAILED: clingo accepted a rule with unbound head variables" >&2
	exit 1
fi
say "ok: clingo rejects a default rule with unbound head variables"
