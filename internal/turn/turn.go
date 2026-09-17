// Package turn advances a period: amendment, finalization of earlier
// decisions, living, report. Prototype: the auctor is a script.
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
	"github.com/heaprip/intercessio/internal/impact"
	"github.com/heaprip/intercessio/internal/journal"
	"github.com/heaprip/intercessio/internal/period"
)

// State is everything a period starts from.
type State struct {
	Period  period.Period
	Corpus  corpus.Version
	Facts   []facts.Fact
	Cases   []cases.Case
	Acts    []cases.Act
	Journal journal.Journal
	Seed    int64
	// Queue holds grievances that did not fit an earlier period's budget.
	Queue []Grievance
}

// Participants is the actors seam as the turn consumes it: a stub or a model.
type Participants interface {
	Petitions(q competence.Query, people, statuses []string, exists func(person, matter string) bool) []actors.Request
	Propose(c cases.Case, q competence.Query) cases.Proposal
	Veto(c cases.Case) bool
	Appeal(c cases.Case) bool
	Attempts(now period.Period) []actors.Attempt
}

// Config holds the explicit inputs that are not state.
type Config struct {
	Strategy entitlement.Strategy
	Actors   Participants
	Auctor   Auctor // nil: the corpus does not change
	Term     int    // periods a case may stay open
	// Budget is how many private cases — grievances and petitions — may be
	// filed in a period. Zero means no limit.
	Budget int
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
	corpus  corpus.Version
	facts   []facts.Fact
	cases   []cases.Case
	acts    []cases.Act
	queue   []Grievance
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
	p := &period_{s: s, cfg: cfg, now: s.Period, corpus: s.Corpus, facts: append([]facts.Fact{}, s.Facts...),
		cases: append([]cases.Case{}, s.Cases...), acts: append([]cases.Act{}, s.Acts...), queue: append([]Grievance{}, s.Queue...)}

	// amendment
	if cfg.Auctor != nil {
		if ams := cfg.Auctor.Amendments(p.now, p.corpus); len(ams) > 0 {
			if err := p.amend(ams); err != nil {
				return Transition{}, err
			}
		}
	}

	x := p.context()
	for i, c := range p.cases {
		c, fs, es := cases.Finalize(x, c)
		c, es2 := cases.Expire(x, c)
		p.cases[i] = c
		p.add(fs, append(es, es2...))
	}
	// admission is checked at the moment of taking office: on the slice before
	// the appointment takes effect, or the new tenure would count against itself
	if err := p.resolve(); err != nil {
		return Transition{}, err
	}
	var usurpations []journal.Entry
	for _, a := range p.acts {
		if a.Status != cases.Decided || a.PerformedAt >= p.now || a.Predicate != "appointment" || len(a.Args) != 4 {
			continue
		}
		person, office := a.Args[0], a.Args[1]
		if ans := p.res.Query(deduction.A("eligible", person, office)); ans.Outcome != deduction.Proved {
			outcome := string(ans.Outcome)
			if ans.Reason != "" {
				outcome += " " + string(ans.Reason)
			}
			e := p.entry(journal.Usurpation, person, office, "appointed by "+a.Actor+" "+a.Office, outcome, "")
			e.Case = a.ID
			usurpations = append(usurpations, e)
		}
	}
	for i, a := range p.acts {
		a, fs, es := cases.FinalizeAct(x, a)
		p.acts[i] = a
		p.add(fs, es)
	}
	p.add(nil, usurpations)
	if err := p.resolve(); err != nil {
		return Transition{}, err
	}

	offices, people, statuses := p.domain("office"), p.domain("person"), p.domain("status_kind")

	exists := func(person, matter string) bool {
		for _, c := range p.cases {
			if c.Person == person && c.Matter == matter {
				return true
			}
		}
		return false
	}
	p.private(offices, people, statuses, exists)

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

	// scripted attempts: an allowed attempt with a fact is an act
	for i, a := range cfg.Actors.Attempts(p.now) {
		if v, rule := competence.Check(p.res.Query, a.Office, a.Kind); v != competence.Allowed {
			p.add(nil, []journal.Entry{p.entry(journal.UltraVires, a.Actor, a.Office, a.Kind, string(v), rule)})
			continue
		}
		if len(a.Fact) == 0 {
			continue
		}
		act, fs, es, ok := cases.Perform(p.context(), fmt.Sprintf("act%d_%d", p.now, i+1), a.Actor, a.Office, a.Kind, a.Subject, a.Fact, p.holders(a.Office), p.quorum(a.Office))
		p.add(fs, es)
		if ok {
			p.acts = append(p.acts, act)
		}
	}

	// decisions
	for i, c := range p.cases {
		if c.Status != cases.Open {
			continue
		}
		actor, office, conflict := p.decider(c, offices)
		if actor == "" {
			p.add(nil, []journal.Entry{p.entry(journal.Vacant, "", c.Office, c.ID, "", "")})
			continue
		}
		if actor != p.occupant(c.Office) || office != c.Office {
			e := p.entry(journal.Recusal, actor, office, c.Person+" "+c.Matter, "from "+p.occupant(c.Office)+" "+c.Office, "")
			e.Case = c.ID
			p.add(nil, []journal.Entry{e})
			c.Office = office
		}
		if conflict {
			e := p.entry(journal.Conflict, actor, office, c.Person+" "+c.Matter, "decides own case", "")
			e.Case = c.ID
			p.add(nil, []journal.Entry{e})
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

	// review
	for i, c := range p.cases {
		if (c.Status != cases.Decided && c.Status != cases.Refused) || c.DecidedAt != p.now || c.Reviewed || !cfg.Actors.Appeal(c) {
			continue
		}
		actor, office := p.reviewer(c)
		if actor == "" {
			e := p.entry(journal.Review, c.Person, c.Office, c.Person+" "+c.Matter, "no-reviewer", "")
			e.Case = c.ID
			p.add(nil, []journal.Entry{e})
			c.Reviewed = true
			p.cases[i] = c
			continue
		}
		c, fs, es := cases.Review(p.context(), c, actor, office, cfg.Actors.Propose)
		p.cases[i] = c
		p.add(fs, es)
	}

	// report
	p.add(nil, []journal.Entry{{
		Period: p.now, Kind: journal.PeriodSummary, Decider: journal.ByRule,
		Subject: fmt.Sprintf("violations=%d checked=%d bearers=%d cases=%d queue=%d", len(violations), checked, len(bearers), len(p.cases), len(p.queue)),
	}})

	next := State{Period: p.now + 1, Corpus: p.corpus, Facts: p.facts, Cases: p.cases, Journal: s.Journal.Append(p.entries...), Seed: s.Seed, Queue: p.queue, Acts: p.acts}
	return Transition{Next: next, Facts: p.added, Entries: next.Journal.Entries[len(s.Journal.Entries):]}, nil
}

// amend applies the auctor's amendments, recounts who they touched and queues
// their grievances.
func (p *period_) amend(ams []Amendment) error {
	before := p.corpus
	after := apply(before, ams)
	from := p.now
	for _, a := range ams {
		if a.Enact != nil && a.Enact.InForce < from {
			from = a.Enact.InForce
		}
		p.add(nil, []journal.Entry{p.entry(journal.Amendment, "auctor", "", a.Subject(), fmt.Sprintf("v%d", after.Number), "")})
	}
	imp, err := impact.Compute(p.cfg.Strategy, before, after, p.facts, from, p.now)
	if err != nil {
		return err
	}
	for _, g := range grievances(imp, p.domain("person"), p.now) {
		p.add(nil, []journal.Entry{p.entry(journal.Grievance, g.Person, "", g.Loss, "queued", "")})
		p.queue = append(p.queue, g)
	}
	p.corpus = after
	return nil
}

// private files grievances from the queue and petitions of the period in one
// order — heavier loss first, older first — until the budget is spent.
// Grievances that do not fit stay queued; petitions that do not fit are asked
// again next period.
func (p *period_) private(offices, people, statuses []string, exists func(person, matter string) bool) {
	type item struct {
		g   *Grievance
		req actors.Request
	}
	var items []item
	for i := range p.queue {
		items = append(items, item{g: &p.queue[i]})
	}
	for _, req := range p.cfg.Actors.Petitions(p.res.Query, people, statuses, exists) {
		if req.Unserved != "" {
			e := p.entry(journal.Unserved, req.Person, "", "petition "+req.Status, req.Unserved, "")
			e.Model, e.Call = req.Model, req.Call
			p.add(nil, []journal.Entry{e})
			continue
		}
		if req.File {
			items = append(items, item{req: req})
		}
	}
	weight := func(it item) (int, period.Period, string) {
		if it.g != nil {
			return it.g.Weight, it.g.Since, it.g.Person
		}
		return weightPetition, p.now, it.req.Person
	}
	sort.SliceStable(items, func(i, j int) bool {
		wa, sa, pa := weight(items[i])
		wb, sb, pb := weight(items[j])
		if wa != wb {
			return wa > wb
		}
		if sa != sb {
			return sa < sb
		}
		return pa < pb
	})

	var keep []Grievance
	filed := 0
	for _, it := range items {
		person, matter := it.req.Person, it.req.Status
		if it.g != nil {
			person, matter = it.g.Person, it.g.Matter
		}
		office := p.firstCompetent(offices, "grant_status")
		if it.g != nil && (matter == "" || office == "") {
			p.add(nil, []journal.Entry{p.entry(journal.Grievance, person, "", it.g.Loss, "no-remedy", "")})
			continue
		}
		if office == "" || exists(person, matter) {
			continue
		}
		if p.cfg.Budget > 0 && filed >= p.cfg.Budget {
			if it.g != nil {
				keep = append(keep, *it.g)
				p.add(nil, []journal.Entry{p.entry(journal.Grievance, person, "", it.g.Loss, "deferred", "")})
			} else {
				p.add(nil, []journal.Entry{p.entry(journal.Deferred, person, "", "petition "+matter, "", "")})
			}
			continue
		}
		filed++
		c, f, e := cases.File(p.context(), cases.Petition, fmt.Sprintf("pt%d_%d", p.now, filed), person, matter, office, person, p.cfg.Term)
		if it.g != nil {
			p.add(nil, []journal.Entry{p.entry(journal.Grievance, person, office, it.g.Loss, "filed", "")})
		} else {
			e.Decider, e.Model, e.Call = it.req.Decider, it.req.Model, it.req.Call
		}
		p.cases = append(p.cases, c)
		p.add([]facts.Fact{f}, []journal.Entry{e})
	}
	p.queue = keep
}

func (p *period_) context() cases.Context {
	x := cases.Context{Now: p.now, Basis: journal.Basis{Strategy: p.cfg.Strategy.Name(), Corpus: p.corpus.Number}}
	if p.res != nil {
		x.Query = p.res.Query
	}
	return x
}

func (p *period_) entry(kind journal.Kind, actor, office, subject, outcome, rule string) journal.Entry {
	return journal.Entry{Period: p.now, Kind: kind, Actor: actor, Office: office, Subject: subject, Outcome: outcome,
		Basis: journal.Basis{Rule: rule, Strategy: p.cfg.Strategy.Name(), Corpus: p.corpus.Number}, Decider: journal.ByRule}
}

func (p *period_) add(fs []facts.Fact, es []journal.Entry) {
	p.facts = append(p.facts, fs...)
	p.added = append(p.added, fs...)
	p.entries = append(p.entries, es...)
}

func (p *period_) resolve() error {
	res, err := entitlement.Resolve(p.cfg.Strategy, p.corpus, p.facts, p.now)
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

// reviewer finds a held office declared to review the deciding office, with a
// holder who is not the person the case is about.
func (p *period_) reviewer(c cases.Case) (actor, office string) {
	stored := p.corpus.DeclarationsAt(p.now)
	for _, f := range p.visible() {
		a := deduction.Atom{Pred: f.Pred}
		for _, v := range f.Args {
			a.Args = append(a.Args, deduction.Value{Const: v.Const, Num: v.Num, IsNum: v.IsNum})
		}
		stored = append(stored, a)
	}
	var offices []string
	for _, a := range stored {
		if a.Pred == "reviews" && len(a.Args) == 2 && a.Args[1].Const == c.Office {
			offices = append(offices, a.Args[0].Const)
		}
	}
	sort.Strings(offices)
	for _, o := range offices {
		if v, _ := competence.Check(p.res.Query, o, "review"); v != competence.Allowed {
			continue
		}
		for _, h := range p.holders(o) {
			if h != c.Person {
				return h, o
			}
		}
	}
	return "", ""
}

// decider picks who decides a case. Nobody decides their own case while
// someone else can: first another holder of the office, then a holder of
// another office competent for the same kind. Only when there is nobody else
// does the concerned holder decide, and that is a conflict.
func (p *period_) decider(c cases.Case, offices []string) (actor, office string, conflict bool) {
	holders := p.holders(c.Office)
	if len(holders) == 0 {
		return "", c.Office, false
	}
	for _, h := range holders {
		if h != c.Person {
			return h, c.Office, false
		}
	}
	for _, o := range offices {
		if o == c.Office {
			continue
		}
		if v, _ := competence.Check(p.res.Query, o, c.ActionKind()); v != competence.Allowed {
			continue
		}
		for _, h := range p.holders(o) {
			if h != c.Person {
				return h, o, false
			}
		}
	}
	return holders[0], c.Office, true
}

func (p *period_) occupant(office string) string {
	if h := p.holders(office); len(h) > 0 {
		return h[0]
	}
	return ""
}

func (p *period_) holders(office string) []string {
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
	return who
}

func (p *period_) quorum(office string) int {
	for _, a := range p.corpus.DeclarationsAt(p.now) {
		if a.Pred == "quorum" && len(a.Args) == 2 && a.Args[0].Const == office {
			return a.Args[1].Num
		}
	}
	for _, f := range p.visible() {
		if f.Pred == "quorum" && len(f.Args) == 2 && f.Args[0].Const == office {
			return f.Args[1].Num
		}
	}
	return 0
}

func (p *period_) capacity(office string) (int, bool) {
	for _, a := range p.corpus.DeclarationsAt(p.now) {
		if a.Pred == "capacity" && len(a.Args) == 2 && a.Args[0].Const == office {
			return a.Args[1].Num, true
		}
	}
	return 0, false
}
