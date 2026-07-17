#!/usr/bin/env python3
"""Second-model check of recorded reasons.

For every recorded move with a reason, a judge answers two questions about the
reason text alone:

  verdict           does the reason argue for the chosen move, against it or
                    for another move, or neither
  self_undermining  does the reason give a purpose for the chosen move and also
                    say the move will not serve that purpose or will harm it

The judge sees only the menu, the chosen move and the reason - never the law, the
condition or who the participant was - so it judges consistency, not legality.

Calibration decides which answers are trusted. The verdict passed calibration;
self_undermining did not, so it is recorded but kept out of the report.

Modes:
  calibrate --config C         runs the judge on hand-labelled calls; every miss is printed
  run --config C RECORDS...    judges every recorded move that has a reason

Standard library only, Python 3.9+. Reuses the network code of spike.py.
"""

from __future__ import annotations

import argparse
import json
import sys
import threading
import time
import urllib.error
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

import spike

HERE = spike.HERE
RECORDS = HERE / 'records'
VERDICTS = ('supports', 'contradicts', 'unclear')

JUDGE_SYSTEM = 'You check whether the reason a person gave supports the action they chose.'

# Hand-labelled calls: (record file, participant, condition, trial, period,
# allowed verdicts, self_undermining or None when either answer is acceptable).
# Labelled before the judge first ran; do not relabel to fit its answers.
CALIBRATION = [
    ('run-20260913-220148.jsonl', 'v4.1-flash:no-thinking', 'closed', 0, 1, {'supports'}, False),
    ('run-20260913-220148.jsonl', 'v4.1-flash:no-thinking', 'closed', 4, 1, {'supports'}, False),
    ('run-20260913-220148.jsonl', 'v4.1-flash:no-thinking', 'career', 2, 1, {'supports'}, False),
    ('run-20260913-220148.jsonl', 'v4.1-flash:no-thinking', 'closed', 1, 1, {'supports', 'unclear'}, True),
    ('run-20260913-220148.jsonl', 'v4.1-flash:no-thinking', 'closed', 6, 1, {'supports', 'unclear'}, True),
    ('run-20260913-220148.jsonl', 'v4.1-flash:no-thinking', 'closed', 9, 1, {'supports', 'unclear'}, True),
    ('run-neutral-20260913-222134.jsonl', 'v4.1-flash:no-thinking@neutral', 'closed', 1, 1, {'contradicts'}, None),
    ('run-neutral-20260913-222651.jsonl', 'v4.1-flash:no-thinking@neutral', 'career', 2, 2, {'contradicts'}, True),
    ('run-neutral-20260913-222651.jsonl', 'v4.1-flash:no-thinking@neutral', 'closed', 1, 1, {'supports'}, False),
    ('run-neutral-20260913-222651.jsonl', 'v4.1-flash:no-thinking@neutral', 'career', 0, 1, {'unclear', 'supports'}, None),
    ('run-neutral-20260913-222651.jsonl', 'v4.1-flash:no-thinking@neutral', 'closed', 4, 2, {'unclear', 'supports'}, None),
    ('run-neutral-20260913-222651.jsonl', 'v4.1-flash:effort-low@neutral', 'career', 0, 2, {'supports'}, False),
    ('run-20260913-220148.jsonl', 'v4.1-flash:effort-low', 'closed', 8, 2, {'supports'}, False),
    ('run-20260913-220148.jsonl', 'v4.1-flash:effort-low', 'closed', 2, 3, {'supports'}, False),
    ('run-20260913-220148.jsonl', 'v4.1-flash:effort-low', 'open', 1, 1, {'supports'}, False),
    ('run-neutral-20260913-222134.jsonl', 'v4.1-flash:effort-low@neutral', 'closed', 1, 1, {'supports'}, False),
]


def menu_lines(rec: dict) -> list:
    world = rec.get('world', 'roman')
    descriptions = rec.get('menu_descriptions')
    if descriptions:
        return [f'- {sid}: {descriptions[sid]}' for sid in rec['menu']]
    lines = []
    for sid in rec['menu']:
        move = spike.canonical_move(world, sid)
        lines.append(f"- {sid}: {spike.WORLDS[world]['moves'][move][1]}")
    return lines


