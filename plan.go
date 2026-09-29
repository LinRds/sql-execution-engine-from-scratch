package engine

import "math"

// TableStats is what the optimizer holds about a table: estimates, not facts.
type TableStats struct {
	Rows   int // estimated row count
	Groups int // estimated number of distinct values (cardinality)
}

// Plan is one candidate execution plan.
type Plan struct {
	Name string
	// Usable reports whether this plan can run at all under the predicate and
	// the index on offer.
	Usable func(p Pred, ix *Index) bool
	// Est is the plan's cost, estimated from the statistics.
	Est func(st TableStats, ix *Index, p Pred) int
}

// ChoosePlan returns the usable plan with the smallest estimated cost.
//
// It returns the zero Plan and false when no plan is usable.
func ChoosePlan(plans []Plan, p Pred, ix *Index, st TableStats) (Plan, bool) {
	cost := math.MaxInt
	var bestPlan *Plan
	for _, plan := range plans {
		if !plan.Usable(p, ix) {
			continue
		}
		curCost := plan.Est(st, ix, p)
		if curCost < cost {
			bestPlan = &plan
			cost = curCost
		}
	}
	if bestPlan != nil {
		return *bestPlan, true
	}
	return Plan{}, false
}
