package linter

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/heaprip/intercessio/internal/corpus"
	d "github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/entitlement"
	"github.com/heaprip/intercessio/internal/impact"
	"github.com/heaprip/intercessio/internal/period"
	"github.com/heaprip/intercessio/internal/scenario"
)

var update = flag.Bool("update", false, "rewrite golden reports")

var strategies = []entitlement.Strategy{entitlement.Hierarchy{}, entitlement.LexPosterior{}}

type world = map[string]any

func loadCasus(t *testing.T, dir string, mutate func(world)) (*scenario.Scenario, *scenario.Report) {
	t.Helper()
	root := filepath.Join("..", "scenario", "testdata")
	schema, err := os.ReadFile(filepath.Join(root, "schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, dir, "scenario.json"))
	if err != nil {
		t.Fatal(err)
	}
	if mutate != nil {
		var w world
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
		t.Fatalf("%s has errors:\n%s", dir, rep)
	}
	return s, rep
}

func lint(t *testing.T, s *scenario.Scenario, rep *scenario.Report, strat entitlement.Strategy, now int, before *corpus.Version, v corpus.Version) *Report {
	t.Helper()
	out, err := Lint(Input{
		Corpus: v, Roles: s.Roles, Facts: s.Facts, Strategy: strat, Now: period.Period(now),
		Loader: append(append([]d.Diagnostic{}, rep.Errors...), rep.Findings...), Before: before,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Errors) > 0 {
		t.Fatalf("lint errors:\n%s", out)
	}
	return out
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update)", err)
	}
	if string(want) != got {
		t.Fatalf("report differs from %s (run with -update):\n%s", path, got)
	}
}

func find(r *Report, failure, place string) *Finding {
	for i, f := range r.Findings {
		if f.Failure == failure && (place == "" || f.Place == place) {
			return &r.Findings[i]
		}
	}
	return nil
}

func mustFind(t *testing.T, r *Report, failure, place string) *Finding {
	t.Helper()
	f := find(r, failure, place)
	if f == nil {
		t.Fatalf("no %s at %s in:\n%s", failure, place, r)
	}
	return f
}

func mustNotFind(t *testing.T, r *Report, failure, place string) {
	t.Helper()
	if f := find(r, failure, place); f != nil {
		t.Fatalf("unexpected %s:\n%s", f, r)
	}
}

// amendC1 is the amendment of the citizenship casus: C1 now requires 30
// periods of service instead of 25.
func amendC1(t *testing.T, v corpus.Version, enacted, inForce int) corpus.Version {
	t.Helper()
	c1, ok := v.Norm("C1")
	if !ok {
		t.Fatal("no C1")
	}
	n := c1
	n.Enacted, n.InForce, n.Rules = period.Period(enacted), period.Period(inForce), nil
	for _, r := range c1.Rules {
		if r.ID == "r_c1" {
			body := append([]d.Literal{}, r.Body...)
			for i, l := range body {
				if l.Pred == ">=" {
					args := append([]d.Term{}, l.Args...)
					args[1] = d.Term{Kind: d.Num, Num: 30}
					body[i].Args = args
				}
			}
			r.Body = body
		}
		n.Rules = append(n.Rules, r)
	}
	return v.Amend(n)
}

// Demonstration and gate, part one: the retroactive amendment of C1 in period
// 35, in force from 28, touches marcus by name; his status stands because it
// rests on the decision in force, not on C1.
func TestLint_RetroactiveAmendmentOfC1(t *testing.T) {
	for _, strat := range strategies {
		t.Run(strat.Name(), func(t *testing.T) {
			s, rep := loadCasus(t, "citizenship", nil)
			after := amendC1(t, s.Corpus, 35, 28)
			out := lint(t, s, rep, strat, 35, &s.Corpus, after)

			f := mustFind(t, out, "retroactivity", "C1")
			if f.Channel != Recount || strings.Join(f.People, ",") != "marcus" {
				t.Fatalf("retroactivity must be recounted and name marcus: %s", f)
			}
			mustNotFind(t, out, "taking-of-vested", "")

			var got []string
			for _, c := range out.Impact.Changes {
				got = append(got, c.String())
				if c.Atom.Pred == "status" {
					t.Errorf("status must not change: %s", c)
				}
			}
			// only 30 and 31: from 32 marcus is civis, and C1 speaks of peregrini
			// under either text. The recount lands on the periods the grant was
			// decided in.
			want := []string{}
			for p := 30; p <= 31; p++ {
				want = append(want,
					impact.Change{Period: period.Period(p), Atom: d.A("entitled", "marcus", "civis")}.String(),
					impact.Change{Period: period.Period(p), Atom: d.A("may_grant", "praetor", "marcus", "civis")}.String())
			}
			if strings.Join(got, "\n") != strings.Join(want, "\n") {
				t.Fatalf("impact:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
			}
			if strat.Name() == "hierarchy" {
				golden(t, "citizenship-c1-retroactive", out.String()+"\n"+out.Impact.Render(out.People))
			}
		})
	}
}

// Control: the same amendment without retroactive force is no retroactivity and
// touches nobody in the period it is made.
func TestLint_AmendmentWithoutRetroactiveForce(t *testing.T) {
	s, rep := loadCasus(t, "citizenship", nil)
	after := amendC1(t, s.Corpus, 35, 35)
	out := lint(t, s, rep, entitlement.Hierarchy{}, 35, &s.Corpus, after)
	mustNotFind(t, out, "retroactivity", "")
	if len(out.Impact.Changes) != 0 {
		t.Fatalf("no one is touched: %v", out.Impact.Changes)
	}
}

// Removing A4 after the strip norm N5 takes marcus's status: a recount names it
// as taking of the vested.
func TestLint_RemovingA4IsTakingOfVested(t *testing.T) {
	for _, strat := range strategies {
		t.Run(strat.Name(), func(t *testing.T) {
			s, rep := loadCasus(t, "citizenship", moveStripToLaterNorm)
			after := s.Corpus.Without("A4")
			out := lint(t, s, rep, strat, 40, &s.Corpus, after)
			f := mustFind(t, out, "taking-of-vested", "status")
			if strings.Join(f.People, ",") != "marcus" {
				t.Fatalf("must name marcus only: %s", f)
			}
		})
	}
}

func moveStripToLaterNorm(w world) {
	var strip any
	for _, n := range w["norms"].([]any) {
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
	w["norms"] = append(w["norms"].([]any), map[string]any{
		"id": "N5", "document": "constitution", "enacted": 40, "in_force": 40, "kind": "lasting", "rules": []any{strip},
	})
	w["declarations"] = append(w["declarations"].([]any),
		map[string]any{"p": "strips_vested", "a": []any{"n5", "civis"}, "period": 40},
		map[string]any{"p": "applies_to", "a": []any{"n5", "marcus"}, "period": 40},
	)
}

// Gate, part two: on the office-eligibility casus the linter names the admission
// condition circumventable through the censor's adrogatio, the censor's closed
// admission and the censor as a dead end of review.
func TestLint_OfficeEligibility(t *testing.T) {
	for _, strat := range strategies {
		t.Run(strat.Name(), func(t *testing.T) {
			s, rep := loadCasus(t, "office-eligibility", nil)
			out := lint(t, s, rep, strat, 13, nil, s.Corpus)
			f := mustFind(t, out, "circumventable-condition", "eligible")
			if !strings.Contains(f.Message, "adrogatio, an act of censor") || !strings.Contains(f.Message, "tribunus") {
				t.Fatalf("must name the censor's adrogatio and the tribunate: %s", f)
			}
			mustFind(t, out, "closed-eligibility", "censor")
			mustFind(t, out, "review-dead-end", "censor")
			mustNotFind(t, out, "competence-gap", "")
			if strat.Name() == "hierarchy" {
				golden(t, "office-eligibility-13", out.String()+"\n"+out.Graph.String())
			}
		})
	}
}

// Control: without the competence to adopt, nothing lawful opens admission; the
// act the rules rest on becomes a competence gap instead.
func TestLint_OfficeEligibilityWithoutAdrogatio(t *testing.T) {
	s, rep := loadCasus(t, "office-eligibility", func(w world) {
		for _, o := range w["offices"].([]any) {
			if m := o.(map[string]any); m["id"] == "censor" {
				m["competences"] = []any{"establish_fact"}
			}
		}
	})
	out := lint(t, s, rep, entitlement.Hierarchy{}, 13, nil, s.Corpus)
	mustNotFind(t, out, "circumventable-condition", "")
	mustNotFind(t, out, "closed-eligibility", "")
	mustFind(t, out, "competence-gap", "adrogatio")
}

// The graph is the one in effect: a tribune on the bench makes the praetor
// reviewable, an empty tribunate does not.
func TestLint_ReviewDeadEndFollowsHolders(t *testing.T) {
	s, rep := loadCasus(t, "citizenship", nil)
	out := lint(t, s, rep, entitlement.Hierarchy{}, 35, nil, s.Corpus)
	f := mustFind(t, out, "review-dead-end", "praetor")
	if !strings.Contains(f.Message, "declared: tribunus, vacant") {
		t.Fatalf("must say the tribunate is declared and vacant: %s", f)
	}
	// a guilty decision is a judgment: the praetor may grant, nobody may judge
	gap := mustFind(t, out, "competence-gap", "judge")
	if !strings.Contains(gap.Message, "r_offense") || strings.Contains(gap.Message, "r_status,") {
		t.Fatalf("judge is required by the rules on guilt, not by the grant: %s", gap)
	}
	mustNotFind(t, out, "competence-gap", "grant_status")

	s, rep = loadCasus(t, "citizenship", func(w world) {
		w["facts"] = append(w["facts"].([]any), map[string]any{
			"id": "occ_tribune", "p": "occupies", "a": []any{"gaius", "tribunus", 35, 35}, "by": "scenario", "period": 35,
		})
	})
	out = lint(t, s, rep, entitlement.Hierarchy{}, 35, nil, s.Corpus)
	mustNotFind(t, out, "review-dead-end", "praetor")
}

// Diagnostics of deduction and scenario come out under catalog slugs.
func TestLint_DiagnosticsSpeakTheCatalog(t *testing.T) {
	s, rep := loadCasus(t, "citizenship", nil)
	out := lint(t, s, rep, entitlement.Hierarchy{}, 35, nil, s.Corpus)
	mustFind(t, out, "indeterminacy", "r_revoke, r_status")
	mustFind(t, out, "dead-conclusion", "holds_right")
	mustFind(t, out, "empty-right", "ius_conubii")
	mustNotFind(t, out, "retroactivity", "")
}

func addOffice(o map[string]any, holders ...string) func(world) {
	return func(w world) {
		w["offices"] = append(w["offices"].([]any), o)
		for _, h := range holders {
			w["people"] = append(w["people"].([]any), map[string]any{"id": h, "status": "civis"})
			w["facts"] = append(w["facts"].([]any), map[string]any{
				"id": "occ_" + h, "p": "occupies", "a": []any{h, o["id"], 30, 40}, "by": "scenario", "period": 30,
			})
		}
	}
}

func setOffice(id, key string, value any) func(world) {
	return func(w world) {
		for _, o := range w["offices"].([]any) {
			if m := o.(map[string]any); m["id"] == id {
				m[key] = value
			}
		}
	}
}

func both(fs ...func(world)) func(world) {
	return func(w world) {
		for _, f := range fs {
			f(w)
		}
	}
}

// A held reviewer makes the reviewed office reviewable; a reviewer without
// holders does not.
func TestLint_ReviewIsARestraint(t *testing.T) {
	consul := map[string]any{"id": "consul", "competences": []any{"review"}, "term": 1, "reviews": []any{"praetor"}}
	s, rep := loadCasus(t, "citizenship", addOffice(consul, "quintus"))
	out := lint(t, s, rep, entitlement.Hierarchy{}, 35, nil, s.Corpus)
	mustNotFind(t, out, "review-dead-end", "praetor")
	mustFind(t, out, "review-dead-end", "consul")
	mustNotFind(t, out, "review-cycle", "")

	s, rep = loadCasus(t, "citizenship", addOffice(consul))
	out = lint(t, s, rep, entitlement.Hierarchy{}, 35, nil, s.Corpus)
	mustFind(t, out, "review-dead-end", "praetor")
}

// Offices reviewing each other make an appeal that never ends.
func TestLint_ReviewCycle(t *testing.T) {
	consul := map[string]any{"id": "consul", "competences": []any{"review"}, "term": 1, "reviews": []any{"praetor"}}
	s, rep := loadCasus(t, "citizenship", both(
		addOffice(consul, "quintus"),
		setOffice("praetor", "competences", []any{"grant_status", "review"}),
		setOffice("praetor", "reviews", []any{"consul"}),
	))
	out := lint(t, s, rep, entitlement.Hierarchy{}, 35, nil, s.Corpus)
	f := mustFind(t, out, "review-cycle", "consul")
	if !strings.Contains(f.Message, "consul -> praetor -> consul") {
		t.Fatalf("cycle path: %s", f)
	}
}

// Two censors who must concur restrain each other: adoption is no longer an act
// nobody can stop. With a single censor the quorum restrains nothing.
func TestLint_ConcurrenceRestrainsTheCensor(t *testing.T) {
	twoCensors := func(w world) {
		setOffice("censor", "quorum", 2)(w)
		w["facts"] = append(w["facts"].([]any),
			map[string]any{"id": "occ_c1", "p": "occupies", "a": []any{"fonteius", "censor", 13, 14}, "by": "scenario", "period": 13},
			map[string]any{"id": "occ_c2", "p": "occupies", "a": []any{"clodius", "censor", 13, 14}, "by": "scenario", "period": 13},
		)
	}
	s, rep := loadCasus(t, "office-eligibility", twoCensors)
	out := lint(t, s, rep, entitlement.Hierarchy{}, 13, nil, s.Corpus)
	mustNotFind(t, out, "review-dead-end", "censor")
	mustNotFind(t, out, "circumventable-condition", "")
	mustFind(t, out, "closed-eligibility", "censor")

	s, rep = loadCasus(t, "office-eligibility", setOffice("censor", "quorum", 2))
	out = lint(t, s, rep, entitlement.Hierarchy{}, 13, nil, s.Corpus)
	mustFind(t, out, "circumventable-condition", "eligible")
}
