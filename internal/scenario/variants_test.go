package scenario

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	d "github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/period"
)

// Variant tests change one input of a casus and check that the rules still
// answer sensibly for a neighbour the story did not cover. They test rules, not
// procedure: procedural facts are manual until case exists.

type world = map[string]any

func loadVariant(t *testing.T, dir string, mutate func(world)) *Scenario {
	t.Helper()
	schema, err := os.ReadFile(filepath.Join("testdata", "schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join("testdata", dir, "scenario.json"))
	if err != nil {
		t.Fatal(err)
	}
	var w world
	if err := json.Unmarshal(raw, &w); err != nil {
		t.Fatal(err)
	}
	mutate(w)
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	s, rep, err := Load(fstest.MapFS{"schema.json": {Data: schema}, "scenario.json": {Data: data}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Errors) > 0 {
		t.Fatalf("variant has errors:\n%s", rep)
	}
	return s
}

func list(w world, key string) []any { return w[key].([]any) }

func setPersonStatus(id, status string) func(world) {
	return func(w world) {
		for _, p := range list(w, "people") {
			if m := p.(map[string]any); m["id"] == id {
				m["status"] = status
			}
		}
	}
}

func dropFact(id string) func(world) {
	return func(w world) {
		var kept []any
		for _, f := range list(w, "facts") {
			if f.(map[string]any)["id"] != id {
				kept = append(kept, f)
			}
		}
		w["facts"] = kept
	}
}

func addFacts(facts ...map[string]any) func(world) {
	return func(w world) {
		for _, f := range facts {
			w["facts"] = append(list(w, "facts"), f)
		}
	}
}

func TestVariants(t *testing.T) {
	tests := []struct {
		name   string
		dir    string
		mutate func(world)
		cases  []derivation
	}{
		{
			name:   "adopted by a patrician keeps patrician status",
			dir:    "office-eligibility",
			mutate: setPersonStatus("fonteius", "patricius"),
			cases: []derivation{
				{12, d.A("status", "clodius", "patricius"), d.Proved, ""},
				// adoption strips every former status; nothing grants plebeius
				{12, d.A("status", "clodius", "plebeius"), d.Disproved, ""},
				{12, d.A("eligible", "clodius", "tribunus"), d.Unknown, d.Gap},
			},
		},
		{
			name:   "adoption not yet in force changes nothing",
			dir:    "office-eligibility",
			mutate: dropFact("a1_in_force"),
			cases: []derivation{
				{12, d.A("status", "clodius", "patricius"), d.Proved, ""},
				{12, d.A("eligible", "clodius", "tribunus"), d.Unknown, d.Gap},
			},
		},
		{
			name: "incompatibility holds both ways",
			dir:  "office-eligibility",
			mutate: addFacts(map[string]any{
				"id": "occ_fonteius", "p": "occupies", "a": []any{"fonteius", "censor", 40, 41}, "by": "scenario",
			}),
			cases: []derivation{
				{40, d.A("eligible", "fonteius", "consul"), d.Disproved, ""},
			},
		},
		{
			name: "exemption lifts the duty of one citizen only",
			dir:  "citizenship",
			mutate: addFacts(
				map[string]any{"id": "e1", "p": "exempt", "a": []any{"e1", "marcus", "munus"}, "by": "praetor", "period": 31},
				map[string]any{"id": "e1_in_force", "p": "in_force", "a": []any{"e1", 32}, "by": "scenario", "period": 32},
			),
			cases: []derivation{
				// the defeater blocks the duty; the violation then has nothing to stand on
				{32, d.A("duty", "marcus", "res_publica", "munus", "none", "maintain", "none"), d.Unknown, d.Blocked},
				{32, d.A("violated", "marcus", "munus", 32), d.Unknown, d.Gap},
				{32, d.A("violated", "gaius", "munus", 32), d.Proved, ""},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := loadVariant(t, tt.dir, tt.mutate)
			for _, c := range tt.cases {
				res, err := d.Evaluate(s.Program(period.Period(c.now)), c.now)
				if err != nil {
					t.Fatal(err)
				}
				if got := res.Query(c.query); got.Outcome != c.outcome || got.Reason != c.reason {
					t.Errorf("period %d, %s: got %s/%s, want %s/%s", c.now, c.query, got.Outcome, got.Reason, c.outcome, c.reason)
				}
			}
		})
	}
}
