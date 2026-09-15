package turn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/heaprip/intercessio/internal/actors"
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/entitlement"
	"github.com/heaprip/intercessio/internal/journal"
	"github.com/heaprip/intercessio/internal/scenario"
)

// eligibilityLived is the office-eligibility casus without the scripted
// adoption facts, with metellus holding the censorship.
func eligibilityLived(t *testing.T, mutate func(map[string]any)) State {
	t.Helper()
	schema, err := os.ReadFile(filepath.Join(testdata, "schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(testdata, "office-eligibility", "scenario.json"))
	if err != nil {
		t.Fatal(err)
	}
	var w map[string]any
	if err := json.Unmarshal(raw, &w); err != nil {
		t.Fatal(err)
	}
	var kept []any
	for _, f := range w["facts"].([]any) {
		switch f.(map[string]any)["id"] {
		case "a1", "a1_performed", "a1_in_force":
		default:
			kept = append(kept, f)
		}
	}
	w["facts"] = append(kept, map[string]any{"id": "occ_metellus", "p": "occupies", "a": []any{"metellus", "censor", 1, 99}, "by": "scenario"})
	w["people"] = append(w["people"].([]any), map[string]any{"id": "metellus", "status": "patricius", "born": 1})
	if mutate != nil {
		mutate(w)
	}
	if raw, err = json.Marshal(w); err != nil {
		t.Fatal(err)
	}
	s, rep, err := scenario.Load(fstest.MapFS{"schema.json": {Data: schema}, "scenario.json": {Data: raw}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Errors) > 0 {
		t.Fatalf("scenario has errors:\n%s", rep)
	}
	return State{Period: s.StartPeriod, Corpus: s.Corpus, Facts: s.Facts, Seed: 7}
}

func adoption(actor, office string) Config {
	return Config{Strategy: entitlement.Hierarchy{}, Actors: actors.Stub{Script: []actors.Attempt{
		{Period: 11, Actor: actor, Office: office, Kind: "adrogatio", Subject: "clodius", Fact: []string{"adrogatio", "clodius", "fonteius"}},
	}}}
}

// Demonstration of the act: the adoption of Clodius is lived, not written in as
// facts. The censor performs it in 11, it enters into force in 12, and from 12
// Clodius is a plebeian eligible for the tribunate.
func TestAdvance_AdoptionIsAnAct(t *testing.T) {
	s := play(t, eligibilityLived(t, nil), adoption("metellus", "censor"), 3)
	if !has(s.Journal, journal.Act, 11, "actor=metellus office=censor") || !has(s.Journal, journal.Finalization, 12, "act11_1") {
		dump(t, s.Journal)
		t.Fatal("the adoption must be performed in 11 and enter into force in 12")
	}
	for now, want := range map[int]deduction.Outcome{11: deduction.Unknown, 12: deduction.Proved} {
		if got := query(t, s, now, deduction.A("eligible", "clodius", "tribunus")).Outcome; got != want {
			t.Fatalf("eligible in %d: got %s, want %s", now, got, want)
		}
	}
	if got := query(t, s, 12, deduction.A("valid", "act11_1")).Outcome; got != deduction.Proved {
		t.Fatalf("the act of a competent office is valid: got %s", got)
	}
}

// Controls: a tribune may not adopt, and a censorship requiring two concurring
// holders cannot act with one.
func TestAdvance_ActNeedsCompetenceAndQuorum(t *testing.T) {
	s := play(t, eligibilityLived(t, nil), adoption("metellus", "tribunus"), 3)
	if !has(s.Journal, journal.UltraVires, 11, "office=tribunus subject=adrogatio") || len(s.Acts) != 0 {
		dump(t, s.Journal)
		t.Fatal("the tribune's adoption must be ultra vires and leave no act")
	}
	quorum := func(w map[string]any) {
		for _, o := range w["offices"].([]any) {
			if m := o.(map[string]any); m["id"] == "censor" {
				m["quorum"] = 2
			}
		}
	}
	s = play(t, eligibilityLived(t, quorum), adoption("metellus", "censor"), 3)
	if !has(s.Journal, journal.NoQuorum, 11, "1 of 2 holders") {
		dump(t, s.Journal)
		t.Fatal("one censor of two required must not act")
	}
	if got := query(t, s, 12, deduction.A("eligible", "clodius", "tribunus")).Outcome; got == deduction.Proved {
		t.Fatal("no act, no change of status")
	}
}
