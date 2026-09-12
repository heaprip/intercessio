package deduction

import (
	"fmt"
	"sort"
	"strings"
)

// Value is a ground argument.
type Value struct {
	Const string
	Num   int
	IsNum bool
}

func (v Value) String() string {
	if v.IsNum {
		return fmt.Sprint(v.Num)
	}
	return v.Const
}

// C and N build values.
func C(s string) Value { return Value{Const: s} }
func N(n int) Value    { return Value{Num: n, IsNum: true} }

// Atom is a ground literal.
type Atom struct {
	Pred string
	Args []Value
	Neg  bool
}

// A builds an atom from strings and ints.
func A(pred string, args ...any) Atom {
	a := Atom{Pred: pred}
	for _, x := range args {
		switch v := x.(type) {
		case int:
			a.Args = append(a.Args, N(v))
		case string:
			a.Args = append(a.Args, C(v))
		case Value:
			a.Args = append(a.Args, v)
		}
	}
	return a
}

// Not negates an atom.
func Not(a Atom) Atom { a.Neg = !a.Neg; return a }

func (a Atom) String() string {
	s := make([]string, len(a.Args))
	for i, v := range a.Args {
		s[i] = v.String()
	}
	out := a.Pred + "(" + strings.Join(s, ", ") + ")"
	if a.Neg {
		out = "¬" + out
	}
	return out
}

// Complement flips the sign.
func (a Atom) Complement() Atom { return Not(a) }

// Program is what the evaluator needs: rules, the filled defeat relation and
// the facts of the slice.
type Program struct {
	Rules   []Rule
	Defeats []Defeat
	Facts   []Atom
}

// Outcome of a query.
type Outcome string

const (
	Proved    Outcome = "proved"
	Disproved Outcome = "disproved"
	Unknown   Outcome = "unknown"
)

// Reason distinguishes the two meanings of unknown.
type Reason string

const (
	// Gap: no applicable rule either way. Fixed by writing a norm.
	Gap Reason = "gap"
	// Deadlock: an applicable pair of opposite rules with no order either way.
	// Fixed by ordering.
	Deadlock Reason = "deadlock"
	// Blocked: every applicable opposite pair is ordered, and still nothing
	// wins, typically because a defeater stops the stronger rule. This is how
	// protection of the vested looks.
	Blocked Reason = "blocked"
	// Undetermined: the proof depends on itself.
	Undetermined Reason = "undetermined"
)

// Answer to a query.
type Answer struct {
	Outcome  Outcome
	Reason   Reason
	Trace    *Trace
	Opposing []string // applicable rules on both sides, for a deadlock
}

// Trace explains a conclusion.
type Trace struct {
	Atom     Atom
	Rule     string // rule id, or "fact"
	Premises []*Trace
	Defeated []string // applicable opposite rules the rule beat
}

func (t *Trace) String() string {
	var b strings.Builder
	var walk func(t *Trace, depth int)
	walk = func(t *Trace, depth int) {
		fmt.Fprintf(&b, "%s%s  [%s]", strings.Repeat("  ", depth), t.Atom, t.Rule)
		if len(t.Defeated) > 0 {
			fmt.Fprintf(&b, "  beats %s", strings.Join(t.Defeated, ", "))
		}
		b.WriteString("\n")
		for _, p := range t.Premises {
			walk(p, depth+1)
		}
	}
	walk(t, 0)
	return b.String()
}

