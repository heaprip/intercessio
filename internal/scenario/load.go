// Package scenario reads the world data of a game and checks its form.
//
// Prototype. A scenario is two files: schema.json with the predicate schema
// and scenario.json with the world, norms, defeats and facts.
package scenario

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"unicode"

	"github.com/heaprip/intercessio/internal/corpus"
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/facts"
	"github.com/heaprip/intercessio/internal/period"
)

// Scenario is the initial value of a game.
type Scenario struct {
	StartPeriod period.Period
	Corpus      corpus.Version
	Facts       []facts.Fact
	Roles       map[string]deduction.Role
}

// Diagnostic is the loader's own report. Prototype: same shape as deduction's.
type Diagnostic = deduction.Diagnostic

// Report splits diagnostics into errors and findings.
type Report struct {
	Errors   []Diagnostic
	Findings []Diagnostic
}

func (r *Report) String() string {
	var b strings.Builder
	for _, d := range append(append([]Diagnostic{}, r.Errors...), r.Findings...) {
		fmt.Fprintf(&b, "%-7s %-26s %-28s %s\n", d.Severity, d.Code, d.Subject, d.Message)
	}
	return b.String()
}

type rawSchema struct {
	Predicates []struct {
		Name  string `json:"name"`
		Arity int    `json:"arity"`
		Role  string `json:"role"`
		None  []int  `json:"none"`
	} `json:"predicates"`
	Constants   []string `json:"constants"`
	ActionKinds []string `json:"action_kinds"`
	DutyKinds   []string `json:"duty_kinds"`
	// Acts say which fact predicates record acts of offices, by which action
	// kinds, and which action kinds stop them.
	Acts []struct {
		Predicate  string            `json:"predicate"`
		Kinds      []string          `json:"kinds"`
		StoppedBy  []string          `json:"stopped_by"`
		OutcomeArg int               `json:"outcome_arg"`
		Outcomes   map[string]string `json:"outcomes"`
	} `json:"acts"`
}

type rawLit struct {
	P   string            `json:"p"`
	A   []json.RawMessage `json:"a"`
	Neg bool              `json:"neg"`
}

type rawRule struct {
	ID   string   `json:"id"`
	S    string   `json:"s"`
	Head rawLit   `json:"head"`
	Body []rawLit `json:"body"`
}

type rawFact struct {
	ID     string            `json:"id"`
	P      string            `json:"p"`
	A      []json.RawMessage `json:"a"`
	By     string            `json:"by"`
	Period int               `json:"period"`
}

type rawScenario struct {
	StartPeriod int `json:"start_period"`
	// WorldNorm names the norm that states the world: bundles of statuses and
	// declarations of offices go into its declarations. Without it they are
	// facts.
	WorldNorm string `json:"world_norm"`
	Statuses  []struct {
		ID        string    `json:"id"`
		Rights    *[]string `json:"rights"`
		Duties    *[]string `json:"duties"`
		Heritable *bool     `json:"heritable"`
	} `json:"statuses"`
	Offices []struct {
		ID                string   `json:"id"`
		Competences       []string `json:"competences"`
		Term              int      `json:"term"`
		RequiresRights    []string `json:"requires_rights"`
		RequiresPrior     []string `json:"requires_prior"`
		MinAge            *int     `json:"min_age"`
		Incompatible      []string `json:"incompatible"`
		IterationBarred   bool     `json:"iteration_barred"`
		ConsecutiveBarred bool     `json:"consecutive_barred"`
		Capacity          *int     `json:"capacity"`
		// Reviews lists the offices whose acts this office reviews.
		Reviews []string `json:"reviews"`
		// Quorum is how many holders must concur for an act of the office.
		Quorum *int `json:"quorum"`
	} `json:"offices"`
	People []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Born   *int   `json:"born"`
	} `json:"people"`
	Documents []struct {
		ID    string `json:"id"`
		Level int    `json:"level"`
	} `json:"documents"`
	Norms []struct {
		ID       string    `json:"id"`
		Document string    `json:"document"`
		Enacted  int       `json:"enacted"`
		InForce  int       `json:"in_force"`
		Kind     string    `json:"kind"`
		Rules    []rawRule `json:"rules"`
	} `json:"norms"`
	Defeats []struct {
		Over  string `json:"over"`
		Under string `json:"under"`
	} `json:"defeats"`
	Declarations []rawFact `json:"declarations"`
	Facts        []rawFact `json:"facts"`
}

