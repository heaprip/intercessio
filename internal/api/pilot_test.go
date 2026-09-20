package api

import (
	"os"
	"testing"
	"testing/fstest"

	"github.com/heaprip/intercessio/internal/agenda"
	"github.com/heaprip/intercessio/internal/llmruntime"
	"github.com/heaprip/intercessio/internal/scenario"
)

// The advisor pilots recorded by `cmd/play -stack 3 -periods 4 -mode live`
// replay offline through the game session: the session asks the same questions
// as the command, every advice comes from the record, none falls back.
func TestPilot_AdvisorReplaysOffline(t *testing.T) {
	for _, model := range []string{"qwen3.7-flash", "granite-4.2-8b"} {
		t.Run(model, func(t *testing.T) {
			recs, err := llmruntime.LoadRecords("../agenda/testdata/advisor-" + model + ".jsonl")
			if err != nil || len(recs) == 0 {
				t.Fatalf("records: %v, %d", err, len(recs))
			}
			schema, err := os.ReadFile("../scenario/testdata/schema.json")
			if err != nil {
				t.Fatal(err)
			}
			world, err := os.ReadFile("../scenario/testdata/playable/scenario.json")
			if err != nil {
				t.Fatal(err)
			}
			sc, _, err := scenario.Load(fstest.MapFS{"schema.json": {Data: schema}, "scenario.json": {Data: world}})
			if err != nil {
				t.Fatal(err)
			}
			rt := llmruntime.New(llmruntime.Replay, "", "", 0, recs)
			g, err := NewGame("pilot", sc, agenda.ModelAdvisor{Runtime: rt, Spec: llmruntime.Models[model]})
			if err != nil {
				t.Fatal(err)
			}
			advised := 0
			for period := 0; period < 4; period++ {
				v := g.Current()
				choices := map[string]string{}
				for _, c := range v.Stack {
					if len(c.Options) > 0 {
						advised++
						if c.Advice.By != "model" || c.Advice.Unserved != "" {
							t.Fatalf("period %d, %s: advice must come from the record: %+v", v.Period, c.Key, c.Advice)
						}
					}
					if c.Amendment {
						choices[c.Key] = "accept"
					}
				}
				if err := g.Decide(choices); err != nil {
					t.Fatal(err)
				}
				if _, err := g.Advance(); err != nil {
					t.Fatal(err)
				}
			}
			// cmd/play in live mode also plays persons and officials with the model;
			// the session keeps stub participants and asks only the advisor
			advisorCalls := 0
			for _, r := range recs {
				if r.Request.Role == "advisor" {
					advisorCalls++
				}
			}
			// the session may have grown since the pilot — a new finding can push a
			// card out of the stack — but every advice it asks for is in the record
			if advised == 0 || advised > advisorCalls {
				t.Fatalf("advised %d cards, recorded %d advisor calls", advised, advisorCalls)
			}
		})
	}
}
