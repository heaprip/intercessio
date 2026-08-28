// Package impact recounts who an amendment touched: it subtracts two runs of
// entitlement over two versions of the corpus, period by period.
//
// Prototype. No cache: a recount is two resolutions per period.
package impact

import (
	"fmt"
	"sort"
	"strings"

	"github.com/heaprip/intercessio/internal/corpus"
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/entitlement"
	"github.com/heaprip/intercessio/internal/facts"
	"github.com/heaprip/intercessio/internal/period"
)

// Change is one conclusion that holds under one version and not the other.
type Change struct {
	Period period.Period
	Atom   deduction.Atom
	// Gained is true when the conclusion holds only after the amendment.
	Gained bool
}

func (c Change) String() string {
	sign := "-"
	if c.Gained {
		sign = "+"
	}
	return fmt.Sprintf("%d %s %s", c.Period, sign, c.Atom)
}

// Report is the recount over a window of periods.
type Report struct {
	Strategy      string
	Before, After int
	From, To      period.Period
	Changes       []Change
}

// Compute resolves both versions at every period of [from, to] and lists the
// proved literals that differ. Facts are the same for both runs: an amendment
// changes the law, not what happened.
func Compute(s entitlement.Strategy, before, after corpus.Version, fs []facts.Fact, from, to period.Period) (Report, error) {
	rep := Report{Strategy: s.Name(), Before: before.Number, After: after.Number, From: from, To: to}
	for p := from; p <= to; p++ {
		b, err := conclusions(s, before, fs, p)
		if err != nil {
			return rep, err
		}
		a, err := conclusions(s, after, fs, p)
		if err != nil {
			return rep, err
		}
		for k, atom := range b {
			if _, ok := a[k]; !ok {
				rep.Changes = append(rep.Changes, Change{Period: p, Atom: atom})
			}
		}
		for k, atom := range a {
			if _, ok := b[k]; !ok {
				rep.Changes = append(rep.Changes, Change{Period: p, Atom: atom, Gained: true})
			}
		}
	}
	sort.Slice(rep.Changes, func(i, j int) bool {
		x, y := rep.Changes[i], rep.Changes[j]
		if x.Period != y.Period {
			return x.Period < y.Period
		}
		return x.Atom.String() < y.Atom.String()
	})
	return rep, nil
}

func conclusions(s entitlement.Strategy, v corpus.Version, fs []facts.Fact, p period.Period) (map[string]deduction.Atom, error) {
	res, err := entitlement.Resolve(s, v, fs, p)
	if err != nil {
		return nil, fmt.Errorf("impact at %d, version %d: %w", p, v.Number, err)
	}
	out := map[string]deduction.Atom{}
	for _, a := range res.Conclusions() {
		out[a.String()] = a
	}
	return out, nil
}

// Person is one affected person and what changed for them.
type Person struct {
	ID      string
	Changes []Change
}

// Affected groups the changes by the people they name. A change naming nobody
// from people is not listed.
func (r Report) Affected(people []string) []Person {
	idx := map[string]int{}
	var out []Person
	for _, id := range people {
		for _, c := range r.Changes {
			if !mentions(c.Atom, id) {
				continue
			}
			i, ok := idx[id]
			if !ok {
				i = len(out)
				idx[id] = i
				out = append(out, Person{ID: id})
			}
			out[i].Changes = append(out[i].Changes, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func mentions(a deduction.Atom, id string) bool {
	for _, v := range a.Args {
		if !v.IsNum && v.Const == id {
			return true
		}
	}
	return false
}

// Render prints the recount by person.
func (r Report) Render(people []string) string {
	var b strings.Builder
	affected := r.Affected(people)
	fmt.Fprintf(&b, "impact v%d -> v%d, periods %d..%d, %s: %d changes, %d people\n", r.Before, r.After, r.From, r.To, r.Strategy, len(r.Changes), len(affected))
	for _, p := range affected {
		fmt.Fprintf(&b, "%s\n", p.ID)
		for _, c := range p.Changes {
			fmt.Fprintf(&b, "  %s\n", c)
		}
	}
	return b.String()
}
