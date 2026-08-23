// Package turn advances a period: finalization of earlier decisions, living,
// report. Prototype: the amendment phase is empty, the stub auctor changes
// nothing.
package turn

import (
	"fmt"
	"sort"

	"github.com/heaprip/intercessio/internal/actors"
	"github.com/heaprip/intercessio/internal/cases"
	"github.com/heaprip/intercessio/internal/competence"
	"github.com/heaprip/intercessio/internal/corpus"
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/entitlement"
	"github.com/heaprip/intercessio/internal/facts"
	"github.com/heaprip/intercessio/internal/journal"
	"github.com/heaprip/intercessio/internal/period"
)

// State is everything a period starts from.
type State struct {
	Period  period.Period
	Corpus  corpus.Version
	Facts   []facts.Fact
	Cases   []cases.Case
	Journal journal.Journal
	Seed    int64
}

// Config holds the explicit inputs that are not state.
type Config struct {
	Strategy entitlement.Strategy
	Actors   actors.Stub
	Term     int // periods a case may stay open
}

// Transition is the result of a period: the next state and what was added.
type Transition struct {
	Next    State
	Facts   []facts.Fact
	Entries []journal.Entry
}

type period_ struct {
	s       State
	cfg     Config
	now     period.Period
	facts   []facts.Fact
	cases   []cases.Case
	added   []facts.Fact
	entries []journal.Entry
	res     *entitlement.Result
}

// Advance lives through one period. It is a function of its inputs: the same
// state and config give the same transition.
func Advance(s State, cfg Config) (Transition, error) {
	if cfg.Term == 0 {
		cfg.Term = 3
	}
	p := &period_{s: s, cfg: cfg, now: s.Period, facts: append([]facts.Fact{}, s.Facts...), cases: append([]cases.Case{}, s.Cases...)}

	x := p.context()
	for i, c := range p.cases {
		c, fs, es := cases.Finalize(x, c)
		c, es2 := cases.Expire(x, c)
		p.cases[i] = c
		p.add(fs, append(es, es2...))
	}
	if err := p.resolve(); err != nil {
		return Transition{}, err
	}

	offices, people, statuses := p.domain("office"), p.domain("person"), p.domain("status_kind")

	// petitions
	exists := func(person, matter string) bool {
		for _, c := range p.cases {
			if c.Person == person && c.Matter == matter {
				return true
			}
		}
		return false
	}
	for i, req := range cfg.Actors.Petitions(p.res.Query, people, statuses, exists) {
		office := p.firstCompetent(offices, "grant_status")
		if office == "" {
			continue
		}
		c, f, e := cases.File(p.context(), cases.Petition, fmt.Sprintf("pt%d_%d", p.now, i+1), req.Person, req.Status, office, req.Person, cfg.Term)
		p.cases = append(p.cases, c)
		p.add([]facts.Fact{f}, []journal.Entry{e})
	}

	// inspections
	isPerson := map[string]bool{}
	for _, x := range people {
		isPerson[x] = true
	}
	var violations []deduction.Atom
	bearers := map[string]bool{}
	for _, a := range p.res.Conclusions() {
		if a.Neg || !isPerson[a.Args[0].Const] {
			continue
		}
		if a.Pred == "violated" && a.Args[2] == deduction.N(int(p.now)) {
			violations = append(violations, a)
		}
		if a.Pred == "duty" && a.Args[4] == deduction.C("maintain") {
			bearers[a.Args[0].Const] = true
		}
	}
	checked := 0
	for _, office := range offices {
		capacity, ok := p.capacity(office)
		if !ok {
			continue
		}
		if v, rule := competence.Check(p.res.Query, office, "inspect"); v != competence.Allowed {
			p.add(nil, []journal.Entry{p.entry(journal.UltraVires, p.occupant(office), office, "inspect", string(v), rule)})
			continue
		}
		sample := cases.Sample(violations, capacity, s.Seed, p.now)
		checked += len(sample)
		for i, v := range sample {
			person, duty := v.Args[0].Const, v.Args[1].Const
			p.add(nil, []journal.Entry{p.entry(journal.ViolationDetected, p.occupant(office), office, v.String(), "", "")})
			if exists(person, duty) {
				continue
			}
			judge := p.firstCompetent(offices, "judge")
			if judge == "" {
				continue
			}
			c, f, e := cases.File(p.context(), cases.Charge, fmt.Sprintf("ch%d_%d", p.now, i+1), person, duty, judge, p.occupant(office), cfg.Term)
			p.cases = append(p.cases, c)
			p.add([]facts.Fact{f}, []journal.Entry{e})
		}
	}
	if err := p.resolve(); err != nil {
		return Transition{}, err
	}

	// scripted attempts
	for _, a := range cfg.Actors.Attempts {
		if period.Period(a.Period) != p.now {
			continue
		}
		if v, rule := competence.Check(p.res.Query, a.Office, a.Kind); v != competence.Allowed {
			p.add(nil, []journal.Entry{p.entry(journal.UltraVires, a.Actor, a.Office, a.Kind, string(v), rule)})
		}
	}

	// decisions
	for i, c := range p.cases {
		if c.Status != cases.Open {
			continue
		}
		actor := p.occupant(c.Office)
		if actor == "" {
			p.add(nil, []journal.Entry{p.entry(journal.Vacant, "", c.Office, c.ID, "", "")})
			continue
		}
		c, fs, es := cases.Decide(p.context(), c, actor, cfg.Actors.Propose)
		p.cases[i] = c
		p.add(fs, es)
	}

	// intercessio
	for i, c := range p.cases {
		if c.Status != cases.Decided || c.DecidedAt != p.now || !cfg.Actors.Veto(c) {
			continue
		}
		office := p.firstCompetent(offices, "intercessio")
		c, fs, es := cases.Veto(p.context(), c, p.occupant(office), office)
		p.cases[i] = c
		p.add(fs, es)
	}

	// report
	p.add(nil, []journal.Entry{{
		Period: p.now, Kind: journal.PeriodSummary, Decider: journal.ByRule,
		Subject: fmt.Sprintf("violations=%d checked=%d bearers=%d cases=%d", len(violations), checked, len(bearers), len(p.cases)),
	}})

	next := State{Period: p.now + 1, Corpus: s.Corpus, Facts: p.facts, Cases: p.cases, Journal: s.Journal.Append(p.entries...), Seed: s.Seed}
	return Transition{Next: next, Facts: p.added, Entries: next.Journal.Entries[len(s.Journal.Entries):]}, nil
}

