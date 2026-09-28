package step5

import "strings"

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
type Stats struct {
	IndexEntriesRead int
	RowsFetched      int
	RowsScanned      int
	RowsReturned     int
	TempTableInserts int
	UsedTempTable    bool
	UsedSort         bool
	UsedICP          bool
}

const (
	FetchCost = 20
	ScanCost  = 1
)

// Cost is the total work, in index-entry units.
func (s Stats) Cost() int {
	return s.IndexEntriesRead + s.RowsFetched*FetchCost + s.RowsScanned*ScanCost
}

func FullScan(t *Table, p Pred) ([]Row, Stats) {
	var s Stats
	out := make([]Row, 0)
	for _, r := range t.Rows {
		s.RowsScanned++
		if p.eval(r) {
			out = append(out, r)
		}
	}
	s.RowsReturned = len(out)
	return out, s
}

func IndexScan(t *Table, ix *Index, lo, hi []int64, p Pred) ([]Row, Stats) {
	var s Stats
	out := make([]Row, 0)
	for _, e := range ix.RangeScan(lo, hi) {
		s.IndexEntriesRead++
		r := t.ByID[e.RowID]
		s.RowsFetched++
		if p.eval(r) {
			out = append(out, r)
		}
	}
	s.RowsReturned = len(out)
	return out, s
}

func CoveringScan(t *Table, ix *Index, lo, hi []int64, p Pred) ([]Row, Stats) {
	var s Stats
	out := make([]Row, 0)
	for _, e := range ix.RangeScan(lo, hi) {
		s.IndexEntriesRead++
		if p.evalEntry(ix, e) {
			out = append(out, t.ByID[e.RowID])
		}
	}
	s.RowsReturned = len(out)
	return out, s
}

func (s Stats) Extra() string {
	var parts []string
	if s.IndexEntriesRead > 0 && s.RowsFetched == 0 && s.RowsScanned == 0 {
		parts = append(parts, "Using index")
	}
	if s.UsedICP {
		parts = append(parts, "Using index condition")
	}
	if s.UsedTempTable {
		parts = append(parts, "Using temporary")
	}
	if s.UsedSort {
		parts = append(parts, "Using filesort")
	}
	return strings.Join(parts, "; ")
}
