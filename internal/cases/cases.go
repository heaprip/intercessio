// Package cases implements the component "case" (a Go keyword): the path of a
// case from petition or charge to execution, and the inspection of duties.
//
// Prototype. The path is shortened: the question to be tried and review stay
// folded; a decision can be stopped by intercessio in the period it is made and
// enters into force in the next one.
package cases

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"

	"github.com/heaprip/intercessio/internal/competence"
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/facts"
	"github.com/heaprip/intercessio/internal/journal"
	"github.com/heaprip/intercessio/internal/period"
)

// Kind of a case: how it was opened.
type Kind string

const (
	// Petition is the private path: a person asks for an outcome.
	Petition Kind = "petition"
	// Charge is the official path: an inspection found a violation.
	Charge Kind = "charge"
)

// Status of a case. Stored in the prototype for simplicity; everything it
// rests on is also a fact.
type Status string

const (
	Open      Status = "open"
	Decided   Status = "decided"
	Vetoed    Status = "vetoed"
	Final     Status = "final"
	Refused   Status = "refused"
	NonLiquet Status = "non-liquet"
	Expired   Status = "expired"
)

// Case is one case.
type Case struct {
	ID        string
	Kind      Kind
	Person    string
	Matter    string // status petitioned for, or duty violated
	Office    string // deciding office
	Opened    period.Period
	Deadline  period.Period
	Decision  string
	Outcome   string
	DecidedAt period.Period
	Status    Status
	Reviewed  bool // appealed once already
}

// ActionKind is the kind of action deciding this case requires.
func (c Case) ActionKind() string {
	if c.Kind == Charge {
		return "judge"
	}
	return "grant_status"
}

// Context is what every step needs.
type Context struct {
	Now   period.Period
	Query competence.Query
	Basis journal.Basis
}

func (x Context) entry(kind journal.Kind, c Case, actor, office, subject, outcome, rule string, by journal.Decider) journal.Entry {
	b := x.Basis
	b.Rule = rule
	return journal.Entry{Period: x.Now, Kind: kind, Actor: actor, Office: office, Case: c.ID, Subject: subject, Outcome: outcome, Basis: b, Decider: by}
}

func fact(id, pred string, now period.Period, by string, args ...any) facts.Fact {
	f := facts.Fact{ID: id, Pred: pred, Prov: facts.Provenance{By: by, Period: now, Source: "case"}}
	for _, a := range args {
		switch v := a.(type) {
		case int:
			f.Args = append(f.Args, facts.Value{Num: v, IsNum: true})
		case period.Period:
			f.Args = append(f.Args, facts.Value{Num: int(v), IsNum: true})
		case string:
			f.Args = append(f.Args, facts.Value{Const: v})
		}
	}
	return f
}

// File registers a case and its founding fact: petition(C, P, S, N) or
// charge(C, P, K, N).
func File(x Context, kind Kind, id, person, matter, office, filer string, term int) (Case, facts.Fact, journal.Entry) {
	c := Case{ID: id, Kind: kind, Person: person, Matter: matter, Office: office, Opened: x.Now, Deadline: x.Now + period.Period(term), Status: Open}
	pred, ek := "petition", journal.PetitionFiled
	if kind == Charge {
		pred, ek = "charge", journal.ChargeFiled
	}
	return c, fact(id, pred, x.Now, filer, id, person, matter, x.Now), x.entry(ek, c, filer, "", person+" "+matter, "", "", journal.ByStub)
}

// Proposal is a participant's proposed outcome and who produced it.
type Proposal struct {
	Outcome  string
	Rule     string // rule the normative result rested on
	Rules    string // every rule of the trace, comma-separated
	Decider  journal.Decider
	Model    string
	Call     string
	Unserved string // set when the model call was not served; the case waits
}

// Propose is the participant's proposal for the outcome. It comes from actors:
// a stub or a model.
type Propose func(c Case, q competence.Query) Proposal

