#!/usr/bin/env python3
"""Toy simulation: do the stand axes behave like meters worth showing a player?

A disposable spike, not part of the design. It builds a tiny polity, lets a
scripted player amend it period after period, writes a journal of what cases
did, computes every candidate metric from the corpus and the journal only, and
then tests each metric for the properties that make a meter useless or unfair:

  - inert:        no player policy moves it beyond seed noise
  - noisy:        seed noise is comparable to the spread between policies
  - scale-bound:  the idle polity lands elsewhere just because the world is larger
  - drifting:     it walks towards an end with no player action at all
  - redundant:    it is the same signal as another metric
  - end reached without the player: the "illogical loss"

Every number in preset() and in the policies is an assumption of the toy
model. Structural findings (a metric is a function of another, a quantity is not
observable from the journal, a count grows with population) survive a change
of numbers; magnitudes do not.

Standard library only. Run: python3 sim.py > report.md
"""

import math
import random
import statistics as st

PERIODS = 40
START = 10            # policies act from this period on; before it every run is idle
WINDOW = 5            # journal metrics look at the last WINDOW periods
TAIL = 10             # "final" value is the mean of the last TAIL periods
SEEDS = 10
POPULATIONS = (50, 200, 1000)
BASE_POP = 200
KINDS = ('grant_status', 'adjudicate', 'establish_fact', 'sanction', 'appoint')
STATUSES = ('determinate', 'discretion', 'conflict', 'gap')


# --- the toy polity ---------------------------------------------------------

def preset():
    return {
        'offices': {
            'consul':   {'holders': 2, 'capacity': 8,  'ambition': 0.04, 'cooptation': False},
            'praetor':  {'holders': 1, 'capacity': 20, 'ambition': 0.03, 'cooptation': False},
            'tribunus': {'holders': 2, 'capacity': 4,  'ambition': 0.02, 'cooptation': False},
            'censor':   {'holders': 2, 'capacity': 6,  'ambition': 0.05, 'cooptation': False},
            'senate':   {'holders': 1, 'capacity': 6,  'ambition': 0.02, 'cooptation': True},
        },
        'competence': {
            'grant_status': ['praetor'],
            'adjudicate': ['praetor'],
            'establish_fact': ['censor'],
            'sanction': ['consul'],
            'appoint': ['senate'],
        },
        'enforcers': ['censor'],
        # (checker, target): checker can block target's decisions before they take effect
        'vetoes': [('tribunus', 'praetor'), ('tribunus', 'consul'), ('consul', 'consul')],
        # (reviewer, target): reviewer re-examines target's blocked decisions
        'reviews': [('senate', 'praetor')],
        # (office, other): eligibility for office changes through acts of other
        'eligibility': [('consul', 'censor'), ('senate', 'censor')],
        'norms': {
            'grant_status': 'determinate', 'adjudicate': 'discretion',
            'establish_fact': 'determinate', 'sanction': 'conflict', 'appoint': 'discretion',
        },
        'demand': {'grant_status': 0.03, 'adjudicate': 0.05, 'establish_fact': 0.02, 'appoint': 0.004},
        'duties': [
            {'name': 'munus', 'rate': 0.06, 'consequence': True, 'private': 0.10},
            {'name': 'census', 'rate': 0.04, 'consequence': False, 'private': 0.0},
        ],
        'case_limit': 3,
    }


def binom(rng, n, p):
    if n <= 0 or p <= 0:
        return 0
    if p >= 1:
        return n
    if n <= 256:
        return sum(rng.random() < p for _ in range(n))
    lam = n * p
    if lam < 30:
        limit, k, prod = math.exp(-lam), 0, rng.random()
        while prod > limit:
            k += 1
            prod *= rng.random()
        return min(k, n)
    return max(0, min(n, round(rng.gauss(lam, math.sqrt(lam * (1 - p))))))


# --- structure helpers ------------------------------------------------------

