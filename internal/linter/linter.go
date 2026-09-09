// Package linter checks a corpus before a turn and speaks the language of the
// failure catalog. The player-facing Finding lives here; it is built from the
// domain-free diagnostics of deduction and scenario, from impact and from the
// power graph.
//
// Prototype. The Finding carries a catalog slug, a channel, a place, a message
// and the people it names; measure, threshold and window are not there yet.
package linter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/heaprip/intercessio/internal/corpus"
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/entitlement"
	"github.com/heaprip/intercessio/internal/facts"
	"github.com/heaprip/intercessio/internal/impact"
	"github.com/heaprip/intercessio/internal/period"
	"github.com/heaprip/intercessio/internal/powergraph"
)

// Channel says what detected a finding, as in the catalog.
type Channel string

const (
	// Linter: a property of what is written.
	Linter Channel = "linter"
	// Recount: the subtraction of two runs over two versions of the corpus.
	Recount Channel = "recount"
)

// Finding is one remark for the player.
type Finding struct {
	// Failure is the catalog slug. Empty for a remark about the form of the
	// corpus that the catalog does not name.
	Failure string
	// Code is the check or diagnostic that produced it.
	Code    string
	Channel Channel
	Place   string
	Message string
	People  []string
}

func (f Finding) String() string {
	name := f.Failure
	if name == "" {
		name = "(form) " + f.Code
	}
	s := fmt.Sprintf("%-26s %-7s %-22s %s", name, f.Channel, f.Place, f.Message)
	if len(f.People) > 0 {
		s += " [" + strings.Join(f.People, ", ") + "]"
	}
	return s
}

// failureOf maps diagnostic codes to catalog slugs. A code missing here is a
// form remark.
var failureOf = map[string]string{
	"underdetermined-condition": "underdetermined-condition",
	"negation-never-concluded":  "underdetermined-condition",
	"unconsumed-conclusion":     "dead-conclusion",
	"unconsumed-right":          "empty-right",
	"unordered-conflict":        "indeterminacy",
}

// Input is what one lint needs.
type Input struct {
	Corpus   corpus.Version
	Roles    map[string]deduction.Role
	Facts    []facts.Fact
	Strategy entitlement.Strategy
	Now      period.Period
	// Loader holds the scenario loader's diagnostics, if the corpus was loaded.
	Loader []deduction.Diagnostic
	// Before is the version the corpus was amended from; nil when there is no
	// amendment and nothing to recount.
	Before *corpus.Version
}

// Report of one lint.
type Report struct {
	Errors   []deduction.Diagnostic
	Findings []Finding
	Impact   *impact.Report
	Graph    *powergraph.Graph
	People   []string
}

