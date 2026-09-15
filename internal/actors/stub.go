// Package actors decides what participants try to do: a deterministic stub or
// a model, on the two sides of one seam.
package actors

import (
	"strings"

	"github.com/heaprip/intercessio/internal/cases"
	"github.com/heaprip/intercessio/internal/competence"
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/journal"
	"github.com/heaprip/intercessio/internal/period"
)

// Request is a person's answer about a petition they could file.
type Request struct {
	Person   string
	Status   string
	File     bool
	Decider  journal.Decider
	Model    string
	Call     string
	Unserved string
}

// Attempt is a scripted action, used to show that competence is checked.
type Attempt struct {
	Period int
	Actor  string
	Office string
	Kind   string
	// Subject and Fact make the attempt an act: Fact is the act predicate and
	// its arguments after the act id, e.g. adrogatio, clodius, fonteius.
	Subject string
	Fact    []string
}

// Stub is the deterministic participant.
type Stub struct {
	// VetoGrantsTo lists persons whose grants the tribune stops.
	VetoGrantsTo map[string]bool
	Script       []Attempt
	// Appeals makes every refused or convicted person appeal once.
	Appeals bool
}

// Appeal says whether the person appeals the decision of this period.
func (s Stub) Appeal(c cases.Case) bool {
	return s.Appeals && !c.Reviewed && (c.Outcome == "refuse" || c.Outcome == "guilty")
}

// Candidates are the petitions the law makes worth filing: a status the person
// is entitled to and does not hold, with no case about it yet.
func Candidates(q competence.Query, people, statuses []string, exists func(person, matter string) bool) []Request {
	var out []Request
	for _, p := range people {
		for _, s := range statuses {
			if exists(p, s) {
				continue
			}
			if q(deduction.A("entitled", p, s)).Outcome == deduction.Proved && q(deduction.A("status", p, s)).Outcome != deduction.Proved {
				out = append(out, Request{Person: p, Status: s})
			}
		}
	}
	return out
}

// Petitions: the stub files every candidate.
func (Stub) Petitions(q competence.Query, people, statuses []string, exists func(person, matter string) bool) []Request {
	out := Candidates(q, people, statuses, exists)
	for i := range out {
		out[i].File, out[i].Decider = true, journal.ByStub
	}
	return out
}

// Normative returns the question a case asks the law, and the outcomes that
// follow from proved, disproved and unknown.
func Normative(c cases.Case) (deduction.Atom, [3]string) {
	if c.Kind == cases.Charge {
		return deduction.A("violated", c.Person, c.Matter, int(c.Opened)), [3]string{"guilty", "acquit", "non-liquet"}
	}
	return deduction.A("entitled", c.Person, c.Matter), [3]string{"grant", "refuse", "non-liquet"}
}

// Propose decides by the normative result: proved grants or convicts,
// disproved refuses or acquits, anything else is non liquet.
func (Stub) Propose(c cases.Case, q competence.Query) cases.Proposal {
	atom, outcomes := Normative(c)
	ans := q(atom)
	p := cases.Proposal{Decider: journal.ByStub}
	if ans.Trace != nil {
		p.Rule, p.Rules = ans.Trace.Rule, strings.Join(ans.Trace.Rules(), ",")
	}
	switch ans.Outcome {
	case deduction.Proved:
		p.Outcome = outcomes[0]
	case deduction.Disproved:
		p.Outcome = outcomes[1]
	default:
		p.Outcome, p.Rule = outcomes[2], string(ans.Reason)
	}
	return p
}

// Veto says whether the tribune stops this decision.
func (s Stub) Veto(c cases.Case) bool {
	return c.Kind == cases.Petition && c.Outcome == "grant" && s.VetoGrantsTo[c.Person]
}

// Attempts returns the scripted attempts of a period.
func (s Stub) Attempts(now period.Period) []Attempt {
	var out []Attempt
	for _, a := range s.Script {
		if period.Period(a.Period) == now {
			out = append(out, a)
		}
	}
	return out
}