def vetoers(s, office):
    """Number of people able to block office's decisions."""
    n = 0
    for checker, target in s['vetoes']:
        if target != office:
            continue
        if checker == office:
            n += max(0, s['offices'][office]['holders'] - 1)   # a colleague
        else:
            n += s['offices'][checker]['holders']
    return n


def reachable(edges, start):
    seen, todo = set(), [start]
    while todo:
        node = todo.pop()
        for a, b in edges:
            if a == node and b not in seen:
                seen.add(b)
                todo.append(b)
    return seen


def review_path(s, office):
    """(has reviewer, review is cyclic) for decisions of office."""
    reviewers = [r for r, t in s['reviews'] if t == office and r != office]
    if not reviewers:
        return False, False
    # edge target -> reviewer: a blocked decision of target goes up to reviewer
    up = [(t, r) for r, t in s['reviews'] if r != t]
    return True, office in reachable(up, office)


# --- one period -------------------------------------------------------------

def run_period(s, period, pop, rng, backlog, journal, truth, amended_recently,
               cap_factor=1.0, rate_factor=1.0, demand_factor=1.0):
    budget = {o: max(1, round(v['capacity'] * v['holders'] * cap_factor)) for o, v in s['offices'].items()}

    def close(case, outcome, **extra):
        rec = {'period': period, 'kind': case['kind'], 'origin': case['origin'],
               'outcome': outcome, 'office': case.get('office')}
        rec.update(extra)
        journal.append(rec)

    # petitions; a kind touched by a recent amendment draws more of them
    for kind, rate in s['demand'].items():
        boost = 2.0 if kind in amended_recently else 1.0
        for _ in range(binom(rng, pop, rate * boost * demand_factor)):
            backlog.append({'kind': kind, 'origin': 'petition', 'born': period, 'state': 'new'})

    # duties: the world breaks them; the polity sees only what it samples or is told
    capacity = cap_factor * sum(s['offices'][o]['capacity'] * s['offices'][o]['holders'] for o in s['enforcers'])
    fraction = min(1.0, capacity / pop)
    for duty in s['duties']:
        actual = binom(rng, pop, duty['rate'] * rate_factor)
        found = binom(rng, actual, fraction)
        told = binom(rng, actual - found, duty['private'])
        truth.append({'period': period, 'actual': actual, 'found': found + told})
        journal.append({'period': period, 'kind': 'duty', 'origin': 'duty', 'outcome': 'sampled',
                        'office': None, 'found': found, 'fraction': fraction, 'told': told})
        for _ in range(found + told):
            if duty['consequence']:
                backlog.append({'kind': 'sanction', 'origin': 'violation', 'born': period, 'state': 'new'})
            else:
                journal.append({'period': period, 'kind': 'duty', 'origin': 'violation',
                                'outcome': 'no_consequence', 'office': None})

    journal.append({'period': period, 'kind': 'load', 'origin': 'world', 'outcome': 'load', 'office': None,
                    'arrived': sum(1 for c in backlog if c['born'] == period),
                    'capacity': sum(budget.values())})

    # cases
    still = []
    for case in backlog:
        if case['state'] == 'blocked':
            has, cyclic = review_path(s, case['office'])
            reviewers = [r for r, t in s['reviews'] if t == case['office'] and r != case['office']]
            free = [r for r in reviewers if budget[r] > 0]
            if has and not cyclic and free:
                budget[free[0]] -= 1
                close(case, 'decided_' + case['status'], reviewed=True)
            else:
                still.append(case)
            continue

        offices = s['competence'].get(case['kind'], [])
        if not offices:
            close(case, 'no_competent')
            continue
        free = [o for o in offices if budget[o] > 0]
        if not free:
            still.append(case)
            continue
        office = free[0]
        budget[office] -= 1
        case['office'] = office
        status = s['norms'][case['kind']]
        if status in ('conflict', 'gap'):
            close(case, 'non_liquet_' + status)
            continue
        p = 0.05 if status == 'determinate' else 0.20
        if rng.random() < 1 - (1 - p) ** vetoers(s, office):
            case['state'], case['status'] = 'blocked', status
            journal.append({'period': period, 'kind': case['kind'], 'origin': case['origin'],
                            'outcome': 'blocked', 'office': office})
            still.append(case)
        else:
            close(case, 'decided_' + status)

    backlog[:] = []
    for case in still:
        if period - case['born'] >= s['case_limit']:
            close(case, 'expired')
        else:
            backlog.append(case)

    # acts outside competence
    for office, v in s['offices'].items():
        ambition = v['ambition'] * (1 + 0.05 * period if v['cooptation'] else 1)
        outside = [k for k in KINDS if office not in s['competence'].get(k, [])]
        for _ in range(binom(rng, v['holders'], ambition)):
            if not outside:
                break
            n = vetoers(s, office)
            blocked = rng.random() < 1 - 0.5 ** n
            journal.append({'period': period, 'kind': rng.choice(outside), 'origin': 'act',
                            'outcome': 'ultra_vires_blocked' if blocked else 'ultra_vires_passed',
                            'office': office})


