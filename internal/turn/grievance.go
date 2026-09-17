package turn

import (
	"sort"
	"strings"

	"github.com/heaprip/intercessio/internal/deduction"

	"github.com/heaprip/intercessio/internal/corpus"
	"github.com/heaprip/intercessio/internal/impact"
	"github.com/heaprip/intercessio/internal/period"
)

// Amendment is one change the auctor makes in the amendment phase.
type Amendment struct {
	Enact  *corpus.Norm // a new norm, or a new text of a norm with the same id
	Repeal string
	// Order declares which rule beats which.
	Order []deduction.Defeat
}

// Subject names the amendment for the journal.
func (a Amendment) Subject() string {
	var parts []string
	if a.Repeal != "" {
		parts = append(parts, "repeal "+a.Repeal)
	}
	if a.Enact != nil {
		parts = append(parts, "enact "+a.Enact.ID)
	}
	for _, d := range a.Order {
		parts = append(parts, "order "+d.Over+" > "+d.Under)
	}
	return strings.Join(parts, "; ")
}

// Apply makes the version an amendment leads to; it is apply, exported for
// those who preview an amendment before the auctor accepts it.
func Apply(v corpus.Version, ams []Amendment) corpus.Version { return apply(v, ams) }

// Auctor gives the amendments of a period. Prototype: a script; the stack of
// proposals comes later.
type Auctor interface {
	Amendments(now period.Period, v corpus.Version) []Amendment
}

// Script is the scripted auctor: amendments by period.
type Script map[period.Period][]Amendment

// Amendments returns the scripted amendments of the period.
func (s Script) Amendments(now period.Period, _ corpus.Version) []Amendment { return s[now] }

// apply makes the new version: repeal, then enact or amend.
func apply(v corpus.Version, ams []Amendment) corpus.Version {
	for _, a := range ams {
		if a.Repeal != "" {
			v = v.Without(a.Repeal)
		}
		if a.Enact != nil {
			if _, ok := v.Norm(a.Enact.ID); ok {
				v = v.Amend(*a.Enact)
			} else {
				v = v.With(*a.Enact)
			}
		}
		if len(a.Order) > 0 {
			v = v.Order(a.Order)
		}
	}
	return v
}

// Grievance is a person an amendment took something from, waiting in the queue
// of private cases.
type Grievance struct {
	Person string
	// Matter is the status the person petitions for; empty when no remedy fits
	// what was lost.
	Matter string
	Weight int
	Since  period.Period
	Loss   string
}

// Weights of losses: a status outweighs an entitlement, which outweighs a
// single right. An ordinary petition weighs as an entitlement.
const (
	weightStatus   = 3
	weightEntitled = 2
	weightRight    = 1
	weightPetition = weightEntitled
)

// grievances turns a recount into at most one grievance per person: the
// heaviest thing they lost.
func grievances(imp impact.Report, people []string, now period.Period) []Grievance {
	isPerson := map[string]bool{}
	for _, p := range people {
		isPerson[p] = true
	}
	by := map[string]Grievance{}
	for _, c := range imp.Changes {
		a := c.Atom
		if c.Gained || a.Neg || len(a.Args) < 2 || !isPerson[a.Args[0].Const] {
			continue
		}
		g := Grievance{Person: a.Args[0].Const, Since: now, Loss: a.String()}
		switch a.Pred {
		case "status":
			g.Weight, g.Matter = weightStatus, a.Args[1].Const
		case "entitled":
			g.Weight, g.Matter = weightEntitled, a.Args[1].Const
		case "holds_right":
			g.Weight = weightRight
		default:
			continue
		}
		if old, ok := by[g.Person]; !ok || g.Weight > old.Weight {
			by[g.Person] = g
		}
	}
	out := make([]Grievance, 0, len(by))
	for _, g := range by {
		out = append(out, g)
	}
	sortGrievances(out)
	return out
}

func sortGrievances(gs []Grievance) {
	sort.SliceStable(gs, func(i, j int) bool {
		a, b := gs[i], gs[j]
		if a.Weight != b.Weight {
			return a.Weight > b.Weight
		}
		if a.Since != b.Since {
			return a.Since < b.Since
		}
		return a.Person < b.Person
	})
}
