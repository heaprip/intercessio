// Package agenda gathers the stack of cards the auctor works through between
// periods: which findings and events of the period deserve a decision on the
// law, in what order, and which may come back.
//
// Prototype. Selection is deterministic; the proposal on a card is a template,
// not an advisor model.
package agenda

import (
	"fmt"
	"sort"
	"strings"

	"github.com/heaprip/intercessio/internal/corpus"
	"github.com/heaprip/intercessio/internal/journal"
	"github.com/heaprip/intercessio/internal/linter"
	"github.com/heaprip/intercessio/internal/period"
	"github.com/heaprip/intercessio/internal/turn"
)

// Card is one reason to change the law.
type Card struct {
	// Key is the root of the card, stable across periods: the failure and the
	// place it rests on.
	Key     string
	Failure string // catalog slug; empty for people touched without a failure
	Root    string
	Channel string
	// Reach is how many findings or journal entries the card gathers.
	Reach  int
	People []string
	// Evidence is the trace of the card: the finding messages and journal
	// entries it was built from. A card without evidence is not built.
	Evidence []string
	Score    int
	Proposal Proposal
	// Reminder: the card was rejected earlier and comes back after the preset
	// silence, without new evidence.
	Reminder bool
	// Persists: the card was accepted earlier and its finding is still there.
	Persists bool
}

func (c Card) String() string {
	name := c.Failure
	if name == "" {
		name = "(event)"
	}
	s := fmt.Sprintf("%-26s %-24s reach=%-3d score=%-3d %s", name, c.Root, c.Reach, c.Score, c.Proposal.Summary)
	if len(c.People) > 0 {
		s += " [" + strings.Join(c.People, ", ") + "]"
	}
	switch {
	case c.Persists:
		s += " (persists after the accepted amendment)"
	case c.Reminder:
		s += " (reminder)"
	}
	return s
}

// Proposal is the draft amendment on a card. With no amendments the auctor has
// to write the norm himself.
type Proposal struct {
	Summary    string
	Amendments []turn.Amendment
}

// Preset is the data of selection.
type Preset struct {
	Size    int
	Weights map[string]int // by failure slug; missing weighs 1
	// Return is how many periods a rejected card stays silent before it comes
	// back as a reminder. Zero: it comes back only with new evidence.
	Return int
}

// Seen is what the auctor saw of a card when he decided on it.
type Seen struct {
	Reach    int
	People   []string
	Period   period.Period
	Accepted bool
}

// Memory holds the decided cards by key. A decided card comes back with new
// evidence — a larger reach or new people; an accepted card whose finding is
// still there comes back at once; a rejected one comes back as a reminder
// after the preset silence.
type Memory map[string]Seen

// Record returns a memory with the card decided in period now.
func (m Memory) Record(c Card, now period.Period, accepted bool) Memory {
	out := Memory{}
	for k, v := range m {
		out[k] = v
	}
	out[c.Key] = Seen{Reach: c.Reach, People: append([]string{}, c.People...), Period: now, Accepted: accepted}
	return out
}

// Input is what a stack is built from.
type Input struct {
	Lint *linter.Report
	// Practice holds the censor's journal findings over the recent window.
	Practice []linter.Finding
	Entries  []journal.Entry // of the period just lived
	Corpus   corpus.Version
	Now      period.Period
	Memory   Memory
	Preset   Preset
}

// Stack is the result: the cards to decide, those that did not fit and go to
// the report, and those held back for lack of new evidence.
type Stack struct {
	Period   period.Period
	Cards    []Card
	Overflow []Card
	Held     []Card
}