# --- player policies --------------------------------------------------------

def amend(s, journal, period, what, kinds=()):
    journal.append({'period': period, 'kind': 'amendment', 'origin': 'auctor',
                    'outcome': what, 'office': None, 'touches': list(kinds)})


def idle(s, period, rng, journal):
    pass


def reformer(s, period, rng, journal):
    for k in KINDS:
        if s['norms'][k] in ('conflict', 'gap'):
            s['norms'][k] = 'determinate'
            return amend(s, journal, period, 'resolve', [k])


def codifier(s, period, rng, journal):
    for k in KINDS:
        if s['norms'][k] == 'discretion':
            s['norms'][k] = 'determinate'
            return amend(s, journal, period, 'codify', [k])


def centralizer(s, period, rng, journal):
    if period % 2 == 0:
        for k in KINDS:
            if s['competence'][k] != ['consul']:
                s['competence'][k] = ['consul']
                return amend(s, journal, period, 'centralize', [k])
    if period % 4 == 1:
        s['vetoes'] = [v for v in s['vetoes'] if v[1] != 'consul']
        amend(s, journal, period, 'remove veto on consul')


def checker(s, period, rng, journal):
    if period % 2:
        return
    for office in s['offices']:
        if office != 'tribunus' and vetoers(s, office) == 0:
            s['vetoes'].append(('tribunus', office))
            return amend(s, journal, period, 'add veto')
        has, cyclic = review_path(s, office)
        if office not in ('senate', 'tribunus') and not has:
            s['reviews'].append(('senate', office))
            return amend(s, journal, period, 'add review')


def over_checker(s, period, rng, journal):
    s['offices']['tribunus']['holders'] = min(12, s['offices']['tribunus']['holders'] + 1)
    target = rng.choice(list(s['offices']))
    source = rng.choice([o for o in s['offices'] if o != target])
    if (source, target) not in s['vetoes']:
        s['vetoes'].append((source, target))
    amend(s, journal, period, 'more vetoes')


def churner(s, period, rng, journal):
    touched = rng.sample(KINDS, 2)
    for k in touched:
        s['norms'][k] = rng.choice(STATUSES)
    amend(s, journal, period, 'churn', touched)
    amend(s, journal, period, 'churn', touched)


def starve_enforcement(s, period, rng, journal):
    if period == START:
        s['offices']['censor']['capacity'] = 1
        amend(s, journal, period, 'starve enforcement')


def expand_capacity(s, period, rng, journal):
    if period == START:
        for v in s['offices'].values():
            v['capacity'] *= 4
        amend(s, journal, period, 'expand capacity')


def review_ring(s, period, rng, journal):
    if period == START:
        s['reviews'] = [('senate', 'praetor'), ('praetor', 'senate')]
        amend(s, journal, period, 'review ring')


