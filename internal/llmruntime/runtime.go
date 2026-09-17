// Package llmruntime calls models in roles and can replay any call from its
// record. It knows nothing about the domain.
//
// Prototype. A call is keyed by the hash of the request the model actually
// sees, so a change that does not touch a participant's request reuses the
// record.
package llmruntime

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// Mode says what a call does when its record is missing.
type Mode string

const (
	// Live uses records as a cache; a miss is a live call, recorded.
	Live Mode = "live"
	// Replay is strict: a miss is replay-divergence, never a live call.
	Replay Mode = "replay"
	// ForkOffline: a miss is unserved, and the caller takes its fallback.
	ForkOffline Mode = "fork-offline"
)

// Model names a model, the provider it is pinned to and how its reasoning
// settings are spelled.
type Model struct {
	Name     string
	ID       string
	Provider string
	Params   map[string]any
}

// Models known to the prototype; see docs/impl for each.
var Models = map[string]Model{
	"deepseek-no-thinking": {Name: "deepseek-no-thinking", ID: "deepseek/deepseek-v4.1-flash", Provider: "DeepSeek",
		Params: map[string]any{"thinking": map[string]any{"type": "disabled"}}},
	"deepseek-low": {Name: "deepseek-low", ID: "deepseek/deepseek-v4.1-flash", Provider: "DeepSeek",
		Params: map[string]any{"reasoning_effort": "low"}},
	"granite-4.2-8b": {Name: "granite-4.2-8b", ID: "ibm-granite/granite-4.2-8b", Provider: "DeepInfra"},
	"qwen3.7-flash":  {Name: "qwen3.7-flash", ID: "qwen/qwen3.7-flash", Provider: "Alibaba"},
}

// Request is everything that determines a call.
type Request struct {
	Role          string         `json:"role"`
	PromptVersion string         `json:"prompt_version"`
	Model         string         `json:"model"`
	Provider      string         `json:"provider"`
	Params        map[string]any `json:"params"`
	System        string         `json:"system"`
	User          string         `json:"user"`
}

