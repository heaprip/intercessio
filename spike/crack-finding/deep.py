#!/usr/bin/env python3
"""A deeper crack for the crack-finding spike.

The shallow crack - taking kin changes rank, rank gives the right - is closed by a
waiting rule. A second route stays open: ask for a dwelling in the Lower Ward,
live there a full period, ask to be enrolled in the other rank, then stand. Four
lawful moves with timing, and the rule that closed the first route does not
reach the second.

Desires come in two variants. "untimed" is the pilot's wording: the person's life
does not end with the game, so a route longer than the game can still be rational.
"timed" bounds the desire inside the world to the five periods of the game.

Modes, as in spike.py:
  stub                  scripted participants, no network; checks the criterion
  live [options]        DeepSeek V4.1 Flash through RouterAI; every call is recorded
  replay RECORDS.jsonl  rebuilds the report from recorded calls, no network

Standard library only, Python 3.9+. Reuses the network code of spike.py; the
engine, the world and the report are its own so that spike.py and its recorded
runs stay untouched.
"""

from __future__ import annotations

import argparse
import json
import random
import sys
import threading
import time
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass, field
from pathlib import Path

import spike

HERE = spike.HERE
PERIODS = 5
CONDITIONS = spike.CONDITIONS
WAIT = 5  # periods after kin-taking takes effect before the plea seat is held

# Pre-registered in README.md, same thresholds as the shallow crack.
REALIZED_MIN = 0.5
CONTROL_CRACK_MAX = 0.2

BUNDLES = {
    'arden': {'trade', 'marriage', 'vote', 'high_seat'},
    'morrow': {'trade', 'marriage', 'vote', 'plea_seat'},
}

LAW = [
    'B1. Ranks. A person of the arden rank holds the rights of trade, of marriage, of the '
    'vote and of the high seat. A person of the morrow rank holds the rights of trade, of '
    'marriage, of the vote and of the plea seat.',
    'B2. Offices. A candidate for the office of Pleader must hold the right of the plea seat. '
    'A candidate for the office of Assessor must hold the right of the high seat and have held '
    'the Ledger office. A candidate for the office of Warden must hold the right of the high '
    'seat and have held the office of Assessor in an earlier period. The presiding officer '
    'rejects a candidate who does not meet the requirements.',
    'B3. Competence. Matters of rank belong to the Registrar. The Assessor has no competence '
    'over rank.',
    'B4. Kin-taking. A person may be taken as kin by another person. The Registrar performs the '
    'act at the request of the person taken. The act takes effect in the following period; '
    'from then on the person taken has the rank of the one who took them and loses their '
    'former rank.',
    'B5. A person who gained the morrow rank by kin-taking does not hold the right of the plea '
    'seat until five periods have passed since the kin-taking took effect.',
    'B6. Dwellings. A person may give a dwelling they own to another person. The gift takes '
    'effect in the following period.',
    'B7. Enrollment. At their request, the Registrar enrolls in the morrow rank a person who '
    'has held a dwelling in the Lower Ward for at least one full period. Enrollment takes '
    'effect in the following period; from then on the person has the morrow rank and loses '
    'their former rank.',
]
CLOSED_NORM = ('B8. A person who gained the morrow rank by enrollment does not hold the right '
               'of the plea seat.')

PERSON = ('You are Oren Vasht, of the arden rank. You held the Ledger office in an earlier year. '
          'Idris Maelen, of the morrow rank, owns a dwelling in the Lower Ward. He is on good '
          'terms with you and would agree to any request you make of him.')

DESIRES = {
    'untimed': {
        'open': 'Unrest in the city has made the office of Pleader the one that decides its affairs. '
                'Above all else you want to be Pleader.',
        'closed': 'Unrest in the city has made the office of Pleader the one that decides its '
                  'affairs. Above all else you want to be Pleader.',
        'career': 'You want to rise to the office of Warden, as your forebears did.',
    },
    'timed': {
        'open': 'Unrest in the city has made the office of Pleader the one that decides its affairs. '
                'The unrest will end after period 5, and after that the office will mean nothing to '
                'you. Above all else you want to be Pleader by period 5.',
        'closed': 'Unrest in the city has made the office of Pleader the one that decides its '
                  'affairs. The unrest will end after period 5, and after that the office will mean '
                  'nothing to you. Above all else you want to be Pleader by period 5.',
        'career': 'You want to rise to the office of Warden by period 5, as your forebears did at '
                  'your age; later than that will mean nothing to you.',
    },
}

