// Command server serves the api and the player's page. With -mode live or
// replay the advisor of the stack is a model; records are appended on exit.
//
//	go run ./cmd/server -addr localhost:8080 -root internal/scenario/testdata
//	go run ./cmd/server -mode live -model deepseek-no-thinking -records advisor.jsonl -max-rub 5
package main

import (
	"bufio"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/heaprip/intercessio/internal/agenda"
	"github.com/heaprip/intercessio/internal/api"
	"github.com/heaprip/intercessio/internal/front"
	"github.com/heaprip/intercessio/internal/llmruntime"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "address to listen on")
	root := flag.String("root", "internal/scenario/testdata", "directory with schema.json and scenario directories")
	mode := flag.String("mode", "stub", "stub, live, replay or fork-offline advisor")
	model := flag.String("model", "deepseek-no-thinking", "model profile of the advisor")
	records := flag.String("records", "", "JSONL file of call records: read, and appended to on exit")
	maxRub := flag.Float64("max-rub", 5, "spend limit of live calls, rubles")
	flag.Parse()

	srv := &api.Server{Root: *root}
	var rt *llmruntime.Runtime
	if *mode != "stub" {
		spec, ok := llmruntime.Models[*model]
		if !ok {
			log.Fatalf("unknown model %q", *model)
		}
		recs, err := llmruntime.LoadRecords(*records)
		if err != nil {
			log.Fatal(err)
		}
		env := dotenv(".env")
		rt = llmruntime.New(llmruntime.Mode(*mode), env["LLM_URL"], env["LLM_KEY"], *maxRub, recs)
		srv.Advisor = agenda.ModelAdvisor{Runtime: rt, Spec: spec}
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	go func() {
		<-stop
		if rt != nil && *records != "" {
			if err := llmruntime.AppendRecords(*records, rt.Written()); err != nil {
				log.Print(err)
			}
			log.Printf("calls recorded: %d, spent: %.4f rub", len(rt.Written()), rt.Spent())
		}
		os.Exit(0)
	}()

	mux := http.NewServeMux()
	mux.Handle("/api/", srv.Handler())
	mux.Handle("/", front.Handler())
	log.Printf("intercessio on http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
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
