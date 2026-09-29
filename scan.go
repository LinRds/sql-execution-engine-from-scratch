package engine

import "strings"

// FullScan reads every row in the table and keeps the ones the predicate
// accepts. No index is involved, so every condition is checked against the
// full row.
func FullScan(t *Table, p Pred) ([]Row, Stats) {
	filtered := make([]Row, 0)
	stats := Stats{}
	for _, row := range t.Rows {
		stats.RowsScanned++
		if p.eval(row) {
			filtered = append(filtered, row)
			stats.RowsReturned++
		}
	}
	return filtered, stats
}

// IndexScan walks the entries in [lo, hi), fetching the full row for each
// one and checking the predicate against it.
//
// The index narrows which rows are worth looking at, but the answer still
// has to come from the row — so every entry in the range costs a fetch.
func IndexScan(t *Table, ix *Index, lo, hi []int64, p Pred) ([]Row, Stats) {
	inRange := ix.RangeScan(lo, hi)
	stats := Stats{
		IndexEntriesRead: len(inRange),
	}
	filtered := make([]Row, 0)
	for _, in := range inRange {
		row := t.ByID[in.RowID]
		stats.RowsFetched++
		if p.eval(row) {
			filtered = append(filtered, row)
			stats.RowsReturned++
		}
	}
	return filtered, stats
}

// CoveringScan answers the predicate from the index entries alone, without
// ever reading a row.
//
// It assumes the caller has already established that the index can serve the
// query: that it carries every column the predicate touches, and every column
// of the row this scan hands back. Choosing this path is the optimizer's job;
// by the time a scan runs, the choice is made.
func CoveringScan(t *Table, ix *Index, lo, hi []int64, p Pred) ([]Row, Stats) {
	if !ix.Covers(p) {
		return IndexScan(t, ix, lo, hi, p)
	}
	inRange := ix.RangeScan(lo, hi)
	filtered := make([]Row, 0)
	stats := Stats{}
	for _, in := range inRange {
		stats.IndexEntriesRead++
		if p.evalEntry(ix, in) {
			stats.RowsReturned++
			filtered = append(filtered, in.ToRow(ix))
		}
	}
	return filtered, stats
}

// Extra renders the EXPLAIN Extra column for this execution.
//
// Report "Using index" only when an index was walked and no row was fetched.
// Both halves matter: an index scan that still read every row is not covered,
// and neither is a full scan — it fetches nothing because it uses no index.
func (s Stats) Extra() string {
	used := make([]string, 0)
	if s.UsedTempTable {
		used = append(used, "Using temporary")
	}
	if s.RowsFetched+s.RowsScanned == 0 {
		used = append(used, "Using index")
	}
	if s.UsedLooseScan {
		used = append(used, "Using index for group-by")
	}
	if s.UsedICP {
		used = append(used, "Using index condition")
	}
	return strings.Join(used, ";")
}
