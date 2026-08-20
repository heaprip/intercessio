package deduction

import (
	"fmt"
	"sort"
	"strings"
)

type adder func(code string, sev Severity, subject, format string, args ...any)

// Check runs the form, safety, defeat, recursion and graph checks. roles holds
// the declared predicates; undeclared ones are skipped by the graph checks and
// are the caller's to report.
func Check(rules []Rule, defeats []Defeat, roles map[string]Role) []Diagnostic {
	var ds []Diagnostic
	seen := map[string]bool{}
	add := func(code string, sev Severity, subject, format string, args ...any) {
		d := Diagnostic{code, sev, subject, fmt.Sprintf(format, args...)}
		key := d.Code + "|" + d.Subject + "|" + d.Message
		if !seen[key] {
			seen[key] = true
			ds = append(ds, d)
		}
	}
	byID := map[string]Rule{}
	for _, r := range rules {
		byID[r.ID] = r
	}
	for _, r := range rules {
		checkForm(r, add)
		checkSafety(r, add)
	}
	checkDefeats(defeats, byID, add)
	checkArithmeticRecursion(rules, add)
	checkGraph(rules, defeats, byID, roles, add)
	Sort(ds)
	return ds
}

// Sort orders diagnostics: errors first, then by code and subject.
func Sort(ds []Diagnostic) {
	sort.SliceStable(ds, func(i, j int) bool {
		a, b := ds[i], ds[j]
		if a.Severity != b.Severity {
			return a.Severity == Error
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Subject != b.Subject {
			return a.Subject < b.Subject
		}
		return a.Message < b.Message
	})
}

func invalidArg(t Term) bool { return t.Kind == Compound || t.Kind == Expr }

func checkForm(r Rule, add adder) {
	if r.Head.Builtin() {
		add("invalid-term", Error, r.ID, "builtin %s in head", r.Head.Pred)
	}
	for _, t := range r.Head.Args {
		switch {
		case invalidArg(t):
			add("invalid-term", Error, r.ID, "head argument %s is not a variable or constant", t)
		case t.Anonymous():
			add("invalid-term", Error, r.ID, "anonymous variable in head %s", r.Head)
		}
	}
	for _, l := range r.Body {
		if l.Builtin() {
			if len(l.Args) != builtinArity[l.Pred] {
				add("invalid-term", Error, r.ID, "builtin %s takes %d arguments", l.Pred, builtinArity[l.Pred])
				continue
			}
			for _, t := range l.Args {
				if invalidArg(t) || t.Anonymous() || t.Kind == Const {
					add("invalid-term", Error, r.ID, "builtin argument %s in %s must be a variable, number or now", t, l)
				}
			}
			continue
		}
		for _, t := range l.Args {
			if invalidArg(t) {
				add("invalid-term", Error, r.ID, "argument %s in %s is not a variable or constant", t, l)
			}
		}
	}
}

func varsOf(t Term, out map[string]bool) {
	switch t.Kind {
	case Var:
		if !t.Anonymous() {
			out[t.Name] = true
		}
	case Compound, Expr:
		for _, a := range t.Args {
			varsOf(a, out)
		}
	}
}

func termVars(ts ...Term) map[string]bool {
	out := map[string]bool{}
	for _, t := range ts {
		varsOf(t, out)
	}
	return out
}

func sortedFree(vs map[string]bool, bound map[string]bool) []string {
	var free []string
	for v := range vs {
		if !bound[v] {
			free = append(free, v)
		}
	}
	sort.Strings(free)
	return free
}

func checkSafety(r Rule, add adder) {
	bound := map[string]bool{}
	for _, l := range r.Body {
		if !l.Builtin() {
			for v := range termVars(l.Args...) {
				bound[v] = true
			}
		}
	}
	for changed := true; changed; {
		changed = false
		for _, l := range r.Body {
			if !l.Arithmetic() || len(l.Args) != 3 {
				continue
			}
			if len(sortedFree(termVars(l.Args[:2]...), bound)) > 0 {
				continue
			}
			if o := l.Args[2]; o.Kind == Var && !o.Anonymous() && !bound[o.Name] {
				bound[o.Name] = true
				changed = true
			}
		}
	}
	for _, l := range r.Body {
		if !l.Builtin() {
			continue
		}
		in := l.Args
		if l.Arithmetic() && len(in) == 3 {
			in = in[:2]
		}
		if free := sortedFree(termVars(in...), bound); len(free) > 0 {
			add("unbound-builtin", Error, r.ID, "%s: %s not bound", l, strings.Join(free, ", "))
		}
	}
	if free := sortedFree(termVars(r.Head.Args...), bound); len(free) > 0 {
		add("unsafe-rule", Error, r.ID, "head variables %s not bound by the body", strings.Join(free, ", "))
	}
}

func checkDefeats(defeats []Defeat, byID map[string]Rule, add adder) {
	next := map[string][]string{}
	for _, d := range defeats {
		for _, id := range []string{d.Over, d.Under} {
			if _, ok := byID[id]; !ok {
				add("unknown-reference", Error, d.Over+" > "+d.Under, "rule %s does not exist", id)
			}
		}
		next[d.Over] = append(next[d.Over], d.Under)
	}
	state := map[string]int{}
	var visit func(n string, path []string)
	visit = func(n string, path []string) {
		state[n] = 1
		path = append(path, n)
		for _, m := range next[n] {
			switch state[m] {
			case 1:
				add("defeat-cycle", Error, strings.Join(append(path, m), " > "), "defeat relation contains a cycle")
			case 0:
				visit(m, path)
			}
		}
		state[n] = 2
	}
	var keys []string
	for k := range next {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if state[k] == 0 {
			visit(k, nil)
		}
	}
}

type position struct {
	pred string
	i    int
}

func (p position) String() string { return fmt.Sprintf("%s/%d", p.pred, p.i) }

type source struct {
	p     position
	arith bool
}

// checkArithmeticRecursion looks for a computed value that can flow, through
// argument positions, back into the input of the computation.
func checkArithmeticRecursion(rules []Rule, add adder) {
	type edge struct {
		from, to position
		arith    bool
		rule     string
	}
	var edges []edge
	for _, r := range rules {
		src := map[string][]source{}
		has := func(v string, s source) bool {
			for _, x := range src[v] {
				if x == s {
					return true
				}
			}
			return false
		}
		for _, l := range r.Body {
			if l.Builtin() {
				continue
			}
			for i, t := range l.Args {
				if t.Kind == Var && !t.Anonymous() {
					src[t.Name] = append(src[t.Name], source{position{l.Pred, i}, false})
				}
			}
		}
		for changed := true; changed; {
			changed = false
			for _, l := range r.Body {
				if !l.Arithmetic() || len(l.Args) != 3 {
					continue
				}
				o := l.Args[2]
				if o.Kind != Var || o.Anonymous() {
					continue
				}
				for v := range termVars(l.Args[:2]...) {
					for _, s := range src[v] {
						s.arith = true
						if !has(o.Name, s) {
							src[o.Name] = append(src[o.Name], s)
							changed = true
						}
					}
				}
			}
		}
		for j, t := range r.Head.Args {
			to := position{r.Head.Pred, j}
			for v := range termVars(t) {
				for _, s := range src[v] {
					edges = append(edges, edge{s.p, to, s.arith || t.Kind == Expr, r.ID})
				}
			}
		}
	}
	adj := map[position][]position{}
	for _, e := range edges {
		adj[e.from] = append(adj[e.from], e.to)
	}
	reach := func(a, b position) bool {
		seen := map[position]bool{a: true}
		queue := []position{a}
		for len(queue) > 0 {
			n := queue[0]
			queue = queue[1:]
			if n == b {
				return true
			}
			for _, m := range adj[n] {
				if !seen[m] {
					seen[m] = true
					queue = append(queue, m)
				}
			}
		}
		return false
	}
	for _, e := range edges {
		if e.arith && reach(e.to, e.from) {
			add("arithmetic-recursion", Error, e.rule, "value computed from %s into %s flows back into its input", e.from, e.to)
		}
	}
}

func opposite(a, b Literal) bool {
	if a.Pred != b.Pred || a.Neg == b.Neg || len(a.Args) != len(b.Args) {
		return false
	}
	for i := range a.Args {
		x, y := a.Args[i], b.Args[i]
		if x.Kind == Var || y.Kind == Var {
			continue
		}
		if x.String() != y.String() {
			return false
		}
	}
	return true
}

func canonical(r Rule) string {
	names := map[string]string{}
	var term func(t Term) string
	term = func(t Term) string {
		switch t.Kind {
		case Var:
			if t.Anonymous() {
				return "_"
			}
			if _, ok := names[t.Name]; !ok {
				names[t.Name] = fmt.Sprintf("V%d", len(names))
			}
			return names[t.Name]
		case Compound, Expr:
			parts := make([]string, len(t.Args))
			for i, a := range t.Args {
				parts[i] = term(a)
			}
			return t.Name + "(" + strings.Join(parts, ",") + ")"
		}
		return t.String()
	}
	lit := func(l Literal) string {
		parts := make([]string, len(l.Args))
		for i, a := range l.Args {
			parts[i] = term(a)
		}
		return fmt.Sprintf("%v%s(%s)", l.Neg, l.Pred, strings.Join(parts, ","))
	}
	var b strings.Builder
	b.WriteString(string(r.Strength) + ":" + lit(r.Head) + ":-")
	for _, l := range r.Body {
		b.WriteString(lit(l) + ";")
	}
	return b.String()
}

func checkGraph(rules []Rule, defeats []Defeat, byID map[string]Rule, roles map[string]Role, add adder) {
	posConc, negConc := map[string][]string{}, map[string][]string{}
	usedPos, usedNeg := map[string]bool{}, map[string]bool{}
	for _, r := range rules {
		if r.Head.Neg {
			negConc[r.Head.Pred] = append(negConc[r.Head.Pred], r.ID)
		} else {
			posConc[r.Head.Pred] = append(posConc[r.Head.Pred], r.ID)
		}
		for _, l := range r.Body {
			if l.Builtin() {
				continue
			}
			if l.Neg {
				usedNeg[l.Pred] = true
			} else {
				usedPos[l.Pred] = true
			}
		}
	}
	preds := map[string]bool{}
	for p := range posConc {
		preds[p] = true
	}
	for p := range negConc {
		preds[p] = true
	}
	for p := range usedPos {
		preds[p] = true
	}
	for p := range usedNeg {
		preds[p] = true
	}
	var names []string
	for p := range preds {
		names = append(names, p)
	}
	sort.Strings(names)

	for _, p := range names {
		role, declared := roles[p]
		if !declared {
			continue
		}
		concluders := append(append([]string{}, posConc[p]...), negConc[p]...)
		sort.Strings(concluders)
		if len(concluders) > 0 {
			switch {
			case role.Stored():
				add("fact-and-conclusion", Finding, p, "%s is stored as %s and concluded by %s", p, role, strings.Join(concluders, ", "))
			case role != RoleOutput && !usedPos[p] && !usedNeg[p]:
				add("unconsumed-conclusion", Finding, p, "%s is concluded by %s and consumed by no rule", p, strings.Join(concluders, ", "))
			}
			if !role.Stored() && len(posConc[p]) == 0 {
				add("negative-only-conclusion", Finding, p, "%s is only ever concluded negatively", p)
			}
		}
		if usedPos[p] && !role.Stored() && len(posConc[p]) == 0 {
			add("underdetermined-condition", Finding, p, "%s is a condition that no rule concludes and no fact supplies", p)
		}
		if usedNeg[p] && len(negConc[p]) == 0 {
			add("negation-never-concluded", Finding, p, "¬%s is a condition that no rule concludes", p)
		}
	}

	groups := map[string][]string{}
	var order []string
	for _, r := range rules {
		c := canonical(r)
		if _, ok := groups[c]; !ok {
			order = append(order, c)
		}
		groups[c] = append(groups[c], r.ID)
	}
	for _, c := range order {
		if ids := groups[c]; len(ids) > 1 {
			sort.Strings(ids)
			add("duplicate-rule", Finding, strings.Join(ids, ", "), "rules have the same head and body")
		}
	}

	next := map[string][]string{}
	for _, d := range defeats {
		next[d.Over] = append(next[d.Over], d.Under)
	}
	stronger := func(a, b string) bool {
		seen := map[string]bool{a: true}
		queue := []string{a}
		for len(queue) > 0 {
			n := queue[0]
			queue = queue[1:]
			for _, m := range next[n] {
				if m == b {
					return true
				}
				if !seen[m] {
					seen[m] = true
					queue = append(queue, m)
				}
			}
		}
		return false
	}
	for i := range rules {
		for j := i + 1; j < len(rules); j++ {
			a, b := rules[i], rules[j]
			if !opposite(a.Head, b.Head) || (a.Strength == Defeater && b.Strength == Defeater) {
				continue
			}
			if !stronger(a.ID, b.ID) && !stronger(b.ID, a.ID) {
				ids := []string{a.ID, b.ID}
				sort.Strings(ids)
				add("unordered-conflict", Finding, strings.Join(ids, ", "), "%s and %s conclude opposite %s with no declared order", ids[0], ids[1], a.Head.Pred)
			}
		}
	}
	for _, d := range defeats {
		over, ok1 := byID[d.Over]
		under, ok2 := byID[d.Under]
		if !ok1 || !ok2 {
			continue
		}
		if !opposite(over.Head, under.Head) {
			add("defeat-without-conflict", Finding, d.Over+" > "+d.Under, "%s and %s do not conclude opposite literals", d.Over, d.Under)
			continue
		}
		body := map[string]bool{}
		for _, l := range under.Body {
			body[l.String()] = true
		}
		subset := true
		for _, l := range over.Body {
			if !body[l.String()] {
				subset = false
				break
			}
		}
		if subset {
			add("always-defeated", Finding, d.Under, "%s loses to %s whenever it applies", d.Under, d.Over)
		}
	}
}
