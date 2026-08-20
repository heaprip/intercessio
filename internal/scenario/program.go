package scenario

import (
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/facts"
	"github.com/heaprip/intercessio/internal/period"
)

// Program is the slice of the initial state at now, ready for evaluation.
//
// Prototype shortcut: the defeat relation is the declared one only. Pairs
// generated from the document hierarchy belong to entitlement, which does not
// exist yet.
func (s *Scenario) Program(now period.Period) deduction.Program {
	p := deduction.Program{Rules: s.Corpus.At(now), Defeats: s.Corpus.Defeats}
	for _, f := range facts.Snapshot(s.Facts, now) {
		a := deduction.Atom{Pred: f.Pred}
		for _, v := range f.Args {
			a.Args = append(a.Args, deduction.Value{Const: v.Const, Num: v.Num, IsNum: v.IsNum})
		}
		p.Facts = append(p.Facts, a)
	}
	return p
}
