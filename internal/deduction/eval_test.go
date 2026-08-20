package deduction

import "testing"

func defeater(id string, head Literal, body ...Literal) Rule {
	r := rule(id, head, body...)
	r.Strength = Defeater
	return r
}

func strict(id string, head Literal, body ...Literal) Rule {
	r := rule(id, head, body...)
	r.Strength = Strict
	return r
}

// Synthetic programs mirroring spike/legal-engine, plus what the spike did not
// cover: defeaters, strict rules, arithmetic with now.
func TestEvaluate_Synthetic(t *testing.T) {
	c1 := rule("r_c1", lit("entitled", v("P"), c("civis")), lit("person", v("P")))
	c2 := rule("r_c2", neg("entitled", v("P"), c("civis")), lit("offense", v("P"), v("O")))
	c3 := rule("r_c3", lit("entitled", v("P"), c("civis")), lit("offense", v("P"), v("O")), lit("merit", v("P"), v("M")))
	chain := []Defeat{{"r_c3", "r_c2"}, {"r_c2", "r_c1"}}
	marcus := A("person", "marcus")
	entitled := A("entitled", "marcus", "civis")

	tests := []struct {
		name    string
		prog    Program
		now     int
		query   Atom
		outcome Outcome
		reason  Reason
	}{
		{name: "exceptio beats the general rule",
			prog:  Program{Rules: []Rule{c1, c2, c3}, Defeats: chain, Facts: []Atom{marcus, A("offense", "marcus", "theft")}},
			query: entitled, outcome: Disproved},
		{name: "replicatio beats the exception",
			prog:  Program{Rules: []Rule{c1, c2, c3}, Defeats: chain, Facts: []Atom{marcus, A("offense", "marcus", "theft"), A("merit", "marcus", "valor")}},
			query: entitled, outcome: Proved},
		{name: "dropped exception: inapplicable rule does not block",
			prog:  Program{Rules: []Rule{c1, c2}, Defeats: chain, Facts: []Atom{marcus}},
			query: entitled, outcome: Proved},
		{name: "unordered conflict is a deadlock",
			prog:  Program{Rules: []Rule{c1, rule("r_x", neg("entitled", v("P"), c("civis")), lit("person", v("P")))}, Facts: []Atom{marcus}},
			query: entitled, outcome: Unknown, reason: Deadlock},
		{name: "no rule is a gap",
			prog:  Program{Rules: []Rule{c2}, Facts: []Atom{marcus}},
			query: entitled, outcome: Unknown, reason: Gap},
		{name: "default rule is displaced by assignment",
			prog: Program{
				Rules: []Rule{
					rule("r_nocomp", neg("competent", v("O"), v("K")), lit("office", v("O")), lit("kind", v("K"))),
					rule("r_comp", lit("competent", v("O"), v("K")), lit("responsibility", v("O"), v("K"))),
				},
				Defeats: []Defeat{{"r_comp", "r_nocomp"}},
				Facts:   []Atom{A("office", "praetor"), A("office", "censor"), A("kind", "grant"), A("responsibility", "praetor", "grant")},
			},
			query: A("competent", "praetor", "grant"), outcome: Proved},
		{name: "default rule holds where nothing displaces it",
			prog: Program{
				Rules: []Rule{
					rule("r_nocomp", neg("competent", v("O"), v("K")), lit("office", v("O")), lit("kind", v("K"))),
					rule("r_comp", lit("competent", v("O"), v("K")), lit("responsibility", v("O"), v("K"))),
				},
				Defeats: []Defeat{{"r_comp", "r_nocomp"}},
				Facts:   []Atom{A("office", "praetor"), A("office", "censor"), A("kind", "grant"), A("responsibility", "praetor", "grant")},
			},
			query: A("competent", "censor", "grant"), outcome: Disproved},
		{name: "defeater blocks without concluding",
			prog: Program{
				Rules: []Rule{
					rule("r_strip", neg("status", v("P"), v("S")), lit("strips", v("S")), lit("person", v("P"))),
					defeater("d_a4", lit("status", v("P"), v("S")), lit("protected"), lit("person", v("P")), lit("kind", v("S"))),
				},
				Defeats: []Defeat{{"d_a4", "r_strip"}},
				Facts:   []Atom{marcus, A("kind", "civis"), A("strips", "civis"), A("protected")},
			},
			query: A("status", "marcus", "civis"), outcome: Unknown, reason: Blocked},
		{name: "strict rule beats defeasible",
			prog: Program{
				Rules: []Rule{strict("s", lit("p", v("X")), lit("q", v("X"))), rule("d", neg("p", v("X")), lit("q", v("X")))},
				Facts: []Atom{A("q", "a")},
			},
			query: A("p", "a"), outcome: Proved},
		{name: "fact beats a defeasible negation",
			prog:  Program{Rules: []Rule{rule("r_veto", neg("in_force", v("A")), lit("vetoed", v("A")))}, Facts: []Atom{A("in_force", "a1"), A("vetoed", "a1")}},
			query: A("in_force", "a1"), outcome: Proved},
		{name: "arithmetic and now",
			prog: Program{
				Rules: []Rule{
					rule("r_tenure", lit("tenure", v("P"), v("D")), lit("served", v("P"), v("F"), v("T")), lit("minus", v("T"), v("F"), v("D"))),
					rule("r_c1", lit("entitled", v("P"), c("civis")), lit("tenure", v("P"), v("D")), lit(">=", v("D"), n(25))),
					rule("r_deadline", lit("due", v("P"), v("D")), lit("person", v("P")), lit("plus", Term{Kind: Now}, n(3), v("D"))),
				},
				Facts: []Atom{marcus, A("served", "marcus", 5, 32)},
			},
			now: 32, query: entitled, outcome: Proved},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Evaluate(tt.prog, tt.now)
			if err != nil {
				t.Fatal(err)
			}
			got := res.Query(tt.query)
			if got.Outcome != tt.outcome || got.Reason != tt.reason {
				t.Fatalf("%s: got %s/%s, want %s/%s\nconclusions: %v", tt.query, got.Outcome, got.Reason, tt.outcome, tt.reason, res.Conclusions())
			}
		})
	}
}

func TestEvaluate_TraceNamesBeatenRules(t *testing.T) {
	prog := Program{
		Rules: []Rule{
			rule("r_c1", lit("entitled", v("P")), lit("person", v("P"))),
			rule("r_c2", neg("entitled", v("P")), lit("offense", v("P"))),
			rule("r_c3", lit("entitled", v("P")), lit("offense", v("P")), lit("merit", v("P"))),
		},
		Defeats: []Defeat{{"r_c3", "r_c2"}, {"r_c2", "r_c1"}},
		Facts:   []Atom{A("person", "m"), A("offense", "m"), A("merit", "m")},
	}
	res, err := Evaluate(prog, 0)
	if err != nil {
		t.Fatal(err)
	}
	ans := res.Query(A("entitled", "m"))
	if ans.Outcome != Proved || ans.Trace.Rule != "r_c3" || len(ans.Trace.Defeated) != 1 || ans.Trace.Defeated[0] != "r_c2" {
		t.Fatalf("unexpected trace:\n%s", ans.Trace)
	}
}
