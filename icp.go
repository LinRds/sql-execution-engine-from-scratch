package engine

// ScanWithoutICP walks the index range and fetches every row it points at,
// then applies the predicate to the fetched row.
//
// The index decides where the walk starts and stops, not which of the rows in
// between survive, so every entry in the range costs one fetch.
func ScanWithoutICP(t *Table, ix *Index, lo, hi []int64, p Pred) ([]Row, Stats) {
	return IndexScan(t, ix, lo, hi, p)
}

// ScanWithICP walks the index range, judges the part of the predicate the index
// carries on each entry, and fetches only the rows whose entry survived.
//
// The part the index does not carry still has to be applied to the fetched row,
// so a surviving entry is a candidate, not a result. UsedICP is set when both
// parts are present: a condition worth pushing, and a row still worth fetching.
func ScanWithICP(t *Table, ix *Index, lo, hi []int64, p Pred) ([]Row, Stats) {
	onEntry, onRow := splitPred(ix, p)
	if len(onEntry) == 0 || len(onRow) == 0 {
		return ScanWithoutICP(t, ix, lo, hi, p)
	}
	inRange := ix.RangeScan(lo, hi)
	stats := Stats{
		IndexEntriesRead: len(inRange),
		UsedICP:          true,
	}
	filtered := make([]Row, 0)
	for _, in := range inRange {
		if !onEntry.evalEntry(ix, in) {
			continue
		}
		row := t.ByID[in.RowID]
		stats.RowsFetched++
		if onRow.eval(row) {
			filtered = append(filtered, row)
			stats.RowsReturned++
		}
	}
	return filtered, stats
}

func splitPred(ix *Index, p Pred) (onEntry, onRow Pred) {
	for _, c := range p {
		if c.Col == "id" || ix.ColIndex(c.Col) >= 0 {
			onEntry = append(onEntry, c)
		} else {
			onRow = append(onRow, c)
		}
	}
	return onEntry, onRow
}
