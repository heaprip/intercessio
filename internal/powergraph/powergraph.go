// Package powergraph builds the power graph in effect in a period: offices with
// their holders, the action kinds they may perform, who can stop whom and whose
// act opens admission to which office.
//
// Prototype. The graph owns no data and is rebuilt for every period.
package powergraph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/heaprip/intercessio/internal/competence"
	"github.com/heaprip/intercessio/internal/corpus"
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/entitlement"
	"github.com/heaprip/intercessio/internal/facts"
	"github.com/heaprip/intercessio/internal/period"
)

// EdgeKind names an edge of the graph.
type EdgeKind string

const (
	// Competence: office may perform an action kind. To is the kind.
	Competence EdgeKind = "competence"
	// Restrains: From can stop the acts of To before they take effect.
	Restrains EdgeKind = "restrains"
	// Admits: an act of From changes what admission to To rests on. Derived by
	// walking the rules back from eligible, not declared.
	Admits EdgeKind = "admits"
)

// Via values of restraint edges that are not act predicates.
const (
	ReviewKind  = "review"
	Concurrence = "concurrence"
)

// Edge of the graph.
type Edge struct {
	Kind     EdgeKind
	From, To string
	Via      string // act predicate or action kind
	// Active is false for a declared restraint whose office has no holder in
	// the period: an empty office stops nobody.
	Active bool
	Path   []deduction.Step
}

// Office is a node.
type Office struct {
	ID          string
	Holders     []string
	Competences []string
}

// Graph is the power graph of one period.
type Graph struct {
	Period  period.Period
	Offices []Office
	Kinds   []string
	Edges   []Edge
}

// Build resolves the slice at now and builds the graph from it.
func Build(s entitlement.Strategy, v corpus.Version, fs []facts.Fact, now period.Period) (*Graph, error) {
	res, err := entitlement.Resolve(s, v, fs, now)
	if err != nil {
		return nil, fmt.Errorf("power graph at %d: %w", now, err)
	}
	stored := Stored(v, fs, now)
	g := &Graph{Period: now, Kinds: Domain(stored, "action_kind")}
	seen := map[string]bool{}
	add := func(e Edge) {
		k := fmt.Sprint(e.Kind, e.From, e.To, e.Via)
		if !seen[k] {
			seen[k] = true
			g.Edges = append(g.Edges, e)
		}
	}

	for _, o := range Domain(stored, "office") {
		off := Office{ID: o}
		for _, a := range stored {
			if a.Pred == "occupies" && len(a.Args) == 4 && a.Args[1].Const == o && a.Args[2].Num <= int(now) && int(now) <= a.Args[3].Num {
				off.Holders = append(off.Holders, a.Args[0].Const)
			}
		}
		sort.Strings(off.Holders)
		for _, k := range g.Kinds {
			if verdict, _ := competence.Check(res.Query, o, k); verdict == competence.Allowed {
				off.Competences = append(off.Competences, k)
				add(Edge{Kind: Competence, From: o, To: k, Via: k, Active: true})
			}
		}
		g.Offices = append(g.Offices, off)
	}

	for _, act := range v.Acts {
		for _, to := range g.holdersOf(act.Kinds) {
			for _, from := range g.holdersOf(act.StoppedBy) {
				if from == to {
					continue
				}
				add(Edge{Kind: Restrains, From: from, To: to, Via: act.Predicate, Active: len(g.Office(from).Holders) > 0})
			}
		}
	}

	// review: a declared reviewer with the competence to review checks every act
	// of the reviewed office anew
	for _, a := range stored {
		if a.Pred != "reviews" || len(a.Args) != 2 {
			continue
		}
		from, to := a.Args[0].Const, a.Args[1].Const
		if from == to || !contains(g.Office(from).Competences, ReviewKind) {
			continue
		}
		add(Edge{Kind: Restrains, From: from, To: to, Via: ReviewKind, Active: len(g.Office(from).Holders) > 0})
	}
	// concurrence: an act needs the consent of several holders, so a colleague
	// restrains the office from within; it acts only with a second holder there
	for _, a := range stored {
		if a.Pred != "quorum" || len(a.Args) != 2 || a.Args[1].Num < 2 {
			continue
		}
		o := a.Args[0].Const
		add(Edge{Kind: Restrains, From: o, To: o, Via: Concurrence, Active: len(g.Office(o).Holders) >= 2})
	}

	var guarded []string
	for _, a := range stored {
		if a.Pred == "requires_right" && len(a.Args) == 2 {
			guarded = append(guarded, a.Args[0].Const)
		}
	}
	sort.Strings(guarded)
	supports := deduction.Supports(v.At(now), "eligible")
	for _, act := range v.Acts {
		path, ok := supports[act.Predicate]
		if !ok {
			continue
		}
		for _, from := range g.holdersOf(act.Kinds) {
			for _, to := range guarded {
				add(Edge{Kind: Admits, From: from, To: to, Via: act.Predicate, Active: true, Path: path})
			}
		}
	}
	return g, nil
}

