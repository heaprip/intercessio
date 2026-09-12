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
	Preset  Preset
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
	decided, nonLiquet, expired, amendments, bearers := 0, 0, 0, 0, 0
	var concerned []string
	for _, e := range in.Journal.Entries {
		if e.Period < in.From || e.Period > in.To {
			continue
		}
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
		case journal.Amendment:
			amendments++
		case journal.PeriodSummary:
			if e.Period == in.To {
				var v, c, b int
				if n, _ := fmt.Sscanf(e.Subject, "violations=%d checked=%d bearers=%d", &v, &c, &b); n == 3 {
					bearers = b
				}
			}
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
		Axis{"enforcement coverage", ratio(capacity, bearers), fmt.Sprintf("capacity %d/bearers %d", capacity, bearers)},
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
	return st
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
