package deduction

import "sort"

// Step is one link of a support path: the predicate and the rule that concludes
// it from the next predicate of the path. The last step has no rule.
type Step struct {
	Pred string
	Rule string
}

// Supports walks the rule graph back from pred through the positive body
// literals of rules with a positive head, and returns every predicate reached
// with one shortest path from pred down to it. It answers "what does this
// condition rest on" without any domain: callers decide which of the reached
// predicates matter.
func Supports(rules []Rule, pred string) map[string][]Step {
	byHead := map[string][]Rule{}
	for _, r := range rules {
		if !r.Head.Neg && r.Strength != Defeater {
			byHead[r.Head.Pred] = append(byHead[r.Head.Pred], r)
		}
	}
	for _, rs := range byHead {
		sort.Slice(rs, func(i, j int) bool { return rs[i].ID < rs[j].ID })
	}
	out := map[string][]Step{pred: {{Pred: pred}}}
	queue := []string{pred}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, r := range byHead[p] {
			for _, l := range r.Body {
				if l.Builtin() || l.Neg {
					continue
				}
				if _, seen := out[l.Pred]; seen {
					continue
				}
				path := append([]Step{}, out[p]...)
				path[len(path)-1].Rule = r.ID
				out[l.Pred] = append(path, Step{Pred: l.Pred})
				queue = append(queue, l.Pred)
			}
		}
	}
	return out
}