def self_perpetuating(s, period, rng, journal):
    if period == START:
        s['eligibility'] += [('senate', 'senate'), ('censor', 'senate'), ('consul', 'consul')]
        s['offices']['censor']['cooptation'] = True
        s['competence']['establish_fact'] = ['senate']
        amend(s, journal, period, 'self-perpetuating senate', ['establish_fact'])


POLICIES = {f.__name__: f for f in (idle, reformer, codifier, centralizer, checker, over_checker,
                                    churner, starve_enforcement, expand_capacity, review_ring,
                                    self_perpetuating)}


# --- metrics: corpus and journal only --------------------------------------

def structural(s):
    kinds = [k for k in KINDS if s['competence'].get(k)]
    constrained = 0
    veto_people = []
    reviewable = 0
    for k in kinds:
        offices = s['competence'][k]
        if any(vetoers(s, o) > 0 or review_path(s, o)[0] for o in offices):
            constrained += 1
        veto_people.append(max(vetoers(s, o) for o in offices))
        if any(review_path(s, o)[0] and not review_path(s, o)[1] for o in offices):
            reviewable += 1
    count = {}
    for k in kinds:
        for o in s['competence'][k]:
            count[o] = count.get(o, 0) + 1
    total = sum(count.values()) or 1
    hhi = sum((c / total) ** 2 for c in count.values())
    offices = list(s['offices'])
    dependent = {a for a, b in s['eligibility']}
    cyclic = {o for o in offices if o in reachable(s['eligibility'], o)}
    return {
        'G1_executive_constraint': constrained / len(KINDS),
        'G2_veto_players': st.mean(veto_people) if veto_people else 0.0,
        'G3_concentration': hhi,
        'G4_review_reachability': reviewable / len(KINDS),
        'G5_eligibility_closure': (len(dependent) + len(cyclic)) / (2 * len(offices)),
    }


def from_journal(w, s):
    closed = [r for r in w if r['outcome'].startswith(('decided', 'non_liquet', 'expired', 'no_competent'))]
    n = len(closed) or 1
    det = sum(r['outcome'] == 'decided_determinate' for r in closed)
    nl = sum(r['outcome'].startswith('non_liquet') or r['outcome'] == 'no_competent' for r in closed)
    exp = sum(r['outcome'] == 'expired' for r in closed)
    pet = [r for r in closed if r['origin'] == 'petition']
    unmet = sum(not r['outcome'].startswith('decided') for r in pet)
    passed = sum(r['outcome'] == 'ultra_vires_passed' for r in w)
    decided = sum(r['outcome'].startswith('decided') for r in w) or 1
    sampled = [r for r in w if r['outcome'] == 'sampled']
    est_actual = sum((r['found'] / r['fraction']) if r['fraction'] > 0 else 0 for r in sampled)
    found = sum(r['found'] + r['told'] for r in sampled)
    applied = {r['kind'] for r in w if r['outcome'].startswith('decided')}
    no_conseq = sum(r['outcome'] == 'no_consequence' for r in w)
    violations = no_conseq + sum(r['origin'] == 'violation' for r in closed)
    amendments = sum(r['kind'] == 'amendment' for r in w)
    loads = [r for r in w if r['outcome'] == 'load']
    load_ratio = sum(r['arrived'] for r in loads) / (sum(r['capacity'] for r in loads) or 1)
    answered = sum(r['outcome'].startswith(('decided', 'non_liquet')) or r['outcome'] == 'no_competent'
                   for r in closed) or 1
    fraction = st.mean(r['fraction'] for r in sampled) if sampled else 0.0
    return {
        'J6_determinacy': det / n,
        'J6b_determinacy_of_answered': det / answered,
        'J7a_unmet_petitions': unmet / (len(pet) or 1),
        'J7b_enforcement_gap_est': 1 - found / est_actual if est_actual else 0.0,
        'J7c_enforcement_gap_structural': 1 - fraction,
        'J8_amendment_rate': amendments / WINDOW,
        'F1_usurpation_count': passed / WINDOW,
        'F1r_usurpation_per_decision': passed / decided,
        'F1b_usurpation_per_office': passed / (WINDOW * len(s['offices'])),
        'F2_non_liquet_share': nl / n,
        'F3_silent_norm_share': 1 - len(applied & set(KINDS)) / len(KINDS),
        'F4_expired_share': exp / n,
        'F5_no_consequence_share': no_conseq / (violations or 1),
        'L_load_ratio': load_ratio,
    }


