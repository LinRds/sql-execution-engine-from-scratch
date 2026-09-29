package engine

// DistinctOrdered deduplicates a column that leads the index.
//
// Equal values sit next to each other in the index, so comparing the current
// value with the previous one is enough to tell whether it is new.
//
// The predicate filters entries; pass nil to keep everything.
func DistinctOrdered(ix *Index, col string, p Pred) ([]int64, Stats) {
	panic("DistinctOrdered is not implemented")
}

// DistinctTempTable deduplicates a column that does not lead the index.
//
// Equal values are scattered, so there is no previous value to compare
// against — every value seen so far has to be remembered.
//
// The predicate filters entries; pass nil to keep everything.
func DistinctTempTable(ix *Index, col string, p Pred) ([]int64, Stats) {
	panic("DistinctTempTable is not implemented")
}
