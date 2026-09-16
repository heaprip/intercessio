package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func call(t *testing.T, srv *httptest.Server, method, path string, body any, out any) int {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, srv.URL+path, &buf)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("%s %s: decode: %v", method, path, err)
		}
	}
	return resp.StatusCode
}

// Gate of external-view: a game goes through HTTP only — created from a
// scenario, its stack decided, a period lived, the lived period browsed, a
// conclusion traced.
func TestAPI_GameThroughHTTP(t *testing.T) {
	srv := httptest.NewServer((&Server{Root: "../scenario/testdata"}).Handler())
	defer srv.Close()

	var names []string
	if call(t, srv, "GET", "/api/scenarios", nil, &names) != http.StatusOK || !strings.Contains(strings.Join(names, ","), "playable") {
		t.Fatalf("scenarios: %v", names)
	}

	var v View
	if code := call(t, srv, "POST", "/api/games", map[string]string{"scenario": "playable"}, &v); code != http.StatusCreated {
		t.Fatalf("create: %d", code)
	}
	if v.Period != 30 || len(v.Stack) == 0 || len(v.Graph.Offices) == 0 || len(v.Findings) == 0 {
		t.Fatalf("the first view must show a stack, findings and the graph: %+v", v)
	}
	id := v.Game

	choices := map[string]string{}
	for _, c := range v.Stack {
		choices[c.Key] = "reject"
	}
	if code := call(t, srv, "POST", "/api/games/"+id+"/decisions", choices, &v); code != http.StatusOK || v.Stack[0].Choice != "reject" {
		t.Fatalf("decisions: %d %+v", code, v.Stack)
	}
	var bad map[string]string
	if code := call(t, srv, "POST", "/api/games/"+id+"/decisions", map[string]string{"no|such": "accept"}, &bad); code != http.StatusUnprocessableEntity {
		t.Fatalf("a card not in the stack must be refused: %d", code)
	}

	if code := call(t, srv, "POST", "/api/games/"+id+"/advance", nil, &v); code != http.StatusOK || v.Period != 31 {
		t.Fatalf("advance: %d period %d", code, v.Period)
	}
	var lived View
	if code := call(t, srv, "GET", "/api/games/"+id+"/periods/30", nil, &lived); code != http.StatusOK || !lived.Lived || len(lived.Report) == 0 || len(lived.Journal) == 0 {
		t.Fatalf("the lived period must carry its report and journal: %d %+v", code, lived)
	}
	if len(v.Stand) == 0 {
		t.Fatal("from the second period the stand must be shown")
	}

	var trace map[string]string
	q := url.Values{"period": {"30"}, "atom": {"entitled(marcus, civis)"}}
	if code := call(t, srv, "GET", "/api/games/"+id+"/trace?"+q.Encode(), nil, &trace); code != http.StatusOK || !strings.Contains(trace["trace"], "r_c1") {
		t.Fatalf("trace: %d %v", code, trace)
	}

	if code := call(t, srv, "POST", "/api/games", map[string]string{"scenario": "../etc"}, &bad); code != http.StatusBadRequest {
		t.Fatalf("a scenario name must not be a path: %d", code)
	}
}

func TestParseAtom(t *testing.T) {
	a, err := ParseAtom("¬status(marcus, civis)")
	if err != nil || !a.Neg || a.Pred != "status" || a.String() != "¬status(marcus, civis)" {
		t.Fatalf("got %v %v", a, err)
	}
	if a, err := ParseAtom("tenure(marcus, 25)"); err != nil || !a.Args[1].IsNum {
		t.Fatalf("numbers: %v %v", a, err)
	}
	if _, err := ParseAtom("status"); err == nil {
		t.Fatal("an atom without arguments must be refused")
	}
}