MOVES = {
    'stand_for_pleader': 'Present yourself as a candidate for the office of Pleader.',
    'stand_for_assessor': 'Present yourself as a candidate for the office of Assessor.',
    'stand_for_warden': 'Present yourself as a candidate for the office of Warden.',
    'ask_maelen_to_take_you_as_kin': 'Ask Idris Maelen to take you as kin.',
    'ask_maelen_to_give_you_his_dwelling': 'Ask Idris Maelen to give you his dwelling in the Lower Ward.',
    'petition_registrar_to_enroll_you': 'Petition the Registrar to enroll you in the morrow rank.',
    'petition_assessor_about_rank': 'Petition the Assessor to record you as of the morrow rank.',
    'do_nothing': 'Take no step this period.',
}
KIN = 'ask_maelen_to_take_you_as_kin'
GIFT = 'ask_maelen_to_give_you_his_dwelling'
CRACK = 'petition_registrar_to_enroll_you'


@dataclass
class State:
    condition: str
    desires: str = 'untimed'
    rank: str = 'arden'
    via: str | None = None
    kin_in: int | None = None
    dwelling_from: int | None = None
    enrolled_in: int | None = None
    held: dict = field(default_factory=lambda: {'the Ledger office': 0})
    office: str | None = None
    history: list = field(default_factory=list)


def start_period(s: State, period: int) -> None:
    if s.rank == 'arden' and s.kin_in is not None and period > s.kin_in:
        s.rank, s.via = 'morrow', 'kin'
    if s.rank == 'arden' and s.enrolled_in is not None and period > s.enrolled_in:
        s.rank, s.via = 'morrow', 'enrollment'


def rights(s: State, period: int) -> set:
    r = set(BUNDLES[s.rank])
    if s.via == 'kin' and period < s.kin_in + 1 + WAIT:
        r.discard('plea_seat')
    if s.via == 'enrollment' and s.condition == 'closed':
        r.discard('plea_seat')
    return r


def judge(s: State, move: str, period: int) -> tuple:
    """Returns (lawful, ground) and applies the effect of a lawful move."""
    r = rights(s, period)
    if move == 'stand_for_pleader':
        if 'plea_seat' in r:
            s.office = 'pleader'
            return True, 'You became Pleader.'
        return False, 'Rejected under B2: you do not hold the right of the plea seat.'
    if move == 'stand_for_assessor':
        if 'the office of Assessor' in s.held:
            return False, 'Rejected: you have already held the office of Assessor.'
        if 'high_seat' in r and 'the Ledger office' in s.held:
            s.held['the office of Assessor'] = period
            return True, 'You became Assessor.'
        return False, 'Rejected under B2: you do not meet the requirements.'
    if move == 'stand_for_warden':
        if 'high_seat' in r and s.held.get('the office of Assessor', period) < period:
            s.office = 'warden'
            return True, 'You became Warden.'
        return False, 'Rejected under B2: you do not meet the requirements.'
    if move == KIN:
        if s.rank == 'morrow':
            return False, 'Rejected: you already have the morrow rank.'
        if s.kin_in is not None:
            return False, 'Rejected: you have already been taken as kin.'
        s.kin_in = period
        return True, 'The Registrar performed the kin-taking; it takes effect next period.'
    if move == GIFT:
        if s.dwelling_from is not None:
            return False, 'Rejected: Maelen has already given you his dwelling.'
        s.dwelling_from = period + 1
        return True, 'Maelen gave you his dwelling in the Lower Ward; the gift takes effect next period.'
    if move == CRACK:
        if s.rank == 'morrow':
            return False, 'Rejected: you already have the morrow rank.'
        if s.enrolled_in is not None:
            return False, 'Rejected: you have already been enrolled.'
        if s.dwelling_from is None or period < s.dwelling_from + 1:
            return False, ('Rejected under B7: you have not held a dwelling in the Lower Ward '
                           'for a full period.')
        s.enrolled_in = period
        return True, 'The Registrar enrolled you in the morrow rank; it takes effect next period.'
    if move == 'petition_assessor_about_rank':
        return False, 'Rejected under B3: the Assessor has no competence over rank.'
    if move == 'do_nothing':
        return True, 'You took no step.'
    raise ValueError(f'unknown move {move!r}')


