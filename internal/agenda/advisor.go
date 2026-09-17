package agenda

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/heaprip/intercessio/internal/llmruntime"
)

// Advice is the advisor's choice on a card: an option by number, 1-based, or
// zero for none, and why.
type Advice struct {
	By       string // stub or model
	Index    int
	Reason   string
	Model    string
	Call     string
	Unserved string // the model was asked and did not serve; the stub's choice stands
}

// Advisor chooses an amendment for a card from the deterministic menu. It may
// choose none; the auctor still decides.
type Advisor interface {
	Advise(c Card, options []Option, before int) Advice
}

// StubAdvisor takes the option that removes the card's finding and leaves the
// fewest findings; failing that, one that leaves fewer findings than now.
type StubAdvisor struct{}

// Advise implements Advisor.
func (StubAdvisor) Advise(c Card, options []Option, before int) Advice {
	best, fixes := 0, false
	for i, o := range options {
		if o.Findings < 0 {
			continue
		}
		switch {
		case best == 0 && (o.Fixes || o.Findings < before):
			best, fixes = i+1, o.Fixes
		case best != 0 && o.Fixes && !fixes:
			best, fixes = i+1, true
		case best != 0 && o.Fixes == fixes && o.Findings < options[best-1].Findings:
			best = i + 1
		}
	}
	a := Advice{By: "stub", Index: best, Reason: "no option improves the corpus"}
	if best > 0 {
		a.Reason = fmt.Sprintf("leaves %d findings of %d", options[best-1].Findings, before)
		if fixes {
			a.Reason = "removes the finding and " + a.Reason
		}
	}
	return a
}

// AdvisorPromptVersion is part of every advisor request.
const AdvisorPromptVersion = "advisor-v1"

const advisorSystem = `You advise the legislator of a small polity in a turn-based simulation.
The polity's law has a problem, described below with evidence. The legislator can take one
of the numbered amendments, or none. For each amendment you see how many problems the law
checker would still report after it, and whether it removes this problem.
Choose what a careful legislator should take: removing the problem matters, but so do side
effects on who holds power and who can stop whom.
Reply with a JSON object only: {"option": <number, 0 for none>, "reason": "<one or two sentences>"}.`

// ModelAdvisor asks a model to choose. When the call is not served, the
// fallback's advice stands with the reason recorded.
type ModelAdvisor struct {
	Runtime  *llmruntime.Runtime
	Spec     llmruntime.Model
	Fallback Advisor
}

// Advise implements Advisor.
func (m ModelAdvisor) Advise(c Card, options []Option, before int) Advice {
	fallback := m.Fallback
	if fallback == nil {
		fallback = StubAdvisor{}
	}
	if len(options) == 0 {
		return fallback.Advise(c, options, before)
	}
	var user strings.Builder
	name := c.Failure
	if name == "" {
		name = "event"
	}
	fmt.Fprintf(&user, "Problem: %s at %s, reach %d\n", name, c.Root, c.Reach)
	if len(c.People) > 0 {
		fmt.Fprintf(&user, "People concerned: %s\n", strings.Join(c.People, ", "))
	}
	user.WriteString("Evidence:\n")
	for i, e := range c.Evidence {
		if i == 6 {
			fmt.Fprintf(&user, "  ... and %d more\n", len(c.Evidence)-i)
			break
		}
		fmt.Fprintf(&user, "  %s\n", strings.Join(strings.Fields(e), " "))
	}
	fmt.Fprintf(&user, "Problems reported now: %d\nAmendments:\n  0. none\n", before)
	for i, o := range options {
		after := fmt.Sprintf("%d problems after", o.Findings)
		if o.Findings < 0 {
			after = "breaks the law: errors after"
		}
		fixes := "keeps this problem"
		if o.Fixes {
			fixes = "removes this problem"
		}
		fmt.Fprintf(&user, "  %d. %s — %s, %s\n", i+1, o.Proposal.Summary, after, fixes)
	}
	var got int
	var reason string
	validate := func(content string) error {
		start, end := strings.Index(content, "{"), strings.LastIndex(content, "}")
		if start < 0 || end < start {
			return fmt.Errorf("the reply contains no JSON object")
		}
		var data struct {
			Option *int   `json:"option"`
			Reason string `json:"reason"`
		}
		if err := json.Unmarshal([]byte(content[start:end+1]), &data); err != nil {
			return fmt.Errorf("the JSON object does not parse: %v", err)
		}
		if data.Option == nil || *data.Option < 0 || *data.Option > len(options) {
			return fmt.Errorf(`"option" must be a number from 0 to %d`, len(options))
		}
		got, reason = *data.Option, data.Reason
		return nil
	}
	req := llmruntime.Request{Role: "advisor", PromptVersion: AdvisorPromptVersion, Model: m.Spec.ID, Provider: m.Spec.Provider,
		Params: m.Spec.Params, System: advisorSystem, User: user.String()}
	reply := m.Runtime.Call(req, validate)
	if reply.Status != llmruntime.Served {
		a := fallback.Advise(c, options, before)
		a.Unserved, a.Model, a.Call = reply.Reason, m.Spec.Name, reply.Hash
		return a
	}
	validate(reply.Content)
	return Advice{By: "model", Index: got, Reason: reason, Model: m.Spec.Name, Call: reply.Hash}
}
