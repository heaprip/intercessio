// Package censor reads the journal and the power graph and shows where the
// construction is heading. It decides nothing and has no competence.
//
// Prototype: a few axes of the stand over a window of periods, and one journal
// failure, mass non liquet.
package censor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/heaprip/intercessio/internal/corpus"
	"github.com/heaprip/intercessio/internal/journal"
	"github.com/heaprip/intercessio/internal/linter"
	"github.com/heaprip/intercessio/internal/period"
	"github.com/heaprip/intercessio/internal/powergraph"
)

// Preset holds the thresholds of the stand.
type Preset struct {
	// NonLiquet is the share of answered cases ending non liquet above which
	// the window is a mass non liquet. Zero disables the finding.
	NonLiquet float64
	// Silent is how many periods a window must span before a norm none of whose
	// rules took part in any decision is reported. Zero disables the finding.
	Silent int
}

// Axis is one measure of the stand with what it was counted from.
type Axis struct {
	Name  string
	Value float64
	Of    string
}

func (a Axis) String() string { return fmt.Sprintf("%s %.2f (%s)", a.Name, a.Value, a.Of) }

// Stand is the result over a window.
type Stand struct {
	From, To period.Period
	Axes     []Axis
	Findings []linter.Finding
}

// Axis returns an axis by name.
func (s Stand) Axis(name string) (Axis, bool) {
	for _, a := range s.Axes {
		if a.Name == name {
			return a, true
		}
	}
	return Axis{}, false
}

func (s Stand) String() string {
	var parts []string
	for _, a := range s.Axes {
		parts = append(parts, a.String())
	}
	return fmt.Sprintf("stand %d..%d: %s", s.From, s.To, strings.Join(parts, "; "))
}

// Input is what the stand is counted from. Graph is the power graph of To.
type Input struct {
	Journal journal.Journal
	From    period.Period
	To      period.Period
	Graph   *powergraph.Graph
	Corpus  corpus.Version
	// People is the population at To: the denominator of enforcement coverage.
	People []string
	Preset Preset
}

