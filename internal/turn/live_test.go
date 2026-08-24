package turn

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/heaprip/intercessio/internal/actors"
	"github.com/heaprip/intercessio/internal/journal"
	"github.com/heaprip/intercessio/internal/llmruntime"
)

// fakeProvider answers like a cooperative model: persons petition, officials
// follow the normative result.
func fakeProvider(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct{ Content string } `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		user := body.Messages[1].Content
		content := `{"move": "petition", "reason": "entitled"}`
		switch {
		case strings.Contains(user, "Allowed outcomes: guilty"):
			content = `{"outcome": "guilty", "reason": "the violation is proved"}`
			if !strings.Contains(user, "Normative result: proved") {
				content = `{"outcome": "non-liquet", "reason": "not proved"}`
			}
		case strings.Contains(user, "Allowed outcomes: grant"):
			content = `{"outcome": "grant", "reason": "entitled"}`
			if !strings.Contains(user, "Normative result: proved") {
				content = `{"outcome": "refuse", "reason": "not entitled"}`
			}
		}
		json.NewEncoder(w).Encode(map[string]any{
			"provider": "Fake",
			"choices":  []any{map[string]any{"message": map[string]any{"content": content}}},
			"usage":    map[string]any{"cost": 0.001},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func modelConfig(rt *llmruntime.Runtime, version string) Config {
	cfg := base
	cfg.Actors = actors.Model{Stub: actors.Stub{Script: script}, Runtime: rt, Spec: llmruntime.Model{Name: "fake", ID: "fake/model", Provider: "Fake"}, Version: version}
	return cfg
}

func journalOf(t *testing.T, cfg Config, periods int) journal.Journal {
	t.Helper()
	return play(t, start(t, nil), cfg, periods).Journal
}

// Gate of live-actors: recorded outputs give an exact replay of the periods,
// without the network.
func TestLive_RecordedCallsReplayExactly(t *testing.T) {
	srv := fakeProvider(t)
	live := llmruntime.New(llmruntime.Live, srv.URL, "k", 1, nil)
	recorded := journalOf(t, modelConfig(live, ""), 3)
	if len(live.Written()) == 0 {
		t.Fatal("no calls were recorded")
	}
	if !has(recorded, journal.Decision, 30, "by=model model=fake") {
		dump(t, recorded)
		t.Fatal("no decision by the model")
	}

	replay := llmruntime.New(llmruntime.Replay, "", "", 0, live.Written())
	replayed := journalOf(t, modelConfig(replay, ""), 3)
	if !reflect.DeepEqual(recorded, replayed) {
		dump(t, replayed)
		t.Fatal("replay from records differs from the recorded run")
	}
}

// Real outputs of DeepSeek V4.1 Flash without reasoning, recorded by
// `cmd/play -mode live`, replay offline into model decisions.
func TestLive_RealPilotReplaysOffline(t *testing.T) {
	recs, err := llmruntime.LoadRecords("testdata/pilot-deepseek-no-thinking.jsonl")
	if err != nil || len(recs) == 0 {
		t.Fatalf("records: %v, %d", err, len(recs))
	}
	rt := llmruntime.New(llmruntime.Replay, "", "", 0, recs)
	cfg := base
	cfg.Actors = actors.Model{Stub: actors.Stub{Script: script}, Runtime: rt, Spec: llmruntime.Models["deepseek-no-thinking"]}
	j := journalOf(t, cfg, 2)
	if len(j.Of(journal.Unserved)) > 0 {
		dump(t, j)
		t.Fatal("recorded pilot must replay without divergence")
	}
	for _, want := range []struct {
		kind     journal.Kind
		now      int
		contains string
	}{
		{journal.PetitionFiled, 30, "by=model model=deepseek-no-thinking"},
		{journal.Decision, 30, "subject=marcus civis outcome=grant rule=r_c1 by=model"},
		{journal.Decision, 31, "subject=marcus munus outcome=guilty"},
	} {
		if !has(j, want.kind, want.now, want.contains) {
			dump(t, j)
			t.Fatalf("no %s in period %d containing %q", want.kind, want.now, want.contains)
		}
	}
}

// A changed prompt changes every request: strict replay does not call the
// model, it reports divergence, and the declared fallbacks apply.
func TestLive_ChangedPromptDiverges(t *testing.T) {
	srv := fakeProvider(t)
	live := llmruntime.New(llmruntime.Live, srv.URL, "k", 1, nil)
	journalOf(t, modelConfig(live, ""), 1)

	replay := llmruntime.New(llmruntime.Replay, "", "", 0, live.Written())
	j := journalOf(t, modelConfig(replay, "v2"), 1)
	if !has(j, journal.Unserved, 30, "outcome=replay-divergence") {
		dump(t, j)
		t.Fatal("no divergence reported")
	}
	if len(j.Of(journal.Decision)) > 0 || len(j.Of(journal.PetitionFiled)) > 0 {
		dump(t, j)
		t.Fatal("fallbacks must not decide or petition")
	}
}

// Offline fork: no records, no network. Persons do nothing, cases would wait.
func TestLive_OfflineForkTakesFallbacks(t *testing.T) {
	offline := llmruntime.New(llmruntime.ForkOffline, "", "", 0, nil)
	j := journalOf(t, modelConfig(offline, ""), 1)
	if !has(j, journal.Unserved, 30, "actor=marcus") || len(j.Of(journal.PetitionFiled)) > 0 {
		dump(t, j)
		t.Fatal("offline person must fall back to doing nothing")
	}
	// the charge path has no model person, so the official is asked and waits
	if !has(j, journal.Unserved, 30, "office=praetor") || !has(j, journal.ChargeFiled, 30, "gaius") {
		dump(t, j)
		t.Fatal("offline official must leave the case open")
	}
}