# bad zones as data: (low, high); None means no bad end on that side
ZONES = {
    'G1_executive_constraint': (0.34, None),
    'G2_veto_players': (0.5, 6.0),
    'G3_concentration': (None, 0.5),
    'G4_review_reachability': (0.2, None),
    'G5_eligibility_closure': (None, 0.4),
    'J6_determinacy': (0.3, 0.95),
    'J6b_determinacy_of_answered': (0.3, 0.95),
    'J7a_unmet_petitions': (None, 0.4),
    'J7b_enforcement_gap_est': (None, 0.8),
    'J7c_enforcement_gap_structural': (None, 0.8),
    'J8_amendment_rate': (None, 1.5),
    'F1_usurpation_count': (None, 0.5),
    'F1r_usurpation_per_decision': (None, 0.05),
    'F1b_usurpation_per_office': (None, 0.1),
    'F2_non_liquet_share': (None, 0.3),
    'F3_silent_norm_share': (None, 0.5),
    'F4_expired_share': (None, 0.3),
    'F5_no_consequence_share': (None, 0.5),
    'L_load_ratio': (None, 1.0),
}


# --- runs -------------------------------------------------------------------

WORLDS = ('static', 'growth', 'plague', 'famine')
CAPACITY_MODES = ('fixed', 'scaled')


def world_state(world, pop, period):
    """(population, duty-rate factor, demand factor) the world imposes this period."""
    if world == 'growth' and period >= START:
        return round(pop * (1 + 2 * (period - START) / (PERIODS - 1 - START))), 1.0, 1.0
    if world == 'plague' and period >= 20:
        return pop // 2, 1.0, 1.0
    if world == 'famine' and 20 <= period < 25:
        return pop, 2.0, 1.5
    return pop, 1.0, 1.0


def run(policy, pop, seed, world='static', capacity='fixed'):
    tag = '' if (world, capacity) == ('static', 'fixed') else f'-{world}-{capacity}'
    rng = random.Random(f'{policy}-{pop}-{seed}{tag}')
    s = preset()
    journal, truth, backlog, rows, starts = [], [], [], [], []
    for period in range(PERIODS):
        starts.append(len(journal))
        if period >= START:
            POLICIES[policy](s, period, rng, journal)
        recent_from = starts[max(0, period - 1)]
        recent = {k for r in journal[recent_from:] if r['kind'] == 'amendment' for k in r['touches']}
        pop_now, rate_f, demand_f = world_state(world, pop, period)
        cap_f = pop_now / pop if capacity == 'scaled' else 1.0
        run_period(s, period, pop_now, rng, backlog, journal, truth, recent, cap_f, rate_f, demand_f)
        row = structural(s)
        row.update(from_journal(journal[starts[max(0, period - WINDOW + 1)]:], s))
        t = [r for r in truth if period - WINDOW < r['period'] <= period]
        actual = sum(r['actual'] for r in t)
        row['_truth_enforcement_gap'] = 1 - sum(r['found'] for r in t) / actual if actual else 0.0
        rows.append(row)
    return rows


def main():
    results = {p: {n: [run(p, n, seed) for seed in range(SEEDS)] for n in POPULATIONS} for p in POLICIES}
    metrics = [m for m in results['idle'][BASE_POP][0][0] if not m.startswith('_')]
    report(results, metrics)
    world_report(metrics)