// Load reads schema.json and scenario.json from fsys. A scenario with errors
// is still returned: the report says what cannot be evaluated.
func Load(fsys fs.FS) (*Scenario, *Report, error) {
	var sch rawSchema
	if err := readJSON(fsys, "schema.json", &sch); err != nil {
		return nil, nil, err
	}
	var raw rawScenario
	if err := readJSON(fsys, "scenario.json", &raw); err != nil {
		return nil, nil, err
	}
	l := &loader{
		roles: map[string]deduction.Role{},
		arity: map[string]int{},
		none:  map[string]map[int]bool{},
		known: map[string]bool{"none": true},
	}
	s, err := l.run(sch, raw)
	if err != nil {
		return nil, nil, err
	}
	rep := &Report{}
	deduction.Sort(l.diags)
	for _, d := range l.diags {
		if d.Severity == deduction.Error {
			rep.Errors = append(rep.Errors, d)
		} else {
			rep.Findings = append(rep.Findings, d)
		}
	}
	return s, rep, nil
}

func readJSON(fsys fs.FS, name string, v any) error {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("decode %s: %w", name, err)
	}
	return nil
}

type loader struct {
	roles map[string]deduction.Role
	arity map[string]int
	none  map[string]map[int]bool
	known map[string]bool
	diags []Diagnostic
}

func (l *loader) add(code string, sev deduction.Severity, subject, format string, args ...any) {
	l.diags = append(l.diags, Diagnostic{Code: code, Severity: sev, Subject: subject, Message: fmt.Sprintf(format, args...)})
}

func (l *loader) declare(name string, arity int, role deduction.Role) {
	l.roles[name] = role
	l.arity[name] = arity
}