func ratio(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

// Build counts the axes and the journal findings.
func Build(in Input) Stand {
	st := Stand{From: in.From, To: in.To}
	decided, nonLiquet, expired, amendments := 0, 0, 0, 0
	var concerned []string
	applied := map[string]bool{}
	reviewedBy := map[string]bool{}
	var usurpers []string
	var usurped []string
	for _, e := range in.Journal.Entries {
		if e.Period < in.From || e.Period > in.To {
			continue
		}
		for _, r := range strings.Split(e.Basis.Rules, ",") {
			applied[r] = true
		}
		applied[e.Basis.Rule] = true
		switch e.Kind {
		case journal.Decision:
			decided++
		case journal.NonLiquet:
			nonLiquet++
			if f := strings.Fields(e.Subject); len(f) > 0 && !contains(concerned, f[0]) {
				concerned = append(concerned, f[0])
			}
		case journal.Expired:
			expired++
		case journal.Review:
			reviewedBy[e.Office] = true
		case journal.Usurpation:
			usurpers = append(usurpers, e.Actor)
			usurped = append(usurped, fmt.Sprintf("%s in %s (%s)", e.Actor, e.Office, e.Subject))
		case journal.Amendment:
			amendments++
		}
	}
	answered := decided + nonLiquet
	share := ratio(nonLiquet, answered)
	st.Axes = append(st.Axes,
		Axis{"non-liquet share", share, fmt.Sprintf("%d/%d answered", nonLiquet, answered)},
		Axis{"expired share", ratio(expired, answered+expired), fmt.Sprintf("%d/%d closed", expired, answered+expired)},
	)
	if in.Graph != nil {
		st.Axes = append(st.Axes, vetoPlayers(in.Graph, in.Corpus))
	}
	capacity := 0
	for _, a := range in.Corpus.DeclarationsAt(in.To) {
		if a.Pred == "capacity" && len(a.Args) == 2 {
			capacity += a.Args[1].Num
		}
	}
	st.Axes = append(st.Axes,
		// per person, not per duty bearer: an amendment stripping citizens of
		// their duties must not make enforcement look better
		Axis{"enforcement coverage", ratio(capacity, len(in.People)), fmt.Sprintf("capacity %d/people %d", capacity, len(in.People))},
		Axis{"amendments", float64(amendments), "in the window"},
	)

	if in.Preset.NonLiquet > 0 && answered > 0 && share > in.Preset.NonLiquet {
		sort.Strings(concerned)
		st.Findings = append(st.Findings, linter.Finding{
			Failure: "mass-non-liquet", Code: "mass-non-liquet", Channel: linter.Journal,
			Place:   fmt.Sprintf("periods %d..%d", in.From, in.To),
			Message: fmt.Sprintf("%d of %d answered cases ended non liquet, above %.2f", nonLiquet, answered, in.Preset.NonLiquet),
			People:  concerned,
		})
	}
	if len(usurpers) > 0 {
		sort.Strings(usurpers)
		st.Findings = append(st.Findings, linter.Finding{
			Failure: "usurpation", Code: "usurpation", Channel: linter.Journal, Place: fmt.Sprintf("periods %d..%d", in.From, in.To),
			Message: "held without admission: " + strings.Join(usurped, "; "), People: usurpers,
		})
	}
	if in.Preset.Silent > 0 && int(in.To-in.From)+1 >= in.Preset.Silent {
		st.Findings = append(st.Findings, silentNorms(in, applied)...)
		st.Findings = append(st.Findings, unusedReview(in, reviewedBy)...)
	}
	return st
}

// silentNorms: norms in force through the window none of whose rules took part
// in any answer. Declarations-only norms are left out: they state data, and
// data is read, not applied.
func silentNorms(in Input, applied map[string]bool) []linter.Finding {
	var out []linter.Finding
	for _, n := range in.Corpus.Norms {
		if len(n.Rules) == 0 || !n.ActiveAt(in.From) || !n.ActiveAt(in.To) {
			continue
		}
		used := false
		var ids []string
		for _, r := range n.Rules {
			ids = append(ids, r.ID)
			used = used || applied[r.ID]
		}
		if used {
			continue
		}
		out = append(out, linter.Finding{
			Failure: "unapplied-norm", Code: "unapplied-norm", Channel: linter.Journal, Place: n.ID,
			Message: fmt.Sprintf("no rule of %s (%s) took part in any decision in periods %d..%d", n.ID, strings.Join(ids, ", "), in.From, in.To),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Place < out[j].Place })
	return out
}

// unusedReview: a path of review in effect at To that nobody took in the
// window. Formally correct and never used — the second silent failure.
func unusedReview(in Input, reviewedBy map[string]bool) []linter.Finding {
	if in.Graph == nil {
		return nil
	}
	var out []linter.Finding
	for _, e := range in.Graph.Find(powergraph.Restrains, "", "") {
		if e.Via != powergraph.ReviewKind || !e.Active || reviewedBy[e.From] {
			continue
		}
		out = append(out, linter.Finding{
			Failure: "unused-review-path", Code: "unused-review-path", Channel: linter.Journal, Place: e.From + " -> " + e.To,
			Message: fmt.Sprintf("%s may review %s, and nobody appealed in periods %d..%d", e.From, e.To, in.From, in.To),
		})
	}
	return out
}

// vetoPlayers counts, for every kind of decision some office may take, the
// holders who can stop it — people, not offices — and averages over kinds.
func vetoPlayers(g *powergraph.Graph, v corpus.Version) Axis {
	kinds := map[string]bool{}
	for _, a := range v.Acts {
		for _, k := range a.Outcomes {
			kinds[k] = true
		}
	}
	total, counted := 0, 0
	var names []string
	for k := range kinds {
		names = append(names, k)
	}
	sort.Strings(names)
	var detail []string
	for _, k := range names {
		deciders := g.Find(powergraph.Competence, "", k)
		if len(deciders) == 0 {
			continue
		}
		var players []string
		for _, d := range deciders {
			for _, e := range g.Find(powergraph.Restrains, "", d.From) {
				if !e.Active {
					continue
				}
				for _, h := range g.Office(e.From).Holders {
					if !contains(players, h) {
						players = append(players, h)
					}
				}
			}
		}
		total += len(players)
		counted++
		detail = append(detail, fmt.Sprintf("%s %d", k, len(players)))
	}
	return Axis{"veto players", ratio(total, counted), strings.Join(detail, ", ")}
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}
