package actors

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/heaprip/intercessio/internal/cases"
	"github.com/heaprip/intercessio/internal/competence"
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/journal"
	"github.com/heaprip/intercessio/internal/llmruntime"
)

// PromptVersion is part of every request; changing a prompt changes it.
const PromptVersion = "v1"

// Model plays officials and persons with a model. The tribune and the script
// stay with the stub. The menu and the legality of a move stay deterministic:
// the model only chooses.
type Model struct {
	Stub
	Runtime *llmruntime.Runtime
	Spec    llmruntime.Model
	// Version overrides PromptVersion, to show replay divergence.
	Version string
}

const officialSystem = `You act as an official in a turn-based simulation of a small polity.
You decide one case within the competence of your office. The facts and the
normative result below come from the polity's law and are correct.
Reply with a JSON object only: {"outcome": "<one of the allowed outcomes>", "reason": "<one or two sentences>"}.`

const personSystem = `You act as a person living in a turn-based simulation of a small polity.
You decide what you do this period. The legal situation below comes from the
polity's law and is correct.
Reply with a JSON object only: {"move": "<one of the allowed moves>", "reason": "<one or two sentences>"}.`

func (m Model) request(role, system, user string) llmruntime.Request {
	v := m.Version
	if v == "" {
		v = PromptVersion
	}
	return llmruntime.Request{Role: role, PromptVersion: v, Model: m.Spec.ID, Provider: m.Spec.Provider, Params: m.Spec.Params, System: system, User: user}
}

func pick(field string, allowed []string) (func(string) error, *string) {
	var got string
	return func(content string) error {
		start, end := strings.Index(content, "{"), strings.LastIndex(content, "}")
		if start < 0 || end < start {
			return fmt.Errorf("the reply contains no JSON object")
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(content[start:end+1]), &data); err != nil {
			return fmt.Errorf("the JSON object does not parse: %v", err)
		}
		v, _ := data[field].(string)
		for _, a := range allowed {
			if v == a {
				got = v
				return nil
			}
		}
		return fmt.Errorf("%q must be exactly one of %s", field, strings.Join(allowed, ", "))
	}, &got
}

func explain(ans deduction.Answer) string {
	switch {
	case ans.Trace != nil:
		return fmt.Sprintf("%s, because:\n%s", ans.Outcome, ans.Trace)
	case ans.Reason != "":
		return fmt.Sprintf("%s (%s)", ans.Outcome, ans.Reason)
	}
	return string(ans.Outcome)
}

// Propose asks the model to decide. If the call is not served, the official's
// declared fallback applies: the case stays in the queue.
func (m Model) Propose(c cases.Case, q competence.Query) cases.Proposal {
	atom, outcomes := Normative(c)
	ans := q(atom)
	rule, rules := "", ""
	if ans.Trace != nil {
		rule, rules = ans.Trace.Rule, strings.Join(ans.Trace.Rules(), ",")
	}
	var user strings.Builder
	fmt.Fprintf(&user, "Office: %s\nCase: %s %s\nPerson: %s\nMatter: %s\nOpened in period: %d\n\n", c.Office, c.Kind, c.ID, c.Person, c.Matter, c.Opened)
	fmt.Fprintf(&user, "Question: %s\nNormative result: %s\n", atom, explain(ans))
	fmt.Fprintf(&user, "Allowed outcomes: %s", strings.Join(outcomes[:], ", "))
	validate, got := pick("outcome", outcomes[:])
	reply := m.Runtime.Call(m.request("official", officialSystem, user.String()), validate)
	p := cases.Proposal{Rule: rule, Rules: rules, Decider: journal.ByModel, Model: m.Spec.Name, Call: reply.Hash}
	if reply.Status != llmruntime.Served {
		p.Unserved = reply.Reason
		return p
	}
	validate(reply.Content)
	p.Outcome = *got
	return p
}

// Petitions asks each person the law makes eligible whether to petition now.
// If the call is not served, the person's fallback is to do nothing.
func (m Model) Petitions(q competence.Query, people, statuses []string, exists func(person, matter string) bool) []Request {
	out := Candidates(q, people, statuses, exists)
	for i, r := range out {
		ans := q(deduction.A("entitled", r.Person, r.Status))
		var user strings.Builder
		fmt.Fprintf(&user, "You are: %s\nYou may petition for the status: %s\n", r.Person, r.Status)
		fmt.Fprintf(&user, "Are you entitled to it: %s\n", explain(ans))
		user.WriteString("Allowed moves: petition, wait")
		validate, got := pick("move", []string{"petition", "wait"})
		reply := m.Runtime.Call(m.request("person", personSystem, user.String()), validate)
		out[i].Decider, out[i].Model, out[i].Call = journal.ByModel, m.Spec.Name, reply.Hash
		if reply.Status != llmruntime.Served {
			out[i].Unserved = reply.Reason
			continue
		}
		validate(reply.Content)
		out[i].File = *got == "petition"
	}
	return out
}
