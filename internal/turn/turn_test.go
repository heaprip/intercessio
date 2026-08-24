package turn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/heaprip/intercessio/internal/actors"
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/entitlement"
	"github.com/heaprip/intercessio/internal/journal"
	"github.com/heaprip/intercessio/internal/period"
	"github.com/heaprip/intercessio/internal/scenario"
)

const testdata = "../scenario/testdata"

func start(t *testing.T, mutate func(map[string]any)) State {
	t.Helper()
	schema, err := os.ReadFile(filepath.Join(testdata, "schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(testdata, "playable", "scenario.json"))
	if err != nil {
		t.Fatal(err)
	}
	if mutate != nil {
		var w map[string]any
		if err := json.Unmarshal(raw, &w); err != nil {
			t.Fatal(err)
		}
		mutate(w)
		if raw, err = json.Marshal(w); err != nil {
			t.Fatal(err)
		}
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

func play(t *testing.T, s State, cfg Config, periods int) State {
	t.Helper()
	for i := 0; i < periods; i++ {
		tr, err := Advance(s, cfg)
		if err != nil {
			t.Fatal(err)
		}
		s = tr.Next
	}
	return s
}

func query(t *testing.T, s State, now int, a deduction.Atom) deduction.Answer {
	t.Helper()
	res, err := entitlement.Resolve(entitlement.Hierarchy{}, s.Corpus, s.Facts, period.Period(now))
	if err != nil {
		t.Fatal(err)
	}
	return res.Query(a)
}

func has(j journal.Journal, kind journal.Kind, now int, contains string) bool {
	for _, e := range j.Of(kind) {
		if int(e.Period) == now && strings.Contains(e.String(), contains) {
			return true
		}
	}
	return false
}

func dump(t *testing.T, j journal.Journal) {
	t.Helper()
	for _, e := range j.Entries {
		t.Log(e)
	}
}

var base = Config{
	Strategy: entitlement.Hierarchy{},
	Actors:   actors.Stub{Script: script},
}

var script = []actors.Attempt{
	{Period: 30, Actor: "appius", Office: "censor", Kind: "grant_status"},
}

// The demonstration of playable-case: the citizenship casus is lived, not
// scripted. Petition, decision and entry into force come from the turn.
func TestAdvance_CitizenshipLived(t *testing.T) {
	s := play(t, start(t, nil), base, 3)
	j := s.Journal
	checks := []struct {
		kind     journal.Kind
		now      int
		contains string
	}{
		{journal.PetitionFiled, 30, "subject=marcus civis"},
		{journal.Decision, 30, "outcome=grant"},
		{journal.UltraVires, 30, "actor=appius office=censor subject=grant_status"},
		{journal.Finalization, 31, "outcome=grant"},
		{journal.ViolationDetected, 30, "violated(gaius, munus, 30)"},
		{journal.ChargeFiled, 30, "subject=gaius munus"},
		{journal.Decision, 30, "outcome=guilty"},
		{journal.Finalization, 31, "outcome=guilty"},
		{journal.ViolationDetected, 31, "violated(marcus, munus, 31)"},
	}
	for _, c := range checks {
		if !has(j, c.kind, c.now, c.contains) {
			dump(t, j)
			t.Fatalf("no %s in period %d containing %q", c.kind, c.now, c.contains)
		}
	}
	if has(j, journal.ViolationDetected, 30, "lucius") {
		t.Fatal("exempt official must not be charged")
	}
	for _, q := range []struct {
		now     int
		atom    deduction.Atom
		outcome deduction.Outcome
	}{
		{30, deduction.A("status", "marcus", "civis"), deduction.Unknown},
		{31, deduction.A("status", "marcus", "civis"), deduction.Proved},
		{30, deduction.A("offense", "gaius", "munus"), deduction.Unknown},
		{31, deduction.A("offense", "gaius", "munus"), deduction.Proved},
	} {
		if got := query(t, s, q.now, q.atom); got.Outcome != q.outcome {
			dump(t, j)
			t.Fatalf("period %d, %s: got %s, want %s", q.now, q.atom, got.Outcome, q.outcome)
		}
	}
}

// The reparation deadline counts from the decision in force, so the penalty
// can itself be violated; the chain ends in a record of offense, not a duty.
func TestAdvance_PenaltyIsViolated(t *testing.T) {
	// periods 30..34: gaius convicted in 30, in force in 31, penalty due by 32,
	// violated and convicted in 33, in force in 34
	s := play(t, start(t, nil), base, 5)
	if !has(s.Journal, journal.ViolationDetected, 33, "violated(gaius, penalty, 33)") {
		dump(t, s.Journal)
		t.Fatal("penalty was never violated")
	}
	if got := query(t, s, 34, deduction.A("offense", "gaius", "penalty")); got.Outcome != deduction.Proved {
		dump(t, s.Journal)
		t.Fatalf("offense for the penalty: got %s", got.Outcome)
	}
	if got := query(t, s, 34, deduction.A("duty", "gaius", "res_publica", "penalty", "penalty", "achieve", 35)); got.Outcome == deduction.Proved {
		t.Fatal("the ladder must end: no penalty for the penalty")
	}
}

func TestAdvance_TribuneStopsGrant(t *testing.T) {
	cfg := base
	cfg.Actors = actors.Stub{Script: script, VetoGrantsTo: map[string]bool{"marcus": true}}
	s := play(t, start(t, nil), cfg, 3)
	if !has(s.Journal, journal.Intercessio, 30, "actor=titus") {
		dump(t, s.Journal)
		t.Fatal("tribune did not intercede")
	}
	if has(s.Journal, journal.Finalization, 31, "marcus") {
		t.Fatal("vetoed decision entered into force")
	}
	if got := query(t, s, 32, deduction.A("status", "marcus", "civis")); got.Outcome == deduction.Proved {
		t.Fatal("marcus became a citizen despite the veto")
	}
}

// With no capacity the violation is derived but never detected: "not
// checked" is not "did not violate".
func TestAdvance_NoCapacityNoCharge(t *testing.T) {
	s := play(t, start(t, func(w map[string]any) {
		for _, o := range w["offices"].([]any) {
			if m := o.(map[string]any); m["id"] == "quaestor" {
				m["capacity"] = 0
			}
		}
	}), base, 2)
	if len(s.Journal.Of(journal.ChargeFiled)) > 0 {
		t.Fatal("charge filed with zero capacity")
	}
	if !has(s.Journal, journal.PeriodSummary, 30, "violations=1 checked=0") {
		dump(t, s.Journal)
		t.Fatal("summary must show a violation that nobody checked")
	}
	if got := query(t, s, 31, deduction.A("offense", "gaius", "munus")); got.Outcome == deduction.Proved {
		t.Fatal("offense without a case")
	}
}

// Gate of playable-case: replaying a period on the same inputs gives the same
// result.
func TestAdvance_Replay(t *testing.T) {
	s := play(t, start(t, nil), base, 1)
	a, err := Advance(s, base)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Advance(s, base)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same state and config gave different transitions")
	}
}