func (l *loader) run(sch rawSchema, raw rawScenario) (*Scenario, error) {
	for _, p := range sch.Predicates {
		l.declare(p.Name, p.Arity, deduction.Role(p.Role))
		l.none[p.Name] = map[int]bool{}
		for _, i := range p.None {
			l.none[p.Name][i] = true
		}
	}
	for _, d := range []string{"person", "office", "status_kind", "right", "action_kind", "duty_kind"} {
		l.declare(d, 1, deduction.RoleDomain)
	}
	for _, list := range [][]string{sch.Constants, sch.ActionKinds, sch.DutyKinds} {
		for _, c := range list {
			l.known[c] = true
		}
	}

	s := &Scenario{StartPeriod: period.Period(raw.StartPeriod), Roles: l.roles}
	var worldDecl []deduction.Atom
	gen := func(pred string, args ...any) {
		f := facts.Fact{ID: pred, Pred: pred, Prov: facts.Provenance{By: "scenario", Source: "world"}}
		a := deduction.Atom{Pred: pred}
		for _, arg := range args {
			switch x := arg.(type) {
			case int:
				f.Args = append(f.Args, facts.Value{Num: x, IsNum: true})
				a.Args = append(a.Args, deduction.N(x))
			case string:
				f.Args = append(f.Args, facts.Value{Const: x})
				a.Args = append(a.Args, deduction.C(x))
			}
		}
		if raw.WorldNorm != "" && l.roles[pred] == deduction.RoleDeclaration {
			l.checkAtom(pred, pred, len(a.Args), true)
			worldDecl = append(worldDecl, a)
			return
		}
		s.Facts = append(s.Facts, f)
	}
	for _, k := range sch.ActionKinds {
		gen("action_kind", k)
	}
	for _, k := range sch.DutyKinds {
		gen("duty_kind", k)
	}

	rights := map[string]bool{}
	for _, st := range raw.Statuses {
		l.known[st.ID] = true
		gen("status_kind", st.ID)
		if st.Rights == nil || st.Duties == nil || st.Heritable == nil {
			l.add("incomplete-status", deduction.Error, st.ID, "status must declare rights, duties and heritable")
		}
		if st.Rights != nil {
			for _, r := range *st.Rights {
				if !rights[r] {
					rights[r] = true
					l.known[r] = true
					gen("right", r)
				}
				gen("bundle", st.ID, r)
			}
		}
		if st.Duties != nil {
			for _, d := range *st.Duties {
				gen("duty_bundle", st.ID, d)
			}
		}
		if st.Heritable != nil && *st.Heritable {
			gen("heritable", st.ID)
		}
	}
	requiredRights := map[string]bool{}
	for _, o := range raw.Offices {
		l.known[o.ID] = true
		gen("office", o.ID)
		for _, k := range o.Competences {
			gen("responsibility", o.ID, k)
		}
		for _, r := range o.RequiresRights {
			requiredRights[r] = true
			gen("requires_right", o.ID, r)
		}
		for _, p := range o.RequiresPrior {
			gen("requires_prior", o.ID, p)
		}
		if o.MinAge != nil {
			gen("min_age", o.ID, *o.MinAge)
		}
		for _, x := range o.Incompatible {
			// incompatibility is symmetric; the world declares it once
			gen("incompatible", o.ID, x)
			gen("incompatible", x, o.ID)
		}
		if o.IterationBarred {
			gen("iteration_barred", o.ID)
		}
		if o.ConsecutiveBarred {
			gen("consecutive_barred", o.ID)
		}
		if o.Capacity != nil {
			gen("capacity", o.ID, *o.Capacity)
		}
		for _, x := range o.Reviews {
			gen("reviews", o.ID, x)
		}
		if o.Quorum != nil {
			gen("quorum", o.ID, *o.Quorum)
		}
	}
	for _, p := range raw.People {
		l.known[p.ID] = true
		gen("person", p.ID)
		if p.Status != "" {
			gen("registered", p.ID, p.Status)
		}
		if p.Born != nil {
			gen("born", p.ID, *p.Born)
		}
	}
	for _, f := range s.Facts {
		l.checkAtom(f.ID, f.Pred, len(f.Args), true)
	}

	for _, group := range []struct {
		items  []rawFact
		source string
	}{{raw.Declarations, "declaration"}, {raw.Facts, "fact"}} {
		for i, rf := range group.items {
			id := rf.ID
			if id == "" {
				id = fmt.Sprintf("%s#%d", group.source, i+1)
			}
			f := facts.Fact{ID: id, Pred: rf.P, Prov: facts.Provenance{By: rf.By, Period: period.Period(rf.Period), Source: group.source}}
			ok := l.checkAtom(id, rf.P, len(rf.A), true)
			for _, a := range rf.A {
				t, err := decodeTerm(a)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", id, err)
				}
				switch t.Kind {
				case deduction.Const:
					l.known[t.Name] = true
					f.Args = append(f.Args, facts.Value{Const: t.Name})
				case deduction.Num:
					f.Args = append(f.Args, facts.Value{Num: t.Num, IsNum: true})
				default:
					ok = false
					l.add("invalid-term", deduction.Error, id, "fact argument %s is not a constant or number", t)
				}
			}
			if ok {
				s.Facts = append(s.Facts, f)
			}
		}
	}

	for _, d := range raw.Documents {
		l.known[d.ID] = true
		s.Corpus.Documents = append(s.Corpus.Documents, corpus.Document{ID: d.ID, Level: d.Level})
	}
	seenRule := map[string]string{}
	var rules []deduction.Rule
	for _, rn := range raw.Norms {
		l.known[rn.ID] = true
	}
	for _, rn := range raw.Norms {
		n := corpus.Norm{ID: rn.ID, Document: rn.Document, Enacted: period.Period(rn.Enacted), InForce: period.Period(rn.InForce), Kind: corpus.Kind(rn.Kind)}
		if !l.known[rn.Document] {
			l.add("unknown-reference", deduction.Error, rn.ID, "document %s does not exist", rn.Document)
		}
		if rn.InForce < rn.Enacted {
			l.add("retroactivity", deduction.Finding, rn.ID, "in force from %d, enacted in %d", rn.InForce, rn.Enacted)
		}
		for _, rr := range rn.Rules {
			r, err := decodeRule(rr)
			if err != nil {
				return nil, fmt.Errorf("rule %s: %w", rr.ID, err)
			}
			if prev, ok := seenRule[r.ID]; ok {
				if prev != r.String() {
					l.add("duplicate-rule-id", deduction.Error, r.ID, "two different rules share the id")
				}
				continue
			}
			seenRule[r.ID] = r.String()
			n.Rules = append(n.Rules, r)
			rules = append(rules, r)
		}
		s.Corpus.Norms = append(s.Corpus.Norms, n)
	}
	if raw.WorldNorm != "" {
		attached := false
		for i := range s.Corpus.Norms {
			if s.Corpus.Norms[i].ID == raw.WorldNorm {
				s.Corpus.Norms[i].Declarations = append(s.Corpus.Norms[i].Declarations, worldDecl...)
				attached = true
			}
		}
		if !attached {
			l.add("unknown-reference", deduction.Error, raw.WorldNorm, "world norm %s does not exist", raw.WorldNorm)
		}
	}
	for _, d := range raw.Defeats {
		s.Corpus.Defeats = append(s.Corpus.Defeats, deduction.Defeat{Over: d.Over, Under: d.Under})
	}
	kinds := map[string]bool{}
	for _, k := range sch.ActionKinds {
		kinds[k] = true
	}
	for _, a := range sch.Acts {
		if role, ok := l.roles[a.Predicate]; !ok || role != deduction.RoleFact {
			l.add("unknown-reference", deduction.Error, a.Predicate, "act predicate %s is not a declared fact", a.Predicate)
		}
		for _, k := range append(append([]string{}, a.Kinds...), a.StoppedBy...) {
			if !kinds[k] {
				l.add("unknown-reference", deduction.Error, a.Predicate, "action kind %s is not declared", k)
			}
		}
		for outcome, k := range a.Outcomes {
			found := false
			for _, x := range a.Kinds {
				found = found || x == k
			}
			if !found {
				l.add("unknown-reference", deduction.Error, a.Predicate, "outcome %s names kind %s the act does not have", outcome, k)
			}
		}
		if len(a.Outcomes) > 0 && (a.OutcomeArg < 0 || a.OutcomeArg >= l.arity[a.Predicate]) {
			l.add("unknown-reference", deduction.Error, a.Predicate, "outcome argument %d is outside %s/%d", a.OutcomeArg, a.Predicate, l.arity[a.Predicate])
		}
		s.Corpus.Acts = append(s.Corpus.Acts, corpus.Act{Predicate: a.Predicate, Kinds: a.Kinds, StoppedBy: a.StoppedBy, OutcomeArg: a.OutcomeArg, Outcomes: a.Outcomes})
	}

	usedConst := map[string]bool{}
	for _, r := range rules {
		lits := append([]deduction.Literal{r.Head}, r.Body...)
		for i, lt := range lits {
			for _, t := range lt.Args {
				walkConsts(t, func(name string) {
					if i > 0 {
						usedConst[name] = true
					}
					if !l.known[name] {
						l.add("unknown-reference", deduction.Error, r.ID, "constant %s is not declared", name)
					}
				})
			}
			if lt.Builtin() {
				continue
			}
			l.checkAtom(r.ID, lt.Pred, len(lt.Args), false)
			for j, t := range lt.Args {
				if t.Kind == deduction.Const && t.Name == "none" && !l.none[lt.Pred][j] {
					l.add("invalid-term", deduction.Error, r.ID, "none is not allowed at %s/%d", lt.Pred, j)
				}
			}
		}
	}
	var rightNames []string
	for r := range rights {
		rightNames = append(rightNames, r)
	}
	sort.Strings(rightNames)
	for _, r := range rightNames {
		if !usedConst[r] && !requiredRights[r] {
			l.add("unconsumed-right", deduction.Finding, r, "right %s enters no condition", r)
		}
	}

	l.diags = append(l.diags, deduction.Check(rules, s.Corpus.Defeats, l.roles)...)
	return s, nil
}

