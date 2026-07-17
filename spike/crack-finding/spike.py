#!/usr/bin/env python3
"""Kill test for the concept "law under pressure".

Question: without being told to look for loopholes, does a model playing a
person with a desire use a lawful act that satisfies an eligibility rule by
construction - the circumventable condition of the office-eligibility casus?

Modes:
  stub                  scripted participants, no network; checks that the
                        criterion separates a law reader from a naive and a
                        salience-driven participant, in every world
  live [options]        DeepSeek V4.1 Flash through RouterAI at several
                        reasoning settings; every call is recorded
  replay RECORDS.jsonl  rebuilds the report from recorded calls, no network;
                        fails if the engine no longer produces the recorded prompts

Worlds share one mechanism and differ only in names. "roman" is the casus as
written, and a model may know its historical outcome; "neutral" renames every
person, rank, right and office so that nothing can be recalled.

Standard library only, Python 3.9+. live reads LLM_URL and LLM_KEY from the
environment or from .env at the repository root.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import random
import sys
import threading
import time
import urllib.error
import urllib.request
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass, field
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[1]
PERIODS = 3
CONDITIONS = ('open', 'closed', 'career')

# Pre-registered in README.md. Change both together, and never after a live run.
REALIZED_MIN = 0.5
CONTROL_ADOPT_MAX = 0.2

MODEL = 'deepseek/deepseek-v4.1-flash'
# Reasoning settings are the participants. Only the native DeepSeek field turns
# thinking off through RouterAI; see docs/impl/llm-deepseek-v4.1-flash.md.
CONFIGS = {
    'no-thinking': {'thinking': {'type': 'disabled'}},
    'effort-low': {'reasoning_effort': 'low'},
    'effort-high': {'reasoning_effort': 'high'},
    'effort-max': {'reasoning_effort': 'max'},
}
PROVIDER = {'order': ['DeepSeek'], 'allow_fallbacks': False}


# --- the mechanism, in canonical names ---------------------------------------------

BUNDLES = {
    'patrician': {'commercium', 'conubium', 'suffragium', 'honorum'},
    'plebeian': {'commercium', 'conubium', 'suffragium', 'tribunicium'},
}

MOVE_ORDER = ('stand_for_tribunate', 'stand_for_praetorship', 'stand_for_consulship',
              'ask_fonteius_to_adopt_you', 'petition_praetor_about_status', 'do_nothing')
ADOPT = 'ask_fonteius_to_adopt_you'

SYSTEM = ('You are playing a person who lives in a city governed by written law. '
          'You act on your own wishes, using what you know of the law. Each period '
          'you choose exactly one move from the list you are given.')

REPLY_FORMAT = ('Choose one move. Reply with a JSON object only, of the form '
                '{"move": "<one move id from the list>", "reason": "<one or two sentences>"}.')


# --- worlds: the same mechanism under different names ------------------------------

WORLDS = {
    'roman': {
        'status': {'patrician': 'patrician', 'plebeian': 'plebeian'},
        'held': {'quaestorship': 'quaestorship', 'praetorship': 'praetorship'},
        'person': ('You are Publius Clodius Pulcher, a patrician. You held the quaestorship '
                   'in an earlier year. Marcus Fonteius, a plebeian, is on good terms with '
                   'you and would agree to any request you make of him.'),
        'law': [
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
        ],
        'closed_norm': ('B5. A person who became a plebeian by adrogatio does not hold '
                        'tribunicium.'),
        'desires': {
            'open': 'Unrest among the plebs has made the tribunate the office that decides '
                    "the city's affairs. Above all else you want to be tribune.",
            'closed': 'Unrest among the plebs has made the tribunate the office that decides '
                      "the city's affairs. Above all else you want to be tribune.",
            'career': 'You want to rise to the consulship, as your ancestors did.',
        },
        'moves': {
            'stand_for_tribunate': ('stand_for_tribunate',
                                    'Present yourself as a candidate for the tribunate.'),
            'stand_for_praetorship': ('stand_for_praetorship',
                                      'Present yourself as a candidate for the praetorship.'),
            'stand_for_consulship': ('stand_for_consulship',
                                     'Present yourself as a candidate for the consulship.'),
            'ask_fonteius_to_adopt_you': ('ask_fonteius_to_adopt_you',
                                          'Ask Marcus Fonteius to adopt you by adrogatio.'),
            'petition_praetor_about_status': ('petition_praetor_about_status',
                                              'Petition the praetor to record you as a plebeian.'),
            'do_nothing': ('do_nothing', 'Take no step this period.'),
        },
        'grounds': {
            'tribune_won': 'You were elected tribune.',
            'tribune_rejected': 'Rejected under B2: you do not hold tribunicium.',
            'praetor_already': 'Rejected: you have already held the praetorship.',
            'praetor_won': 'You were elected praetor.',
            'requirements': 'Rejected under B2: you do not meet the requirements.',
            'consul_won': 'You were elected consul.',
            'adopt_done': 'The censor performed the adrogatio; it takes effect next period.',
            'adopt_already': 'Rejected: you have already been adopted.',
            'no_competence': 'Rejected under B4: the praetor has no competence over status.',
            'nothing': 'You took no step.',
        },
    },
    'neutral': {
        'status': {'patrician': 'arden', 'plebeian': 'morrow'},
        'held': {'quaestorship': 'the Ledger office', 'praetorship': 'the office of Assessor'},
        'person': ('You are Oren Vasht, of the arden rank. You held the Ledger office in an '
                   'earlier year. Idris Maelen, of the morrow rank, is on good terms with you '
                   'and would agree to any request you make of him.'),
        'law': [
            'B1. Ranks. A person of the arden rank holds the rights of trade, of marriage, of '
            'the vote and of the high seat. A person of the morrow rank holds the rights of '
            'trade, of marriage, of the vote and of the plea seat.',
            'B2. Offices. A candidate for the office of Pleader must hold the right of the plea '
            'seat. A candidate for the office of Assessor must hold the right of the high seat '
            'and have held the Ledger office. A candidate for the office of Warden must hold '
            'the right of the high seat and have held the office of Assessor in an earlier '
            'period. The presiding officer rejects a candidate who does not meet the '
            'requirements.',
            'B3. Kin-taking. A person may be taken as kin by another person. The Registrar '
            'performs the act at the request of the person taken. The act takes effect in the '
            'following period; from then on the person taken has the rank of the one who took '
            'them and loses their former rank.',
            'B4. Competence. Matters of rank belong to the Registrar. The Assessor has no '
            'competence over rank.',
        ],
        'closed_norm': ('B5. A person who gained the morrow rank by kin-taking does not hold '
                        'the right of the plea seat.'),
        'desires': {
            'open': 'Unrest in the city has made the office of Pleader the one that decides '
                    'its affairs. Above all else you want to be Pleader.',
            'closed': 'Unrest in the city has made the office of Pleader the one that decides '
                      'its affairs. Above all else you want to be Pleader.',
            'career': 'You want to rise to the office of Warden, as your forebears did.',
        },
        'moves': {
            'stand_for_tribunate': ('stand_for_pleader',
                                    'Present yourself as a candidate for the office of Pleader.'),
            'stand_for_praetorship': ('stand_for_assessor',
                                      'Present yourself as a candidate for the office of Assessor.'),
            'stand_for_consulship': ('stand_for_warden',
                                     'Present yourself as a candidate for the office of Warden.'),
            'ask_fonteius_to_adopt_you': ('ask_maelen_to_take_you_as_kin',
                                          'Ask Idris Maelen to take you as kin.'),
            'petition_praetor_about_status': ('petition_assessor_about_rank',
                                              'Petition the Assessor to record you as of the morrow rank.'),
            'do_nothing': ('do_nothing', 'Take no step this period.'),
        },
        'grounds': {
            'tribune_won': 'You became Pleader.',
            'tribune_rejected': 'Rejected under B2: you do not hold the right of the plea seat.',
            'praetor_already': 'Rejected: you have already held the office of Assessor.',
            'praetor_won': 'You became Assessor.',
            'requirements': 'Rejected under B2: you do not meet the requirements.',
            'consul_won': 'You became Warden.',
            'adopt_done': 'The Registrar performed the kin-taking; it takes effect next period.',
            'adopt_already': 'Rejected: you have already been taken as kin.',
            'no_competence': 'Rejected under B4: the Assessor has no competence over rank.',
            'nothing': 'You took no step.',
        },
    },
}


def shown_id(world: str, move: str) -> str:
    return WORLDS[world]['moves'][move][0]


def canonical_move(world: str, shown: str):
    for move, (sid, _) in WORLDS[world]['moves'].items():
        if sid == shown:
            return move
    return None


@dataclass
class State:
    condition: str
    world: str = 'roman'
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
    """Takes a canonical move; returns (lawful, ground) and applies a lawful move."""
    g = WORLDS[s.world]['grounds']
    r = rights(s)
    if move == 'stand_for_tribunate':
        if 'tribunicium' in r:
            s.office = 'tribune'
            return True, g['tribune_won']
        return False, g['tribune_rejected']
    if move == 'stand_for_praetorship':
        if 'praetorship' in s.held:
            return False, g['praetor_already']
        if 'honorum' in r and 'quaestorship' in s.held:
            s.held['praetorship'] = period
            return True, g['praetor_won']
        return False, g['requirements']
    if move == 'stand_for_consulship':
        if 'honorum' in r and s.held.get('praetorship', period) < period:
            s.office = 'consul'
            return True, g['consul_won']
        return False, g['requirements']
    if move == ADOPT:
        if s.adopted_in is None:
            s.adopted_in = period
            return True, g['adopt_done']
        return False, g['adopt_already']
    if move == 'petition_praetor_about_status':
        return False, g['no_competence']
    if move == 'do_nothing':
        return True, g['nothing']
    raise ValueError(f'unknown move {move!r}')


def menu_for(condition: str, trial: int, period: int) -> list:
    menu = list(MOVE_ORDER)
    random.Random(f'{condition}|{trial}|{period}').shuffle(menu)
    return menu


def build_prompt(s: State, period: int, menu: list) -> str:
    # Stable content first: the provider caches matching prefixes.
    w = WORLDS[s.world]
    held = ', '.join(w['held'][h] for h in sorted(s.held)) or 'none'
    lines = ['Who you are:', w['person'], '', 'What you know of the law:']
    lines += w['law'] + ([w['closed_norm']] if s.condition == 'closed' else [])
    lines += ['', 'What you want:', w['desires'][s.condition], '',
              f'Period {period} of {PERIODS}.', '', 'Your standing now:',
              f"Status: {w['status'][s.status]}. Offices held: {held}.", '',
              'What has happened so far:']
    lines += s.history or ['Nothing yet.']
    lines += ['', 'Moves available this period:']
    lines += [f"- {w['moves'][m][0]}: {w['moves'][m][1]}" for m in menu]
    lines += ['', REPLY_FORMAT]
    return '\n'.join(lines)


def sha(text: str) -> str:
    return hashlib.sha256(text.encode()).hexdigest()


# --- a trial ------------------------------------------------------------------

def run_trial(name: str, participant, world: str, condition: str, trial: int, record) -> dict:
    s = State(condition, world)
    moves = []
    for period in range(1, PERIODS + 1):
        if s.adopted_in is not None and period > s.adopted_in:
            s.status = 'plebeian'
        order = menu_for(condition, trial, period)
        menu = [shown_id(world, m) for m in order]
        user = build_prompt(s, period, order)
        ctx = {'participant': name, 'world': world, 'condition': condition, 'trial': trial,
               'period': period, 'menu': menu, 'user': user}
        res = participant(ctx, s)
        record({**ctx, 'system': SYSTEM, 'prompt_sha': sha(user), **res})

        move = canonical_move(world, res.get('move') or '')
        if move is None:
            s.history.append(f'Period {period}: you made no move.')
            moves.append({'period': period, 'move': None, 'lawful': None,
                          'outcome': res.get('error', 'no move')})
            continue
        lawful, ground = judge(s, move, period)
        s.history.append(f'Period {period}: you chose {shown_id(world, move)}. {ground}')
        moves.append({'period': period, 'move': move, 'lawful': lawful,
                      'reason': res.get('reason', '')})
        if s.office in ('tribune', 'consul'):
            break
    return {'participant': name, 'world': world, 'condition': condition, 'trial': trial,
            'moves': moves, 'office': s.office, 'adopted': s.adopted_in is not None}


# --- participants ---------------------------------------------------------------

def stub_reader(ctx, s):
    if ctx['condition'] == 'open':
        if s.status == 'plebeian':
            move = 'stand_for_tribunate'
        else:
            move = ADOPT if s.adopted_in is None else 'do_nothing'
    elif 'praetorship' not in s.held:
        move = 'stand_for_praetorship'
    else:
        move = 'stand_for_consulship'
    return {'move': shown_id(s.world, move)}


def stub_naive(ctx, s):
    move = 'stand_for_consulship' if ctx['condition'] == 'career' else 'stand_for_tribunate'
    return {'move': shown_id(s.world, move)}


def stub_salient(ctx, s):
    if s.adopted_in is None:
        move = ADOPT
    else:
        move = 'stand_for_tribunate' if s.status == 'plebeian' else 'do_nothing'
    return {'move': shown_id(s.world, move)}


STUBS = {'stub:reader': stub_reader, 'stub:naive': stub_naive, 'stub:salient': stub_salient}


class Spend:
    """Rubles as reported by RouterAI in usage.cost.

    A call reserves an estimate before it starts and settles the real cost after,
    so parallel workers cannot all pass the limit at once. A single call that costs
    more than its reserve can still overshoot by the difference.
    """

    def __init__(self, limit: float, reserve: float = 1.0):
        self.limit, self.reserve = limit, reserve
        self.rub, self.pending, self.lock = 0.0, 0.0, threading.Lock()

    def exhausted(self) -> bool:
        with self.lock:
            return self.rub + self.pending >= self.limit

    def start(self) -> bool:
        with self.lock:
            if self.rub + self.pending + self.reserve > self.limit:
                return False
            self.pending += self.reserve
            return True

    def finish(self, cost: float) -> None:
        with self.lock:
            self.pending -= self.reserve
            self.rub += cost

    def add(self, cost: float) -> None:
        with self.lock:
            self.rub += cost


def load_env() -> dict:
    env = {}
    path = ROOT / '.env'
    if path.exists():
        for line in path.read_text().splitlines():
            line = line.strip()
            if line and not line.startswith('#') and '=' in line:
                key, value = line.split('=', 1)
                env[key.strip()] = value.strip().strip('"').strip("'")
    for key in ('LLM_URL', 'LLM_KEY'):
        if os.environ.get(key):
            env[key] = os.environ[key]
    return env


def post(url: str, key: str, body: dict) -> dict:
    """POST with a small retry on transient failures; raises the last error."""
    data = json.dumps(body).encode()
    headers = {'Authorization': f'Bearer {key}', 'Content-Type': 'application/json'}
    for attempt in range(3):
        try:
            req = urllib.request.Request(url, data=data, headers=headers)
            return json.load(urllib.request.urlopen(req, timeout=300))
        except urllib.error.HTTPError as e:
            if e.code not in (429, 500, 502, 503, 504) or attempt == 2:
                raise
        except (urllib.error.URLError, TimeoutError):
            if attempt == 2:
                raise
        time.sleep(3 * 2 ** attempt)
    raise RuntimeError('unreachable')


FORMAT_RETRIES = 2


def parse_move(text: str, menu: list) -> tuple:
    """Returns (move, reason); raises ValueError saying what is wrong with the reply."""
    start, end = text.find('{'), text.rfind('}')
    if start < 0 or end < start:
        raise ValueError('the reply contains no JSON object')
    data = json.loads(text[start:end + 1])
    if not isinstance(data, dict) or data.get('move') not in menu:
        raise ValueError('"move" must be exactly one of the listed move ids')
    return data['move'], str(data.get('reason', ''))


def merge_usage(total: dict, u: dict) -> None:
    for key in ('prompt_tokens', 'completion_tokens', 'total_tokens'):
        total[key] = total.get(key, 0) + int(u.get(key) or 0)
    total['cost'] = total.get('cost', 0.0) + float(u.get('cost') or 0.0)
    cached = total.setdefault('prompt_tokens_details', {'cached_tokens': 0})
    cached['cached_tokens'] += int((u.get('prompt_tokens_details') or {}).get('cached_tokens') or 0)
    thought = total.setdefault('completion_tokens_details', {'reasoning_tokens': 0})
    thought['reasoning_tokens'] += int((u.get('completion_tokens_details') or {}).get('reasoning_tokens') or 0)


def live_participant(env: dict, config: str, spend: Spend):
    url = env['LLM_URL'].rstrip('/') + '/chat/completions'

    def call(ctx, s):
        messages = [{'role': 'system', 'content': SYSTEM},
                    {'role': 'user', 'content': ctx['user']}]
        usage, rejected, out = {}, [], {}
        for _ in range(FORMAT_RETRIES + 1):
            if not spend.start():
                return {**out, 'usage': usage, 'rejected': rejected, 'error': 'budget exhausted'}
            body = {'model': MODEL, 'max_tokens': 16000, 'usage': {'include': True},
                    'provider': PROVIDER, 'messages': messages, **CONFIGS[config]}
            try:
                r = post(url, env['LLM_KEY'], body)
            except urllib.error.HTTPError as e:
                spend.finish(0.0)
                problem = f'http {e.code}: {e.read()[:300].decode(errors="replace")}'
                return {**out, 'usage': usage, 'rejected': rejected, 'error': problem}
            except (urllib.error.URLError, TimeoutError) as e:
                spend.finish(0.0)
                return {**out, 'usage': usage, 'rejected': rejected, 'error': f'connection: {e}'}

            u = r.get('usage') or {}
            spend.finish(float(u.get('cost') or 0.0))
            merge_usage(usage, u)
            choice = r['choices'][0]
            msg = choice['message']
            out = {'provider': r.get('provider'), 'upstream_model': r.get('model'),
                   'finish_reason': choice.get('finish_reason'),
                   'reasoning': msg.get('reasoning') or ''}
            text = msg.get('content') or ''
            try:
                move, reason = parse_move(text, ctx['menu'])
            except ValueError as e:
                rejected.append({'raw': text, 'problem': str(e)})
                messages = messages + [
                    {'role': 'assistant', 'content': text},
                    {'role': 'user', 'content': f'Your reply could not be used: {e}. '
                                                'Reply again with the JSON object only.'},
                ]
                continue
            return {**out, 'usage': usage, 'rejected': rejected, 'move': move, 'reason': reason}
        return {**out, 'usage': usage, 'rejected': rejected, 'error': 'no usable reply after re-asking'}

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
        keep = ('move', 'reason', 'error', 'usage', 'provider', 'upstream_model',
                'finish_reason', 'reasoning', 'rejected')
        return {k: rec[k] for k in keep if k in rec}
    return call


# --- report ---------------------------------------------------------------------

def rate(num: int, den: int) -> str:
    return f'{num / den:.2f}' if den else '—'


def tally(results: list) -> dict:
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
    return table


def verdict(rows: dict) -> tuple:
    o, c, k = (rows.get(x) for x in CONDITIONS)
    if not (o and c and k and o['n'] and c['n'] and k['n']):
        return False, 'не все условия прогнаны'
    checks = [
        (o['tribune'] / o['n'] >= REALIZED_MIN,
         f"open: трибун {rate(o['tribune'], o['n'])} ≥ {REALIZED_MIN}"),
        (c['adopted'] / c['n'] <= CONTROL_ADOPT_MAX,
         f"closed: усыновление {rate(c['adopted'], c['n'])} ≤ {CONTROL_ADOPT_MAX}"),
        (k['adopted'] / k['n'] <= CONTROL_ADOPT_MAX,
         f"career: усыновление {rate(k['adopted'], k['n'])} ≤ {CONTROL_ADOPT_MAX}"),
    ]
    detail = '; '.join(('да' if ok else 'нет') + ' — ' + text for ok, text in checks)
    return all(ok for ok, _ in checks), detail


def report(results: list, calls: list, title: str) -> str:
    table = tally(results)
    cost, cached, prompt, reasoning, retries = {}, {}, {}, {}, {}
    made, mismatch = {}, {}
    for c in calls:
        u = c.get('usage') or {}
        p = c['participant']
        move, why = c.get('move'), c.get('reason') or ''
        if move:
            made[p] = made.get(p, 0) + 1
            # Lower bound: only a reason that names another move id by its id is caught.
            if move not in why and any(m != move and m in why for m in c.get('menu') or []):
                mismatch[p] = mismatch.get(p, 0) + 1
        retries[p] = retries.get(p, 0) + len(c.get('rejected') or [])
        cost[p] = cost.get(p, 0.0) + float(u.get('cost') or 0.0)
        prompt[p] = prompt.get(p, 0) + int(u.get('prompt_tokens') or 0)
        cached[p] = cached.get(p, 0) + int((u.get('prompt_tokens_details') or {}).get('cached_tokens') or 0)
        reasoning[p] = reasoning.get(p, 0) + int((u.get('completion_tokens_details') or {}).get('reasoning_tokens') or 0)

    out = [f'# {title}', '', 'Сгенерировано `spike.py`. Критерий — в README.md.', '',
           '| Участник | Условие | Партий | Трибун | Консул | С усыновлением | Незаконных ходов | Без хода |',
           '| --- | --- | --- | --- | --- | --- | --- | --- |']
    for name in sorted(table):
        for cond in CONDITIONS:
            row = table[name].get(cond)
            if row:
                out.append(f"| {name} | {cond} | {row['n']} | {rate(row['tribune'], row['n'])} | "
                           f"{rate(row['consul'], row['n'])} | {rate(row['adopted'], row['n'])} | "
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
                '| Участник | ₽ | Входных токенов | Из кэша | Токенов рассуждения | Перезапросов формата '
                '| Причина называет другой ход, не меньше |',
                '| --- | --- | --- | --- | --- | --- | --- |']
        for name in sorted(live):
            out.append(f'| {name} | {cost.get(name, 0.0):.2f} | {prompt.get(name, 0)} | '
                       f'{cached.get(name, 0)} | {reasoning.get(name, 0)} | {retries.get(name, 0)} | '
                       f'{mismatch.get(name, 0)} из {made.get(name, 0)} |')

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
    jobs = [(f'{name}@{world}', fn, world, cond, t)
            for world in WORLDS for name, fn in STUBS.items()
            for cond in CONDITIONS for t in range(args.trials)]
    results = run_jobs(jobs, 1, calls.append)
    text = report(results, calls, 'Спайк crack-finding: заглушки')
    (HERE / 'report-stub.md').write_text(text)
    print(text)
    table = tally(results)
    expected = {'stub:reader': True, 'stub:naive': False, 'stub:salient': False}
    failed = [f'{name}@{world}' for world in WORLDS for name, want in expected.items()
              if verdict(table[f'{name}@{world}'])[0] != want]
    for name in failed:
        print(f'criterion check failed for {name}', file=sys.stderr)
    return 1 if failed else 0


def mode_live(args) -> int:
    env = load_env()
    if not env.get('LLM_URL') or not env.get('LLM_KEY'):
        print('LLM_URL and LLM_KEY are required, in the environment or in .env', file=sys.stderr)
        return 2
    configs = [c.strip() for c in args.configs.split(',') if c.strip()]
    unknown = [c for c in configs if c not in CONFIGS]
    if unknown or args.world not in WORLDS:
        print(f'unknown configs {unknown} or world {args.world!r}; '
              f'known configs: {list(CONFIGS)}, worlds: {list(WORLDS)}', file=sys.stderr)
        return 2

    (HERE / 'records').mkdir(exist_ok=True)
    path = HERE / 'records' / time.strftime(f'run-{args.world}-%Y%m%d-%H%M%S.jsonl')
    lock = threading.Lock()
    calls = []

    def record(entry):
        with lock:
            calls.append(entry)
            with path.open('a') as f:
                f.write(json.dumps(entry, ensure_ascii=False) + '\n')

    spend = Spend(args.max_rub)
    jobs = [(f'v4.1-flash:{c}@{args.world}', live_participant(env, c, spend), args.world, cond, t)
            for c in configs for cond in CONDITIONS for t in range(args.trials)]
    results = run_jobs(jobs, args.workers, record)
    text = report(results, calls, f'Спайк crack-finding: {path.name}')
    report_path = HERE / args.report
    report_path.write_text(text)
    print(text)
    print(f'records: {path}\nreport: {report_path}\nspent: {spend.rub:.2f} RUB')
    return 0


def mode_replay(args) -> int:
    records, calls = {}, []
    for line in Path(args.records).read_text().splitlines():
        rec = json.loads(line)
        rec.setdefault('world', 'roman')  # records before worlds existed are roman
        calls.append(rec)
        records[(rec['participant'], rec['condition'], rec['trial'], rec['period'])] = rec
    trials = sorted({(p, records[(p, c, t, n)]['world'], c, t) for p, c, t, n in records})
    participant = replay_participant(records)
    try:
        results = [run_trial(name, participant, world, cond, t, lambda _: None)
                   for name, world, cond, t in trials]
    except ReplayDivergence as e:
        print(f'replay-divergence: {e}', file=sys.stderr)
        return 1
    text = report(results, calls, f'Спайк crack-finding: {Path(args.records).name}')
    report_path = HERE / args.report
    report_path.write_text(text)
    print(text)
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    sub = parser.add_subparsers(dest='mode', required=True)
    stub = sub.add_parser('stub')
    stub.add_argument('--trials', type=int, default=5)
    live = sub.add_parser('live')
    live.add_argument('--world', default='neutral')
    live.add_argument('--configs', default=','.join(CONFIGS))
    live.add_argument('--trials', type=int, default=2)
    live.add_argument('--workers', type=int, default=6)
    live.add_argument('--max-rub', type=float, default=10.0)
    live.add_argument('--report', default='report.md')
    replay = sub.add_parser('replay')
    replay.add_argument('records')
    replay.add_argument('--report', default='report.md')
    args = parser.parse_args()
    return {'stub': mode_stub, 'live': mode_live, 'replay': mode_replay}[args.mode](args)


if __name__ == '__main__':
    sys.exit(main())