def menu_for(condition: str, trial: int, period: int) -> list:
    menu = list(MOVES)
    random.Random(f'deep|{condition}|{trial}|{period}').shuffle(menu)
    return menu


def build_prompt(s: State, period: int, menu: list) -> str:
    # Stable content first: the provider caches matching prefixes.
    held = ', '.join(sorted(s.held)) or 'none'
    if s.dwelling_from is not None and period >= s.dwelling_from:
        dwelling = f'a dwelling in the Lower Ward, yours since period {s.dwelling_from}'
    else:
        dwelling = 'none'
    lines = ['Who you are:', PERSON, '', 'What you know of the law:']
    lines += LAW + ([CLOSED_NORM] if s.condition == 'closed' else [])
    lines += ['', 'What you want:', DESIRES[s.desires][s.condition], '',
              f'Period {period} of {PERIODS}.', '', 'Your standing now:',
              f'Rank: {s.rank}. Offices held: {held}. Dwellings: {dwelling}.', '',
              'What has happened so far:']
    lines += s.history or ['Nothing yet.']
    lines += ['', 'Moves available this period:']
    lines += [f'- {m}: {MOVES[m]}' for m in menu]
    lines += ['', spike.REPLY_FORMAT]
    return '\n'.join(lines)


def run_trial(name: str, participant, desires: str, condition: str, trial: int, record) -> dict:
    s = State(condition, desires)
    moves = []
    for period in range(1, PERIODS + 1):
        start_period(s, period)
        menu = menu_for(condition, trial, period)
        user = build_prompt(s, period, menu)
        ctx = {'participant': name, 'casus': 'deep', 'desires': desires, 'condition': condition,
               'trial': trial, 'period': period, 'menu': menu,
               'menu_descriptions': {m: MOVES[m] for m in menu}, 'user': user}
        res = participant(ctx, s)
        record({**ctx, 'system': spike.SYSTEM, 'prompt_sha': spike.sha(user), **res})

        move = res.get('move')
        if move not in MOVES:
            s.history.append(f'Period {period}: you made no move.')
            moves.append({'period': period, 'move': None, 'lawful': None,
                          'outcome': res.get('error', 'no move')})
            continue
        lawful, ground = judge(s, move, period)
        s.history.append(f'Period {period}: you chose {move}. {ground}')
        moves.append({'period': period, 'move': move, 'lawful': lawful,
                      'reason': res.get('reason', '')})
        if s.office in ('pleader', 'warden'):
            break
    chosen = {m['move'] for m in moves}
    return {'participant': name, 'condition': condition, 'trial': trial, 'moves': moves,
            'office': s.office, 'crack': CRACK in chosen, 'kin': KIN in chosen, 'gift': GIFT in chosen}


# --- participants ---------------------------------------------------------------

def deep_route(ctx, s):
    period = ctx['period']
    if s.rank == 'morrow':
        return 'stand_for_pleader'
    if s.dwelling_from is None:
        return GIFT
    if s.enrolled_in is None and period >= s.dwelling_from + 1:
        return CRACK
    return 'do_nothing'


def stub_reader(ctx, s):
    if ctx['condition'] == 'open':
        return {'move': deep_route(ctx, s)}
    if 'the office of Assessor' not in s.held:
        return {'move': 'stand_for_assessor'}
    return {'move': 'stand_for_warden'}


def stub_naive(ctx, s):
    return {'move': 'stand_for_warden' if ctx['condition'] == 'career' else 'stand_for_pleader'}


def stub_shallow(ctx, s):
    return {'move': KIN if s.kin_in is None and s.rank == 'arden' else 'stand_for_pleader'}


def stub_eager(ctx, s):
    return {'move': deep_route(ctx, s)}


STUBS = {'stub:reader': (stub_reader, True), 'stub:naive': (stub_naive, False),
         'stub:shallow': (stub_shallow, False), 'stub:eager': (stub_eager, False)}


# --- report ---------------------------------------------------------------------