// checkAtom reports undeclared predicates and arity mismatches. stored says
// whether the atom is data rather than a rule literal.
func (l *loader) checkAtom(subject, pred string, arity int, stored bool) bool {
	role, ok := l.roles[pred]
	if !ok {
		l.add("undeclared-predicate", deduction.Error, subject, "predicate %s/%d is not declared", pred, arity)
		return false
	}
	if l.arity[pred] != arity {
		l.add("arity-mismatch", deduction.Error, subject, "%s used with %d arguments, declared %d", pred, arity, l.arity[pred])
		return false
	}
	if stored && !role.Stored() {
		l.add("fact-and-conclusion", deduction.Finding, subject, "%s is %s but given as data", pred, role)
	}
	return true
}

func walkConsts(t deduction.Term, f func(string)) {
	switch t.Kind {
	case deduction.Const:
		f(t.Name)
	case deduction.Expr:
		for _, a := range t.Args {
			walkConsts(a, f)
		}
	case deduction.Compound:
		for _, a := range t.Args {
			walkConsts(a, f)
		}
	}
}

func decodeRule(rr rawRule) (deduction.Rule, error) {
	r := deduction.Rule{ID: rr.ID, Strength: deduction.Defeasible}
	switch rr.S {
	case "", "defeasible":
	case "strict":
		r.Strength = deduction.Strict
	case "defeater":
		r.Strength = deduction.Defeater
	default:
		return r, fmt.Errorf("unknown strength %q", rr.S)
	}
	var err error
	if r.Head, err = decodeLit(rr.Head); err != nil {
		return r, err
	}
	for _, b := range rr.Body {
		lt, err := decodeLit(b)
		if err != nil {
			return r, err
		}
		r.Body = append(r.Body, lt)
	}
	return r, nil
}

