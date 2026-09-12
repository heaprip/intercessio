package report

import (
	"strings"
	"testing"

	"github.com/heaprip/intercessio/internal/agenda"
	"github.com/heaprip/intercessio/internal/journal"
	"github.com/heaprip/intercessio/internal/linter"
)

func section(r Report, title string) string {
	for _, s := range r.Sections {
		if s.Title == title {
			return strings.Join(s.Lines, "\n")
		}
	}
	return ""
}

// The report keeps apart what the player must not confuse: a gap from a
// deadlock, not violating from not being checked.
func TestBuild_KeepsDiagnosesApart(t *testing.T) {
	entries := []journal.Entry{
		{Kind: journal.Amendment, Actor: "auctor", Subject: "repeal A4", Outcome: "v2"},
		{Kind: journal.Grievance, Actor: "gaius", Outcome: "filed"},
		{Kind: journal.Grievance, Actor: "titus", Outcome: "deferred"},
		{Kind: journal.PetitionFiled, Subject: "gaius civis"},
		{Kind: journal.Decision, Subject: "gaius civis", Outcome: "refuse"},
		{Kind: journal.NonLiquet, Subject: "appius civis", Basis: journal.Basis{Rule: "gap"}},
		{Kind: journal.NonLiquet, Subject: "tiberius in_force", Basis: journal.Basis{Rule: "deadlock"}},
		{Kind: journal.Expired, Subject: "tiberius law", Case: "c9"},
		{Kind: journal.Conflict, Actor: "lucius", Case: "pt32_1"},
		{Kind: journal.PeriodSummary, Subject: "violations=3 checked=1 bearers=4 cases=5 queue=1"},
	}
	lint := &linter.Report{Findings: []linter.Finding{
		{Failure: "taking-of-vested", Channel: linter.Recount, Place: "status", Message: "status taken", People: []string{"gaius", "titus"}},
		{Failure: "", Code: "duplicate-rule", Place: "r_x", Message: "same"},
	}}
	over := []agenda.Card{{Key: "dead-conclusion|valid", Failure: "dead-conclusion", Root: "valid", Reach: 1, Score: 1}}
	r := Build(Input{Period: 32, Entries: entries, Lint: lint, Overflow: over})
	if testing.Verbose() {
		t.Log("\n" + r.String())
	}

	cases := section(r, "cases")
	for _, want := range []string{"non liquet, gap: 1 [appius civis]", "non liquet, deadlock: 1 [tiberius in_force]", "expired: tiberius law", "conflict: lucius decides own case pt32_1", "decided refuse 1"} {
		if !strings.Contains(cases, want) {
			t.Fatalf("cases section lacks %q:\n%s", want, cases)
		}
	}
	if got := section(r, "enforcement"); !strings.Contains(got, "violations 3, checked 1, not checked 2") {
		t.Fatalf("enforcement: %s", got)
	}
	if got := section(r, "amendment"); !strings.Contains(got, "grievances deferred: 1 [titus]") {
		t.Fatalf("amendment: %s", got)
	}
	linterLines := section(r, "linter")
	if !strings.Contains(linterLines, "[gaius, titus]") || strings.Contains(linterLines, "duplicate-rule") {
		t.Fatalf("linter must name people and leave form remarks out:\n%s", linterLines)
	}
	if !strings.Contains(section(r, "not in the stack"), "dead-conclusion") {
		t.Fatal("overflow of the stack must reach the report")
	}
}

// With the previous lint the report shows what is new and what went away, and
// only counts what persists.
func TestBuild_FindingsSincePreviousPeriod(t *testing.T) {
	stay := linter.Finding{Failure: "indeterminacy", Place: "r_a, r_b", Message: "same"}
	gone := linter.Finding{Failure: "retroactivity", Place: "C2", Message: "enacted in 30, applies from 1"}
	fresh := linter.Finding{Failure: "judge-in-own-cause", Place: "grant_status", Message: "only lucius", People: []string{"lucius"}}
	r := Build(Input{Period: 31,
		Lint:     &linter.Report{Findings: []linter.Finding{stay, fresh}},
		Previous: &linter.Report{Findings: []linter.Finding{stay, gone}},
	})
	got := section(r, "linter")
	for _, want := range []string{"new: judge-in-own-cause", "gone: retroactivity", "1 findings persist"} {
		if !strings.Contains(got, want) {
			t.Fatalf("linter section lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "indeterminacy") {
		t.Fatalf("a persisting finding must only be counted:\n%s", got)
	}
}
