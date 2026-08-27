// Package entitlement is the single answerer of "which normative result applies
// to this case in this period". It fills the defeat relation and hands it to
// deduction, which knows nothing about documents.
//
// Prototype.
package entitlement

import (
	"fmt"
	"strings"

	"github.com/heaprip/intercessio/internal/corpus"
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/facts"
	"github.com/heaprip/intercessio/internal/period"
)

// Source says where a defeat pair came from. The trace must carry it, or the
// player cannot see why one rule beat another.
type Source string

const (
	Declared  Source = "declared"
	Generated Source = "generated"
)

// Pair is one filled defeat pair.
type Pair struct {
	Over, Under string
	Source      Source
}

// Strategy fills the defeat relation for a slice of the corpus. It is the seam
// of norm resolution: replacing the model of resolution means replacing this.
type Strategy interface {
	Name() string
	Fill(v corpus.Version, now period.Period) []Pair
}

type ruleInNorm struct {
	rule  deduction.Rule
	norm  corpus.Norm
	level int
}

// fill keeps every declared pair and generates the rest for conflicting rules
// where better says one norm outranks the other. A declared pair in either
// direction suppresses the generated one: declared beats generated.
func fill(v corpus.Version, now period.Period, better func(a, b ruleInNorm) bool) []Pair {
	var pairs []Pair
	declared := map[[2]string]bool{}
	for _, d := range v.Defeats {
		pairs = append(pairs, Pair{d.Over, d.Under, Declared})
		declared[[2]string{d.Over, d.Under}] = true
	}
	var rules []ruleInNorm
	for _, n := range v.Norms {
		if !n.ActiveAt(now) {
			continue
		}
		doc, _ := v.Document(n.Document)
		for _, r := range n.Rules {
			rules = append(rules, ruleInNorm{r, n, doc.Level})
		}
	}
	for i := range rules {
		for j := i + 1; j < len(rules); j++ {
			a, b := rules[i], rules[j]
			if !deduction.Conflicting(a.rule, b.rule) {
				continue
			}
			if declared[[2]string{a.rule.ID, b.rule.ID}] || declared[[2]string{b.rule.ID, a.rule.ID}] {
				continue
			}
			switch {
			case better(a, b):
				pairs = append(pairs, Pair{a.rule.ID, b.rule.ID, Generated})
			case better(b, a):
				pairs = append(pairs, Pair{b.rule.ID, a.rule.ID, Generated})
			}
		}
	}
	return pairs
}

// Hierarchy: a rule of a higher document beats a conflicting rule of a lower
// one. Level 1 is the constitution.
type Hierarchy struct{}

func (Hierarchy) Name() string { return "hierarchy" }

func (Hierarchy) Fill(v corpus.Version, now period.Period) []Pair {
	return fill(v, now, func(a, b ruleInNorm) bool { return a.level < b.level })
}

// LexPosterior: a rule of a later enacted norm beats a conflicting rule of an
// earlier one. The substitute strategy of the substitutability test.
type LexPosterior struct{}

func (LexPosterior) Name() string { return "lex-posterior" }

func (LexPosterior) Fill(v corpus.Version, now period.Period) []Pair {
	return fill(v, now, func(a, b ruleInNorm) bool { return a.norm.Enacted > b.norm.Enacted })
}

// Result is a resolved slice.
type Result struct {
	*deduction.Result
	Pairs  []Pair
	source map[[2]string]Source
}

// Resolve takes the slice at now, fills the defeat relation with the strategy
// and evaluates.
func Resolve(s Strategy, v corpus.Version, fs []facts.Fact, now period.Period) (*Result, error) {
	pairs := s.Fill(v, now)
	prog := deduction.Program{Rules: v.At(now)}
	src := map[[2]string]Source{}
	for _, p := range pairs {
		prog.Defeats = append(prog.Defeats, deduction.Defeat{Over: p.Over, Under: p.Under})
		src[[2]string{p.Over, p.Under}] = p.Source
	}
	prog.Facts = append(prog.Facts, v.DeclarationsAt(now)...)
	for _, f := range facts.Snapshot(fs, now) {
		a := deduction.Atom{Pred: f.Pred}
		for _, x := range f.Args {
			a.Args = append(a.Args, deduction.Value{Const: x.Const, Num: x.Num, IsNum: x.IsNum})
		}
		prog.Facts = append(prog.Facts, a)
	}
	res, err := deduction.Evaluate(prog, int(now))
	if err != nil {
		return nil, fmt.Errorf("resolve with %s at %d: %w", s.Name(), now, err)
	}
	return &Result{Result: res, Pairs: pairs, source: src}, nil
}

// Source tells where the pair over > under came from.
func (r *Result) Source(over, under string) Source { return r.source[[2]string{over, under}] }

// Explain renders a trace with the source of every defeat.
func (r *Result) Explain(t *deduction.Trace) string {
	var b strings.Builder
	var walk func(t *deduction.Trace, depth int)
	walk = func(t *deduction.Trace, depth int) {
		fmt.Fprintf(&b, "%s%s  [%s]", strings.Repeat("  ", depth), t.Atom, t.Rule)
		if len(t.Defeated) > 0 {
			beaten := make([]string, len(t.Defeated))
			for i, d := range t.Defeated {
				beaten[i] = fmt.Sprintf("%s (%s)", d, r.Source(t.Rule, d))
			}
			fmt.Fprintf(&b, "  beats %s", strings.Join(beaten, ", "))
		}
		b.WriteString("\n")
		for _, p := range t.Premises {
			walk(p, depth+1)
		}
	}
	walk(t, 0)
	return b.String()
}
