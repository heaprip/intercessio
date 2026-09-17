package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"testing/fstest"

	"github.com/heaprip/intercessio/internal/agenda"
	"github.com/heaprip/intercessio/internal/scenario"
)

// Server holds games in memory and serves them over HTTP.
type Server struct {
	// Root is the directory with schema.json and one directory per scenario.
	Root string
	// Advisor chooses amendments for cards; nil is the stub.
	Advisor agenda.Advisor
	mu      sync.Mutex
	games   map[string]*Game
	next    int
}

var scenarioName = regexp.MustCompile(`^[a-z0-9-]+$`)

// Handler routes the API:
//
//	GET  /api/scenarios                      scenario names
//	POST /api/games {"scenario": "playable"} start a game, returns the current view
//	GET  /api/games/{id}                     the current view
//	GET  /api/games/{id}/periods/{n}         a lived or the current period
//	POST /api/games/{id}/decisions {"key": "accept"|"reject"}
//	POST /api/games/{id}/advance             live the period, returns the next view
//	GET  /api/games/{id}/trace?period=n&atom=entitled(marcus, civis)
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/scenarios", s.scenarios)
	mux.HandleFunc("POST /api/games", s.create)
	mux.HandleFunc("GET /api/games/{id}", s.withGame(func(w http.ResponseWriter, r *http.Request, g *Game) {
		reply(w, http.StatusOK, g.Current())
	}))
	mux.HandleFunc("GET /api/games/{id}/periods/{n}", s.withGame(func(w http.ResponseWriter, r *http.Request, g *Game) {
		n, err := strconv.Atoi(r.PathValue("n"))
		if err != nil {
			fail(w, http.StatusBadRequest, "period must be a number")
			return
		}
		v, err := g.Period(n)
		if err != nil {
			fail(w, http.StatusNotFound, err.Error())
			return
		}
		reply(w, http.StatusOK, v)
	}))
	mux.HandleFunc("POST /api/games/{id}/decisions", s.withGame(func(w http.ResponseWriter, r *http.Request, g *Game) {
		var choices map[string]string
		if err := json.NewDecoder(r.Body).Decode(&choices); err != nil {
			fail(w, http.StatusBadRequest, "body must be an object of card keys to accept or reject")
			return
		}
		if err := g.Decide(choices); err != nil {
			fail(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		reply(w, http.StatusOK, g.Current())
	}))
	mux.HandleFunc("POST /api/games/{id}/advance", s.withGame(func(w http.ResponseWriter, r *http.Request, g *Game) {
		v, err := g.Advance()
		if err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		reply(w, http.StatusOK, v)
	}))
	mux.HandleFunc("GET /api/games/{id}/trace", s.withGame(func(w http.ResponseWriter, r *http.Request, g *Game) {
		n, err := strconv.Atoi(r.URL.Query().Get("period"))
		if err != nil {
			fail(w, http.StatusBadRequest, "period must be a number")
			return
		}
		text, err := g.Trace(n, r.URL.Query().Get("atom"))
		if err != nil {
			fail(w, http.StatusBadRequest, err.Error())
			return
		}
		reply(w, http.StatusOK, map[string]string{"trace": text})
	}))
	return mux
}

func (s *Server) scenarios(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(s.Root)
	if err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	names := []string{}
	for _, e := range entries {
		if e.IsDir() && scenarioName.MatchString(e.Name()) {
			if _, err := os.Stat(filepath.Join(s.Root, e.Name(), "scenario.json")); err == nil {
				names = append(names, e.Name())
			}
		}
	}
	reply(w, http.StatusOK, names)
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Scenario string `json:"scenario"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !scenarioName.MatchString(body.Scenario) {
		fail(w, http.StatusBadRequest, "body must name a scenario: lowercase letters, digits and dashes")
		return
	}
	schema, err := os.ReadFile(filepath.Join(s.Root, "schema.json"))
	if err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	world, err := os.ReadFile(filepath.Join(s.Root, body.Scenario, "scenario.json"))
	if err != nil {
		fail(w, http.StatusNotFound, fmt.Sprintf("no scenario %q", body.Scenario))
		return
	}
	sc, rep, err := scenario.Load(fstest.MapFS{"schema.json": {Data: schema}, "scenario.json": {Data: world}})
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if len(rep.Errors) > 0 {
		fail(w, http.StatusUnprocessableEntity, "scenario has errors:\n"+rep.String())
		return
	}
	s.mu.Lock()
	if s.games == nil {
		s.games = map[string]*Game{}
	}
	s.next++
	id := fmt.Sprintf("g%d", s.next)
	s.mu.Unlock()
	g, err := NewGame(id, sc, s.Advisor)
	if err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.mu.Lock()
	s.games[id] = g
	s.mu.Unlock()
	reply(w, http.StatusCreated, g.Current())
}

func (s *Server) withGame(h func(http.ResponseWriter, *http.Request, *Game)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		g := s.games[r.PathValue("id")]
		s.mu.Unlock()
		if g == nil {
			fail(w, http.StatusNotFound, "no such game")
			return
		}
		h(w, r, g)
	}
}

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, status int, msg string) {
	reply(w, status, map[string]string{"error": msg})
}
