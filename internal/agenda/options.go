package agenda

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/heaprip/intercessio/internal/corpus"
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/linter"
	"github.com/heaprip/intercessio/internal/turn"
)

// Option is one amendment the auctor may take for a card, with what the linter
// says the corpus would look like after it.
type Option struct {
	Proposal Proposal
	// Findings is how many catalog findings remain after the amendment; -1
	// when the amended corpus has errors.
	Findings int
	// Fixes says the card's own finding is gone after the amendment.
	Fixes bool
}

// options lists the amendments the code can draft for a card. The menu is
// deterministic: an advisor only chooses from it.
func options(c Card, in Input) []Proposal {
	var out []Proposal
	if p := propose(c, in.Corpus); len(p.Amendments) > 0 {
		out = append(out, p)
	}
	graph := in.Lint.Graph
	held := func(o string) bool { return graph != nil && len(graph.Office(o).Holders) > 0 }
	offices := func() []string {
		var ids []string
		if graph != nil {
			for _, o := range graph.Offices {
				ids = append(ids, o.ID)
			}
		}
		return ids
	}
	declare := func(i int, summary string, atoms ...deduction.Atom) {
		n := corpus.Norm{ID: fmt.Sprintf("N%d_%s_%d", in.Now, slug(c.Key), i), Document: constitution(in.Corpus),
			Enacted: in.Now, InForce: in.Now, Kind: corpus.Lasting, Declarations: atoms}
		out = append(out, Proposal{Summary: summary, Amendments: []turn.Amendment{{Enact: &n}}})
	}
	switch c.Failure {
	case "indeterminacy":
		// one option per rule of the root: it beats every rule it conflicts with
		partners := map[string][]string{}
		for _, place := range c.Places {
			pair := strings.Split(place, ", ")
			if len(pair) != 2 {
				continue
			}
			partners[pair[0]] = append(partners[pair[0]], pair[1])
			partners[pair[1]] = append(partners[pair[1]], pair[0])
		}
		var rules []string
		for r := range partners {
			rules = append(rules, r)
		}
		sort.Strings(rules)
		for _, r := range rules {
			var order []deduction.Defeat
			var under []string
			for _, other := range partners[r] {
				order = append(order, deduction.Defeat{Over: r, Under: other})
				under = append(under, other)
			}
			sort.Strings(under)
			out = append(out, Proposal{Summary: fmt.Sprintf("declare that %s beats %s", r, strings.Join(under, ", ")),
				Amendments: []turn.Amendment{{Order: order}}})
		}
	case "judge-in-own-cause":
		for i, o := range offices() {
			if held(o) && !contains(graph.Office(o).Competences, c.Root) {
				declare(i, fmt.Sprintf("give %s the power to %s", o, c.Root), deduction.A("responsibility", o, c.Root))
			}
		}
	case "review-dead-end", "unchecked-act":
		target := c.Root
		if _, office, ok := strings.Cut(c.Root, " by "); ok {
			target = office
		}
		for i, o := range offices() {
			if o != target && held(o) {
				declare(i, fmt.Sprintf("let %s review %s", o, target),
					deduction.A("reviews", o, target), deduction.A("responsibility", o, "review"))
			}
		}
	case "dangling-succession":
		declare(0, fmt.Sprintf("fill %s by a constitutive act", c.Root), deduction.A("constituted", c.Root))
	}
	return out
}

var nonWord = regexp.MustCompile(`[^A-Za-z0-9]+`)

func slug(s string) string { return strings.Trim(nonWord.ReplaceAllString(s, "_"), "_") }

// constitution is the document of the highest level.
func constitution(v corpus.Version) string {
	best, id := 0, ""
	for _, d := range v.Documents {
		if id == "" || d.Level < best {
			best, id = d.Level, d.ID
		}
	}
	return id
}

func countFindings(r *linter.Report) int {
	n := 0
	for _, f := range r.Findings {
		if f.Failure != "" {
			n++
		}
	}
	return n
}

// preview lints the corpus each option leads to.
func preview(c Card, in Input, ps []Proposal) []Option {
	var out []Option
	for _, p := range ps {
		o := Option{Proposal: p, Findings: -1}
		v := turn.Apply(in.Corpus, p.Amendments)
		rep, err := linter.Lint(linter.Input{Corpus: v, Roles: in.Roles, Facts: in.Facts, Strategy: in.Strategy, Now: in.Now})
		if err == nil && len(rep.Errors) == 0 {
			o.Findings, o.Fixes = countFindings(rep), true
			for _, f := range rep.Findings {
				if f.Failure == c.Failure && root(f) == c.Root {
					o.Fixes = false
				}
			}
		}
		out = append(out, o)
	}
	return out
}