def judge_prompt(rec: dict) -> str:
    return '\n'.join(
        ['A person chose one move from this list:'] + menu_lines(rec) + [
            '',
            f"Chosen move: {rec['move']}",
            f"Stated reason: \"{rec['reason']}\"",
            '',
            'Judge only what the reason says, not whether it is correct.',
            '1. verdict: "supports" if the reason argues for the chosen move; "contradicts" '
            'if it argues for a different move or against the chosen move; "unclear" if it '
            'does not commit either way.',
            '2. self_undermining: true if the reason gives a purpose for the chosen move and '
            'also says that the move will not serve that purpose or will harm it; otherwise false.',
            'Reply with a JSON object only, of the form {"verdict": "supports" | "contradicts" '
            '| "unclear", "argued_move": "<the move id the reason argues for, or null>", '
            '"self_undermining": true | false}.',
        ])


def parse_verdict(text: str, menu: list) -> dict:
    start, end = text.find('{'), text.rfind('}')
    if start < 0 or end < start:
        raise ValueError('the reply contains no JSON object')
    data = json.loads(text[start:end + 1])
    if not isinstance(data, dict) or data.get('verdict') not in VERDICTS:
        raise ValueError('"verdict" must be "supports", "contradicts" or "unclear"')
    if not isinstance(data.get('self_undermining'), bool):
        raise ValueError('"self_undermining" must be true or false')
    argued = data.get('argued_move')
    return {'verdict': data['verdict'], 'self_undermining': data['self_undermining'],
            'argued_move': argued if argued in menu else None}


def ask(env: dict, config: str, spend: spike.Spend, rec: dict) -> dict:
    url = env['LLM_URL'].rstrip('/') + '/chat/completions'
    messages = [{'role': 'system', 'content': JUDGE_SYSTEM},
                {'role': 'user', 'content': judge_prompt(rec)}]
    usage, rejected = {}, []
    for _ in range(spike.FORMAT_RETRIES + 1):
        if not spend.start():
            return {'usage': usage, 'rejected': rejected, 'error': 'budget exhausted'}
        body = {'model': spike.MODEL, 'max_tokens': 8000, 'usage': {'include': True},
                'provider': spike.PROVIDER, 'messages': messages, **spike.CONFIGS[config]}
        try:
            r = spike.post(url, env['LLM_KEY'], body)
        except urllib.error.HTTPError as e:
            spend.finish(0.0)
            return {'usage': usage, 'rejected': rejected,
                    'error': f'http {e.code}: {e.read()[:300].decode(errors="replace")}'}
        except (urllib.error.URLError, TimeoutError) as e:
            spend.finish(0.0)
            return {'usage': usage, 'rejected': rejected, 'error': f'connection: {e}'}
        u = r.get('usage') or {}
        spend.finish(float(u.get('cost') or 0.0))
        spike.merge_usage(usage, u)
        text = r['choices'][0]['message'].get('content') or ''
        try:
            return {**parse_verdict(text, rec['menu']), 'usage': usage, 'rejected': rejected}
        except ValueError as e:
            rejected.append({'raw': text, 'problem': str(e)})
            messages = messages + [
                {'role': 'assistant', 'content': text},
                {'role': 'user', 'content': f'Your reply could not be used: {e}. '
                                            'Reply again with the JSON object only.'},
            ]
    return {'usage': usage, 'rejected': rejected, 'error': 'no usable reply after re-asking'}


def load_records(path: Path) -> dict:
    out = {}
    for line in path.read_text().splitlines():
        rec = json.loads(line)
        rec.setdefault('world', 'roman')
        out[(rec['participant'], rec['condition'], rec['trial'], rec['period'])] = rec
    return out


def judge_all(env, config, spend, recs, workers, record):
    def one(rec):
        res = ask(env, config, spend, rec)
        entry = {'judge': config, 'participant': rec['participant'], 'world': rec['world'],
                 'condition': rec['condition'], 'trial': rec['trial'], 'period': rec['period'],
                 'move': rec['move'], 'reason': rec['reason'], **res}
        record(entry)
        return entry
    with ThreadPoolExecutor(max_workers=workers) as pool:
        return list(pool.map(one, recs))


def writer(path: Path):
    lock = threading.Lock()

    def record(entry):
        with lock, path.open('a') as f:
            f.write(json.dumps(entry, ensure_ascii=False) + '\n')
    return record


