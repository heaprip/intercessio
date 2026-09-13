package turn

import (
	"strings"
	"testing"

	"github.com/heaprip/intercessio/internal/actors"
	"github.com/heaprip/intercessio/internal/corpus"
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/journal"
)

var stripped = []string{"appius", "gaius", "lucius", "servius", "titus"}

// stripCitizens is the amendment of period 31: A4 no longer protects the
// vested, and N5 strips civis from five registered citizens.
func stripCitizens() Script {
	n5 := corpus.Norm{ID: "N5", Document: "constitution", Enacted: 31, InForce: 31, Kind: corpus.Lasting}
	n5.Declarations = append(n5.Declarations, deduction.A("strips_vested", "n5", "civis"))
	for _, p := range stripped {
		n5.Declarations = append(n5.Declarations, deduction.A("applies_to", "n5", p))
	}
	return Script{31: {{Repeal: "A4"}, {Enact: &n5}}}
}

func count(j journal.Journal, kind journal.Kind, now int, outcome string) int {
	n := 0
	for _, e := range j.Of(kind) {
		if int(e.Period) == now && (outcome == "" || e.Outcome == outcome) {
			n++
		}
	}
	return n
}

// Demonstration and gate of self-generated-cases: an amendment raises a wave of
// grievances, and the number of cases a period files is the budget, not how
// many grievances there are. The rest wait in the queue, heavier and older
// first.
func TestAdvance_AmendmentRaisesGrievancesWithinBudget(t *testing.T) {
	cfg := base
	cfg.Auctor = stripCitizens()
	cfg.Budget = 2
	s := play(t, start(t, nil), cfg, 4) // periods 30..33
	j := s.Journal
	if testing.Verbose() {
		dump(t, j)
	}

	if got := count(j, journal.Amendment, 31, ""); got != 2 {
		dump(t, j)
		t.Fatalf("amendments in 31: %d", got)
	}
	if got := count(j, journal.Grievance, 31, "queued"); got != len(stripped) {
		dump(t, j)
		t.Fatalf("grievances queued in 31: got %d, want %d", got, len(stripped))
	}
	for now, want := range map[int]int{31: 2, 32: 2, 33: 1} {
		if got := count(j, journal.Grievance, now, "filed"); got != want {
			dump(t, j)
			t.Fatalf("grievances filed in %d: got %d, want %d", now, got, want)
		}
		if got := count(j, journal.PetitionFiled, now, ""); got > cfg.Budget {
			t.Fatalf("period %d filed %d private cases over a budget of %d", now, got, cfg.Budget)
		}
	}
	if got := count(j, journal.Grievance, 31, "deferred"); got != 3 {
		dump(t, j)
		t.Fatalf("deferred in 31: %d", got)
	}
	if len(s.Queue) != 0 {
		t.Fatalf("queue must be empty by 33: %v", s.Queue)
	}
	if got := query(t, s, 31, deduction.A("status", "gaius", "civis")); got.Outcome == deduction.Proved {
		t.Fatal("the amendment did not take gaius's status")
	}
}

// Control: without a budget every grievance is filed at once.
func TestAdvance_GrievancesWithoutBudget(t *testing.T) {
	cfg := base
	cfg.Auctor = stripCitizens()
	s := play(t, start(t, nil), cfg, 2)
	if got := count(s.Journal, journal.Grievance, 31, "filed"); got != len(stripped) {
		dump(t, s.Journal)
		t.Fatalf("filed in 31: got %d, want %d", got, len(stripped))
	}
}

