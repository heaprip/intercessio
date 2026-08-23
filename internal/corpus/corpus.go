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
	Kind     Kind
	Rules    []deduction.Rule
	// Declarations are the data the norm states: bundles of statuses,
	// competences and requirements of offices, capacities.
	Declarations []deduction.Atom
}

// Version is one immutable state of the corpus.
type Version struct {
	Number    int
	Documents []Document
	Norms     []Norm
	Defeats   []deduction.Defeat
}

// DeclarationsAt returns the declarations of the norms in force at now.
func (v Version) DeclarationsAt(now period.Period) []deduction.Atom {
	var out []deduction.Atom
	for _, n := range v.Norms {
		if n.InForce <= now {
			out = append(out, n.Declarations...)
		}
	}
	return out
}

// Without returns a new version with the norm removed. The receiver is not
// changed: old versions stay alive for impact.
func (v Version) Without(normID string) Version {
	out := Version{Number: v.Number + 1, Documents: v.Documents, Defeats: v.Defeats}
	for _, n := range v.Norms {
		if n.ID != normID {
			out.Norms = append(out.Norms, n)
		}
	}
	return out
}

// With returns a new version with the norm added.
func (v Version) With(n Norm) Version {
	out := Version{Number: v.Number + 1, Documents: v.Documents, Defeats: v.Defeats}
	out.Norms = append(append([]Norm{}, v.Norms...), n)
	return out
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
		if n.InForce <= now {
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