def mode_calibrate(args, env) -> int:
    cache, recs, labels = {}, [], []
    for file, participant, cond, trial, period, allowed, undermining in CALIBRATION:
        if file not in cache:
            cache[file] = load_records(RECORDS / file)
        recs.append(cache[file][(participant, cond, trial, period)])
        labels.append((allowed, undermining))
    spend = spike.Spend(args.max_rub)
    path = RECORDS / time.strftime(f'judge-calibration-{args.config}-%Y%m%d-%H%M%S.jsonl')
    results = judge_all(env, args.config, spend, recs, args.workers, writer(path))
    verdict_misses, undermining_misses, undermining_labelled = 0, 0, 0
    for rec, res, (allowed, undermining) in zip(recs, results, labels):
        verdict_ok = res.get('verdict') in allowed
        undermining_ok = undermining is None or res.get('self_undermining') == undermining
        verdict_misses += not verdict_ok
        undermining_labelled += undermining is not None
        undermining_misses += not undermining_ok
        print(f"verdict {'ok  ' if verdict_ok else 'MISS'} undermining {'ok  ' if undermining_ok else 'MISS'} "
              f"{rec['participant']} {rec['condition']} t{rec['trial']} p{rec['period']}: "
              f"got {res.get('verdict')}/{res.get('self_undermining')} want {sorted(allowed)}/{undermining}"
              f"{' error: ' + res['error'] if 'error' in res else ''}")
    print(f'\njudge {args.config}: verdict {len(recs) - verdict_misses} of {len(recs)}; '
          f'self_undermining {undermining_labelled - undermining_misses} of {undermining_labelled} labelled; '
          f'spent {spend.rub:.2f} RUB; records {path.name}')
    return 0 if verdict_misses == 0 else 1


def mode_run(args, env) -> int:
    spend = spike.Spend(args.max_rub)
    sections = [f'# Разбор причин судьёй `{args.config}`', '',
                'Сгенерировано `judge.py`. Судья видит только меню, выбранный ход и причину, '
                'но не закон, условие и участника.', '',
                'В отчёте только ось согласованности причины с ходом: она прошла калибровку на '
                'размеченных случаях. Ось самоопровержения записывается в `records/judge-*`, но '
                'калибровку не прошла и сюда не выводится.', '']
    for source in args.records:
        source = Path(source)
        recs = [r for r in load_records(source).values() if r.get('move') and r.get('reason')]
        recs.sort(key=lambda r: (r['participant'], r['condition'], r['trial'], r['period']))
        path = RECORDS / f'judge-{args.config}-{source.name}'
        path.unlink(missing_ok=True)
        results = judge_all(env, args.config, spend, recs, args.workers, writer(path))

        table = {}
        for res in results:
            row = table.setdefault((res['participant'], res['condition']),
                                   {'n': 0, 'contradicts': 0, 'unclear': 0, 'errors': 0})
            row['n'] += 1
            if 'error' in res:
                row['errors'] += 1
                continue
            row['contradicts'] += res['verdict'] == 'contradicts'
            row['unclear'] += res['verdict'] == 'unclear'
        sections += [f'## {source.name}', '',
                     '| Участник | Условие | Ходов с причиной | Противоречит ходу | Неясно | Без ответа судьи |',
                     '| --- | --- | --- | --- | --- | --- |']
        for (participant, cond), row in sorted(table.items()):
            sections.append(f"| {participant} | {cond} | {row['n']} | {row['contradicts']} | "
                            f"{row['unclear']} | {row['errors']} |")
        flagged = [r for r in results if r.get('verdict') == 'contradicts']
        if flagged:
            sections += ['', 'Причины, противоречащие ходу:', '']
            for r in flagged:
                argued = f", причина за `{r['argued_move']}`" if r.get('argued_move') else ''
                sections.append(f"- {r['participant']}, {r['condition']}, партия {r['trial']}, "
                                f"период {r['period']}, ход `{r['move']}`{argued}: {r['reason']}")
        sections.append('')
    sections.append(f'Расход судьи: {spend.rub:.2f} ₽.')
    report = HERE / args.report
    report.write_text('\n'.join(sections) + '\n')
    print('\n'.join(sections))
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    sub = parser.add_subparsers(dest='mode', required=True)
    cal = sub.add_parser('calibrate')
    cal.add_argument('--config', default='no-thinking', choices=list(spike.CONFIGS))
    cal.add_argument('--workers', type=int, default=6)
    cal.add_argument('--max-rub', type=float, default=3.0)
    run = sub.add_parser('run')
    run.add_argument('records', nargs='+')
    run.add_argument('--config', default='no-thinking', choices=list(spike.CONFIGS))
    run.add_argument('--workers', type=int, default=8)
    run.add_argument('--max-rub', type=float, default=15.0)
    run.add_argument('--report', default='report-judge.md')
    args = parser.parse_args()
    env = spike.load_env()
    if not env.get('LLM_URL') or not env.get('LLM_KEY'):
        print('LLM_URL and LLM_KEY are required, in the environment or in .env', file=sys.stderr)
        return 2
    return {'calibrate': mode_calibrate, 'run': mode_run}[args.mode](args, env)


if __name__ == '__main__':
    sys.exit(main())