func decodeLit(rl rawLit) (deduction.Literal, error) {
	lt := deduction.Literal{Pred: rl.P, Neg: rl.Neg}
	for _, a := range rl.A {
		t, err := decodeTerm(a)
		if err != nil {
			return lt, fmt.Errorf("%s: %w", rl.P, err)
		}
		lt.Args = append(lt.Args, t)
	}
	return lt, nil
}

func decodeTerm(raw json.RawMessage) (deduction.Term, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		switch {
		case s == "":
			return deduction.Term{}, fmt.Errorf("empty term")
		case s == "now":
			return deduction.Term{Kind: deduction.Now}, nil
		case s == "_" || unicode.IsUpper([]rune(s)[0]):
			return deduction.Term{Kind: deduction.Var, Name: s}, nil
		default:
			return deduction.Term{Kind: deduction.Const, Name: s}, nil
		}
	}
	var num int
	if err := json.Unmarshal(raw, &num); err == nil {
		return deduction.Term{Kind: deduction.Num, Num: num}, nil
	}
	var obj struct {
		F    string            `json:"f"`
		Args []json.RawMessage `json:"args"`
		Op   string            `json:"op"`
		L    json.RawMessage   `json:"l"`
		R    json.RawMessage   `json:"r"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return deduction.Term{}, fmt.Errorf("term %s: %w", raw, err)
	}
	switch {
	case obj.F != "":
		t := deduction.Term{Kind: deduction.Compound, Name: obj.F}
		for _, a := range obj.Args {
			x, err := decodeTerm(a)
			if err != nil {
				return t, err
			}
			t.Args = append(t.Args, x)
		}
		return t, nil
	case obj.Op != "":
		l, err := decodeTerm(obj.L)
		if err != nil {
			return deduction.Term{}, err
		}
		r, err := decodeTerm(obj.R)
		if err != nil {
			return deduction.Term{}, err
		}
		return deduction.Term{Kind: deduction.Expr, Name: obj.Op, Args: []deduction.Term{l, r}}, nil
	}
	return deduction.Term{}, fmt.Errorf("term %s is neither a compound nor an expression", raw)
}
