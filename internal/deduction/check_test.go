package deduction

import "testing"

func v(n string) Term { return Term{Kind: Var, Name: n} }
func c(n string) Term { return Term{Kind: Const, Name: n} }
func n(i int) Term    { return Term{Kind: Num, Num: i} }

func lit(p string, args ...Term) Literal { return Literal{Pred: p, Args: args} }
func neg(p string, args ...Term) Literal { return Literal{Pred: p, Args: args, Neg: true} }

func rule(id string, head Literal, body ...Literal) Rule {
	return Rule{ID: id, Head: head, Body: body, Strength: Defeasible}
}

func codes(ds []Diagnostic) map[string]bool {
	out := map[string]bool{}
	for _, d := range ds {
		out[d.Code] = true
	}
	return out
}

func TestCheck_Codes(t *testing.T) {
	roles := map[string]Role{"p": RoleDerived, "q": RoleFact, "office": RoleDomain, "kind": RoleDomain, "competent": RoleOutput}
	tests := []struct {
		name    string
		rules   []Rule
		defeats []Defeat
		want    string
		absent  bool
	}{
		{name: "unsafe head variable", rules: []Rule{rule("r", lit("p", v("X")), lit("q", v("Y")))}, want: "unsafe-rule"},
		{name: "default rule over domain predicates is safe", rules: []Rule{rule("r", neg("competent", v("O"), v("K")), lit("office", v("O")), lit("kind", v("K")))}, want: "unsafe-rule", absent: true},
		{name: "arithmetic output binds", rules: []Rule{rule("r", lit("p", v("B")), lit("q", v("A")), lit("plus", v("A"), n(1), v("B")))}, want: "unsafe-rule", absent: true},
		{name: "unbound comparison", rules: []Rule{rule("r", lit("p", v("A")), lit("q", v("A")), lit(">", v("N"), v("A")))}, want: "unbound-builtin"},
		{name: "now is bound", rules: []Rule{rule("r", lit("p", v("A")), lit("q", v("A")), lit(">", Term{Kind: Now}, v("A")))}, want: "unbound-builtin", absent: true},
		{name: "recursion through arithmetic", rules: []Rule{rule("r", lit("p", v("B")), lit("p", v("A")), lit("plus", v("A"), n(1), v("B")))}, want: "arithmetic-recursion"},
		{name: "comparison does not recurse", rules: []Rule{rule("r", lit("p", v("A")), lit("p", v("A")), lit(">", v("A"), n(1)))}, want: "arithmetic-recursion", absent: true},
		{name: "anonymous variable in head", rules: []Rule{rule("r", lit("p", c("_x"), v("_")), lit("q", v("A")))}, want: "invalid-term"},
		{name: "compound term", rules: []Rule{rule("r", lit("p", Term{Kind: Compound, Name: "f", Args: []Term{v("A")}}), lit("q", v("A")))}, want: "invalid-term"},
		{name: "defeat cycle", rules: []Rule{rule("a", lit("p", v("X")), lit("q", v("X"))), rule("b", neg("p", v("X")), lit("q", v("X")))}, defeats: []Defeat{{"a", "b"}, {"b", "a"}}, want: "defeat-cycle"},
		{name: "unordered conflict", rules: []Rule{rule("a", lit("p", v("X")), lit("q", v("X"))), rule("b", neg("p", v("X")), lit("q", v("X")))}, want: "unordered-conflict"},
		{name: "ordered conflict", rules: []Rule{rule("a", lit("p", v("X")), lit("q", v("X"))), rule("b", neg("p", v("X")), lit("q", v("X")))}, defeats: []Defeat{{"a", "b"}}, want: "unordered-conflict", absent: true},
		{name: "negation never concluded", rules: []Rule{rule("a", lit("competent", v("X"), v("Y")), neg("p", v("X")), lit("q", v("Y")))}, want: "negation-never-concluded"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := codes(Check(tt.rules, tt.defeats, roles))
			if got[tt.want] == tt.absent {
				t.Fatalf("code %s present=%v, want present=%v; got %v", tt.want, got[tt.want], !tt.absent, got)
			}
		})
	}
}