// holdersOf lists the offices competent for any of the kinds.
func (g *Graph) holdersOf(kinds []string) []string {
	var out []string
	for _, o := range g.Offices {
		for _, k := range o.Competences {
			if contains(kinds, k) {
				out = append(out, o.ID)
				break
			}
		}
	}
	return out
}

// Office returns the node by id.
func (g *Graph) Office(id string) Office {
	for _, o := range g.Offices {
		if o.ID == id {
			return o
		}
	}
	return Office{ID: id}
}

// Find lists edges of a kind; an empty from or to matches any.
func (g *Graph) Find(kind EdgeKind, from, to string) []Edge {
	var out []Edge
	for _, e := range g.Edges {
		if e.Kind == kind && (from == "" || e.From == from) && (to == "" || e.To == to) {
			out = append(out, e)
		}
	}
	return out
}

func (g *Graph) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "power graph, period %d\n", g.Period)
	for _, o := range g.Offices {
		holders := "vacant"
		if len(o.Holders) > 0 {
			holders = strings.Join(o.Holders, ", ")
		}
		fmt.Fprintf(&b, "office %-10s %-16s may %s\n", o.ID, "["+holders+"]", strings.Join(o.Competences, ", "))
	}
	for _, e := range g.Edges {
		if e.Kind == Competence {
			continue
		}
		note := ""
		if !e.Active {
			note = " (vacant)"
		}
		if len(e.Path) > 0 {
			note += ": " + PathString(e.Path)
		}
		fmt.Fprintf(&b, "%-9s %s -> %s via %s%s\n", e.Kind, e.From, e.To, e.Via, note)
	}
	return b.String()
}

// PathString renders a support path from the condition down.
func PathString(path []deduction.Step) string {
	var b strings.Builder
	for i, s := range path {
		b.WriteString(s.Pred)
		if i < len(path)-1 {
			fmt.Fprintf(&b, " <-%s- ", s.Rule)
		}
	}
	return b.String()
}

// Stored returns the facts and declarations visible at now as atoms.
func Stored(v corpus.Version, fs []facts.Fact, now period.Period) []deduction.Atom {
	out := v.DeclarationsAt(now)
	for _, f := range facts.Snapshot(fs, now) {
		a := deduction.Atom{Pred: f.Pred}
		for _, x := range f.Args {
			a.Args = append(a.Args, deduction.Value{Const: x.Const, Num: x.Num, IsNum: x.IsNum})
		}
		out = append(out, a)
	}
	return out
}

// Domain lists the sorted values of a unary domain predicate.
func Domain(stored []deduction.Atom, pred string) []string {
	set := map[string]bool{}
	for _, a := range stored {
		if a.Pred == pred && len(a.Args) == 1 {
			set[a.Args[0].Const] = true
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}
