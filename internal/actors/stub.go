// Package actors decides what participants try to do. Prototype: deterministic
// stubs only; the model participant is the other side of the seam.
package actors

import (
	"github.com/heaprip/intercessio/internal/cases"
	"github.com/heaprip/intercessio/internal/competence"
	"github.com/heaprip/intercessio/internal/deduction"
)

// Request is a petition a person wants to file.
type Request struct {
	Person string
	Status string
}

// Attempt is a scripted action, used to show that competence is checked.
type Attempt struct {
	Period int
	Actor  string
	Office string
	Kind   string
}

// Stub is the deterministic participant.
type Stub struct {
	// VetoGrantsTo lists persons whose grants the tribune stops.
	VetoGrantsTo map[string]bool
	Attempts     []Attempt
}

// Petitions: a person petitions for a status they are entitled to and do not
// hold, unless a case about it already exists.
func (Stub) Petitions(q competence.Query, people, statuses []string, exists func(person, matter string) bool) []Request {
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

// Propose decides by the normative result: proved grants or convicts,
// disproved refuses or acquits, anything else is non liquet.
func (Stub) Propose(c cases.Case, q competence.Query) (string, string) {
	var ans deduction.Answer
	yes, no := "grant", "refuse"
	switch c.Kind {
	case cases.Petition:
		ans = q(deduction.A("entitled", c.Person, c.Matter))
	case cases.Charge:
		ans = q(deduction.A("violated", c.Person, c.Matter, int(c.Opened)))
		yes, no = "guilty", "acquit"
	}
	rule := ""
	if ans.Trace != nil {
		rule = ans.Trace.Rule
	}
	switch ans.Outcome {
	case deduction.Proved:
		return yes, rule
	case deduction.Disproved:
		return no, rule
	}
	return "non-liquet", string(ans.Reason)
}

// Veto says whether the tribune stops this decision.
func (s Stub) Veto(c cases.Case) bool {
	return c.Kind == cases.Petition && c.Outcome == "grant" && s.VetoGrantsTo[c.Person]
}
