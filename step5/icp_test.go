package step5

import (
	"strings"
	"testing"

	"sql-execution-engine-from-scratch"
)

// sampleTable builds 1000 rows. staff_id has two values, 500 rows each;
// id and created_at are unique.
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

func condLt(col string, v int64) engine.Cond {
	return engine.Cond{Col: col, Ok: func(x int64) bool { return x < v }}
}

// E1: a condition the index carries is settled on the entry, before the row is
// fetched. The primary key is carried by every index, so id counts too.
func TestICP_FiltersBeforeFetchingTheRow(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_staff", []string{"staff_id"}, tbl.Rows)
	p := engine.Pred{
		condLt("id", 901),
		condGe("created_at", 1700000500),
	}

	withICP, s := engine.ScanWithICP(tbl, ix, []int64{1}, []int64{2}, p)
	withoutICP, plain := engine.ScanWithoutICP(tbl, ix, []int64{1}, []int64{2}, p)

	if !s.UsedICP {
		t.Fatal("ICP did not engage — id is on every entry and created_at is not in this " +
			"index, so the predicate has both a condition worth pushing and a row still " +
			"worth fetching")
	}
	if s.RowsFetched >= plain.RowsFetched {
		t.Fatalf("fetched %d rows with ICP and %d without — id is on every entry, so the "+
			"entries whose id is too large should have been rejected before any row was "+
			"looked up", s.RowsFetched, plain.RowsFetched)
	}
	if len(withICP) != len(withoutICP) {
		t.Fatalf("ICP returned %d rows and the plain scan returned %d — pushing a condition down "+
			"decides when it is judged, not what the answer is", len(withICP), len(withoutICP))
	}
}

// E2: without ICP the index only decides where the walk starts and stops.
// Every entry in the range is fetched, and the predicate runs on the row.
func TestNoICP_FiltersAfterFetchingTheRow(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, tbl.Rows)
	p := engine.Pred{
		condEq("staff_id", 1),
		condGe("created_at", 1700000800),
		condGe("id", 900),
	}

	_, s := engine.ScanWithoutICP(tbl, ix, []int64{1, 0}, []int64{2, 0}, p)

	if s.RowsFetched != s.IndexEntriesRead {
		t.Fatalf("read %d entries and fetched %d rows — nothing was judged at the entry, so every "+
			"entry in the range has to cost exactly one fetch",
			s.IndexEntriesRead, s.RowsFetched)
	}
	if s.RowsReturned >= s.RowsFetched {
		t.Fatalf("returned %d rows out of %d fetched — if every fetched row survived, the "+
			"predicate never ran on the row", s.RowsReturned, s.RowsFetched)
	}
}

// E3: ICP needs two things at once — a condition the index can judge, and a row
// that still has to be fetched. Either one missing and there is nothing to push.
func TestICP_NeedsBothATableLookupAndAPushableCond(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, tbl.Rows)
	created := engine.NewIndex("idx_created", []string{"created_at"}, tbl.Rows)

	covering := engine.Pred{condEq("staff_id", 1), condGe("created_at", 1700000800)}
	_, s := engine.ScanWithICP(tbl, ix, []int64{1, 0}, []int64{2, 0}, covering)
	if s.UsedICP {
		t.Fatal("UsedICP is set for a predicate this index answers on its own — every " +
			"condition can be settled on the entry, so no row has to be fetched to decide " +
			"the query and there is no fetch left for a pushed condition to save")
	}
	if s.RowsFetched != s.IndexEntriesRead {
		t.Fatalf("read %d entries and fetched %d rows — with nothing left to push this is "+
			"the plain index scan: every entry in the range costs one fetch",
			s.IndexEntriesRead, s.RowsFetched)
	}

	keyed := engine.Pred{condEq("staff_id", 1), condGe("id", 900)}
	if _, s := engine.ScanWithICP(tbl, ix, []int64{1, 0}, []int64{2, 0}, keyed); s.UsedICP {
		t.Fatal("UsedICP is set for a predicate this index answers on its own — staff_id is " +
			"a column of this index and id is on every index, so both conditions can be " +
			"settled on the entry and no row has to be fetched to decide the query")
	}

	unpushable := engine.Pred{condEq("staff_id", 1)}
	if _, s := engine.ScanWithICP(tbl, created, []int64{1700000800}, []int64{1700001000}, unpushable); s.UsedICP {
		t.Fatal("UsedICP is set for a predicate with no condition on an indexed column — " +
			"staff_id is not in this index, so an entry has nothing to judge and every row " +
			"in the range still has to be fetched and tested there")
	}
}

