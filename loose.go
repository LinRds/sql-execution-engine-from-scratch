package engine

// RangeCond is one condition on a column: Lo <= value <= Hi.
//
// Lo == Hi means the condition is an equality.
type RangeCond struct {
	Col string
	Lo  int64
	Hi  int64
}

// TightScan returns the distinct values of col by walking every index entry.
//
// Equal values sit next to each other, so comparing the current value with the
// previous one is enough to tell whether it is new.
//
// The predicate filters entries; pass nil to keep everything.
func TightScan(ix *Index, col string, p Pred) ([]int64, Stats) {
	panic("TightScan is not implemented")
}

// LooseScan returns the distinct values of col by reading one entry per group
// and seeking past the rest of that group.
//
// The predicate is evaluated on the entry the scan lands on for each group, so
// conditions on columns other than col are assumed to hold for the whole group.
//
// It sets UsedLooseScan, which Extra() reports as Using index for group-by.
func LooseScan(ix *Index, col string, p Pred) ([]int64, Stats) {
	panic("LooseScan is not implemented")
}

// CanLooseScan reports whether a loose scan over col is allowed.
//
// The index has to carry col, and every condition on a column other than col
// has to be an equality.
func CanLooseScan(ix *Index, col string, conds []RangeCond) bool {
	panic("CanLooseScan is not implemented")
}
