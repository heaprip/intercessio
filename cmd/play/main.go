// Command play lives a scenario through several periods with stub participants
// and prints the journal.
//
//	go run ./cmd/play -periods 3 internal/scenario/testdata/schema.json internal/scenario/testdata/playable/scenario.json
//	go run ./cmd/play -periods 3 -veto marcus ...
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing/fstest"

	"github.com/heaprip/intercessio/internal/actors"
	"github.com/heaprip/intercessio/internal/entitlement"
	"github.com/heaprip/intercessio/internal/scenario"
	"github.com/heaprip/intercessio/internal/turn"
)

func main() {
	periods := flag.Int("periods", 3, "how many periods to live")
	veto := flag.String("veto", "", "comma-separated persons whose grants the tribune stops")
	seed := flag.Int64("seed", 7, "seed of inspection sampling")
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: play [-periods N] [-veto a,b] [-seed S] schema.json scenario.json")
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
	stub := actors.Stub{VetoGrantsTo: map[string]bool{}}
	for _, p := range strings.Split(*veto, ",") {
		if p != "" {
			stub.VetoGrantsTo[p] = true
		}
	}
	cfg := turn.Config{Strategy: entitlement.Hierarchy{}, Actors: stub}
	st := turn.State{Period: s.StartPeriod, Corpus: s.Corpus, Facts: s.Facts, Seed: *seed}
	for i := 0; i < *periods; i++ {
		tr, err := turn.Advance(st, cfg)
		if err != nil {
			fail(err)
		}
		for _, e := range tr.Entries {
			fmt.Println(e)
		}
		st = tr.Next
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
