// Package api gives a game to its one consumer, the player's front: start a
// game from a scenario, look at a period, decide the cards of the stack, live
// the period, ask for the trace of a conclusion.
//
// Prototype. Plain HTTP with JSON on the standard library, games in memory.
// Transport types are views built here; the domain does not know them.
package api

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/heaprip/intercessio/internal/actors"
	"github.com/heaprip/intercessio/internal/agenda"
	"github.com/heaprip/intercessio/internal/censor"
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/entitlement"
	"github.com/heaprip/intercessio/internal/journal"
	"github.com/heaprip/intercessio/internal/linter"
	"github.com/heaprip/intercessio/internal/period"
	"github.com/heaprip/intercessio/internal/powergraph"
	"github.com/heaprip/intercessio/internal/report"
	"github.com/heaprip/intercessio/internal/scenario"
	"github.com/heaprip/intercessio/internal/turn"
)

// Game is one game in progress. Its methods are safe for concurrent use.
type Game struct {
	mu       sync.Mutex
	id       string
	scenario *scenario.Scenario
	cfg      turn.Config
	state    turn.State
	memory   agenda.Memory
	preset   agenda.Preset
	stand    censor.Preset
	lived    []View
	pending  pending
	previous *linter.Report
	last     []journal.Entry
}

type pending struct {
	lint   *linter.Report
	stack  agenda.Stack
	stand  *censor.Stand
	graph  *powergraph.Graph
	chosen map[string]string // card key -> accept or reject
}

// NewGame starts a game at the scenario's start period with stub participants.
func NewGame(id string, s *scenario.Scenario) (*Game, error) {
	g := &Game{
		id:       id,
		scenario: s,
		cfg:      turn.Config{Strategy: entitlement.Hierarchy{}, Actors: actors.Stub{}},
		state:    turn.State{Period: s.StartPeriod, Corpus: s.Corpus, Facts: s.Facts, Seed: 7},
		memory:   agenda.Memory{},
		preset: agenda.Preset{Size: 3, Return: 3, Weights: map[string]int{
			"retroactivity": 5, "taking-of-vested": 5, "usurpation": 5, "judge-in-own-cause": 4,
			"circumventable-condition": 4, "unchecked-act": 4, "indeterminacy": 2,
		}},
		stand: censor.Preset{NonLiquet: 0.3, Silent: 3},
	}
	if err := g.prepare(); err != nil {
		return nil, err
	}
	return g, nil
}

// prepare lints the corpus of the coming period and builds its stack.
func (g *Game) prepare() error {
	st := g.state
	lint, err := linter.Lint(linter.Input{Corpus: st.Corpus, Roles: g.scenario.Roles, Facts: st.Facts, Strategy: g.cfg.Strategy, Now: st.Period})
	if err != nil {
		return err
	}
	graph, err := powergraph.Build(g.cfg.Strategy, st.Corpus, st.Facts, st.Period)
	if err != nil {
		return err
	}
	p := pending{lint: lint, graph: graph, chosen: map[string]string{}}
	var findings []linter.Finding
	if st.Period > g.scenario.StartPeriod {
		to := st.Period - 1
		from := to - 2
		if from < g.scenario.StartPeriod {
			from = g.scenario.StartPeriod
		}
		lastGraph, err := powergraph.Build(g.cfg.Strategy, st.Corpus, st.Facts, to)
		if err != nil {
			return err
		}
		stand := censor.Build(censor.Input{Journal: st.Journal, From: from, To: to, Graph: lastGraph, Corpus: st.Corpus,
			People: powergraph.Domain(powergraph.Stored(st.Corpus, st.Facts, to), "person"), Preset: g.stand})
		p.stand = &stand
		findings = stand.Findings
	}
	p.stack = agenda.Build(agenda.Input{Lint: lint, Practice: findings, Entries: g.last, Corpus: st.Corpus, Now: st.Period, Memory: g.memory, Preset: g.preset})
	g.pending = p
	return nil
}

// View is what the front shows of one period: the stack to decide before it is
// lived, and after it is lived what happened.
type View struct {
	Game     string        `json:"game"`
	Period   int           `json:"period"`
	Lived    bool          `json:"lived"`
	Stack    []CardView    `json:"stack"`
	Overflow []CardView    `json:"overflow"`
	Held     int           `json:"held"`
	Findings []FindingView `json:"findings"`
	Stand    []AxisView    `json:"stand"`
	Graph    GraphView     `json:"graph"`
	Report   []SectionView `json:"report"`
	Journal  []string      `json:"journal"`
	Current  int           `json:"current"`
}

