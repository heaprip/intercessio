// Package report assembles the report of a period: what happened, what the
// linter found, who an amendment touched, and what did not fit the stack.
//
// Prototype: plain text sections, no front.
package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/heaprip/intercessio/internal/agenda"
	"github.com/heaprip/intercessio/internal/journal"
	"github.com/heaprip/intercessio/internal/linter"
	"github.com/heaprip/intercessio/internal/period"
)

// Section is one titled block of lines.
type Section struct {
	Title string
	Lines []string
}

// Report of one period.
type Report struct {
	Period   period.Period
	Sections []Section
}

// Input is what a report is built from.
type Input struct {
	Period   period.Period
	Entries  []journal.Entry // of the period
	Lint     *linter.Report  // of the corpus the period was lived under
	Overflow []agenda.Card   // cards that did not fit the stack
}

// Build assembles the sections. An empty section is left out.
func Build(in Input) Report {
	r := Report{Period: in.Period}
	add := func(title string, lines []string) {
		if len(lines) > 0 {
			r.Sections = append(r.Sections, Section{title, lines})
		}
	}
	add("amendment", amendment(in.Entries))
	add("cases", casesOf(in.Entries))
	add("enforcement", enforcement(in.Entries))
	add("linter", findings(in.Lint))
	var over []string
	for _, c := range in.Overflow {
		over = append(over, c.String())
	}
	add("not in the stack", over)
	return r
}

func amendment(es []journal.Entry) []string {
	var lines []string
	grievances := map[string][]string{}
	for _, e := range es {
		switch e.Kind {
		case journal.Amendment:
			lines = append(lines, fmt.Sprintf("%s -> %s", e.Subject, e.Outcome))
		case journal.Grievance:
			grievances[e.Outcome] = append(grievances[e.Outcome], e.Actor)
		}
	}
	for _, outcome := range []string{"queued", "filed", "deferred", "no-remedy"} {
		if names := grievances[outcome]; len(names) > 0 {
			lines = append(lines, fmt.Sprintf("grievances %s: %d [%s]", outcome, len(names), strings.Join(names, ", ")))
		}
	}
	return lines
}

// casesOf counts decisions by outcome and keeps the reasons of non liquet
// apart: a gap is fixed by writing a norm, a deadlock by declaring an order.
func casesOf(es []journal.Entry) []string {
	outcomes := map[string]int{}
	reasons := map[string][]string{}
	var lines []string
	for _, e := range es {
		switch e.Kind {
		case journal.PetitionFiled, journal.ChargeFiled:
			outcomes[string(e.Kind)]++
		case journal.Decision:
			outcomes["decided "+e.Outcome]++
		case journal.NonLiquet:
			reasons[e.Basis.Rule] = append(reasons[e.Basis.Rule], e.Subject)
		case journal.Intercessio:
			lines = append(lines, fmt.Sprintf("intercessio by %s on %s", e.Actor, e.Subject))
		case journal.Expired:
			lines = append(lines, fmt.Sprintf("expired: %s (%s)", e.Subject, e.Case))
		case journal.UltraVires:
			lines = append(lines, fmt.Sprintf("ultra vires: %s as %s tried %s", e.Actor, e.Office, e.Subject))
		case journal.Vacant:
			lines = append(lines, fmt.Sprintf("vacant: %s, case %s waits", e.Office, e.Subject))
		case journal.Recusal:
			lines = append(lines, fmt.Sprintf("recusal: %s goes to %s %s", e.Case, e.Actor, e.Office))
		case journal.Conflict:
			lines = append(lines, fmt.Sprintf("conflict: %s decides own case %s", e.Actor, e.Case))
		}
	}
	var counts []string
	for k, n := range outcomes {
		counts = append(counts, fmt.Sprintf("%s %d", k, n))
	}
	sort.Strings(counts)
	if len(counts) > 0 {
		lines = append([]string{strings.Join(counts, ", ")}, lines...)
	}
	var rs []string
	for k := range reasons {
		rs = append(rs, k)
	}
	sort.Strings(rs)
	for _, k := range rs {
		lines = append(lines, fmt.Sprintf("non liquet, %s: %d [%s]", k, len(reasons[k]), strings.Join(reasons[k], "; ")))
	}
	return lines
}

// enforcement keeps "did not violate" apart from "nobody checked".
func enforcement(es []journal.Entry) []string {
	for _, e := range es {
		if e.Kind != journal.PeriodSummary {
			continue
		}
		var violations, checked, bearers, cases, queue int
		if n, _ := fmt.Sscanf(e.Subject, "violations=%d checked=%d bearers=%d cases=%d queue=%d", &violations, &checked, &bearers, &cases, &queue); n < 2 {
			return nil
		}
		return []string{fmt.Sprintf("duty bearers %d; violations %d, checked %d, not checked %d", bearers, violations, checked, violations-checked)}
	}
	return nil
}

// findings lists catalog findings; those naming people show them by name.
func findings(l *linter.Report) []string {
	if l == nil {
		return nil
	}
	var lines []string
	for _, f := range l.Findings {
		if f.Failure != "" {
			lines = append(lines, f.String())
		}
	}
	return lines
}

func (r Report) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "report, period %d\n", r.Period)
	for _, s := range r.Sections {
		fmt.Fprintf(&b, "  %s\n", s.Title)
		for _, l := range s.Lines {
			fmt.Fprintf(&b, "    %s\n", l)
		}
	}
	return b.String()
}