// Rules lists, sorted and once each, every rule that took part in the trace:
// the rules that concluded and the opposite rules they beat.
func (t *Trace) Rules() []string {
	seen := map[string]bool{}
	var walk func(t *Trace)
	walk = func(t *Trace) {
		if t == nil {
			return
		}
		if t.Rule != "fact" && t.Rule != "?" && t.Rule != "" {
			seen[t.Rule] = true
		}
		for _, d := range t.Defeated {
			seen[d] = true
		}
		for _, p := range t.Premises {
			walk(p)
		}
	}
	walk(t)
	out := make([]string, 0, len(seen))
	for r := range seen {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

type instance struct {
	rule Rule
	head Atom
	body []Atom
}

// Result of an evaluation on one period.
type Result struct {
	now      int
	facts    map[string]bool
	atoms    map[string]Atom
	plus     map[string]bool
	minus    map[string]bool
	proof    map[string]*instance
	rulesFor map[string][]*instance
	stronger map[string]map[string]bool
}

// Evaluate grounds the rules against the facts and resolves conflicts by the
// defeat relation. now is the period the slice is taken at.
func Evaluate(p Program, now int) (*Result, error) {
	r := &Result{
		now:      now,
		facts:    map[string]bool{},
		atoms:    map[string]Atom{},
		plus:     map[string]bool{},
		minus:    map[string]bool{},
		proof:    map[string]*instance{},
		rulesFor: map[string][]*instance{},
		stronger: direct(p.Defeats),
	}
	insts, err := r.ground(p)
	if err != nil {
		return nil, err
	}
	for _, in := range insts {
		k := in.head.String()
		r.rulesFor[k] = append(r.rulesFor[k], in)
		r.note(in.head)
		for _, b := range in.body {
			r.note(b)
		}
	}
	r.resolve()
	return r, nil
}

func (r *Result) note(a Atom) {
	r.atoms[a.String()] = a
	c := a.Complement()
	r.atoms[c.String()] = c
}

// direct reads the defeat relation as given. It is not transitive: a chain
// r_status > r_status_out > d_a4 > r_strip speaks of three different conflicts
// and must not order r_status over r_strip.
func direct(defeats []Defeat) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, d := range defeats {
		if out[d.Over] == nil {
			out[d.Over] = map[string]bool{}
		}
		out[d.Over][d.Under] = true
	}
	return out
}

type binding map[string]Value

func (b binding) copy() binding {
	c := make(binding, len(b))
	for k, v := range b {
		c[k] = v
	}
	return c
}

func (r *Result) value(t Term, b binding) (Value, bool) {
	switch t.Kind {
	case Const:
		return C(t.Name), true
	case Num:
		return N(t.Num), true
	case Now:
		return N(r.now), true
	case Var:
		v, ok := b[t.Name]
		return v, ok && !t.Anonymous()
	}
	return Value{}, false
}

func (r *Result) match(t Term, v Value, b binding) bool {
	if t.Anonymous() {
		return true
	}
	if t.Kind == Var {
		if bound, ok := b[t.Name]; ok {
			return bound == v
		}
		b[t.Name] = v
		return true
	}
	want, ok := r.value(t, b)
	return ok && want == v
}

// builtins evaluates comparisons and arithmetic once their inputs are bound.
func (r *Result) builtins(ls []Literal, b binding) bool {
	pending := append([]Literal{}, ls...)
	for len(pending) > 0 {
		var rest []Literal
		for _, l := range pending {
			x, okx := r.value(l.Args[0], b)
			y, oky := r.value(l.Args[1], b)
			if !okx || !oky {
				rest = append(rest, l)
				continue
			}
			if !x.IsNum || !y.IsNum {
				return false
			}
			switch l.Pred {
			case "<":
				if !(x.Num < y.Num) {
					return false
				}
			case "=<":
				if !(x.Num <= y.Num) {
					return false
				}
			case ">":
				if !(x.Num > y.Num) {
					return false
				}
			case ">=":
				if !(x.Num >= y.Num) {
					return false
				}
			case "plus", "minus":
				out := x.Num + y.Num
				if l.Pred == "minus" {
					out = x.Num - y.Num
				}
				if !r.match(l.Args[2], N(out), b) {
					return false
				}
			}
		}
		if len(rest) == len(pending) {
			return false
		}
		pending = rest
	}
	return true
}

func (r *Result) ground(p Program) ([]*instance, error) {
	index := map[string][]Atom{}
	possible := map[string]bool{}
	addAtom := func(a Atom) bool {
		k := a.String()
		if possible[k] {
			return false
		}
		possible[k] = true
		index[fmt.Sprint(a.Neg, a.Pred)] = append(index[fmt.Sprint(a.Neg, a.Pred)], a)
		return true
	}
	for _, f := range p.Facts {
		r.facts[f.String()] = true
		r.note(f)
		addAtom(f)
	}

	seen := map[string]bool{}
	var insts []*instance
	for round := 0; ; round++ {
		if round > 1000 {
			return nil, fmt.Errorf("grounding did not reach a fixpoint")
		}
		changed := false
		for _, rule := range p.Rules {
			var preds, builtins []Literal
			for _, l := range rule.Body {
				if l.Builtin() {
					builtins = append(builtins, l)
				} else {
					preds = append(preds, l)
				}
			}
			var rec func(i int, b binding, matched []Atom)
			rec = func(i int, b binding, matched []Atom) {
				if i < len(preds) {
					l := preds[i]
					for _, a := range index[fmt.Sprint(l.Neg, l.Pred)] {
						if len(a.Args) != len(l.Args) {
							continue
						}
						nb := b.copy()
						ok := true
						for j, t := range l.Args {
							if !r.match(t, a.Args[j], nb) {
								ok = false
								break
							}
						}
						if ok {
							rec(i+1, nb, append(append([]Atom{}, matched...), a))
						}
					}
					return
				}
				if !r.builtins(builtins, b) {
					return
				}
				head := Atom{Pred: rule.Head.Pred, Neg: rule.Head.Neg}
				for _, t := range rule.Head.Args {
					v, ok := r.value(t, b)
					if !ok {
						return
					}
					head.Args = append(head.Args, v)
				}
				key := rule.ID + "|" + head.String()
				for _, m := range matched {
					key += "|" + m.String()
				}
				if seen[key] {
					return
				}
				seen[key] = true
				insts = append(insts, &instance{rule: rule, head: head, body: matched})
				if rule.Strength != Defeater && addAtom(head) {
					changed = true
				}
			}
			rec(0, binding{}, nil)
		}
		if !changed {
			return insts, nil
		}
	}
}

func (r *Result) applicable(in *instance) bool {
	for _, b := range in.body {
		if !r.plus[b.String()] {
			return false
		}
	}
	return true
}

func (r *Result) discarded(in *instance) bool {
	for _, b := range in.body {
		if r.minus[b.String()] {
			return true
		}
	}
	return false
}

func (r *Result) beats(a, b string) bool { return r.stronger[a][b] }

func (r *Result) strictApplicable(key string) bool {
	if r.facts[key] {
		return true
	}
	for _, in := range r.rulesFor[key] {
		if in.rule.Strength == Strict && r.applicable(in) {
			return true
		}
	}
	return false
}

// resolve tags every literal proved or refuted until nothing changes. Both
// sets only grow, so the order of rules does not matter.
func (r *Result) resolve() {
	for k := range r.facts {
		r.plus[k] = true
	}
	keys := make([]string, 0, len(r.atoms))
	for k := range r.atoms {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for changed := true; changed; {
		changed = false
		for _, key := range keys {
			if r.plus[key] || r.minus[key] {
				continue
			}
			comp := r.atoms[key].Complement().String()
			var proof *instance
			for _, in := range r.rulesFor[key] {
				if in.rule.Strength == Strict && r.applicable(in) {
					proof = in
					break
				}
			}
			if proof == nil && !r.strictApplicable(comp) {
				for _, in := range r.rulesFor[key] {
					if in.rule.Strength != Defeasible || !r.applicable(in) {
						continue
					}
					wins := true
					for _, s := range r.rulesFor[comp] {
						if !r.discarded(s) && !r.beats(in.rule.ID, s.rule.ID) {
							wins = false
							break
						}
					}
					if wins {
						proof = in
						break
					}
				}
			}
			if proof != nil {
				r.plus[key] = true
				r.proof[key] = proof
				changed = true
				continue
			}
			refuted := true
			if !r.strictApplicable(comp) {
				for _, in := range r.rulesFor[key] {
					if in.rule.Strength == Defeater || r.discarded(in) {
						continue
					}
					beaten := false
					for _, s := range r.rulesFor[comp] {
						if r.applicable(s) && !r.beats(in.rule.ID, s.rule.ID) {
							beaten = true
							break
						}
					}
					if !beaten {
						refuted = false
						break
					}
				}
			}
			if refuted {
				r.minus[key] = true
				changed = true
			}
		}
	}
}

// Query answers whether an atom holds in the slice.
func (r *Result) Query(a Atom) Answer {
	key, comp := a.String(), a.Complement().String()
	if _, known := r.atoms[key]; !known {
		return Answer{Outcome: Unknown, Reason: Gap}
	}
	switch {
	case r.plus[key]:
		return Answer{Outcome: Proved, Trace: r.trace(key, map[string]bool{})}
	case r.plus[comp]:
		return Answer{Outcome: Disproved, Trace: r.trace(comp, map[string]bool{})}
	case !r.minus[key] || !r.minus[comp]:
		return Answer{Outcome: Unknown, Reason: Undetermined}
	}
	var opposing []string
	sides := [2][]*instance{}
	concluding := false
	for i, k := range []string{key, comp} {
		for _, in := range r.rulesFor[k] {
			if r.applicable(in) {
				sides[i] = append(sides[i], in)
				opposing = append(opposing, in.rule.ID)
				if in.rule.Strength != Defeater {
					concluding = true
				}
			}
		}
	}
	sort.Strings(opposing)
	if !concluding {
		return Answer{Outcome: Unknown, Reason: Gap, Opposing: opposing}
	}
	for _, x := range sides[0] {
		for _, y := range sides[1] {
			if x.rule.Strength == Defeater && y.rule.Strength == Defeater {
				continue
			}
			if !r.beats(x.rule.ID, y.rule.ID) && !r.beats(y.rule.ID, x.rule.ID) {
				return Answer{Outcome: Unknown, Reason: Deadlock, Opposing: opposing}
			}
		}
	}
	return Answer{Outcome: Unknown, Reason: Blocked, Opposing: opposing}
}

func (r *Result) trace(key string, visiting map[string]bool) *Trace {
	t := &Trace{Atom: r.atoms[key]}
	if r.facts[key] {
		t.Rule = "fact"
		return t
	}
	in := r.proof[key]
	if in == nil || visiting[key] {
		t.Rule = "?"
		return t
	}
	visiting[key] = true
	defer delete(visiting, key)
	t.Rule = in.rule.ID
	for _, b := range in.body {
		t.Premises = append(t.Premises, r.trace(b.String(), visiting))
	}
	for _, s := range r.rulesFor[r.atoms[key].Complement().String()] {
		if r.applicable(s) {
			t.Defeated = append(t.Defeated, s.rule.ID)
		}
	}
	sort.Strings(t.Defeated)
	return t
}

// Conclusions lists every proved literal that is not a stored fact.
func (r *Result) Conclusions() []Atom {
	var out []Atom
	for k := range r.plus {
		if !r.facts[k] {
			out = append(out, r.atoms[k])
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}