// Decide checks the office's competence before anything else, then records the
// proposed outcome as a decision.
func Decide(x Context, c Case, actor string, propose Propose) (Case, []facts.Fact, []journal.Entry) {
	verdict, rule := competence.Check(x.Query, c.Office, c.ActionKind())
	if verdict != competence.Allowed {
		return c, nil, []journal.Entry{x.entry(journal.UltraVires, c, actor, c.Office, c.ActionKind(), string(verdict), rule, journal.ByRule)}
	}
	p := propose(c, x.Query)
	rules := p.Rules
	if rule != "" {
		rules = strings.Trim(rules+","+rule, ",")
	}
	stamp := func(e journal.Entry) []journal.Entry {
		e.Decider, e.Model, e.Call = p.Decider, p.Model, p.Call
		e.Basis.Rules = rules
		return []journal.Entry{e}
	}
	if p.Unserved != "" {
		// declared fallback of an official: the case stays in the queue, the term runs
		e := x.entry(journal.Unserved, c, actor, c.Office, c.Person+" "+c.Matter, p.Unserved, p.Rule, journal.ByRule)
		e.Model, e.Call = p.Model, p.Call
		return c, nil, []journal.Entry{e}
	}
	outcome, basis := p.Outcome, p.Rule
	if outcome == "non-liquet" {
		c.Status = NonLiquet
		return c, nil, stamp(x.entry(journal.NonLiquet, c, actor, c.Office, c.Person+" "+c.Matter, outcome, basis, journal.ByStub))
	}
	c.Decision = fmt.Sprintf("d_%s", c.ID)
	c.Outcome = outcome
	c.DecidedAt = x.Now
	c.Status = Decided
	if outcome == "refuse" || outcome == "acquit" {
		c.Status = Refused
	}
	f := fact(c.Decision, "decision", x.Now, actor, c.Decision, c.ID, outcome, c.Office, x.Now)
	return c, []facts.Fact{f}, stamp(x.entry(journal.Decision, c, actor, c.Office, c.Person+" "+c.Matter, outcome, basis, journal.ByStub))
}

// Veto stops a decision before it enters into force, if the office may.
func Veto(x Context, c Case, actor, office string) (Case, []facts.Fact, []journal.Entry) {
	verdict, rule := competence.Check(x.Query, office, "intercessio")
	if verdict != competence.Allowed {
		return c, nil, []journal.Entry{x.entry(journal.UltraVires, c, actor, office, "intercessio", string(verdict), rule, journal.ByRule)}
	}
	c.Status = Vetoed
	f := fact("v_"+c.Decision, "vetoes", x.Now, actor, actor, c.Decision)
	return c, []facts.Fact{f}, []journal.Entry{x.entry(journal.Intercessio, c, actor, office, c.Decision, "stopped", rule, journal.ByStub)}
}

// Review decides an appealed case anew by the reviewing office. The same
// outcome upholds the decision; another one replaces it before it enters into
// force.
func Review(x Context, c Case, actor, office string, propose Propose) (Case, []facts.Fact, []journal.Entry) {
	verdict, rule := competence.Check(x.Query, office, "review")
	if verdict != competence.Allowed {
		return c, nil, []journal.Entry{x.entry(journal.UltraVires, c, actor, office, "review", string(verdict), rule, journal.ByRule)}
	}
	c.Reviewed = true
	p := propose(c, x.Query)
	if p.Unserved != "" {
		e := x.entry(journal.Unserved, c, actor, office, "review "+c.ID, p.Unserved, p.Rule, journal.ByRule)
		e.Model, e.Call = p.Model, p.Call
		return c, nil, []journal.Entry{e}
	}
	e := x.entry(journal.Review, c, actor, office, c.Person+" "+c.Matter, "upheld "+c.Outcome, p.Rule, p.Decider)
	e.Model, e.Call, e.Basis.Rules = p.Model, p.Call, p.Rules
	if p.Outcome == c.Outcome {
		return c, nil, []journal.Entry{e}
	}
	e.Outcome = "changed to " + p.Outcome
	c.Outcome, c.DecidedAt = p.Outcome, x.Now
	switch p.Outcome {
	case "non-liquet":
		c.Status, c.Decision = NonLiquet, ""
		return c, nil, []journal.Entry{e}
	case "refuse", "acquit":
		c.Status = Refused
	default:
		c.Status = Decided
	}
	c.Decision = "rv_" + c.ID
	f := fact(c.Decision, "decision", x.Now, actor, c.Decision, c.ID, p.Outcome, office, x.Now)
	return c, []facts.Fact{f}, []journal.Entry{e}
}

// Finalize puts a decision made in an earlier period into force. Its effects
// are then derived by the rules; execution only records that.
func Finalize(x Context, c Case) (Case, []facts.Fact, []journal.Entry) {
	if c.Status != Decided || c.DecidedAt >= x.Now {
		return c, nil, nil
	}
	c.Status = Final
	f := fact("f_"+c.Decision, "in_force", x.Now, "case", c.Decision, x.Now)
	return c, []facts.Fact{f}, []journal.Entry{
		x.entry(journal.Finalization, c, "", c.Office, c.Decision, c.Outcome, "", journal.ByRule),
		x.entry(journal.Execution, c, "", c.Office, c.Person+" "+c.Matter, c.Outcome, "", journal.ByRule),
	}
}