// CardView is a card with the auctor's choice, if made.
type CardView struct {
	Key       string   `json:"key"`
	Failure   string   `json:"failure"`
	Root      string   `json:"root"`
	Reach     int      `json:"reach"`
	Score     int      `json:"score"`
	Proposal  string   `json:"proposal"`
	Amendment bool     `json:"amendment"`
	People    []string `json:"people"`
	Evidence  []string `json:"evidence"`
	Reminder  bool     `json:"reminder"`
	Persists  bool     `json:"persists"`
	Choice    string   `json:"choice"`
}

// FindingView is one finding of the linter or the censor.
type FindingView struct {
	Failure string   `json:"failure"`
	Channel string   `json:"channel"`
	Place   string   `json:"place"`
	Message string   `json:"message"`
	People  []string `json:"people"`
}

// AxisView is one axis of the stand.
type AxisView struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Of    string  `json:"of"`
}

// GraphView is the power graph of the period.
type GraphView struct {
	Offices []OfficeView `json:"offices"`
	Edges   []EdgeView   `json:"edges"`
}

// OfficeView is a node.
type OfficeView struct {
	ID          string   `json:"id"`
	Holders     []string `json:"holders"`
	Competences []string `json:"competences"`
}

// EdgeView is an edge other than competence.
type EdgeView struct {
	Kind   string `json:"kind"`
	From   string `json:"from"`
	To     string `json:"to"`
	Via    string `json:"via"`
	Active bool   `json:"active"`
}

// SectionView is a section of the report.
type SectionView struct {
	Title string   `json:"title"`
	Lines []string `json:"lines"`
}

func cards(cs []agenda.Card, chosen map[string]string) []CardView {
	out := []CardView{}
	for _, c := range cs {
		out = append(out, CardView{Key: c.Key, Failure: c.Failure, Root: c.Root, Reach: c.Reach, Score: c.Score,
			Proposal: c.Proposal.Summary, Amendment: len(c.Proposal.Amendments) > 0, People: c.People, Evidence: c.Evidence,
			Reminder: c.Reminder, Persists: c.Persists, Choice: chosen[c.Key]})
	}
	return out
}

func graphView(g *powergraph.Graph) GraphView {
	out := GraphView{Offices: []OfficeView{}, Edges: []EdgeView{}}
	if g == nil {
		return out
	}
	for _, o := range g.Offices {
		out.Offices = append(out.Offices, OfficeView{ID: o.ID, Holders: o.Holders, Competences: o.Competences})
	}
	for _, e := range g.Edges {
		if e.Kind != powergraph.Competence {
			out.Edges = append(out.Edges, EdgeView{Kind: string(e.Kind), From: e.From, To: e.To, Via: e.Via, Active: e.Active})
		}
	}
	return out
}

func (g *Game) pendingView() View {
	p := g.pending
	v := View{Game: g.id, Period: int(g.state.Period), Current: int(g.state.Period),
		Stack: cards(p.stack.Cards, p.chosen), Overflow: cards(p.stack.Overflow, nil), Held: len(p.stack.Held),
		Findings: []FindingView{}, Stand: []AxisView{}, Report: []SectionView{}, Journal: []string{}, Graph: graphView(p.graph)}
	for _, f := range p.lint.Findings {
		if f.Failure != "" {
			v.Findings = append(v.Findings, FindingView{f.Failure, string(f.Channel), f.Place, f.Message, f.People})
		}
	}
	if p.stand != nil {
		for _, a := range p.stand.Axes {
			v.Stand = append(v.Stand, AxisView{a.Name, a.Value, a.Of})
		}
		for _, f := range p.stand.Findings {
			v.Findings = append(v.Findings, FindingView{f.Failure, string(f.Channel), f.Place, f.Message, f.People})
		}
	}
	return v
}

// Current returns the view of the period waiting to be lived.
func (g *Game) Current() View {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.pendingView()
}

// Period returns the view of a lived period or of the current one.
func (g *Game) Period(n int) (View, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if n == int(g.state.Period) {
		return g.pendingView(), nil
	}
	for _, v := range g.lived {
		if v.Period == n {
			v.Current = int(g.state.Period)
			return v, nil
		}
	}
	return View{}, fmt.Errorf("period %d is neither lived nor current", n)
}

