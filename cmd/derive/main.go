// Command derive loads a scenario, evaluates it at a period and prints what
// holds, or the answer and trace for one predicate.
//
//	go run ./cmd/derive -now 12 internal/scenario/testdata/schema.json internal/scenario/testdata/office-eligibility/scenario.json
//	go run ./cmd/derive -now 12 -pred eligible ...
package main

import (
	"flag"
	"fmt"
	"os"
	"testing/fstest"

	"github.com/heaprip/intercessio/internal/deduction"
	"github.com/heaprip/intercessio/internal/period"
	"github.com/heaprip/intercessio/internal/scenario"
)

func main() {
	now := flag.Int("now", 0, "period to take the slice at")
	pred := flag.String("pred", "", "print only this predicate, with traces")
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: derive -now N [-pred name] schema.json scenario.json")
		os.Exit(2)
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
	res, err := deduction.Evaluate(s.Program(period.Period(*now)), *now)
	if err != nil {
		fail(err)
	}
	for _, a := range res.Conclusions() {
		if *pred != "" && a.Pred != *pred {
			continue
		}
		fmt.Println(a)
		if *pred != "" {
			fmt.Print(res.Query(a).Trace)
			fmt.Println()
		}
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
