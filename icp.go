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
	canUseICP := canUseICP(ix, p)
	if !canUseICP {
		return ScanWithoutICP(t, ix, lo, hi, p)
	}
	inRange := ix.RangeScan(lo, hi)
	stats := Stats{
		IndexEntriesRead: len(inRange),
		UsedICP:          canUseICP,
	}
	filtered := make([]Row, 0)
	for _, in := range inRange {
		if !p.evalICP(ix, in) {
			continue
		}
		row := t.ByID[in.RowID]
		stats.RowsFetched++
		if p.eval(row) {
			filtered = append(filtered, row)
			stats.RowsReturned++
		}
	}
	return filtered, stats
}

func canUseICP(ix *Index, p Pred) bool {
	needback := false
	canFilter := false
	for _, c := range p {
		idx := ix.ColIndex(c.Col)
		if idx < 0 && c.Col != "id" {
			needback = true
		} else {
			canFilter = true
		}
	}
	return needback && canFilter
}
