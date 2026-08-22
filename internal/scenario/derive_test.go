package scenario

import (
	"testing"

	d "github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/period"
)

type derivation struct {
	now     int
	query   d.Atom
	outcome d.Outcome
	reason  d.Reason
}

func checkDerivations(t *testing.T, dir string, cases []derivation) {
	t.Helper()
	s, rep := loadScenario(t, dir)
	if len(rep.Errors) > 0 {
		t.Fatalf("scenario has errors:\n%s", rep)
	}
	for _, c := range cases {
		res, err := d.Evaluate(s.Program(period.Period(c.now)), c.now)
		if err != nil {
			t.Fatal(err)
		}
		got := res.Query(c.query)
		if got.Outcome != c.outcome || got.Reason != c.reason {
			t.Errorf("period %d, %s: got %s/%s, want %s/%s", c.now, c.query, got.Outcome, got.Reason, c.outcome, c.reason)
			if got.Trace != nil {
				t.Logf("trace:\n%s", got.Trace)
			}
		}
	}
}

// The casus walk-through: the admission rule is satisfied by construction.
func TestDerive_OfficeEligibility(t *testing.T) {
	checkDerivations(t, "office-eligibility", []derivation{
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
	})
}

func TestDerive_Citizenship(t *testing.T) {
	checkDerivations(t, "citizenship", []derivation{
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
		{32, d.A("duty", "marcus", "res_publica", "penalty", "munus", "achieve", 33), d.Proved, ""},
		{32, d.A("competent", "censor", "grant_status"), d.Disproved, ""},
	})
}