// Hash keys the record. encoding/json sorts map keys, so it is stable.
func (r Request) Hash() string {
	data, _ := json.Marshal(r)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Record is a call as it happened, served or not.
type Record struct {
	Hash      string   `json:"hash"`
	Request   Request  `json:"request"`
	Content   string   `json:"content,omitempty"`
	Reasoning string   `json:"reasoning,omitempty"`
	Provider  string   `json:"provider,omitempty"`
	Cost      float64  `json:"cost"`
	Rejected  []string `json:"rejected,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// Status of a reply.
type Status string

const (
	Served   Status = "served"
	Unserved Status = "unserved"
)

// Reply is the outcome of a call. Unserved is an outcome, not an error.
type Reply struct {
	Status     Status
	Reason     string
	Content    string
	Hash       string
	FromRecord bool
}

// Runtime is one run's access to models.
type Runtime struct {
	Mode   Mode
	URL    string
	Key    string
	MaxRub float64
	HTTP   *http.Client

	mu      sync.Mutex
	records map[string]Record
	written []Record
	spent   float64
}

// New builds a runtime over existing records.
func New(mode Mode, url, key string, maxRub float64, records []Record) *Runtime {
	rt := &Runtime{Mode: mode, URL: url, Key: key, MaxRub: maxRub, HTTP: &http.Client{Timeout: 2 * time.Minute}, records: map[string]Record{}}
	for _, r := range records {
		rt.records[r.Hash] = r
	}
	return rt
}

// Written returns the records made by live calls of this run.
func (rt *Runtime) Written() []Record {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return append([]Record{}, rt.written...)
}

// Spent returns rubles spent by live calls of this run.
func (rt *Runtime) Spent() float64 {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.spent
}

const formatRetries = 2

// Call answers a request. validate checks the content; a live call re-asks the
// model with the problem at most formatRetries times.
func (rt *Runtime) Call(req Request, validate func(content string) error) Reply {
	h := req.Hash()
	rt.mu.Lock()
	rec, ok := rt.records[h]
	rt.mu.Unlock()
	if ok {
		if rec.Error != "" {
			return Reply{Status: Unserved, Reason: rec.Error, Hash: h, FromRecord: true}
		}
		if err := validate(rec.Content); err != nil {
			return Reply{Status: Unserved, Reason: "invalid record: " + err.Error(), Hash: h, FromRecord: true}
		}
		return Reply{Status: Served, Content: rec.Content, Hash: h, FromRecord: true}
	}
	switch rt.Mode {
	case Replay:
		return Reply{Status: Unserved, Reason: "replay-divergence", Hash: h}
	case ForkOffline:
		return Reply{Status: Unserved, Reason: "offline", Hash: h}
	}
	rec = rt.live(req, validate)
	rec.Hash = h
	rt.mu.Lock()
	rt.records[h] = rec
	rt.written = append(rt.written, rec)
	rt.mu.Unlock()
	if rec.Error != "" {
		return Reply{Status: Unserved, Reason: rec.Error, Hash: h}
	}
	return Reply{Status: Served, Content: rec.Content, Hash: h}
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (rt *Runtime) live(req Request, validate func(string) error) Record {
	rec := Record{Request: req}
	if rt.URL == "" || rt.Key == "" {
		rec.Error = "no provider configured"
		return rec
	}
	msgs := []message{{"system", req.System}, {"user", req.User}}
	for attempt := 0; attempt <= formatRetries; attempt++ {
		rt.mu.Lock()
		exhausted := rt.spent >= rt.MaxRub
		rt.mu.Unlock()
		if exhausted {
			rec.Error = "budget exhausted"
			return rec
		}
		body := map[string]any{
			"model": req.Model, "max_tokens": 16000, "usage": map[string]any{"include": true},
			"provider": map[string]any{"order": []string{req.Provider}, "allow_fallbacks": false},
			"messages": msgs,
		}
		for k, v := range req.Params {
			body[k] = v
		}
		resp, err := rt.post(body)
		if err != nil {
			rec.Error = err.Error()
			return rec
		}
		rt.mu.Lock()
		rt.spent += resp.Usage.Cost
		rt.mu.Unlock()
		rec.Cost += resp.Usage.Cost
		rec.Provider = resp.Provider
		if len(resp.Choices) == 0 {
			rec.Error = "provider returned no choices"
			return rec
		}
		content := resp.Choices[0].Message.Content
		rec.Reasoning = resp.Choices[0].Message.Reasoning
		if err := validate(content); err != nil {
			rec.Rejected = append(rec.Rejected, content)
			msgs = append(msgs,
				message{"assistant", content},
				message{"user", fmt.Sprintf("Your reply could not be used: %v. Reply again with the JSON object only.", err)})
			continue
		}
		rec.Content = content
		return rec
	}
	rec.Error = "no usable reply after re-asking"
	return rec
}

type chatResponse struct {
	Provider string `json:"provider"`
	Choices  []struct {
		Message struct {
			Content   string `json:"content"`
			Reasoning string `json:"reasoning"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		Cost float64 `json:"cost"`
	} `json:"usage"`
}

func (rt *Runtime) post(body map[string]any) (*chatResponse, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(1<<attempt) * time.Second)
		}
		httpReq, err := http.NewRequest(http.MethodPost, rt.URL+"/chat/completions", bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Authorization", "Bearer "+rt.Key)
		httpReq.Header.Set("Content-Type", "application/json")
		res, err := rt.HTTP.Do(httpReq)
		if err != nil {
			last = fmt.Errorf("connection: %w", err)
			continue
		}
		raw, _ := io.ReadAll(res.Body)
		res.Body.Close()
		switch {
		case res.StatusCode == http.StatusOK:
			var out chatResponse
			if err := json.Unmarshal(raw, &out); err != nil {
				return nil, fmt.Errorf("decode response: %w", err)
			}
			return &out, nil
		case res.StatusCode == 429 || res.StatusCode >= 500:
			last = fmt.Errorf("http %d", res.StatusCode)
		default:
			return nil, fmt.Errorf("http %d: %.300s", res.StatusCode, raw)
		}
	}
	return nil, last
}

// LoadRecords reads a JSONL file; a missing file is no records.
func LoadRecords(path string) ([]Record, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Record
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 64<<20)
	for sc.Scan() {
		var r Record
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out = append(out, r)
	}
	return out, sc.Err()
}

// AppendRecords appends records to a JSONL file.
func AppendRecords(path string, records []Record) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, r := range records {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	return nil
}
