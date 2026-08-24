// Command play lives a scenario through several periods and prints the journal.
// Participants are stubs, or a model through llm-runtime.
//
//	go run ./cmd/play -periods 3 internal/scenario/testdata/schema.json internal/scenario/testdata/playable/scenario.json
//	go run ./cmd/play -mode live -model deepseek-no-thinking -records run.jsonl -max-rub 2 ...
//	go run ./cmd/play -mode replay -records run.jsonl ...
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing/fstest"

	"github.com/heaprip/intercessio/internal/actors"
	"github.com/heaprip/intercessio/internal/entitlement"
	"github.com/heaprip/intercessio/internal/llmruntime"
	"github.com/heaprip/intercessio/internal/scenario"
	"github.com/heaprip/intercessio/internal/turn"
)

func main() {
	periods := flag.Int("periods", 3, "how many periods to live")
	veto := flag.String("veto", "", "comma-separated persons whose grants the tribune stops")
	seed := flag.Int64("seed", 7, "seed of inspection sampling")
	mode := flag.String("mode", "stub", "stub, live, replay or fork-offline")
	model := flag.String("model", "deepseek-no-thinking", "model profile for live participants")
	records := flag.String("records", "", "JSONL file of call records: read, and appended to in live mode")
	maxRub := flag.Float64("max-rub", 1, "spend limit of live calls, rubles")
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: play [flags] schema.json scenario.json")
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

	var rt *llmruntime.Runtime
	if *mode != "stub" {
		spec, ok := llmruntime.Models[*model]
		if !ok {
			fail(fmt.Errorf("unknown model %q", *model))
		}
		recs, err := llmruntime.LoadRecords(*records)
		if err != nil {
			fail(err)
		}
		env := dotenv(".env")
		rt = llmruntime.New(llmruntime.Mode(*mode), env["LLM_URL"], env["LLM_KEY"], *maxRub, recs)
		cfg.Actors = actors.Model{Stub: stub, Runtime: rt, Spec: spec}
	}

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
	if rt != nil {
		written := rt.Written()
		if *records != "" && len(written) > 0 {
			if err := llmruntime.AppendRecords(*records, written); err != nil {
				fail(err)
			}
		}
		fmt.Printf("calls recorded: %d, spent: %.4f rub\n", len(written), rt.Spent())
	}
}

// dotenv reads KEY=VALUE lines; the environment wins.
func dotenv(path string) map[string]string {
	env := map[string]string{}
	if f, err := os.Open(path); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
				continue
			}
			k, v, _ := strings.Cut(line, "=")
			env[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
		}
		f.Close()
	}
	for _, k := range []string{"LLM_URL", "LLM_KEY"} {
		if v := os.Getenv(k); v != "" {
			env[k] = v
		}
	}
	env["LLM_URL"] = strings.TrimRight(env["LLM_URL"], "/")
	return env
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