# --- analysis ---------------------------------------------------------------

def seed_finals(results, policy, pop, m):
    return [st.mean(r[m] for r in rows[-TAIL:]) for rows in results[policy][pop]]


def traj(results, policy, pop, m):
    return [st.mean(rows[t][m] for rows in results[policy][pop]) for t in range(PERIODS)]


def sd(xs):
    return st.pstdev(xs) if len(xs) > 1 else 0.0


def pearson(xs, ys):
    mx, my = st.mean(xs), st.mean(ys)
    sx = math.sqrt(sum((x - mx) ** 2 for x in xs))
    sy = math.sqrt(sum((y - my) ** 2 for y in ys))
    if sx == 0 or sy == 0:
        return None
    return sum((x - mx) * (y - my) for x, y in zip(xs, ys)) / (sx * sy)


def zone_of(m, value):
    low, high = ZONES[m]
    if low is not None and value < low:
        return 'low'
    if high is not None and value > high:
        return 'high'
    return None


def report(results, metrics):
    out = []
    p = out.append
    p('# Отчёт спайка метрик\n')
    p(f'Сгенерировано `spike/metrics/sim.py`. Периодов {PERIODS}, политика игрока действует с периода '
      f'{START}, окно журнала {WINDOW}, зёрен {SEEDS}, население {", ".join(map(str, POPULATIONS))}; '
      f'базовое население {BASE_POP}. Значения — средние последних {TAIL} периодов.\n')

    rows_out = []
    verdicts = {}
    for m in metrics:
        base = {pol: seed_finals(results, pol, BASE_POP, m) for pol in POLICIES}
        means = {pol: st.mean(v) for pol, v in base.items()}
        spread = max(means.values()) - min(means.values())
        idle_sd = sd(base['idle'])
        effects = {}
        for pol in POLICIES:
            if pol == 'idle':
                continue
            pooled = math.sqrt((sd(base[pol]) ** 2 + idle_sd ** 2) / 2)
            gap = abs(means[pol] - means['idle'])
            effects[pol] = gap / pooled if pooled else (float('inf') if gap > 1e-12 else 0.0)
        best = max(effects, key=effects.get)
        lo_pol = min(means, key=means.get)
        hi_pol = max(means, key=means.get)
        noise = st.mean(sd([rows[t][m] for rows in results['idle'][BASE_POP]]) for t in range(START, PERIODS))
        snr = spread / noise if noise else float('inf')
        idle_small = st.mean(seed_finals(results, 'idle', POPULATIONS[0], m))
        idle_large = st.mean(seed_finals(results, 'idle', POPULATIONS[-1], m))
        scale = abs(idle_large - idle_small) / spread if spread else 0.0
        it = traj(results, 'idle', BASE_POP, m)
        drift = abs(st.mean(it[-5:]) - st.mean(it[START:START + 5])) / spread if spread else 0.0
        pt = traj(results, best, BASE_POP, m)
        diff = [a - b for a, b in zip(pt, it)]
        target = st.mean(diff[-TAIL:])
        lag = next((t - START for t in range(START, PERIODS) if target and abs(diff[t]) >= 0.9 * abs(target)), None)

        low_by, high_by, idle_hits = [], [], []
        for pol in POLICIES:
            for pop in POPULATIONS:
                finals = seed_finals(results, pol, pop, m)
                zones = [zone_of(m, v) for v in finals]
                for z, bucket in (('low', low_by), ('high', high_by)):
                    if zones.count(z) > SEEDS / 2:
                        label = pol if pop == BASE_POP else f'{pol}@{pop}'
                        if pol == 'idle':
                            idle_hits.append(f'{z}@{pop}')
                        elif pop == BASE_POP:
                            bucket.append(label)

        flags = []
        if effects[best] < 1.0:
            flags.append('инертна')
        if snr < 2.0:
            flags.append('шумна')
        if scale > 0.5:
            flags.append('зависит от размера мира')
        if drift > 0.5:
            flags.append('дрейфует без игрока')
        if idle_hits:
            flags.append('край без игрока: ' + ', '.join(idle_hits))
        low, high = ZONES[m]
        if low is not None and not low_by and not any(h.startswith('low') for h in idle_hits):
            flags.append('нижний край недостижим')
        if high is not None and not high_by and not any(h.startswith('high') for h in idle_hits):
            flags.append('верхний край недостижим')
        verdicts[m] = flags
        rows_out.append((m, means['idle'], f'{means[lo_pol]:.2f} {lo_pol}', f'{means[hi_pol]:.2f} {hi_pol}',
                         best, effects[best], snr, scale, drift, lag,
                         ', '.join(low_by) or '—', ', '.join(high_by) or '—', flags))

    p('## Метрики\n')
    p('| Метрика | idle | мин | макс | сильнее всего двигает | эффект | сигнал/шум | размер мира | дрейф | лаг | в нижней зоне | в верхней зоне | Флаги |')
    p('| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |')
    for m, idle_v, mn, mx, best, eff, snr, scale, drift, lag, lo, hi, flags in rows_out:
        eff_s = 'детерм.' if eff == float('inf') else f'{eff:.1f}'
        snr_s = '∞' if snr == float('inf') else f'{snr:.1f}'
        p(f'| `{m}` | {idle_v:.2f} | {mn} | {mx} | {best} | {eff_s} | {snr_s} | {scale:.2f} | '
          f'{drift:.2f} | {"—" if lag is None else lag} | {lo} | {hi} | {"; ".join(flags) or "—"} |')

    p('\nЭффект — разница с `idle` в стандартных отклонениях зёрен. Сигнал/шум — разброс средних по политикам, '
      'делённый на шум зёрен у `idle`. Размер мира и дрейф — доля разброса политик. Лаг — периодов от начала '
      'политики до 90% итогового сдвига. «детерм.» — метрика не зависит от зерна.\n')

    p('## Пары, которые почти один сигнал\n')
    samples = {m: [] for m in metrics}
    for pol in POLICIES:
        for pop in POPULATIONS:
            for rows in results[pol][pop]:
                for r in rows[START:]:
                    for m in metrics:
                        samples[m].append(r[m])
    pairs = []
    for i, a in enumerate(metrics):
        for b in metrics[i + 1:]:
            r = pearson(samples[a], samples[b])
            if r is not None and abs(r) >= 0.8:
                pairs.append((abs(r), a, b, r))
    p('| Метрика | Метрика | r |')
    p('| --- | --- | --- |')
    for _, a, b, r in sorted(pairs, reverse=True):
        p(f'| `{a}` | `{b}` | {r:.2f} |')
    if not pairs:
        p('| — | — | — |')

    p('\n## Оценка разрыва принуждения против истины\n')
    p('Журнал не знает, сколько обязанностей нарушено на самом деле. `J7b` оценивает это по доле выборки; '
      'симуляция знает истину.\n')
    p('| Население | Политика | оценка по журналу `J7b` | структурная `J7c` | истина |')
    p('| --- | --- | --- | --- | --- |')
    for pop in POPULATIONS:
        for pol in ('idle', 'starve_enforcement', 'expand_capacity'):
            est = st.mean(seed_finals(results, pol, pop, 'J7b_enforcement_gap_est'))
            tru = st.mean(seed_finals(results, pol, pop, '_truth_enforcement_gap'))
            stru = st.mean(seed_finals(results, pol, pop, 'J7c_enforcement_gap_structural'))
            p(f'| {pop} | {pol} | {est:.2f} | {stru:.2f} | {tru:.2f} |')

    p('\n## Средние итоговые значения по политикам\n')
    p('| Политика | ' + ' | '.join(f'`{m}`' for m in metrics) + ' |')
    p('| --- |' + ' --- |' * len(metrics))
    for pol in POLICIES:
        vals = [st.mean(seed_finals(results, pol, BASE_POP, m)) for m in metrics]
        p(f'| {pol} | ' + ' | '.join(f'{v:.2f}' for v in vals) + ' |')

    p('\n## Размер мира у idle\n')
    p('| Метрика | ' + ' | '.join(str(n) for n in POPULATIONS) + ' |')
    p('| --- |' + ' --- |' * len(POPULATIONS))
    for m in metrics:
        p(f'| `{m}` | ' + ' | '.join(f'{st.mean(seed_finals(results, "idle", n, m)):.2f}' for n in POPULATIONS) + ' |')

    print('\n'.join(out))


