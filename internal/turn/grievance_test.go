package turn

import (
	"testing"

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
