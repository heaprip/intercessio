// Command derive loads a scenario, resolves it at a period and prints what
// holds, or every conclusion of one predicate with its trace.
//
//	go run ./cmd/derive -now 12 internal/scenario/testdata/schema.json internal/scenario/testdata/office-eligibility/scenario.json
//	go run ./cmd/derive -now 12 -pred eligible -strategy lex-posterior ...
//	go run ./cmd/derive -now 40 -without A4 ...
package main

import (
	"flag"
	"fmt"
	"os"
	"testing/fstest"

	"github.com/heaprip/intercessio/internal/entitlement"
	"github.com/heaprip/intercessio/internal/period"
	"github.com/heaprip/intercessio/internal/scenario"
)

func main() {
	now := flag.Int("now", 0, "period to take the slice at")
	pred := flag.String("pred", "", "print only this predicate, with traces")
	strategy := flag.String("strategy", "hierarchy", "resolution strategy: hierarchy or lex-posterior")
	without := flag.String("without", "", "remove this norm from the corpus first")
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: derive -now N [-pred name] [-strategy s] [-without norm] schema.json scenario.json")
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
	if len(rep.Errors) > 0 {
		fmt.Print(rep)
		os.Exit(1)
	}
	v := s.Corpus
	if *without != "" {
		v = v.Without(*without)
	}
	res, err := entitlement.Resolve(strat, v, s.Facts, period.Period(*now))
	if err != nil {
		fail(err)
	}
	for _, a := range res.Conclusions() {
		if *pred != "" && a.Pred != *pred {
			continue
		}
		fmt.Println(a)
		if *pred != "" {
			fmt.Print(res.Explain(res.Query(a).Trace))
			fmt.Println()
		}
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