// The amendment phase keeps the period a function of its inputs.
func TestAdvance_ReplayWithAmendment(t *testing.T) {
	cfg := base
	cfg.Auctor = stripCitizens()
	cfg.Budget = 2
	s := play(t, start(t, nil), cfg, 1)
	a, err := Advance(s, cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Advance(s, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Entries) != len(b.Entries) || a.Next.Corpus.Number != b.Next.Corpus.Number {
		t.Fatal("same state and config gave different transitions")
	}
	for i := range a.Entries {
		if a.Entries[i] != b.Entries[i] {
			t.Fatalf("entry %d differs: %s / %s", i, a.Entries[i], b.Entries[i])
		}
	}
}

func caseOf(j journal.Journal, person string, now int) string {
	for _, e := range j.Of(journal.PetitionFiled) {
		if int(e.Period) == now && strings.HasPrefix(e.Subject, person+" ") {
			return e.Case
		}
	}
	return ""
}

func decidedBy(j journal.Journal, id string) string {
	for _, e := range j.Entries {
		if e.Case == id && (e.Kind == journal.Decision || e.Kind == journal.NonLiquet) {
			return e.Actor
		}
	}
	return ""
}

// The praetor stripped of citizenship has nobody to hand his grievance to: he
// decides it himself, and the journal records the conflict.
func TestAdvance_OfficialDecidesOwnGrievanceAsConflict(t *testing.T) {
	cfg := base
	cfg.Auctor = stripCitizens()
	s := play(t, start(t, nil), cfg, 2)
	id := caseOf(s.Journal, "lucius", 31)
	if id == "" || decidedBy(s.Journal, id) != "lucius" {
		dump(t, s.Journal)
		t.Fatalf("lucius's case %q decided by %q", id, decidedBy(s.Journal, id))
	}
	if !has(s.Journal, journal.Conflict, 31, "case="+id) {
		dump(t, s.Journal)
		t.Fatal("no conflict recorded")
	}
}

// With a second praetor on the bench the concerned one steps aside.
func TestAdvance_RecusalToColleague(t *testing.T) {
	cfg := base
	cfg.Auctor = stripCitizens()
	s := play(t, start(t, func(w map[string]any) {
		w["facts"] = append(w["facts"].([]any), map[string]any{
			"id": "occ_marcus", "p": "occupies", "a": []any{"marcus", "praetor", 31, 99}, "by": "scenario", "period": 31,
		})
	}), cfg, 2)
	id := caseOf(s.Journal, "lucius", 31)
	if got := decidedBy(s.Journal, id); got != "marcus" {
		dump(t, s.Journal)
		t.Fatalf("lucius's case decided by %q, want marcus", got)
	}
	if !has(s.Journal, journal.Recusal, 31, "case="+id) || has(s.Journal, journal.Conflict, 31, "case="+id) {
		dump(t, s.Journal)
		t.Fatal("want a recusal and no conflict")
	}
}

// consulReviews adds a held consul who reviews the praetor.
func consulReviews(w map[string]any) {
	w["offices"] = append(w["offices"].([]any), map[string]any{"id": "consul", "competences": []any{"review"}, "term": 1, "reviews": []any{"praetor"}})
	w["people"] = append(w["people"].([]any), map[string]any{"id": "quintus", "status": "civis"})
	w["facts"] = append(w["facts"].([]any),
		map[string]any{"id": "occ_quintus", "p": "occupies", "a": []any{"quintus", "consul", 1, 99}, "by": "scenario"},
		map[string]any{"id": "ex_quintus", "p": "exempt", "a": []any{"ex_quintus", "quintus", "munus"}, "by": "scenario"},
		map[string]any{"id": "ex_quintus_in_force", "p": "in_force", "a": []any{"ex_quintus", 1}, "by": "scenario"},
	)
}

// A convicted person appeals; the consul reviews by the same law and upholds,
// and the conviction still enters into force.
func TestAdvance_AppealIsReviewed(t *testing.T) {
	cfg := base
	cfg.Actors = actors.Stub{Script: script, Appeals: true}
	s := play(t, start(t, consulReviews), cfg, 2)
	if !has(s.Journal, journal.Review, 30, "actor=quintus office=consul case=ch30_1 subject=gaius munus outcome=upheld guilty") {
		dump(t, s.Journal)
		t.Fatal("gaius's appeal was not reviewed by the consul")
	}
	if !has(s.Journal, journal.Finalization, 31, "outcome=guilty") {
		dump(t, s.Journal)
		t.Fatal("an upheld conviction enters into force")
	}

	s = play(t, start(t, nil), cfg, 1)
	if !has(s.Journal, journal.Review, 30, "outcome=no-reviewer") {
		dump(t, s.Journal)
		t.Fatal("an appeal with nobody to review it must be recorded")
	}
}
