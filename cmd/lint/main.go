// Command lint loads a scenario and prints the linter's findings at a period,
// the power graph and, for a corpus with a norm removed, the recount by person.
//
//	go run ./cmd/lint -now 13 -graph internal/scenario/testdata/schema.json internal/scenario/testdata/office-eligibility/scenario.json
//	go run ./cmd/lint -now 35 -without A4 internal/scenario/testdata/schema.json internal/scenario/testdata/citizenship/scenario.json
package main

import (
	"flag"
	"fmt"
	"os"
	"testing/fstest"

	"github.com/heaprip/intercessio/internal/corpus"
	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/entitlement"
	"github.com/heaprip/intercessio/internal/linter"
	"github.com/heaprip/intercessio/internal/period"
	"github.com/heaprip/intercessio/internal/scenario"
)

func main() {
	now := flag.Int("now", 0, "period to lint at")
	strategy := flag.String("strategy", "hierarchy", "resolution strategy: hierarchy or lex-posterior")
	without := flag.String("without", "", "remove this norm and recount who it touches")
	graph := flag.Bool("graph", false, "print the power graph")
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: lint -now N [-strategy s] [-without norm] [-graph] schema.json scenario.json")
		os.Exit(2)
	}
	var strat entitlement.Strategy
	switch *strategy {
	case "hierarchy":
		strat = entitlement.Hierarchy{}
	case "lex-posterior":
		strat = entitlement.LexPosterior{}
	default:
		fail(fmt.Errorf("unknown strategy %q", *strategy))
	}
	schema, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fail(err)
	}
	world, err := os.ReadFile(flag.Arg(1))
	if err != nil {
		fail(err)
	}
	s, rep, err := scenario.Load(fstest.MapFS{"schema.json": {Data: schema}, "scenario.json": {Data: world}})
	if err != nil {
		fail(err)
	}
	in := linter.Input{Corpus: s.Corpus, Roles: s.Roles, Facts: s.Facts, Strategy: strat, Now: period.Period(*now),
		Loader: append(append([]deduction.Diagnostic{}, rep.Errors...), rep.Findings...)}
	if *without != "" {
		before := s.Corpus
		in.Before = &before
		in.Corpus = corpus.Version(s.Corpus).Without(*without)
	}
	out, err := linter.Lint(in)
	if err != nil {
		fail(err)
	}
	fmt.Print(out)
	fmt.Printf("%d errors, %d findings\n", len(out.Errors), len(out.Findings))
	if out.Impact != nil {
		fmt.Println()
		fmt.Print(out.Impact.Render(out.People))
	}
	if *graph && out.Graph != nil {
		fmt.Println()
		fmt.Print(out.Graph)
	}
	if len(out.Errors) > 0 {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
