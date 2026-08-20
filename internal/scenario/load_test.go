package scenario

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

var update = flag.Bool("update", false, "rewrite golden reports")

func load(t *testing.T, dir string) *Report {
	t.Helper()
	_, rep := loadScenario(t, dir)
	return rep
}

func loadScenario(t *testing.T, dir string) (*Scenario, *Report) {
	t.Helper()
	schema, err := os.ReadFile(filepath.Join("testdata", "schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	world, err := os.ReadFile(filepath.Join("testdata", dir, "scenario.json"))
	if err != nil {
		t.Fatal(err)
	}
	s, rep, err := Load(fstest.MapFS{"schema.json": {Data: schema}, "scenario.json": {Data: world}})
	if err != nil {
		t.Fatal(err)
	}
	return s, rep
}

func golden(t *testing.T, dir, got string) {
	t.Helper()
	path := filepath.Join("testdata", dir, "report.golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update)", err)
	}
	if string(want) != got {
		t.Fatalf("report differs from %s (run with -update):\n%s", path, got)
	}
}

func TestLoad_OriginalCasus(t *testing.T) {
	rep := load(t, "originals")
	if len(rep.Errors) == 0 {
		t.Fatal("original casus must not load cleanly")
	}
	golden(t, "originals", rep.String())
}

func TestLoad_RewrittenCasus(t *testing.T) {
	for _, dir := range []string{"citizenship", "office-eligibility"} {
		t.Run(dir, func(t *testing.T) {
			rep := load(t, dir)
			if len(rep.Errors) > 0 {
				t.Fatalf("rewritten casus must load without errors:\n%s", rep)
			}
			golden(t, dir, rep.String())
		})
	}
}
