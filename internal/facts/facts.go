// Package facts holds immutable facts with provenance. Prototype: types only.
package facts

import (
	"fmt"
	"strings"

	"github.com/heaprip/intercessio/internal/period"
)

// Value is a ground argument: a constant or a number.
type Value struct {
	Const string
	Num   int
	IsNum bool
}

func (v Value) String() string {
	if v.IsNum {
		return fmt.Sprint(v.Num)
	}
	return v.Const
}

// Provenance says who stated the fact, when and from where.
type Provenance struct {
	By     string
	Period period.Period
	Source string
}

// Fact is a stored ground atom.
type Fact struct {
	ID   string
	Pred string
	Args []Value
	Prov Provenance
}

func (f Fact) String() string {
	s := make([]string, len(f.Args))
	for i, a := range f.Args {
		s[i] = a.String()
	}
	return f.Pred + "(" + strings.Join(s, ", ") + ")"
}
