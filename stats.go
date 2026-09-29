package engine

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
	TempTableInserts int
	UsedTempTable    bool
	UsedSort         bool
	UsedLooseScan    bool
	UsedICP          bool
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
