// Package competence answers whether an office may perform an action of a
// kind. It owns nothing but the check: competences are norms of the corpus.
package competence

import "github.com/heaprip/intercessio/internal/deduction"

// Verdict of a check.
type Verdict string

const (
	Allowed Verdict = "allowed"
	Denied  Verdict = "denied"
	Unclear Verdict = "unclear"
)

// Query asks the resolved slice.
type Query func(deduction.Atom) deduction.Answer

// Check asks competent(office, kind) and returns the verdict with the rule it
// rested on.
func Check(q Query, office, kind string) (Verdict, string) {
	ans := q(deduction.A("competent", office, kind))
	rule := ""
	if ans.Trace != nil {
		rule = ans.Trace.Rule
	}
	switch ans.Outcome {
	case deduction.Proved:
		return Allowed, rule
	case deduction.Disproved:
		return Denied, rule
	}
	return Unclear, rule
}
