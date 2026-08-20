// Command scenario-check loads a scenario and prints the loader's report.
//
//	go run ./cmd/scenario-check internal/scenario/testdata/schema.json internal/scenario/testdata/originals/scenario.json
package main

import (
	"fmt"
	"os"
	"testing/fstest"

	"github.com/heaprip/intercessio/internal/scenario"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: scenario-check schema.json scenario.json")
		os.Exit(2)
	}
	schema, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	world, err := os.ReadFile(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	_, rep, err := scenario.Load(fstest.MapFS{"schema.json": {Data: schema}, "scenario.json": {Data: world}})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	fmt.Print(rep)
	fmt.Printf("%d errors, %d findings\n", len(rep.Errors), len(rep.Findings))
	if len(rep.Errors) > 0 {
		os.Exit(1)
	}
}