// Build gathers, groups, orders and cuts.
func Build(in Input) Stack {
	byKey := map[string]*Card{}
	var order []string
	add := func(failure, root, channel, evidence string, people ...string) {
		if evidence == "" {
			return
		}
		key := failure + "|" + root
		c, ok := byKey[key]
		if !ok {
			c = &Card{Key: key, Failure: failure, Root: root, Channel: channel}
			byKey[key] = c
			order = append(order, key)
		}
		c.Reach++
		c.Evidence = append(c.Evidence, evidence)
		for _, p := range people {
			if p != "" && !contains(c.People, p) {
				c.People = append(c.People, p)
			}
		}
	}

	if in.Lint != nil {
		for _, f := range in.Lint.Findings {
			if f.Failure == "" {
				continue // a remark on form is not a reason to change the law
			}
			add(f.Failure, root(f), string(f.Channel), f.String(), f.People...)
		}
	}
	for _, f := range in.Practice {
		add(f.Failure, root(f), string(f.Channel), f.String(), f.People...)
	}
	for _, e := range in.Entries {
		switch {
		case e.Kind == journal.Conflict:
			add("judge-in-own-cause", e.Office, "journal", e.String(), e.Actor)
		case e.Kind == journal.Grievance && e.Outcome == "queued":
			add("", "grievances", "recount", e.String(), e.Actor)
		case e.Kind == journal.NonLiquet:
			add("", "non-liquet "+e.Basis.Rule, "journal", e.String(), firstWord(e.Subject))
		}
	}

	st := Stack{Period: in.Now}
	var cards []Card
	for _, k := range order {
		c := *byKey[k]
		sort.Strings(c.People)
		w, ok := in.Preset.Weights[c.Failure]
		if !ok {
			w = 1
		}
		c.Score = w * c.Reach
		c.Proposal = propose(c, in.Corpus)
		if seen, ok := in.Memory[c.Key]; ok && c.Reach <= seen.Reach && subset(c.People, seen.People) {
			switch {
			case seen.Accepted:
				c.Persists = true
			case in.Preset.Return > 0 && in.Now-seen.Period >= period.Period(in.Preset.Return):
				c.Reminder = true
			default:
				st.Held = append(st.Held, c)
				continue
			}
		}
		cards = append(cards, c)
	}
	sort.SliceStable(cards, func(i, j int) bool {
		if cards[i].Score != cards[j].Score {
			return cards[i].Score > cards[j].Score
		}
		return cards[i].Key < cards[j].Key
	})
	for i, c := range cards {
		if in.Preset.Size > 0 && i >= in.Preset.Size {
			st.Overflow = append(st.Overflow, c)
			continue
		}
		st.Cards = append(st.Cards, c)
	}
	return st
}

// root groups findings that one fix would answer: every unordered conflict over
// the same predicate is one card, whatever pairs of rules make it.
func root(f linter.Finding) string {
	if f.Failure == "mass-non-liquet" {
		return "cases" // the window moves every period; the root does not
	}
	if f.Failure == "indeterminacy" {
		if _, after, ok := strings.Cut(f.Message, "opposite "); ok {
			return strings.Fields(after)[0]
		}
	}
	return f.Place
}

// propose drafts the amendment. Only a mechanical fix is drafted; the rest asks
// the auctor to write the norm.
func propose(c Card, v corpus.Version) Proposal {
	switch c.Failure {
	case "retroactivity":
		if n, ok := v.Norm(c.Root); ok {
			fixed := n
			fixed.InForce = n.Enacted
			return Proposal{
				Summary:    fmt.Sprintf("re-enact %s in force from its enactment in %d", n.ID, n.Enacted),
				Amendments: []turn.Amendment{{Enact: &fixed}},
			}
		}
	case "indeterminacy":
		return Proposal{Summary: "declare which rule wins over " + c.Root}
	case "judge-in-own-cause":
		return Proposal{Summary: "give another office the power to decide on " + c.Root}
	case "":
		return Proposal{Summary: "decide whether " + c.Root + " call for a norm"}
	}
	return Proposal{Summary: "write a norm against " + c.Failure + " at " + c.Root}
}

func firstWord(s string) string {
	if f := strings.Fields(s); len(f) > 0 {
		return f[0]
	}
	return ""
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

func subset(xs, of []string) bool {
	for _, x := range xs {
		if !contains(of, x) {
			return false
		}
	}
	return true
}