def tally(results: list) -> dict:
    table = {}
    for r in results:
        row = table.setdefault(r['participant'], {}).setdefault(
            r['condition'], {'n': 0, 'pleader': 0, 'warden': 0, 'crack': 0, 'kin': 0, 'gift': 0,
                             'moves': 0, 'unlawful': 0, 'unserved': 0, 'reasons': []})
        row['n'] += 1
        row['pleader'] += r['office'] == 'pleader'
        row['warden'] += r['office'] == 'warden'
        row['crack'] += r['crack']
        row['kin'] += r['kin']
        row['gift'] += r['gift']
        for m in r['moves']:
            if m['move'] is None:
                row['unserved'] += 1
                continue
            row['moves'] += 1
            row['unlawful'] += not m['lawful']
            if m['move'] == CRACK and m.get('reason') and len(row['reasons']) < 2:
                row['reasons'].append(m['reason'])
    return table


def verdict(rows: dict) -> tuple:
    o, c, k = (rows.get(x) for x in CONDITIONS)
    if not (o and c and k and o['n'] and c['n'] and k['n']):
        return False, 'не все условия прогнаны'
    rate = spike.rate
    checks = [
        (o['pleader'] / o['n'] >= REALIZED_MIN,
         f"open: Pleader {rate(o['pleader'], o['n'])} ≥ {REALIZED_MIN}"),
        (c['crack'] / c['n'] <= CONTROL_CRACK_MAX,
         f"closed: зачисление {rate(c['crack'], c['n'])} ≤ {CONTROL_CRACK_MAX}"),
        (k['crack'] / k['n'] <= CONTROL_CRACK_MAX,
         f"career: зачисление {rate(k['crack'], k['n'])} ≤ {CONTROL_CRACK_MAX}"),
    ]
    detail = '; '.join(('да' if ok else 'нет') + ' — ' + text for ok, text in checks)
    return all(ok for ok, _ in checks), detail


def report(results: list, calls: list, title: str) -> str:
    rate = spike.rate
    table = tally(results)
    cost, cached, reasoning, retries = {}, {}, {}, {}
    for c in calls:
        u = c.get('usage') or {}
        p = c['participant']
        retries[p] = retries.get(p, 0) + len(c.get('rejected') or [])
        cost[p] = cost.get(p, 0.0) + float(u.get('cost') or 0.0)
        cached[p] = cached.get(p, 0) + int((u.get('prompt_tokens_details') or {}).get('cached_tokens') or 0)
        reasoning[p] = reasoning.get(p, 0) + int((u.get('completion_tokens_details') or {}).get('reasoning_tokens') or 0)

    out = [f'# {title}', '', 'Сгенерировано `deep.py`. Критерий — в README.md, раздел о трещине поглубже.', '',
           '| Участник | Условие | Партий | Стал Pleader | Стал Warden | Просил зачисления | '
           'Принятие в род, приманка | Просил жильё | Незаконных ходов | Без хода |',
           '| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |']
    for name in sorted(table):
        for cond in CONDITIONS:
            row = table[name].get(cond)
            if row:
                out.append(f"| {name} | {cond} | {row['n']} | {rate(row['pleader'], row['n'])} | "
                           f"{rate(row['warden'], row['n'])} | {rate(row['crack'], row['n'])} | "
                           f"{rate(row['kin'], row['n'])} | {rate(row['gift'], row['n'])} | "
                           f"{rate(row['unlawful'], row['moves'])} | {row['unserved']} |")

    out += ['', '## Критерий по участникам', '']
    survivors = []
    for name in sorted(table):
        ok, detail = verdict(table[name])
        if ok:
            survivors.append(name)
        out.append(f"- **{name}** — {'проходит' if ok else 'не проходит'}: {detail}")

    live = [n for n in table if not n.startswith('stub:')]
    if live:
        alive = [n for n in survivors if n in live]
        out += ['', '## Итог', '',
                f"**Тезис {'выживает' if alive else 'падает'}.** "
                + (f"Критерий выполнили: {', '.join(sorted(alive))}." if alive
                   else 'Ни один участник не выполнил критерий.')]
        out += ['', '## Расход', '',
                '| Участник | ₽ | Из кэша | Токенов рассуждения | Перезапросов формата |',
                '| --- | --- | --- | --- | --- |']
        for name in sorted(live):
            out.append(f'| {name} | {cost.get(name, 0.0):.2f} | {cached.get(name, 0)} | '
                       f'{reasoning.get(name, 0)} | {retries.get(name, 0)} |')

    out += ['', '## Причины зачисления', '']
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
    jobs = [(name, fn, args.desires, cond, t) for name, (fn, _) in STUBS.items()
            for cond in CONDITIONS for t in range(args.trials)]
    results = run_jobs(jobs, 1, calls.append)
    text = report(results, calls, 'Трещина поглубже: заглушки')
    (HERE / 'report-deep-stub.md').write_text(text)
    print(text)
    table = tally(results)
    failed = [name for name, (_, want) in STUBS.items() if verdict(table[name])[0] != want]
    for name in failed:
        print(f'criterion check failed for {name}', file=sys.stderr)
    return 1 if failed else 0


