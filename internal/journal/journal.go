// Package journal records everything that happened, with provenance.
//
// Prototype: an append-only value, written in the same transition as the rest
// of the state.
package journal

import (
	"fmt"
	"strings"

	"github.com/heaprip/intercessio/internal/period"
)

// Kind is the closed set of entry kinds.
type Kind string

const (
	PetitionFiled     Kind = "petition-filed"
	ChargeFiled       Kind = "charge-filed"
	ViolationDetected Kind = "violation-detected"
	Decision          Kind = "decision"
	NonLiquet         Kind = "non-liquet"
	UltraVires        Kind = "ultra-vires"
	Intercessio       Kind = "intercessio"
	Finalization      Kind = "finalization"
	Execution         Kind = "execution"
	Expired           Kind = "expired"
	Vacant            Kind = "vacant"
	// Unserved: a model call did not happen and the declared fallback was taken.
	Unserved Kind = "unserved"
	// Amendment: the auctor changed the corpus in the amendment phase.
	Amendment Kind = "amendment"
	// Grievance: a person touched by an amendment; the outcome says what became
	// of it: queued, filed, deferred or no-remedy.
	Grievance Kind = "grievance"
	// Deferred: a petition that did not fit the case budget of the period.
	Deferred Kind = "deferred"
	// Recusal: the case concerns its decider and passed to another holder.
	Recusal Kind = "recusal"
	// Review: a decision was appealed and reviewed; the outcome says upheld,
	// changed or no-reviewer.
	Review Kind = "review"
	// Act: an office performed an act outside a case.
	Act Kind = "act"
	// Usurpation: a person holds an office by an appointment in force without
	// being eligible for it.
	Usurpation Kind = "usurpation"
	// NoQuorum: an act needed more concurring holders than the office had.
	NoQuorum Kind = "no-quorum"
	// Conflict: the case concerns its decider and nobody else could take it.
	Conflict      Kind = "conflict"
	PeriodSummary Kind = "period-summary"
)

// Decider says who made the step: a rule, a stub or a model.
type Decider string

const (
	ByRule  Decider = "rule"
	ByStub  Decider = "stub"
	ByModel Decider = "model"
)

// Basis is what the step rested on.
type Basis struct {
	Rule string // the rule the answer rested on, if any
	// Rules is every rule that took part, comma-separated: the trace of the
	// answer and the competence it was checked against. A string keeps the
	// entry comparable.
	Rules    string
	Corpus   int    // corpus version
	Strategy string // resolution strategy
}

// Entry is one record.
type Entry struct {
	Period  period.Period
	Seq     int
	Kind    Kind
	Actor   string
	Office  string
	Case    string
	Subject string
	Outcome string
	Basis   Basis
	Decider Decider
	Model   string // model name when a model decided
	Call    string // hash of the call record; prompts never enter the journal
}

func (e Entry) String() string {
	var parts []string
	add := func(k, v string) {
		if v != "" {
			parts = append(parts, k+"="+v)
		}
	}
	add("actor", e.Actor)
	add("office", e.Office)
	add("case", e.Case)
	add("subject", e.Subject)
	add("outcome", e.Outcome)
	add("rule", e.Basis.Rule)
	add("by", string(e.Decider))
	add("model", e.Model)
	if len(e.Call) > 8 {
		add("call", e.Call[:8])
	}
	return fmt.Sprintf("%3d #%-3d %-19s %s", e.Period, e.Seq, e.Kind, strings.Join(parts, " "))
}

// Journal is the sequence of entries.
type Journal struct {
	Entries []Entry
}

// Append returns a new journal with the entries numbered after the last one.
func (j Journal) Append(es ...Entry) Journal {
	out := Journal{Entries: append([]Entry{}, j.Entries...)}
	for _, e := range es {
		e.Seq = len(out.Entries) + 1
		out.Entries = append(out.Entries, e)
	}
	return out
}

// Of returns the entries of a kind.
func (j Journal) Of(k Kind) []Entry {
	var out []Entry
	for _, e := range j.Entries {
		if e.Kind == k {
			out = append(out, e)
		}
	}
	return out
}
