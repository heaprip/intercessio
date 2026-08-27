package deduction

import (
	"reflect"
	"testing"
)

func TestSupports_WalksPositiveBodiesBack(t *testing.T) {
	rules := []Rule{
		rule("r_goal", lit("goal", v("X")), lit("mid", v("X")), lit(">", v("X"), n(1))),
		rule("r_mid", lit("mid", v("X")), lit("act", v("X"))),
		rule("r_block", neg("goal", v("X")), lit("barrier", v("X"))),
		rule("r_unless", lit("goal", v("X")), neg("other", v("X"))),
	}
	got := Supports(rules, "goal")
	want := []Step{{"goal", "r_goal"}, {"mid", "r_mid"}, {"act", ""}}
	if !reflect.DeepEqual(got["act"], want) {
		t.Fatalf("path to act: got %v, want %v", got["act"], want)
	}
	for _, p := range []string{"barrier", "other", ">"} {
		if _, ok := got[p]; ok {
			t.Errorf("%s must not support goal: negative heads, negative bodies and builtins are not supports", p)
		}
	}
}