// Decide records the auctor's choice on cards of the current stack: accept or
// reject. A card not in the stack is an error; a card left undecided when the
// period is lived counts as rejected.
func (g *Game) Decide(choices map[string]string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	in := map[string]bool{}
	for _, c := range g.pending.stack.Cards {
		in[c.Key] = true
	}
	for key, choice := range choices {
		if !in[key] {
			return fmt.Errorf("card %q is not in the stack of period %d", key, g.state.Period)
		}
		if choice != "accept" && choice != "reject" {
			return fmt.Errorf("choice for %q must be accept or reject, got %q", key, choice)
		}
		g.pending.chosen[key] = choice
	}
	return nil
}

// Advance lives the current period with the accepted amendments and prepares
// the next one.
func (g *Game) Advance() (View, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	p := g.pending
	script := turn.Script{}
	for _, c := range p.stack.Cards {
		accepted := p.chosen[c.Key] == "accept" && len(c.Proposal.Amendments) > 0
		if accepted {
			script[g.state.Period] = append(script[g.state.Period], c.Proposal.Amendments...)
		}
		g.memory = g.memory.Record(c, g.state.Period, accepted)
	}
	cfg := g.cfg
	cfg.Auctor = script
	lived := g.pendingView()
	tr, err := turn.Advance(g.state, cfg)
	if err != nil {
		return View{}, err
	}
	rep := report.Build(report.Input{Period: g.state.Period, Entries: tr.Entries, Lint: p.lint, Previous: g.previous, Overflow: p.stack.Overflow})
	for _, s := range rep.Sections {
		lived.Report = append(lived.Report, SectionView{s.Title, s.Lines})
	}
	for _, e := range tr.Entries {
		lived.Journal = append(lived.Journal, e.String())
	}
	lived.Lived = true
	g.lived = append(g.lived, lived)
	g.previous, g.last, g.state = p.lint, tr.Entries, tr.Next
	if err := g.prepare(); err != nil {
		return View{}, err
	}
	return g.pendingView(), nil
}

// Trace explains a conclusion at a period: the rule tree under the current
// corpus and the facts known now.
func (g *Game) Trace(n int, atom string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	a, err := ParseAtom(atom)
	if err != nil {
		return "", err
	}
	res, err := entitlement.Resolve(g.cfg.Strategy, g.state.Corpus, g.state.Facts, period.Period(n))
	if err != nil {
		return "", err
	}
	ans := res.Query(a)
	head := fmt.Sprintf("%s in period %d: %s", a, n, ans.Outcome)
	if ans.Reason != "" {
		head += " (" + string(ans.Reason) + ")"
	}
	if len(ans.Opposing) > 0 {
		head += "; opposing rules: " + strings.Join(ans.Opposing, ", ")
	}
	if ans.Trace == nil {
		return head + "\n", nil
	}
	return head + "\n" + res.Explain(ans.Trace), nil
}

// ParseAtom reads pred(a, b, 3), with a leading ¬ or "not " for a negative
// atom. Arguments are constants or integers.
func ParseAtom(s string) (deduction.Atom, error) {
	s = strings.TrimSpace(s)
	var a deduction.Atom
	switch {
	case strings.HasPrefix(s, "¬"):
		a.Neg, s = true, strings.TrimSpace(strings.TrimPrefix(s, "¬"))
	case strings.HasPrefix(s, "not "):
		a.Neg, s = true, strings.TrimSpace(strings.TrimPrefix(s, "not "))
	}
	open, end := strings.Index(s, "("), strings.LastIndex(s, ")")
	if open <= 0 || end != len(s)-1 {
		return a, fmt.Errorf("atom %q must look like pred(a, b)", s)
	}
	a.Pred = strings.TrimSpace(s[:open])
	for _, part := range strings.Split(s[open+1:end], ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return a, fmt.Errorf("atom %q has an empty argument", s)
		}
		if n, err := strconv.Atoi(part); err == nil {
			a.Args = append(a.Args, deduction.N(n))
		} else {
			a.Args = append(a.Args, deduction.C(part))
		}
	}
	return a, nil
}

// Periods lists the lived periods and the current one.
func (g *Game) Periods() []int {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []int
	for _, v := range g.lived {
		out = append(out, v.Period)
	}
	out = append(out, int(g.state.Period))
	sort.Ints(out)
	return out
}
