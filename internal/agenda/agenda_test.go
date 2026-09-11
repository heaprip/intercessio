package agenda

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/heaprip/intercessio/internal/actors"
	"github.com/heaprip/intercessio/internal/entitlement"
	"github.com/heaprip/intercessio/internal/journal"
	"github.com/heaprip/intercessio/internal/linter"
	"github.com/heaprip/intercessio/internal/scenario"
	"github.com/heaprip/intercessio/internal/turn"
)

var preset = Preset{Size: 3, Weights: map[string]int{
	"retroactivity": 5, "taking-of-vested": 5, "judge-in-own-cause": 4, "circumventable-condition": 4, "indeterminacy": 2,
}}

// retroactiveC2 makes the exception C2 of the playable casus enacted in 30 and
// in force from 1.
func retroactiveC2(w map[string]any) {
	for _, n := range w["norms"].([]any) {
		if m := n.(map[string]any); m["id"] == "C2" {
			m["enacted"] = 30
		}
	}
}

func load(t *testing.T) (*scenario.Scenario, turn.State) {
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
	var w map[string]any
	if err := json.Unmarshal(raw, &w); err != nil {
		t.Fatal(err)
	}
	retroactiveC2(w)
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
	return s, turn.State{Period: s.StartPeriod, Corpus: s.Corpus, Facts: s.Facts, Seed: 7}
}

type round struct {
	stack   Stack
	entries []journal.Entry
}

// play is the game loop: lint, build the stack, let a stub auctor accept every
// card with a drafted amendment and reject the rest, then live the period.
func play(t *testing.T, periods int) []round {
	t.Helper()
	s, st := load(t)
	cfg := turn.Config{Strategy: entitlement.Hierarchy{}, Actors: actors.Stub{}}
	memory := Memory{}
	var entries []journal.Entry
	var rounds []round
	for i := 0; i < periods; i++ {
		lint, err := linter.Lint(linter.Input{Corpus: st.Corpus, Roles: s.Roles, Facts: st.Facts, Strategy: cfg.Strategy, Now: st.Period})
		if err != nil {
			t.Fatal(err)
		}
		stack := Build(Input{Lint: lint, Entries: entries, Corpus: st.Corpus, Now: st.Period, Memory: memory, Preset: preset})
		script := turn.Script{}
		for _, c := range stack.Cards {
			script[st.Period] = append(script[st.Period], c.Proposal.Amendments...)
			memory = memory.Record(c)
		}
		cfg.Auctor = script
		tr, err := turn.Advance(st, cfg)
		if err != nil {
			t.Fatal(err)
		}
		rounds = append(rounds, round{stack, tr.Entries})
		entries, st = tr.Entries, tr.Next
	}
	return rounds
}

func card(s Stack, failure, root string) *Card {
	for i, c := range s.Cards {
		if c.Failure == failure && c.Root == root {
			return &s.Cards[i]
		}
	}
	return nil
}

func show(t *testing.T, rounds []round) {
	t.Helper()
	for _, r := range rounds {
		t.Logf("period %d: %d cards, %d overflow, %d held", r.stack.Period, len(r.stack.Cards), len(r.stack.Overflow), len(r.stack.Held))
		for _, c := range r.stack.Cards {
			t.Logf("  %s", c)
		}
	}
}

// Gate of the stack: a game of several periods goes only through stacks. No
// stack is larger than the preset, a card never decides a case, an accepted
// amendment removes its finding, and a decided card does not come back without
// new evidence.
func TestStack_GameGoesThroughStacks(t *testing.T) {
	rounds := play(t, 4)
	if testing.Verbose() {
		show(t, rounds)
	}

	first := rounds[0].stack
	retro := card(first, "retroactivity", "C2")
	if retro == nil || len(retro.Proposal.Amendments) != 1 {
		show(t, rounds)
		t.Fatal("period 30 must bring a card to re-enact C2")
	}
	status := card(first, "indeterminacy", "status")
	if status == nil || status.Reach < 2 {
		show(t, rounds)
		t.Fatal("every unordered conflict over status must be one card")
	}
	if len(first.Overflow) == 0 {
		t.Fatal("what does not fit must go to the report, not vanish")
	}

	for i, r := range rounds {
		if len(r.stack.Cards) > preset.Size {
			t.Fatalf("period %d: %d cards over the size %d", r.stack.Period, len(r.stack.Cards), preset.Size)
		}
		for _, c := range r.stack.Cards {
			if len(c.Evidence) == 0 {
				t.Fatalf("card without evidence: %s", c)
			}
		}
		for _, e := range r.entries {
			if e.Actor == "auctor" && e.Kind != journal.Amendment {
				t.Fatalf("the auctor did something other than amend: %s", e)
			}
		}
		if i == 0 {
			continue
		}
		if card(r.stack, "retroactivity", "C2") != nil {
			t.Fatalf("period %d: the accepted re-enactment did not remove the finding", r.stack.Period)
		}
		for _, c := range r.stack.Cards {
			for _, prev := range rounds[i-1].stack.Cards {
				if c.Key == prev.Key && c.Reach <= prev.Reach && subset(c.People, prev.People) {
					t.Fatalf("period %d: %s came back without new evidence", r.stack.Period, c.Key)
				}
			}
		}
	}
	if !strings.Contains(rounds[0].entries[0].String(), "enact C2") {
		t.Fatalf("the accepted card must amend at the start of the period: %s", rounds[0].entries[0])
	}
}

// A card held back returns when its reach grows.
func TestStack_CardReturnsWithNewEvidence(t *testing.T) {
	lint := &linter.Report{Findings: []linter.Finding{
		{Failure: "judge-in-own-cause", Place: "grant_status", Message: "only gaius", People: []string{"gaius"}, Channel: linter.Linter},
	}}
	first := Build(Input{Lint: lint, Preset: preset})
	if len(first.Cards) != 1 {
		t.Fatalf("cards: %v", first.Cards)
	}
	memory := Memory{}.Record(first.Cards[0])
	if again := Build(Input{Lint: lint, Memory: memory, Preset: preset}); len(again.Cards) != 0 || len(again.Held) != 1 {
		t.Fatalf("the same card must be held: %+v", again)
	}
	lint.Findings[0].People = []string{"gaius", "lucius"}
	if grown := Build(Input{Lint: lint, Memory: memory, Preset: preset}); len(grown.Cards) != 1 {
		t.Fatalf("a new person is new evidence: %+v", grown)
	}
}