func (p *period_) context() cases.Context {
	x := cases.Context{Now: p.now, Basis: journal.Basis{Strategy: p.cfg.Strategy.Name()}}
	if p.res != nil {
		x.Query = p.res.Query
	}
	return x
}

func (p *period_) entry(kind journal.Kind, actor, office, subject, outcome, rule string) journal.Entry {
	return journal.Entry{Period: p.now, Kind: kind, Actor: actor, Office: office, Subject: subject, Outcome: outcome,
		Basis: journal.Basis{Rule: rule, Strategy: p.cfg.Strategy.Name()}, Decider: journal.ByRule}
}

func (p *period_) add(fs []facts.Fact, es []journal.Entry) {
	p.facts = append(p.facts, fs...)
	p.added = append(p.added, fs...)
	p.entries = append(p.entries, es...)
}

func (p *period_) resolve() error {
	res, err := entitlement.Resolve(p.cfg.Strategy, p.s.Corpus, p.facts, p.now)
	if err != nil {
		return err
	}
	p.res = res
	return nil
}

func (p *period_) visible() []facts.Fact { return facts.Snapshot(p.facts, p.now) }

func (p *period_) domain(pred string) []string {
	var out []string
	for _, f := range p.visible() {
		if f.Pred == pred && len(f.Args) == 1 {
			out = append(out, f.Args[0].Const)
		}
	}
	sort.Strings(out)
	return out
}

func (p *period_) firstCompetent(offices []string, kind string) string {
	for _, o := range offices {
		if v, _ := competence.Check(p.res.Query, o, kind); v == competence.Allowed {
			return o
		}
	}
	return ""
}

func (p *period_) occupant(office string) string {
	var who []string
	for _, f := range p.visible() {
		if f.Pred != "occupies" || len(f.Args) != 4 || f.Args[1].Const != office {
			continue
		}
		if f.Args[2].Num <= int(p.now) && int(p.now) <= f.Args[3].Num {
			who = append(who, f.Args[0].Const)
		}
	}
	sort.Strings(who)
	if len(who) == 0 {
		return ""
	}
	return who[0]
}

func (p *period_) capacity(office string) (int, bool) {
	for _, a := range p.s.Corpus.DeclarationsAt(p.now) {
		if a.Pred == "capacity" && len(a.Args) == 2 && a.Args[0].Const == office {
			return a.Args[1].Num, true
		}
	}
	return 0, false
}
