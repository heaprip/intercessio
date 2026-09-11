// Package corpus holds versions of the normative corpus. Prototype: types only.
package corpus

import (
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/period"
)

// Kind of a norm.
type Kind string

const (
	Constitutive Kind = "constitutive"
	Lasting      Kind = "lasting"
)

// Document carries a level in the hierarchy.
type Document struct {
	ID    string
	Level int
}

// Norm carries rules and dates.
type Norm struct {
	ID       string
	Document string
	Enacted  period.Period
	InForce  period.Period
	// Until is the first period the text no longer applies to; zero means open.
	// An amendment closes the old text at the in-force period of the new one.
	Until period.Period
	Kind  Kind
	Rules []deduction.Rule
	// Declarations are the data the norm states: bundles of statuses,
	// competences and requirements of offices, capacities.
	Declarations []deduction.Atom
}

// ActiveAt reports whether the text of the norm applies to period now.
func (n Norm) ActiveAt(now period.Period) bool {
	return n.InForce <= now && (n.Until == 0 || now < n.Until)
}

// Retroactive reports whether the norm applies to periods before its enactment.
func (n Norm) Retroactive() bool { return n.InForce < n.Enacted }

// Act declares that a stored fact predicate records an act of an office: which
// action kinds produce it and which action kinds can stop it before it takes
// effect. These are the declared edges of the power graph.
type Act struct {
	Predicate string
	Kinds     []string
	StoppedBy []string
	// Outcomes map the outcome constant at position OutcomeArg to the kind that
	// produces an act with that outcome: a decision to grant is not a judgment.
	// Empty when one kind covers every outcome.
	OutcomeArg int
	Outcomes   map[string]string
}

// KindFor returns the action kind an act literal with these arguments needs,
// or false when the outcome is not a known constant.
func (a Act) KindFor(args []deduction.Term) (string, bool) {
	if len(a.Outcomes) == 0 || a.OutcomeArg >= len(args) || args[a.OutcomeArg].Kind != deduction.Const {
		return "", false
	}
	k, ok := a.Outcomes[args[a.OutcomeArg].Name]
	return k, ok
}

// Version is one immutable state of the corpus.
type Version struct {
	Number    int
	Documents []Document
	Norms     []Norm
	Defeats   []deduction.Defeat
	Acts      []Act
}

// DeclarationsAt returns the declarations of the norms in force at now.
func (v Version) DeclarationsAt(now period.Period) []deduction.Atom {
	var out []deduction.Atom
	for _, n := range v.Norms {
		if n.ActiveAt(now) {
			out = append(out, n.Declarations...)
		}
	}
	return out
}

func (v Version) next() Version {
	return Version{Number: v.Number + 1, Documents: v.Documents, Defeats: v.Defeats, Acts: v.Acts}
}

// Without returns a new version with the norm removed. The receiver is not
// changed: old versions stay alive for impact. A repealed norm takes with it the
// declared defeat pairs that name its rules: a pair orders two rules, and with
// one of them gone it orders nothing.
func (v Version) Without(normID string) Version {
	out := v.next()
	gone := map[string]bool{}
	for _, n := range v.Norms {
		if n.ID != normID {
			out.Norms = append(out.Norms, n)
			continue
		}
		for _, r := range n.Rules {
			gone[r.ID] = true
		}
	}
	out.Defeats = nil
	for _, d := range v.Defeats {
		if !gone[d.Over] && !gone[d.Under] {
			out.Defeats = append(out.Defeats, d)
		}
	}
	return out
}

// With returns a new version with the norm added.
func (v Version) With(n Norm) Version {
	out := v.next()
	out.Norms = append(append([]Norm{}, v.Norms...), n)
	return out
}

// Amend returns a new version where the open text of the norm with the same id
// stops applying from n.InForce and n applies from then on. Periods before stay
// under the old text, as tempus regit actum reads them.
func (v Version) Amend(n Norm) Version {
	out := v.next()
	for _, old := range v.Norms {
		if old.ID == n.ID && old.Until == 0 {
			old.Until = n.InForce
		}
		out.Norms = append(out.Norms, old)
	}
	out.Norms = append(out.Norms, n)
	return out
}

// Norm returns the open text of a norm.
func (v Version) Norm(id string) (Norm, bool) {
	for _, n := range v.Norms {
		if n.ID == id && n.Until == 0 {
			return n, true
		}
	}
	return Norm{}, false
}

// Document looks a document up by id.
func (v Version) Document(id string) (Document, bool) {
	for _, d := range v.Documents {
		if d.ID == id {
			return d, true
		}
	}
	return Document{}, false
}

// At returns the rules of the norms in force at now.
func (v Version) At(now period.Period) []deduction.Rule {
	var out []deduction.Rule
	for _, n := range v.Norms {
		if n.ActiveAt(now) {
			out = append(out, n.Rules...)
		}
	}
	return out
}

// Rules lists the rules of all norms.
func (v Version) Rules() []deduction.Rule {
	var out []deduction.Rule
	for _, n := range v.Norms {
		out = append(out, n.Rules...)
	}
	return out
}