// E4: a pushed condition thins the queue, it does not empty it. Whatever
// survives still has to be looked up, because the rest of the predicate was
// never in the index to begin with.
func TestICP_ReducesFetchesButNeverEliminatesThem(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_created", []string{"created_at"}, tbl.Rows)
	p := engine.Pred{
		condLt("created_at", 1700000900),
		condEq("staff_id", 1),
	}

	_, s := engine.ScanWithICP(tbl, ix, []int64{1700000800}, []int64{1700001000}, p)

	if s.RowsFetched == 0 {
		t.Fatal("no row was fetched — but staff_id is not in this index, so the entries that " +
			"survived created_at still have to be looked up before staff_id can be judged")
	}
	if s.RowsFetched >= s.IndexEntriesRead {
		t.Fatalf("fetched %d rows for %d entries — the pushed condition was supposed to cut the "+
			"queue down before the table was touched", s.RowsFetched, s.IndexEntriesRead)
	}
	if s.RowsFetched <= s.RowsReturned {
		t.Fatalf("fetched %d rows and returned %d — some of what was fetched has to be thrown "+
			"away, otherwise the row was not deciding anything",
			s.RowsFetched, s.RowsReturned)
	}
}

// E5: a pushed condition can be reported and still save nothing. When the range
// already pins the indexed column to the values the condition accepts, there is
// nothing left for it to reject.
func TestICP_SavesNothingWhenTheRangeIsAlreadyExact(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_created", []string{"created_at"}, tbl.Rows)
	p := engine.Pred{
		condGe("created_at", 1700000500),
		condLt("created_at", 1700000600),
		condEq("staff_id", 1),
	}

	_, withICP := engine.ScanWithICP(tbl, ix, []int64{1700000500}, []int64{1700000600}, p)
	_, withoutICP := engine.ScanWithoutICP(tbl, ix, []int64{1700000500}, []int64{1700000600}, p)

	if !withICP.UsedICP {
		t.Fatal("ICP did not engage — created_at is in this index and staff_id is not, so the " +
			"predicate has both a condition worth pushing and a row still worth fetching")
	}
	if withICP.RowsFetched != withoutICP.RowsFetched {
		t.Fatalf("fetched %d rows with ICP and %d without — the range [1700000500, 1700000600) "+
			"already admits only the created_at values the condition accepts, so the pushed "+
			"condition had nothing to reject and could not save a single fetch",
			withICP.RowsFetched, withoutICP.RowsFetched)
	}
	if withICP.IndexEntriesRead != withoutICP.IndexEntriesRead {
		t.Fatalf("read %d entries with ICP and %d without — pushing a condition down changes "+
			"when an entry is judged, never how much of the index is walked",
			withICP.IndexEntriesRead, withoutICP.IndexEntriesRead)
	}
}

// E6: the flag reaches EXPLAIN. A condition judged on the entry is what the
// Extra column reports as "Using index condition".
func TestICP_ReportsUsingIndexCondition(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_created", []string{"created_at"}, tbl.Rows)
	p := engine.Pred{
		condLt("created_at", 1700000900),
		condEq("staff_id", 1),
	}

	_, s := engine.ScanWithICP(tbl, ix, []int64{1700000800}, []int64{1700001000}, p)
	if !strings.Contains(s.Extra(), "Using index condition") {
		t.Fatalf("Extra is %q — created_at was judged on the entry before any row was fetched, "+
			"and that is what EXPLAIN reports as Using index condition", s.Extra())
	}

	_, plain := engine.ScanWithoutICP(tbl, ix, []int64{1700000800}, []int64{1700001000}, p)
	if strings.Contains(plain.Extra(), "Using index condition") {
		t.Fatalf("Extra is %q — this scan judged nothing on the entry, so it has no pushed "+
			"condition to report", plain.Extra())
	}
}
