package scenario

import (
	"testing"

	"github.com/heaprip/intercessio/internal/corpus"
	d "github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/entitlement"
	"github.com/heaprip/intercessio/internal/period"
)

type derivation struct {
	now     int
	query   d.Atom
	outcome d.Outcome
	reason  d.Reason
}

func expect(t *testing.T, s *Scenario, strat entitlement.Strategy, v corpus.Version, cases []derivation) {
	t.Helper()
	for _, c := range cases {
		res, err := entitlement.Resolve(strat, v, s.Facts, period.Period(c.now))
		if err != nil {
			t.Fatal(err)
		}
		got := res.Query(c.query)
		if got.Outcome != c.outcome || got.Reason != c.reason {
			t.Errorf("%s, period %d, %s: got %s/%s, want %s/%s", strat.Name(), c.now, c.query, got.Outcome, got.Reason, c.outcome, c.reason)
			if got.Trace != nil {
				t.Logf("trace:\n%s", res.Explain(got.Trace))
			}
		}
	}
}

func checkDerivations(t *testing.T, dir string, cases []derivation) {
	t.Helper()
	s, rep := loadScenario(t, dir)
	if len(rep.Errors) > 0 {
		t.Fatalf("scenario has errors:\n%s", rep)
	}
	expect(t, s, entitlement.Hierarchy{}, s.Corpus, cases)
}

// The casus walk-through: the admission rule is satisfied by construction.
func TestDerive_OfficeEligibility(t *testing.T) {
	checkDerivations(t, "office-eligibility", officeEligibility)
}

var officeEligibility = []derivation{
	{10, d.A("eligible", "clodius", "tribunus"), d.Unknown, d.Gap},
	{11, d.A("status", "clodius", "plebeius"), d.Unknown, d.Gap},
	{12, d.A("status", "clodius", "plebeius"), d.Proved, ""},
	{12, d.A("holds_right", "clodius", "ius_tribunicium"), d.Proved, ""},
	{12, d.A("eligible", "clodius", "tribunus"), d.Proved, ""},
	// adoption strips every former status, the adopter's status beats that
	{12, d.A("status", "clodius", "patricius"), d.Disproved, ""},
	{12, d.A("eligible", "clodius", "censor"), d.Disproved, ""},
	{12, d.A("eligible", "clodius", "consul"), d.Disproved, ""},
	{13, d.A("lawful_tenure", "clodius", "tribunus"), d.Proved, ""},
	{13, d.A("usurped", "clodius", "tribunus"), d.Unknown, d.Gap},
}

func TestDerive_Citizenship(t *testing.T) {
	checkDerivations(t, "citizenship", citizenship)
}

var citizenship = []derivation{
	{30, d.A("tenure", "marcus", 25), d.Proved, ""},
	{30, d.A("entitled", "marcus", "civis"), d.Proved, ""},
	{30, d.A("may_grant", "praetor", "marcus", "civis"), d.Proved, ""},
	{29, d.A("entitled", "marcus", "civis"), d.Unknown, d.Gap},
	{30, d.A("duty", "praetor", "marcus", "decide", "p1", "achieve", 33), d.Proved, ""},
	{31, d.A("status", "marcus", "civis"), d.Unknown, d.Gap},
	{32, d.A("status", "marcus", "civis"), d.Proved, ""},
	{32, d.A("status", "marcus", "peregrinus"), d.Disproved, ""},
	{31, d.A("status", "marcus", "peregrinus"), d.Proved, ""},
	{32, d.A("duty", "praetor", "marcus", "decide", "p1", "achieve", 33), d.Unknown, d.Gap},
	{32, d.A("violated", "marcus", "munus", 32), d.Proved, ""},
	// the penalty follows a decision in force, not the derived violation
	{32, d.A("duty", "marcus", "res_publica", "penalty", "munus", "achieve", 33), d.Unknown, d.Gap},
	{32, d.A("competent", "censor", "grant_status"), d.Disproved, ""},
}

var strategies = []entitlement.Strategy{entitlement.Hierarchy{}, entitlement.LexPosterior{}}

// Gate of derived-state: the casus passes on a substituted resolution strategy,
// and its consumers do not change.
func TestSubstitutability(t *testing.T) {
	for _, strat := range strategies {
		t.Run(strat.Name(), func(t *testing.T) {
			for dir, cases := range map[string][]derivation{"citizenship": citizenship, "office-eligibility": officeEligibility} {
				s, _ := loadScenario(t, dir)
				expect(t, s, strat, s.Corpus, cases)
			}
		})
	}
}

// Demonstration of derived-state: the strip norm N5 is blocked while A4
// protects the vested; removing A4 from the corpus takes the status away.
func TestDerive_RemovingA4TakesStatus(t *testing.T) {
	for _, strat := range strategies {
		t.Run(strat.Name(), func(t *testing.T) {
			s := loadVariant(t, "citizenship", moveStripToLaterNorm)
			civis := d.A("status", "marcus", "civis")
			expect(t, s, strat, s.Corpus, []derivation{
				{39, civis, d.Proved, ""},
				{40, civis, d.Unknown, d.Blocked},
				{40, d.A("status", "gaius", "civis"), d.Proved, ""},
			})
			expect(t, s, strat, s.Corpus.Without("A4"), []derivation{
				{40, civis, d.Disproved, ""},
				{40, d.A("status", "gaius", "civis"), d.Proved, ""},
			})
		})
	}
}

// moveStripToLaterNorm takes r_strip out of A1 into N5, a constitutional norm
// enacted in period 40 that strips marcus of civis.
func moveStripToLaterNorm(w world) {
	var strip any
	for _, n := range list(w, "norms") {
		norm := n.(map[string]any)
		if norm["id"] != "A1" {
			continue
		}
		var kept []any
		for _, r := range norm["rules"].([]any) {
			if r.(map[string]any)["id"] == "r_strip" {
				strip = r
				continue
			}
			kept = append(kept, r)
		}
		norm["rules"] = kept
	}
	w["norms"] = append(list(w, "norms"), map[string]any{
		"id": "N5", "document": "constitution", "enacted": 40, "in_force": 40, "kind": "lasting", "rules": []any{strip},
	})
	w["declarations"] = append(list(w, "declarations"),
		map[string]any{"p": "strips_vested", "a": []any{"n5", "civis"}, "period": 40},
		map[string]any{"p": "applies_to", "a": []any{"n5", "marcus"}, "period": 40},
	)
}
