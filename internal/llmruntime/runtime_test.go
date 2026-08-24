package llmruntime

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func fakeProvider(t *testing.T, replies ...string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := int(n.Add(1)) - 1
		if i >= len(replies) {
			i = len(replies) - 1
		}
		json.NewEncoder(w).Encode(map[string]any{
			"provider": "Fake",
			"choices":  []any{map[string]any{"message": map[string]any{"content": replies[i]}}},
			"usage":    map[string]any{"cost": 0.01},
		})
	}))
	t.Cleanup(srv.Close)
	return srv, &n
}

func mustJSON(s string) error {
	if !strings.HasPrefix(s, "{") {
		return errors.New("not a JSON object")
	}
	return nil
}

var req = Request{Role: "official", PromptVersion: "v1", Model: "m", Provider: "Fake", System: "s", User: "u"}

func TestCall_ReasksOnInvalidAndRecords(t *testing.T) {
	srv, calls := fakeProvider(t, "garbage", `{"outcome":"grant"}`)
	rt := New(Live, srv.URL, "k", 1, nil)
	got := rt.Call(req, mustJSON)
	if got.Status != Served || got.Content != `{"outcome":"grant"}` || calls.Load() != 2 {
		t.Fatalf("got %+v after %d calls", got, calls.Load())
	}
	w := rt.Written()
	if len(w) != 1 || len(w[0].Rejected) != 1 || w[0].Cost != 0.02 {
		t.Fatalf("record: %+v", w)
	}
}

func TestCall_ReplayUsesRecordWithoutNetwork(t *testing.T) {
	srv, _ := fakeProvider(t, `{"outcome":"grant"}`)
	live := New(Live, srv.URL, "k", 1, nil)
	live.Call(req, mustJSON)

	replay := New(Replay, "", "", 0, live.Written())
	if got := replay.Call(req, mustJSON); got.Status != Served || !got.FromRecord {
		t.Fatalf("replay: %+v", got)
	}
	changed := req
	changed.PromptVersion = "v2"
	if got := replay.Call(changed, mustJSON); got.Status != Unserved || got.Reason != "replay-divergence" {
		t.Fatalf("changed request: %+v", got)
	}
	if got := New(ForkOffline, "", "", 0, nil).Call(req, mustJSON); got.Status != Unserved || got.Reason != "offline" {
		t.Fatalf("offline: %+v", got)
	}
}

func TestCall_BudgetExhaustedIsUnserved(t *testing.T) {
	srv, calls := fakeProvider(t, `{"outcome":"grant"}`)
	rt := New(Live, srv.URL, "k", 0, nil)
	if got := rt.Call(req, mustJSON); got.Status != Unserved || got.Reason != "budget exhausted" || calls.Load() != 0 {
		t.Fatalf("got %+v after %d calls", got, calls.Load())
	}
}
