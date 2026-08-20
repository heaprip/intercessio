// Package deduction holds the rule language and the checks over it.
//
// Prototype: the evaluator is not written yet; only the types and the checks
// the scenario loader needs.
package deduction

import (
	"fmt"
	"strings"
)

// TermKind tells what a term is.
type TermKind int

const (
	Var TermKind = iota
	Const
	Num
	Now
	// Compound and Expr are invalid by the rule-form convention. They exist so
	// that a loader can represent the input and report it instead of failing.
	Compound
	Expr
)

// Term is an argument of a literal.
type Term struct {
	Kind TermKind
	Name string // variable, constant, functor or operator
	Num  int
	Args []Term
}

func (t Term) String() string {
	switch t.Kind {
	case Var, Const:
		return t.Name
	case Num:
		return fmt.Sprint(t.Num)
	case Now:
		return "now"
	case Compound:
		return t.Name + "(" + joinTerms(t.Args) + ")"
	case Expr:
		return t.Args[0].String() + " " + t.Name + " " + t.Args[1].String()
	}
	return "?"
}

// Anonymous reports whether the term is the anonymous variable.
func (t Term) Anonymous() bool { return t.Kind == Var && t.Name == "_" }

func joinTerms(ts []Term) string {
	s := make([]string, len(ts))
	for i, t := range ts {
		s[i] = t.String()
	}
	return strings.Join(s, ", ")
}

// Literal is a predicate applied to terms, possibly negated. Negation is a
// conclusion about absence, not negation as failure.
type Literal struct {
	Pred string
	Args []Term
	Neg  bool
}

var builtinArity = map[string]int{"<": 2, "=<": 2, ">": 2, ">=": 2, "plus": 3, "minus": 3}

// Builtin reports whether the literal is a built-in comparison or arithmetic.
func (l Literal) Builtin() bool { _, ok := builtinArity[l.Pred]; return ok }

// Arithmetic reports whether the literal is plus or minus.
func (l Literal) Arithmetic() bool { return l.Pred == "plus" || l.Pred == "minus" }

func (l Literal) String() string {
	if l.Builtin() && !l.Arithmetic() && len(l.Args) == 2 {
		return l.Args[0].String() + " " + l.Pred + " " + l.Args[1].String()
	}
	s := l.Pred + "(" + joinTerms(l.Args) + ")"
	if l.Neg {
		s = "¬" + s
	}
	return s
}

// Strength of a rule.
type Strength string

const (
	Strict     Strength = "strict"
	Defeasible Strength = "defeasible"
	Defeater   Strength = "defeater"
)

// Rule is a head, a body and a strength.
type Rule struct {
	ID       string
	Head     Literal
	Body     []Literal
	Strength Strength
}

func (r Rule) String() string {
	body := make([]string, len(r.Body))
	for i, l := range r.Body {
		body[i] = l.String()
	}
	arrow := " ⇒ "
	if r.Strength == Defeater {
		arrow = " ⇝ "
	}
	return strings.Join(body, ", ") + arrow + r.Head.String()
}

// Defeat declares that rule Over beats rule Under.
type Defeat struct{ Over, Under string }

// Role of a predicate in the schema.
type Role string

const (
	RoleFact        Role = "fact"
	RoleDeclaration Role = "declaration"
	RoleDomain      Role = "domain"
	RoleDerived     Role = "derived"
	RoleOutput      Role = "output"
)

// Stored reports whether the predicate is stored rather than concluded.
func (r Role) Stored() bool { return r == RoleFact || r == RoleDeclaration || r == RoleDomain }

// Severity of a diagnostic.
type Severity string

const (
	// Error means evaluation is impossible.
	Error Severity = "error"
	// Finding means the corpus loads and is a bad constitution.
	Finding Severity = "finding"
)

// Diagnostic is a domain-free report about rules.
type Diagnostic struct {
	Code     string
	Severity Severity
	Subject  string
	Message  string
}