// Expire closes an open case past its deadline.
func Expire(x Context, c Case) (Case, []journal.Entry) {
	if c.Status != Open || x.Now <= c.Deadline {
		return c, nil
	}
	c.Status = Expired
	return c, []journal.Entry{x.entry(journal.Expired, c, "", c.Office, c.Person+" "+c.Matter, "", "", journal.ByRule)}
}

// Sample picks at most capacity violations to inspect. Deterministic by seed
// and period, so a period replays.
func Sample(violations []deduction.Atom, capacity int, seed int64, now period.Period) []deduction.Atom {
	v := append([]deduction.Atom{}, violations...)
	sort.Slice(v, func(i, j int) bool { return v[i].String() < v[j].String() })
	r := rand.New(rand.NewPCG(uint64(seed), uint64(now)))
	r.Shuffle(len(v), func(i, j int) { v[i], v[j] = v[j], v[i] })
	if capacity < len(v) {
		v = v[:capacity]
	}
	return v
}

// Act is a decision without a case: an office changes a status, a duty or an
// office directly, within its competence. It enters into force in the next
// period, like a decision.
type Act struct {
	ID          string
	Kind        string
	Office      string
	Actor       string
	Subject     string
	Predicate   string
	Args        []string
	PerformedAt period.Period
	Status      Status // Decided when performed, Final in force
}

// Perform checks the office's competence and the quorum of its holders, then
// records the act: performed/4, the act fact itself and, for a collegial office,
// concurrence/2. Holders are the office's holders in the period.
func Perform(x Context, id, actor, office, kind, subject string, fact []string, holders []string, quorum int) (Act, []facts.Fact, []journal.Entry, bool) {
	a := Act{ID: id, Kind: kind, Office: office, Actor: actor, Subject: subject, PerformedAt: x.Now}
	entry := func(k journal.Kind, outcome, rule string) journal.Entry {
		e := x.entry(k, Case{ID: id}, actor, office, kind+" "+subject, outcome, rule, journal.ByStub)
		return e
	}
	if verdict, rule := competence.Check(x.Query, office, kind); verdict != competence.Allowed {
		e := entry(journal.UltraVires, string(verdict), rule)
		e.Subject, e.Decider = kind, journal.ByRule
		return a, nil, []journal.Entry{e}, false
	}
	if quorum > 1 && len(holders) < quorum {
		return a, nil, []journal.Entry{entry(journal.NoQuorum, fmt.Sprintf("%d of %d holders", len(holders), quorum), "")}, false
	}
	if len(fact) == 0 {
		return a, nil, nil, false
	}
	a.Predicate, a.Args, a.Status = fact[0], fact[1:], Decided
	args := []any{id}
	for _, v := range a.Args {
		if n, err := strconv.Atoi(v); err == nil {
			args = append(args, n)
			continue
		}
		args = append(args, v)
	}
	fs := []facts.Fact{
		fact_(id+"_performed", "performed", x.Now, actor, id, kind, office, subject),
		fact_(id, a.Predicate, x.Now, actor, args...),
	}
	if quorum > 1 {
		fs = append(fs, fact_(id+"_concurrence", "concurrence", x.Now, actor, id, office))
	}
	return a, fs, []journal.Entry{entry(journal.Act, strings.Join(fact, " "), "")}, true
}

func fact_(id, pred string, now period.Period, by string, args ...any) facts.Fact {
	f := fact(id, pred, now, by, args...)
	f.Prov.Source = "act"
	return f
}

// FinalizeAct puts an act performed in an earlier period into force.
func FinalizeAct(x Context, a Act) (Act, []facts.Fact, []journal.Entry) {
	if a.Status != Decided || a.PerformedAt >= x.Now {
		return a, nil, nil
	}
	a.Status = Final
	fs := []facts.Fact{fact("f_"+a.ID, "in_force", x.Now, "case", a.ID, x.Now)}
	es := []journal.Entry{x.entry(journal.Finalization, Case{ID: a.ID}, "", a.Office, a.ID, a.Kind, "", journal.ByRule)}
	// execution of an appointment: the person holds the office for the term
	if a.Predicate == "appointment" && len(a.Args) == 4 {
		from, err1 := strconv.Atoi(a.Args[2])
		to, err2 := strconv.Atoi(a.Args[3])
		if err1 == nil && err2 == nil {
			fs = append(fs, fact("occ_"+a.ID, "occupies", x.Now, a.Actor, a.Args[0], a.Args[1], from, to))
			es = append(es, x.entry(journal.Execution, Case{ID: a.ID}, "", a.Office, a.Args[0]+" "+a.Args[1], a.Kind, "", journal.ByRule))
		}
	}
	return a, fs, es
}
