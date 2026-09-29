package engine

// ScanWithoutICP walks the index range and fetches every row it points at,
// then applies the predicate to the fetched row.
//
// The index decides where the walk starts and stops, not which of the rows in
// between survive, so every entry in the range costs one fetch.
func ScanWithoutICP(t *Table, ix *Index, lo, hi []int64, p Pred) ([]Row, Stats) {
	panic("ScanWithoutICP is not implemented")
}

// ScanWithICP walks the index range, judges the part of the predicate the index
// carries on each entry, and fetches only the rows whose entry survived.
//
// The part the index does not carry still has to be applied to the fetched row,
// so a surviving entry is a candidate, not a result. UsedICP is set when both
// parts are present: a condition worth pushing, and a row still worth fetching.
func ScanWithICP(t *Table, ix *Index, lo, hi []int64, p Pred) ([]Row, Stats) {
	panic("ScanWithICP is not implemented")
}
