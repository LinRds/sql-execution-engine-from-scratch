package step2

// Table holds the rows an index points into.
type Table struct {
	Rows []Row
	ByID map[int64]Row
}

// Cond is one condition: a column has to satisfy a predicate.
type Cond struct {
	Col string
	Ok  func(int64) bool
}

// Pred is a list of conditions, all of which must hold.
type Pred []Cond

// Stats is what one execution cost.
//
// The three counters are kept apart because they are not the same kind of
// work: reading an index entry, fetching a row at random, and reading a row
// in a sequential scan all cost different amounts. See Cost.
type Stats struct {
	IndexEntriesRead int
	RowsFetched      int
	RowsScanned      int
	RowsReturned     int
	UsedTempTable    bool
	UsedSort         bool
}

const (
	// FetchCost is what one random row fetch costs, in index-entry units.
	// A fetched row is wider than an index entry, and reaching it means a
	// random seek rather than a walk. Both make it cost far more.
	FetchCost = 20

	// ScanCost is what one row costs in a sequential full scan. The scan
	// reads everything anyway, in order, so each row is cheap.
	ScanCost = 1
)

// Cost is the total work, in index-entry units.
func (s Stats) Cost() int {
	return s.IndexEntriesRead + s.RowsFetched*FetchCost + s.RowsScanned*ScanCost
}

// FullScan reads every row in the table and keeps the ones the predicate
// accepts. No index is involved, so every condition is checked against the
// full row.
func FullScan(t *Table, p Pred) ([]Row, Stats) {
	panic("not implemented")
}

// IndexScan walks the entries in [lo, hi), fetching the full row for each
// one and checking the predicate against it.
//
// The index narrows which rows are worth looking at, but the answer still
// has to come from the row — so every entry in the range costs a fetch.
func IndexScan(t *Table, ix *Index, lo, hi []int64, p Pred) ([]Row, Stats) {
	panic("not implemented")
}

// CoveringScan answers the predicate from the index entries alone, without
// ever reading a row.
//
// It is only correct when the index carries every column the predicate
// touches — checking that is the caller's job, via ix.Covers.
func CoveringScan(t *Table, ix *Index, lo, hi []int64, p Pred) ([]Row, Stats) {
	panic("not implemented")
}

// Extra renders the EXPLAIN Extra column for this execution.
//
// Report "Using index" exactly when no row was fetched — that flag means
// the index alone answered the query, not merely that an index was used.
func (s Stats) Extra() string {
	panic("not implemented")
}
