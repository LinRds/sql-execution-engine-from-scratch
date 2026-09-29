package step2

import (
	"testing"

	"sql-execution-engine-from-scratch"
)

// sampleTable builds 1000 rows. staff_id has two values, 500 rows each;
// id and created_at are unique. The contrast is what makes the cost
// difference in B5 visible.
func sampleTable() *engine.Table {
	rows := make([]engine.Row, 0, 1000)
	for i := 0; i < 1000; i++ {
		rows = append(rows, engine.Row{
			ID:        int64(i + 1),
			StaffID:   int64(i%2 + 1),
			CreatedAt: 1700000000 + int64(i),
		})
	}
	t := &engine.Table{Rows: rows, ByID: make(map[int64]engine.Row, len(rows))}
	for _, r := range rows {
		t.ByID[r.ID] = r
	}
	return t
}

func condEq(col string, v int64) engine.Cond {
	return engine.Cond{Col: col, Ok: func(x int64) bool { return x == v }}
}

func condGe(col string, v int64) engine.Cond {
	return engine.Cond{Col: col, Ok: func(x int64) bool { return x >= v }}
}

// B1: an index entry carries the key and the row id, nothing else. Anything
// the predicate needs beyond the key has to come from the row.
func TestIndexScan_FetchesTheFullRowForEveryHit(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_staff", []string{"staff_id"}, tbl.Rows)

	_, s := engine.IndexScan(tbl, ix, []int64{1}, []int64{2}, engine.Pred{condEq("staff_id", 1)})

	if s.RowsFetched == 0 {
		t.Fatal("no row was fetched — but the predicate asks about staff_id, and the index " +
			"only narrows the walk; the answer still has to come from the row")
	}
	if s.RowsFetched != s.IndexEntriesRead {
		t.Fatalf("read %d entries but fetched %d rows — with nothing pushable, every entry "+
			"in the range costs exactly one fetch", s.IndexEntriesRead, s.RowsFetched)
	}
}

// B2: when the index carries every column the predicate touches, no row is
// needed at all.
func TestCoveringScan_AnswersFromTheIndexAlone(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, tbl.Rows)

	_, s := engine.CoveringScan(tbl, ix, []int64{1, 0}, []int64{2, 0}, engine.Pred{
		condEq("staff_id", 1),
		condGe("created_at", 1700000100),
	})

	if !ix.Covers(engine.Pred{condEq("staff_id", 1), condGe("created_at", 1700000100)}) {
		t.Fatal("(staff_id, created_at) should cover both conditions — check ColIndex")
	}
	if s.RowsFetched != 0 {
		t.Fatalf("fetched %d rows, want 0 — every column the predicate needs is in the index, "+
			"so there is nothing left to read from the table", s.RowsFetched)
	}
}

// B3: "Using index" is not "an index was used". It means no row was fetched.
func TestExtra_SaysUsingIndexOnlyWhenNoRowWasFetched(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, tbl.Rows)
	p := engine.Pred{condEq("staff_id", 1)}

	_, fetched := engine.IndexScan(tbl, ix, []int64{1, 0}, []int64{2, 0}, p)
	_, covered := engine.CoveringScan(tbl, ix, []int64{1, 0}, []int64{2, 0}, p)

	if got := fetched.Extra(); got == "Using index" {
		t.Fatalf("Extra() = %q for a scan that fetched %d rows — "+
			"Using index means the index answered it alone, not that an index was walked",
			got, fetched.RowsFetched)
	}
	if got := covered.Extra(); got != "Using index" {
		t.Fatalf("Extra() = %q for a scan that fetched no rows, want %q", got, "Using index")
	}
}

// B4: a fetched row costs far more than an index entry — it is wider, and
// reaching it is a random seek rather than a walk.
func TestCost_ARowFetchCostsMoreThanAnIndexEntry(t *testing.T) {
	oneEntry := engine.Stats{IndexEntriesRead: 1}
	oneFetch := engine.Stats{RowsFetched: 1}

	if oneFetch.Cost() <= oneEntry.Cost() {
		t.Fatalf("one fetch costs %d and one index entry costs %d — a fetched row is wider "+
			"and randomly placed, so it cannot be as cheap as an entry",
			oneFetch.Cost(), oneEntry.Cost())
	}
}

// B5: Seek finds a starting point, not a short tail. When the key repeats
// often, fetching every hit costs more than reading the whole table once.
func TestCost_ALowCardinalityIndexLosesToAFullScan(t *testing.T) {
	tbl := sampleTable()

	byStaff := engine.NewIndex("idx_staff", []string{"staff_id"}, tbl.Rows)
	byID := engine.NewIndex("idx_id", []string{"id"}, tbl.Rows)

	_, full := engine.FullScan(tbl, engine.Pred{condEq("staff_id", 1)})
	_, lowCard := engine.IndexScan(tbl, byStaff, []int64{1}, []int64{2}, engine.Pred{condEq("staff_id", 1)})
	_, highCard := engine.IndexScan(tbl, byID, []int64{500}, []int64{501}, engine.Pred{condEq("id", 500)})

	if lowCard.RowsFetched <= highCard.RowsFetched {
		t.Fatalf("staff_id 1 has 500 rows and id 500 has 1 — fetched %d vs %d",
			lowCard.RowsFetched, highCard.RowsFetched)
	}
	if lowCard.Cost() <= full.Cost() {
		t.Fatalf("index scan cost %d, full scan cost %d — when the index leaves 500 rows to "+
			"fetch, going through it costs more than reading the table once",
			lowCard.Cost(), full.Cost())
	}
	if highCard.Cost() >= full.Cost() {
		t.Fatalf("index scan cost %d, full scan cost %d — a unique key leaves one row to fetch, "+
			"which has to beat reading everything", highCard.Cost(), full.Cost())
	}
}
