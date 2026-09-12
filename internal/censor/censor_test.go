package censor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/heaprip/intercessio/internal/actors"
	"github.com/heaprip/intercessio/internal/corpus"
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/entitlement"
	"github.com/heaprip/intercessio/internal/powergraph"
	"github.com/heaprip/intercessio/internal/scenario"
	"github.com/heaprip/intercessio/internal/turn"
)

func start(t *testing.T, mutate func(map[string]any)) turn.State {
	t.Helper()
	root := filepath.Join("..", "scenario", "testdata")
	schema, err := os.ReadFile(filepath.Join(root, "schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "playable", "scenario.json"))
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
	return turn.State{Period: s.StartPeriod, Corpus: s.Corpus, Facts: s.Facts, Seed: 7}
}

// stripCitizens: in 31 A4 is repealed and N5 strips five citizens.
func stripCitizens() turn.Script {
	n5 := corpus.Norm{ID: "N5", Document: "constitution", Enacted: 31, InForce: 31, Kind: corpus.Lasting}
	n5.Declarations = append(n5.Declarations, deduction.A("strips_vested", "n5", "civis"))
	for _, p := range []string{"appius", "gaius", "lucius", "servius", "titus"} {
		n5.Declarations = append(n5.Declarations, deduction.A("applies_to", "n5", p))
	}
	return turn.Script{31: {{Repeal: "A4"}, {Enact: &n5}}}
}

func stand(t *testing.T, mutate func(map[string]any), auctor turn.Auctor, periods int) Stand {
	t.Helper()
	cfg := turn.Config{Strategy: entitlement.Hierarchy{}, Actors: actors.Stub{}, Auctor: auctor}
	st := start(t, mutate)
	from := st.Period
	for i := 0; i < periods; i++ {
		tr, err := turn.Advance(st, cfg)
		if err != nil {
			t.Fatal(err)
		}
		st = tr.Next
	}
	to := st.Period - 1
	g, err := powergraph.Build(cfg.Strategy, st.Corpus, st.Facts, to)
	if err != nil {
		t.Fatal(err)
	}
	out := Build(Input{Journal: st.Journal, From: from, To: to, Graph: g, Corpus: st.Corpus, Preset: Preset{NonLiquet: 0.3}})
	if testing.Verbose() {
		t.Log(out)
	}
	return out
}

func axis(t *testing.T, s Stand, name string) float64 {
	t.Helper()
	a, ok := s.Axis(name)
	if !ok {
		t.Fatalf("no axis %s in %s", name, s)
	}
	return a.Value
}

// Gate of censor-stand: the stand moves with the player's decisions and with
// the holders in office, and mass non liquet is found by the journal alone.
func TestStand_MovesWithTheGame(t *testing.T) {
	quiet := stand(t, nil, nil, 3)
	stripped := stand(t, nil, stripCitizens(), 3)

	if axis(t, quiet, "non-liquet share") != 0 || len(quiet.Findings) != 0 {
		t.Fatalf("a quiet game has no non liquet: %s %v", quiet, quiet.Findings)
	}
	if axis(t, stripped, "non-liquet share") <= 0.3 {
		t.Fatalf("stripping citizens must raise non liquet: %s", stripped)
	}
	if len(stripped.Findings) != 1 || stripped.Findings[0].Failure != "mass-non-liquet" {
		t.Fatalf("mass non liquet must be found: %v", stripped.Findings)
	}
	if axis(t, quiet, "amendments") != 0 || axis(t, stripped, "amendments") != 2 {
		t.Fatal("amendments must be counted")
	}

	if got := axis(t, quiet, "veto players"); got != 1 {
		t.Fatalf("one tribune can stop the praetor: got %.2f", got)
	}
	noTribune := stand(t, func(w map[string]any) {
		var kept []any
		for _, f := range w["facts"].([]any) {
			if f.(map[string]any)["id"] != "occ_titus" {
				kept = append(kept, f)
			}
		}
		w["facts"] = kept
	}, nil, 3)
	if got := axis(t, noTribune, "veto players"); got != 0 {
		t.Fatalf("an empty tribunate stops nobody: got %.2f", got)
	}

	if axis(t, quiet, "enforcement coverage") <= 0 {
		t.Fatalf("the quaestor covers bearers: %s", quiet)
	}

}