def world_report(metrics):
    """A world that changes by itself: does a meter reach an end without the player,
    and does the change drown the effect of the player's decisions?"""
    watched = [m for m in metrics if not m.startswith(('G', 'J8'))]
    combos = [(w, c) for w in WORLDS for c in CAPACITY_MODES]
    policies = ('idle', 'reformer', 'checker', 'starve_enforcement')
    data = {(pol, w, c): [run(pol, BASE_POP, seed, w, c) for seed in range(SEEDS)]
            for pol in policies for w, c in combos}

    def traj(pol, w, c, m):
        return [st.mean(rows[t][m] for rows in data[(pol, w, c)]) for t in range(PERIODS)]

    def first_zone(values, m):
        for t in range(START, PERIODS):
            z = zone_of(m, values[t])
            if z:
                return f'{z}@{t}'
        return '—'

    out = []
    p = out.append
    p('\n## Мир меняется без игрока\n')
    p('Бездействующий игрок, население 200. `growth` — население растёт до трёх раз с периода 10; '
      '`plague` — падает вдвое в период 20; `famine` — периоды 20–24 вдвое больше нарушений и в полтора раза '
      'больше прошений. `fixed` — пропускная способность должностей постоянна; `scaled` — пресет растит её '
      'вместе с населением. Ячейка — первый период, когда средняя траектория вошла в плохую зону.\n')
    p('| Метрика | ' + ' | '.join(f'{w}/{c}' for w, c in combos) + ' |')
    p('| --- |' + ' --- |' * len(combos))
    for m in watched:
        cells = [first_zone(traj('idle', w, c, m), m) for w, c in combos]
        p(f'| `{m}` | ' + ' | '.join(cells) + ' |')

    p('\n## Заглушает ли мир решения игрока\n')
    p('Эффект политики против бездействия в стандартных отклонениях зёрен, среднее последних '
      f'{TAIL} периодов. Если эффект падает в меняющемся мире, игрок хуже видит, что сделал.\n')
    shown = ['J6_determinacy', 'J6b_determinacy_of_answered', 'F2_non_liquet_share', 'F4_expired_share',
             'J7a_unmet_petitions', 'J7b_enforcement_gap_est', 'J7c_enforcement_gap_structural', 'L_load_ratio']
    for pol in policies[1:]:
        p(f'\n**{pol}**\n')
        p('| Метрика | ' + ' | '.join(f'{w}/{c}' for w, c in combos) + ' |')
        p('| --- |' + ' --- |' * len(combos))
        for m in shown:
            cells = []
            for w, c in combos:
                a = [st.mean(r[m] for r in rows[-TAIL:]) for rows in data[(pol, w, c)]]
                b = [st.mean(r[m] for r in rows[-TAIL:]) for rows in data[('idle', w, c)]]
                pooled = math.sqrt((sd(a) ** 2 + sd(b) ** 2) / 2)
                gap = st.mean(a) - st.mean(b)
                cells.append('детерм.' if not pooled and abs(gap) > 1e-12 else
                             ('0' if not pooled else f'{gap / pooled:+.1f}'))
            p(f'| `{m}` | ' + ' | '.join(cells) + ' |')
    print('\n'.join(out))


if __name__ == '__main__':
    main()