// Lint runs every check. Errors stop it before evaluation: a corpus with errors
// cannot be resolved, and the checks that need a resolution are skipped.
func Lint(in Input) (*Report, error) {
	rep := &Report{}
	seen := map[string]bool{}
	var ds []deduction.Diagnostic
	add := func(d deduction.Diagnostic) {
		if d.Code == "retroactivity" {
			return // recounted below
		}
		k := d.Code + "|" + d.Subject + "|" + d.Message
		if !seen[k] {
			seen[k] = true
			ds = append(ds, d)
		}
	}
	for _, d := range in.Loader {
		add(d)
	}
	// The slice is checked with the pairs among its own rules: a pair naming a
	// norm not yet in force is not an error of this period, and a pair naming no
	// rule at all is already the loader's error.
	slice := in.Corpus.At(in.Now)
	inSlice := map[string]bool{}
	for _, r := range slice {
		inSlice[r.ID] = true
	}
	var pairs []deduction.Defeat
	for _, p := range in.Corpus.Defeats {
		if inSlice[p.Over] && inSlice[p.Under] {
			pairs = append(pairs, p)
		}
	}
	for _, d := range deduction.Check(slice, pairs, in.Roles) {
		add(d)
	}
	deduction.Sort(ds)
	for _, d := range ds {
		if d.Severity == deduction.Error {
			rep.Errors = append(rep.Errors, d)
			continue
		}
		rep.Findings = append(rep.Findings, Finding{Failure: failureOf[d.Code], Code: d.Code, Channel: Linter, Place: d.Subject, Message: d.Message})
	}
	if len(rep.Errors) > 0 {
		return rep, nil
	}
	rep.People = powergraph.Domain(powergraph.Stored(in.Corpus, in.Facts, in.Now), "person")

	changed := map[string]bool{}
	if in.Before != nil {
		changed = changedNorms(*in.Before, in.Corpus)
		from := in.Now
		for _, n := range in.Corpus.Norms {
			if n.Until == 0 && changed[n.ID] && n.InForce < from {
				from = n.InForce
			}
		}
		imp, err := impact.Compute(in.Strategy, *in.Before, in.Corpus, in.Facts, from, in.Now)
		if err != nil {
			return nil, err
		}
		rep.Impact = &imp
	}
	rep.retroactivity(in, changed)
	rep.takingOfVested(in)

	g, err := powergraph.Build(in.Strategy, in.Corpus, in.Facts, in.Now)
	if err != nil {
		return nil, err
	}
	rep.Graph = g
	rep.reviewDeadEnd()
	rep.competenceGap(in)
	rep.circumventable()
	rep.closedEligibility()

	sort.SliceStable(rep.Findings, func(i, j int) bool {
		a, b := rep.Findings[i], rep.Findings[j]
		if (a.Failure == "") != (b.Failure == "") {
			return a.Failure != ""
		}
		if a.Failure != b.Failure {
			return a.Failure < b.Failure
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Place < b.Place
	})
	return rep, nil
}

// changedNorms lists the ids whose open text was added, amended or removed.
func changedNorms(before, after corpus.Version) map[string]bool {
	out := map[string]bool{}
	sig := func(n corpus.Norm) string {
		s := fmt.Sprint(n.Document, n.Enacted, n.InForce, n.Kind)
		for _, r := range n.Rules {
			s += "|" + r.ID + ":" + r.String()
		}
		for _, a := range n.Declarations {
			s += "|" + a.String()
		}
		return s
	}
	for _, n := range after.Norms {
		if n.Until != 0 {
			continue
		}
		if old, ok := before.Norm(n.ID); !ok || sig(old) != sig(n) {
			out[n.ID] = true
		}
	}
	for _, n := range before.Norms {
		if _, ok := after.Norm(n.ID); n.Until == 0 && !ok {
			out[n.ID] = true
		}
	}
	return out
}

func (r *Report) retroactivity(in Input, changed map[string]bool) {
	for _, n := range in.Corpus.Norms {
		if n.Until != 0 || !n.Retroactive() {
			continue
		}
		f := Finding{Failure: "retroactivity", Code: "retroactivity", Channel: Linter, Place: n.ID,
			Message: fmt.Sprintf("enacted in %d, applies from %d", n.Enacted, n.InForce)}
		if r.Impact != nil && changed[n.ID] {
			window := impact.Report{Changes: nil}
			for _, c := range r.Impact.Changes {
				if c.Period >= n.InForce && c.Period < n.Enacted {
					window.Changes = append(window.Changes, c)
				}
			}
			f.Channel = Recount
			for _, p := range window.Affected(r.People) {
				f.People = append(f.People, p.ID)
			}
			f.Message += fmt.Sprintf("; recount over %d..%d changes %d conclusions", n.InForce, n.Enacted-1, len(window.Changes))
		}
		r.Findings = append(r.Findings, f)
	}
}

// takingOfVested reports statuses the amendment takes away in the current
// period: a status proved before and no longer, or its absence newly proved.
func (r *Report) takingOfVested(in Input) {
	if r.Impact == nil {
		return
	}
	var taken []string
	people := map[string]bool{}
	for _, c := range r.Impact.Changes {
		if c.Period != in.Now || c.Atom.Pred != "status" || c.Gained == !c.Atom.Neg {
			continue
		}
		taken = append(taken, c.String())
		for _, p := range r.People {
			if len(c.Atom.Args) > 0 && c.Atom.Args[0].Const == p {
				people[p] = true
			}
		}
	}
	if len(taken) == 0 {
		return
	}
	f := Finding{Failure: "taking-of-vested", Code: "taking-of-vested", Channel: Recount, Place: "status",
		Message: "status taken: " + strings.Join(taken, "; ")}
	for p := range people {
		f.People = append(f.People, p)
	}
	sort.Strings(f.People)
	r.Findings = append(r.Findings, f)
}

func (r *Report) reviewDeadEnd() {
	for _, o := range r.Graph.Offices {
		if len(o.Competences) == 0 {
			continue
		}
		active := false
		var vacant []string
		for _, e := range r.Graph.Find(powergraph.Restrains, "", o.ID) {
			if e.Active {
				active = true
			} else if !contains(vacant, e.From) {
				vacant = append(vacant, e.From)
			}
		}
		if active {
			continue
		}
		msg := fmt.Sprintf("no office in effect can stop %s (%s)", o.ID, strings.Join(o.Competences, ", "))
		if len(vacant) > 0 {
			msg += "; declared: " + strings.Join(vacant, ", ") + ", vacant"
		}
		r.Findings = append(r.Findings, Finding{Failure: "review-dead-end", Code: "review-dead-end", Channel: Linter, Place: o.ID, Message: msg, People: o.Holders})
	}
}

// competenceGap: an action the corpus requires and no office may perform.
// Required means an action kind named in a competent/2 condition, or an act
// predicate some condition rests on — the act needs any one of its kinds, since
// the linter cannot tell which kind a given condition means.
func (r *Report) competenceGap(in Input) {
	acts := map[string][]string{}
	for _, a := range in.Corpus.Acts {
		acts[a.Predicate] = a.Kinds
	}
	required := map[string][]string{} // place -> rules
	anyOf := map[string][]string{}    // place -> kinds, any one suffices
	for _, rule := range in.Corpus.At(in.Now) {
		for _, l := range rule.Body {
			if l.Pred == "competent" && len(l.Args) == 2 && l.Args[1].Kind == deduction.Const {
				k := l.Args[1].Name
				required[k] = appendNew(required[k], rule.ID)
				anyOf[k] = []string{k}
			}
			if kinds, ok := acts[l.Pred]; ok {
				required[l.Pred] = appendNew(required[l.Pred], rule.ID)
				anyOf[l.Pred] = kinds
			}
		}
	}
	var places []string
	for p := range required {
		places = append(places, p)
	}
	sort.Strings(places)
	for _, p := range places {
		covered := false
		for _, k := range anyOf[p] {
			if len(r.Graph.Find(powergraph.Competence, "", k)) > 0 {
				covered = true
			}
		}
		if covered {
			continue
		}
		r.Findings = append(r.Findings, Finding{Failure: "competence-gap", Code: "competence-gap", Channel: Linter, Place: p,
			Message: fmt.Sprintf("%s is required by %s and no office may perform %s", p, strings.Join(required[p], ", "), strings.Join(anyOf[p], " or "))})
	}
}

// circumventable: admission rests on an act whose office nothing in effect can
// stop. The rule is then satisfied by construction rather than broken.
func (r *Report) circumventable() {
	done := map[string]bool{}
	for _, e := range r.Graph.Find(powergraph.Admits, "", "") {
		key := e.From + "|" + e.Via
		if done[key] || r.restrained(e.From) {
			continue
		}
		done[key] = true
		var opened []string
		for _, x := range r.Graph.Find(powergraph.Admits, e.From, "") {
			if x.Via == e.Via {
				opened = append(opened, x.To)
			}
		}
		r.Findings = append(r.Findings, Finding{Failure: "circumventable-condition", Code: "circumventable-condition", Channel: Linter, Place: "eligible",
			Message: fmt.Sprintf("admission to %s rests on %s, an act of %s that no office in effect can stop: %s",
				strings.Join(opened, ", "), e.Via, e.From, powergraph.PathString(e.Path)),
			People: r.Graph.Office(e.From).Holders})
	}
}

func (r *Report) restrained(office string) bool {
	for _, e := range r.Graph.Find(powergraph.Restrains, "", office) {
		if e.Active {
			return true
		}
	}
	return false
}

// closedEligibility: an office whose act opens admission, along admits edges,
// back to the office itself.
func (r *Report) closedEligibility() {
	for _, o := range r.Graph.Offices {
		path := r.admitsCycle(o.ID)
		if path == nil {
			continue
		}
		r.Findings = append(r.Findings, Finding{Failure: "closed-eligibility", Code: "closed-eligibility", Channel: Linter, Place: o.ID,
			Message: "an act of the office reopens admission to it: " + strings.Join(path, " -> "), People: o.Holders})
	}
}

func (r *Report) admitsCycle(start string) []string {
	type node struct {
		office string
		path   []string
	}
	seen := map[string]bool{}
	queue := []node{{start, []string{start}}}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		for _, e := range r.Graph.Find(powergraph.Admits, n.office, "") {
			path := append(append([]string{}, n.path...), fmt.Sprintf("(%s) %s", e.Via, e.To))
			if e.To == start {
				return path
			}
			if !seen[e.To] {
				seen[e.To] = true
				queue = append(queue, node{e.To, path})
			}
		}
	}
	return nil
}

// String renders errors and findings, one per line.
func (r *Report) String() string {
	var b strings.Builder
	for _, d := range r.Errors {
		fmt.Fprintf(&b, "error %-26s %-22s %s\n", d.Code, d.Subject, d.Message)
	}
	for _, f := range r.Findings {
		b.WriteString(f.String() + "\n")
	}
	return b.String()
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

func appendNew(xs []string, x string) []string {
	if contains(xs, x) {
		return xs
	}
	return append(xs, x)
}
