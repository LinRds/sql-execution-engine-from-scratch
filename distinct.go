package engine

// DistinctOrdered deduplicates a column that leads the index.
//
// Equal values sit next to each other in the index, so comparing the current
// value with the previous one is enough to tell whether it is new.
//
// The predicate filters entries; pass nil to keep everything.
func DistinctOrdered(ix *Index, col string, p Pred) ([]int64, Stats) {
	stats := Stats{}
	idx := ix.ColIndex(col)
	if idx == -1 {
		return nil, stats
	}
	filtered := make([]int64, 0)
	for _, entry := range ix.Keys {
		stats.IndexEntriesRead++
		if !p.evalEntry(ix, entry) {
			continue
		}
		if len(filtered) > 0 && filtered[len(filtered)-1] == entry.Key[idx] {
			continue
		}
		stats.RowsReturned++
		filtered = append(filtered, entry.Key[idx])
	}
	return filtered, stats
}

// DistinctTempTable deduplicates a column that does not lead the index.
//
// Equal values are scattered, so there is no previous value to compare
// against — every value seen so far has to be remembered.
//
// The predicate filters entries; pass nil to keep everything.
func DistinctTempTable(ix *Index, col string, p Pred) ([]int64, Stats) {
	stats := Stats{}
	idx := ix.ColIndex(col)
	if idx == -1 {
		return nil, stats
	}
	seen := make(map[int64]struct{})
	stats.UsedTempTable = true
	filtered := make([]int64, 0)
	for _, entry := range ix.Keys {
		stats.IndexEntriesRead++
		if !p.evalEntry(ix, entry) {
			continue
		}
		if _, ok := seen[entry.Key[idx]]; ok {
			continue
		}
		stats.RowsReturned++
		filtered = append(filtered, entry.Key[idx])
		seen[entry.Key[idx]] = struct{}{}
		stats.TempTableInserts++
	}
	return filtered, stats
}
