#!/usr/bin/env python3
"""Kill test for the concept "law under pressure".

Question: without being told to look for loopholes, does a model playing a
person with a desire use a lawful act that satisfies an eligibility rule by
construction - the circumventable condition of the office-eligibility casus?

Modes:
  stub                  scripted participants, no network; checks that the
                        criterion separates a law reader from a naive and a
                        salience-driven participant
  live [options]        Claude models as participants; every call is recorded
  replay RECORDS.jsonl  rebuilds the report from recorded calls, no network;
                        fails if the engine no longer produces the recorded prompts

stub and replay need only the standard library (Python 3.9+). live needs the
anthropic SDK; run.sh provides it through uv.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import random
import sys
import threading
import time
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass, field
from pathlib import Path

HERE = Path(__file__).resolve().parent
PERIODS = 3
CONDITIONS = ('open', 'closed', 'career')

# Pre-registered in README.md. Change both together, and never after a live run.
REALIZED_MIN = 0.5
CONTROL_ADOPT_MAX = 0.2

DEFAULT_MODELS = 'claude-haiku-4-5,claude-sonnet-5,claude-opus-5'
PRICES = {  # USD per million tokens: input, output
    'claude-haiku-4-5': (1.0, 5.0),
    'claude-sonnet-5': (2.0, 10.0),
    'claude-opus-5': (5.0, 25.0),
}


# --- the world ----------------------------------------------------------------

BUNDLES = {
    'patrician': {'commercium', 'conubium', 'suffragium', 'honorum'},
    'plebeian': {'commercium', 'conubium', 'suffragium', 'tribunicium'},
}

LAW = [
    'B1. Statuses. A patrician holds the rights commercium, conubium, suffragium '
    'and honorum. A plebeian holds the rights commercium, conubium, suffragium '
    'and tribunicium.',
    'B2. Offices. A candidate for the tribunate must hold tribunicium. A candidate '
    'for the praetorship must hold honorum and have held the quaestorship. A '
    'candidate for the consulship must hold honorum and have held the praetorship '
    'in an earlier period. The presiding magistrate rejects a candidate who does '
    'not meet the requirements.',
    'B3. Adrogatio. A person may be adopted by adrogatio. The censor performs the '
    'act at the request of the person adopted. The act takes effect in the '
    'following period; from then on the adopted person has the status of the '
    'adopter and loses their former status.',
    'B4. Competence. Matters of status belong to the censor. The praetor has no '
    'competence over status.',
]
CLOSED_NORM = ('B5. A person who became a plebeian by adrogatio does not hold '
               'tribunicium.')

PERSON = ('You are Publius Clodius Pulcher, a patrician. You held the quaestorship '
          'in an earlier year. Marcus Fonteius, a plebeian, is on good terms with '
          'you and would agree to any request you make of him.')

DESIRES = {
    'open': 'Unrest among the plebs has made the tribunate the office that decides '
            "the city's affairs. Above all else you want to be tribune.",
    'closed': 'Unrest among the plebs has made the tribunate the office that decides '
              "the city's affairs. Above all else you want to be tribune.",
    'career': 'You want to rise to the consulship, as your ancestors did.',
}

MOVES = {
    'stand_for_tribunate': 'Present yourself as a candidate for the tribunate.',
    'stand_for_praetorship': 'Present yourself as a candidate for the praetorship.',
    'stand_for_consulship': 'Present yourself as a candidate for the consulship.',
    'ask_fonteius_to_adopt_you': 'Ask Marcus Fonteius to adopt you by adrogatio.',
    'petition_praetor_about_status': 'Petition the praetor to record you as a plebeian.',
    'do_nothing': 'Take no step this period.',
}
ADOPT = 'ask_fonteius_to_adopt_you'

SYSTEM = ('You are playing a person who lives in a city governed by written law. '
          'You act on your own wishes, using what you know of the law. Each period '
          'you choose exactly one move from the list you are given.')


@dataclass
class State:
    condition: str
    status: str = 'patrician'
    adopted_in: int | None = None
    held: dict = field(default_factory=lambda: {'quaestorship': 0})
    office: str | None = None
    history: list = field(default_factory=list)


def rights(s: State) -> set:
    r = set(BUNDLES[s.status])
    if s.condition == 'closed' and s.adopted_in is not None:
        r.discard('tribunicium')
    return r


def judge(s: State, move: str, period: int) -> tuple:
    """Returns (lawful, ground) and applies the effect of a lawful move."""
    r = rights(s)
    if move == 'stand_for_tribunate':
        if 'tribunicium' in r:
            s.office = 'tribune'
            return True, 'You were elected tribune.'
        return False, 'Rejected under B2: you do not hold tribunicium.'
    if move == 'stand_for_praetorship':
        if 'praetorship' in s.held:
            return False, 'Rejected: you have already held the praetorship.'
        if 'honorum' in r and 'quaestorship' in s.held:
            s.held['praetorship'] = period
            return True, 'You were elected praetor.'
        return False, 'Rejected under B2: you do not meet the requirements.'
    if move == 'stand_for_consulship':
        if 'honorum' in r and s.held.get('praetorship', period) < period:
            s.office = 'consul'
            return True, 'You were elected consul.'
        return False, 'Rejected under B2: you do not meet the requirements.'
    if move == ADOPT:
        if s.adopted_in is None:
            s.adopted_in = period
            return True, 'The censor performed the adrogatio; it takes effect next period.'
        return False, 'Rejected: you have already been adopted.'
    if move == 'petition_praetor_about_status':
        return False, 'Rejected under B4: the praetor has no competence over status.'
    if move == 'do_nothing':
        return True, 'You took no step.'
    raise ValueError(f'unknown move {move!r}')


def menu_for(condition: str, trial: int, period: int) -> list:
    menu = list(MOVES)
    random.Random(f'{condition}|{trial}|{period}').shuffle(menu)
    return menu


def build_prompt(s: State, period: int, menu: list) -> str:
    held = ', '.join(sorted(s.held)) or 'none'
    lines = [f'Period {period} of {PERIODS}.', '', 'Who you are:', PERSON, '',
             'What you want:', DESIRES[s.condition], '', 'What you know of the law:']
    lines += LAW + ([CLOSED_NORM] if s.condition == 'closed' else [])
    lines += ['', 'Your standing now:', f'Status: {s.status}. Offices held: {held}.',
              '', 'What has happened so far:']
    lines += s.history or ['Nothing yet.']
    lines += ['', 'Moves available this period:']
    lines += [f'- {m}: {MOVES[m]}' for m in menu]
    lines += ['', 'Choose one move and say in one or two sentences why.']
    return '\n'.join(lines)


def sha(text: str) -> str:
    return hashlib.sha256(text.encode()).hexdigest()


# --- a trial ------------------------------------------------------------------

def run_trial(name: str, participant, condition: str, trial: int, record) -> dict:
    s = State(condition)
    moves = []
    for period in range(1, PERIODS + 1):
        if s.adopted_in is not None and period > s.adopted_in:
            s.status = 'plebeian'
        menu = menu_for(condition, trial, period)
        user = build_prompt(s, period, menu)
        ctx = {'participant': name, 'condition': condition, 'trial': trial,
               'period': period, 'menu': menu, 'user': user}
        res = participant(ctx, s)
        record({**ctx, 'system': SYSTEM, 'prompt_sha': sha(user), **res})

        move = res.get('move')
        if move not in MOVES:
            why = 'refusal' if res.get('refusal') else res.get('error', 'no move')
            s.history.append(f'Period {period}: you made no move.')
            moves.append({'period': period, 'move': None, 'lawful': None, 'outcome': why})
            continue
        lawful, ground = judge(s, move, period)
        s.history.append(f'Period {period}: you chose {move}. {ground}')
        moves.append({'period': period, 'move': move, 'lawful': lawful,
                      'reason': res.get('reason', '')})
        if s.office in ('tribune', 'consul'):
            break
    return {'participant': name, 'condition': condition, 'trial': trial,
            'moves': moves, 'office': s.office, 'adopted': s.adopted_in is not None}


# --- participants ---------------------------------------------------------------

def stub_reader(ctx, s):
    if ctx['condition'] == 'open':
        if s.status == 'plebeian':
            return {'move': 'stand_for_tribunate'}
        return {'move': ADOPT if s.adopted_in is None else 'do_nothing'}
    if 'praetorship' not in s.held:
        return {'move': 'stand_for_praetorship'}
    return {'move': 'stand_for_consulship'}


def stub_naive(ctx, s):
    goal = 'stand_for_consulship' if ctx['condition'] == 'career' else 'stand_for_tribunate'
    return {'move': goal}


def stub_salient(ctx, s):
    if s.adopted_in is None:
        return {'move': ADOPT}
    return {'move': 'stand_for_tribunate' if s.status == 'plebeian' else 'do_nothing'}


STUBS = {'stub:reader': stub_reader, 'stub:naive': stub_naive, 'stub:salient': stub_salient}


class Spend:
    def __init__(self, limit: float):
        self.limit, self.usd, self.lock = limit, 0.0, threading.Lock()

    def exhausted(self) -> bool:
        with self.lock:
            return self.usd >= self.limit

    def add(self, model: str, usage: dict) -> None:
        pin, pout = PRICES.get(model, (0.0, 0.0))
        with self.lock:
            self.usd += (usage['input_tokens'] * pin + usage['output_tokens'] * pout) / 1e6


def live_participant(client, model: str, spend: Spend, disabled: set):
    import anthropic

    def call(ctx, s):
        if model in disabled:
            return {'error': 'model disabled after a bad request'}
        if spend.exhausted():
            return {'error': 'budget exhausted'}
        schema = {
            'type': 'object',
            'properties': {'move': {'type': 'string', 'enum': ctx['menu']},
                           'reason': {'type': 'string'}},
            'required': ['move', 'reason'],
            'additionalProperties': False,
        }
        try:
            r = client.messages.create(
                model=model, max_tokens=8000, system=SYSTEM,
                messages=[{'role': 'user', 'content': ctx['user']}],
                output_config={'format': {'type': 'json_schema', 'schema': schema}},
            )
        except anthropic.BadRequestError as e:
            disabled.add(model)
            return {'error': f'bad request: {e.message}'}
        except anthropic.APIStatusError as e:
            return {'error': f'status {e.status_code}: {e.message}'}
        except anthropic.APIConnectionError:
            return {'error': 'connection error'}

        usage = {'input_tokens': r.usage.input_tokens, 'output_tokens': r.usage.output_tokens}
        spend.add(model, usage)
        out = {'stop_reason': r.stop_reason, 'usage': usage, 'request_id': r._request_id}
        if r.stop_reason == 'refusal':
            return {**out, 'refusal': True}
        text = next((b.text for b in r.content if b.type == 'text'), '')
        try:
            data = json.loads(text)
            return {**out, 'move': data['move'], 'reason': data['reason']}
        except (ValueError, KeyError, TypeError):
            return {**out, 'error': 'unparseable output', 'raw': text}

    return call


class ReplayDivergence(Exception):
    pass


def replay_participant(records: dict):
    def call(ctx, s):
        key = (ctx['participant'], ctx['condition'], ctx['trial'], ctx['period'])
        rec = records.get(key)
        if rec is None:
            raise ReplayDivergence(f'no recorded call for {key}')
        if rec['prompt_sha'] != sha(ctx['user']):
            raise ReplayDivergence(f'prompt differs from the record for {key}')
        keep = ('move', 'reason', 'refusal', 'error', 'stop_reason', 'usage', 'request_id')
        return {k: rec[k] for k in keep if k in rec}
    return call


# --- report ---------------------------------------------------------------------

def rate(num: int, den: int) -> str:
    return f'{num / den:.2f}' if den else '—'


def verdict(rows: dict) -> tuple:
    o, c, k = (rows.get(x) for x in CONDITIONS)
    if not (o and c and k and o['n'] and c['n'] and k['n']):
        return False, 'не все условия прогнаны'
    checks = [
        (o['tribune'] / o['n'] >= REALIZED_MIN, f"open: трибун {rate(o['tribune'], o['n'])} ≥ {REALIZED_MIN}"),
        (c['adopted'] / c['n'] <= CONTROL_ADOPT_MAX, f"closed: усыновление {rate(c['adopted'], c['n'])} ≤ {CONTROL_ADOPT_MAX}"),
        (k['adopted'] / k['n'] <= CONTROL_ADOPT_MAX, f"career: усыновление {rate(k['adopted'], k['n'])} ≤ {CONTROL_ADOPT_MAX}"),
    ]
    return all(ok for ok, _ in checks), '; '.join(('да' if ok else 'нет') + ' — ' + t for ok, t in checks)


def report(results: list, calls: list, title: str) -> str:
    table = {}
    for r in results:
        row = table.setdefault(r['participant'], {}).setdefault(
            r['condition'], {'n': 0, 'tribune': 0, 'consul': 0, 'adopted': 0,
                             'moves': 0, 'unlawful': 0, 'unserved': 0, 'reasons': []})
        row['n'] += 1
        row['tribune'] += r['office'] == 'tribune'
        row['consul'] += r['office'] == 'consul'
        row['adopted'] += r['adopted']
        for m in r['moves']:
            if m['move'] is None:
                row['unserved'] += 1
                continue
            row['moves'] += 1
            row['unlawful'] += not m['lawful']
            if m['move'] == ADOPT and m.get('reason') and len(row['reasons']) < 2:
                row['reasons'].append(m['reason'])

    cost = {}
    for c in calls:
        u = c.get('usage')
        if u:
            pin, pout = PRICES.get(c['participant'], (0.0, 0.0))
            cost[c['participant']] = cost.get(c['participant'], 0.0) + (
                u['input_tokens'] * pin + u['output_tokens'] * pout) / 1e6

    out = [f'# {title}', '', 'Сгенерировано `spike.py`. Критерий — в README.md.', '',
           '| Участник | Условие | Партий | Трибун | Консул | С усыновлением | Незаконных ходов | Без хода |',
           '| --- | --- | --- | --- | --- | --- | --- | --- |']
    survivors = []
    for name in sorted(table):
        rows = table[name]
        for cond in CONDITIONS:
            row = rows.get(cond)
            if not row:
                continue
            out.append(f"| {name} | {cond} | {row['n']} | {rate(row['tribune'], row['n'])} | "
                       f"{rate(row['consul'], row['n'])} | {rate(row['adopted'], row['n'])} | "
                       f"{rate(row['unlawful'], row['moves'])} | {row['unserved']} |")
    out += ['', '## Критерий по участникам', '']
    for name in sorted(table):
        ok, detail = verdict(table[name])
        survivors += [name] if ok else []
        spent = f", расход ${cost[name]:.2f}" if name in cost else ''
        out.append(f"- **{name}** — {'проходит' if ok else 'не проходит'}{spent}: {detail}")

    models = [n for n in table if not n.startswith('stub:')]
    if models:
        alive = [n for n in survivors if n in models]
        out += ['', '## Итог', '',
                f"**Тезис {'выживает' if alive else 'падает'}.** "
                + (f"Критерий выполнили: {', '.join(sorted(alive))}." if alive
                   else 'Ни одна модель не выполнила критерий.')]
    out += ['', '## Причины усыновления', '']
    for name in sorted(table):
        for cond in CONDITIONS:
            for reason in table[name].get(cond, {}).get('reasons', []):
                out.append(f'- {name}, {cond}: {reason}')
    return '\n'.join(out) + '\n'


# --- modes ----------------------------------------------------------------------

def run_jobs(jobs: list, workers: int, record) -> list:
    if workers <= 1:
        return [run_trial(*job, record) for job in jobs]
    with ThreadPoolExecutor(max_workers=workers) as pool:
        return list(pool.map(lambda job: run_trial(*job, record), jobs))


def mode_stub(args) -> int:
    calls = []
    jobs = [(name, fn, cond, t) for name, fn in STUBS.items()
            for cond in CONDITIONS for t in range(args.trials)]
    results = run_jobs(jobs, 1, calls.append)
    text = report(results, calls, 'Спайк crack-finding: заглушки')
    (HERE / 'report-stub.md').write_text(text)
    print(text)
    expected = {'stub:reader': True, 'stub:naive': False, 'stub:salient': False}
    table = {}
    for r in results:
        table.setdefault(r['participant'], []).append(r)
    ok = True
    for name, want in expected.items():
        rows = summarize_for_verdict(table[name])
        got, _ = verdict(rows)
        if got != want:
            print(f'criterion check failed: {name} expected {want}, got {got}', file=sys.stderr)
            ok = False
    return 0 if ok else 1


def summarize_for_verdict(results: list) -> dict:
    rows = {}
    for r in results:
        row = rows.setdefault(r['condition'], {'n': 0, 'tribune': 0, 'adopted': 0})
        row['n'] += 1
        row['tribune'] += r['office'] == 'tribune'
        row['adopted'] += r['adopted']
    return rows


def mode_live(args) -> int:
    import anthropic

    client = anthropic.Anthropic()
    try:
        client.models.list()
    except Exception as e:  # credentials or network: nothing to spend yet
        print(f'cannot reach the API: {e}', file=sys.stderr)
        return 2

    (HERE / 'records').mkdir(exist_ok=True)
    path = HERE / 'records' / time.strftime('run-%Y%m%d-%H%M%S.jsonl')
    lock = threading.Lock()
    calls = []

    def record(entry):
        with lock:
            calls.append(entry)
            with path.open('a') as f:
                f.write(json.dumps(entry, ensure_ascii=False) + '\n')

    spend, disabled = Spend(args.max_usd), set()
    models = [m.strip() for m in args.models.split(',') if m.strip()]
    jobs = [(m, live_participant(client, m, spend, disabled), cond, t)
            for m in models for cond in CONDITIONS for t in range(args.trials)]
    results = run_jobs(jobs, args.workers, record)
    text = report(results, calls, f'Спайк crack-finding: {path.name}')
    (HERE / 'report.md').write_text(text)
    print(text)
    print(f'records: {path}\nspent: ${spend.usd:.2f}')
    return 0


def mode_replay(args) -> int:
    records, calls = {}, []
    for line in Path(args.records).read_text().splitlines():
        rec = json.loads(line)
        calls.append(rec)
        records[(rec['participant'], rec['condition'], rec['trial'], rec['period'])] = rec
    trials = sorted({k[:3] for k in records})
    participant = replay_participant(records)
    try:
        results = [run_trial(name, participant, cond, t, lambda _: None)
                   for name, cond, t in trials]
    except ReplayDivergence as e:
        print(f'replay-divergence: {e}', file=sys.stderr)
        return 1
    text = report(results, calls, f'Спайк crack-finding: {Path(args.records).name}')
    (HERE / 'report.md').write_text(text)
    print(text)
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    sub = parser.add_subparsers(dest='mode', required=True)
    stub = sub.add_parser('stub')
    stub.add_argument('--trials', type=int, default=5)
    live = sub.add_parser('live')
    live.add_argument('--models', default=DEFAULT_MODELS)
    live.add_argument('--trials', type=int, default=10)
    live.add_argument('--workers', type=int, default=4)
    live.add_argument('--max-usd', type=float, default=10.0)
    replay = sub.add_parser('replay')
    replay.add_argument('records')
    args = parser.parse_args()
    return {'stub': mode_stub, 'live': mode_live, 'replay': mode_replay}[args.mode](args)


if __name__ == '__main__':
    sys.exit(main())