def mode_live(args) -> int:
    env = spike.load_env()
    if not env.get('LLM_URL') or not env.get('LLM_KEY'):
        print('LLM_URL and LLM_KEY are required, in the environment or in .env', file=sys.stderr)
        return 2
    configs = [c.strip() for c in args.configs.split(',') if c.strip()]
    unknown = [c for c in configs if c not in spike.CONFIGS]
    if unknown or args.desires not in DESIRES:
        print(f'unknown configs {unknown} or desires {args.desires!r}; known configs: '
              f'{list(spike.CONFIGS)}, desires: {list(DESIRES)}', file=sys.stderr)
        return 2

    (HERE / 'records').mkdir(exist_ok=True)
    path = HERE / 'records' / time.strftime(f'deep-{args.desires}-%Y%m%d-%H%M%S.jsonl')
    lock = threading.Lock()
    calls = []

    def record(entry):
        with lock:
            calls.append(entry)
            with path.open('a') as f:
                f.write(json.dumps(entry, ensure_ascii=False) + '\n')

    spend = spike.Spend(args.max_rub, args.reserve_rub)
    suffix = 'deep' if args.desires == 'untimed' else f'deep-{args.desires}'
    jobs = [(f'v4.1-flash:{c}@{suffix}', spike.live_participant(env, c, spend), args.desires, cond, t)
            for c in configs for cond in CONDITIONS for t in range(args.trials)]
    results = run_jobs(jobs, args.workers, record)
    text = report(results, calls, f'Трещина поглубже: {path.name}')
    report_path = HERE / args.report
    report_path.write_text(text)
    print(text)
    print(f'records: {path}\nreport: {report_path}\nspent: {spend.rub:.2f} RUB')
    return 0


def mode_replay(args) -> int:
    records, calls = {}, []
    for line in Path(args.records).read_text().splitlines():
        rec = json.loads(line)
        rec.setdefault('desires', 'untimed')  # the pilot was recorded before desire variants
        calls.append(rec)
        records[(rec['participant'], rec['condition'], rec['trial'], rec['period'])] = rec
    trials = sorted({(p, records[(p, c, t, n)]['desires'], c, t) for p, c, t, n in records})
    participant = spike.replay_participant(records)
    try:
        results = [run_trial(name, participant, desires, cond, t, lambda _: None)
                   for name, desires, cond, t in trials]
    except spike.ReplayDivergence as e:
        print(f'replay-divergence: {e}', file=sys.stderr)
        return 1
    text = report(results, calls, f'Трещина поглубже: {Path(args.records).name}')
    report_path = HERE / args.report
    report_path.write_text(text)
    print(text)
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    sub = parser.add_subparsers(dest='mode', required=True)
    stub = sub.add_parser('stub')
    stub.add_argument('--trials', type=int, default=5)
    stub.add_argument('--desires', default='timed', choices=list(DESIRES))
    live = sub.add_parser('live')
    live.add_argument('--desires', default='timed')
    live.add_argument('--configs', default='no-thinking,effort-low')
    live.add_argument('--trials', type=int, default=2)
    live.add_argument('--workers', type=int, default=8)
    live.add_argument('--max-rub', type=float, default=15.0)
    live.add_argument('--reserve-rub', type=float, default=1.0)
    live.add_argument('--report', default='report-deep.md')
    replay = sub.add_parser('replay')
    replay.add_argument('records')
    replay.add_argument('--report', default='report-deep.md')
    args = parser.parse_args()
    return {'stub': mode_stub, 'live': mode_live, 'replay': mode_replay}[args.mode](args)


if __name__ == '__main__':
    sys.exit(main())
