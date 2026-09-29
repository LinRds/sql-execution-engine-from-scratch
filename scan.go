package engine

// FullScan reads every row in the table and keeps the ones the predicate
// accepts. No index is involved, so every condition is checked against the
// full row.
func FullScan(t *Table, p Pred) ([]Row, Stats) {
	panic("FullScan is not implemented")
}

// IndexScan walks the entries in [lo, hi), fetching the full row for each
// one and checking the predicate against it.
//
// The index narrows which rows are worth looking at, but the answer still
// has to come from the row — so every entry in the range costs a fetch.
func IndexScan(t *Table, ix *Index, lo, hi []int64, p Pred) ([]Row, Stats) {
	panic("IndexScan is not implemented")
}

// CoveringScan answers the predicate from the index entries alone, without
// ever reading a row.
//
// It is only correct when the index carries every column the predicate
// touches — checking that is the caller's job, via ix.Covers.
func CoveringScan(t *Table, ix *Index, lo, hi []int64, p Pred) ([]Row, Stats) {
	panic("CoveringScan is not implemented")
}

// Extra renders the EXPLAIN Extra column for this execution.
//
// Report "Using index" exactly when no row was fetched — that flag means
// the index alone answered the query, not merely that an index was used.
func (s Stats) Extra() string {
	panic("Extra is not implemented")
}
