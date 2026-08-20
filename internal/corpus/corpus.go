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
}

// Version is one immutable state of the corpus.
type Version struct {
	Documents []Document
	Norms     []Norm
	Defeats   []deduction.Defeat
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
