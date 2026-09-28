package step3

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// sampleTable builds 1000 rows. staff_id has two values, 500 rows each;
// id and created_at are unique.
func sampleTable() *Table {
	rows := make([]Row, 0, 1000)
	for i := 0; i < 1000; i++ {
		rows = append(rows, Row{
			ID:        int64(i + 1),
			StaffID:   int64(i%2 + 1),
			CreatedAt: 1700000000 + int64(i),
		})
	}
	t := &Table{Rows: rows, ByID: make(map[int64]Row, len(rows))}
	for _, r := range rows {
		t.ByID[r.ID] = r
	}
	return t
}

func condEq(col string, v int64) Cond {
	return Cond{Col: col, Ok: func(x int64) bool { return x == v }}
}

// C1: when equal values are adjacent, one variable is enough. Nothing has to
// be remembered across the whole scan, so no temporary table appears.
func TestDistinctOrdered_KeepsOneValuePerRun(t *testing.T) {
	tbl := sampleTable()
	ix := NewIndex("idx_staff", []string{"staff_id"}, tbl.Rows)

	got, s := DistinctOrdered(ix, "staff_id", nil)

	if s.UsedTempTable {
		t.Fatal("a temporary table was used — but staff_id leads this index, so equal values " +
			"arrive together and the previous value is all you need to compare against")
	}
	if !reflect.DeepEqual(got, []int64{1, 2}) {
		t.Fatalf("got %v, want [1 2] — staff_id has exactly two values", got)
	}
}

// C2: when equal values are scattered, there is no previous value to lean on.
// Every value seen so far has to be remembered.
func TestDistinctTempTable_RemembersEveryValueItHasSeen(t *testing.T) {
	tbl := sampleTable()
	ix := NewIndex("idx_created_staff", []string{"created_at", "staff_id"}, tbl.Rows)

	got, s := DistinctTempTable(ix, "staff_id", nil)

	if !s.UsedTempTable {
		t.Fatal("no temporary table was used — but staff_id trails this index, so equal values " +
			"are scattered and nothing short of remembering them all can work")
	}
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	if !reflect.DeepEqual(got, []int64{1, 2}) {
		t.Fatalf("got %v, want [1 2] — staff_id has exactly two values", got)
	}
}

// C3: the deciding question is whether the column leads the index. The same
// column is contiguous in one index and scattered in another.
func TestDistinct_LeadingColumnIsOrderedTrailingIsNot(t *testing.T) {
	tbl := sampleTable()
	leading := NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, tbl.Rows)
	trailing := NewIndex("idx_created_staff", []string{"created_at", "staff_id"}, tbl.Rows)

	if !columnGroupsAreContiguous(leading, "staff_id") {
		t.Fatal("staff_id leads (staff_id, created_at), so its values must form unbroken runs")
	}
	if columnGroupsAreContiguous(trailing, "staff_id") {
		t.Fatal("staff_id trails (created_at, staff_id), so its values are scattered — " +
			"if they look contiguous, the index was built in the wrong order")
	}
}

// C4: Using temporary is EXPLAIN's way of saying "the data was not in the
// order this step needed".
func TestExtra_SaysUsingTemporaryWhenTheDataIsOutOfOrder(t *testing.T) {
	tbl := sampleTable()
	trailing := NewIndex("idx_created_staff", []string{"created_at", "staff_id"}, tbl.Rows)

	_, s := DistinctTempTable(trailing, "staff_id", nil)

	if got := s.Extra(); !strings.Contains(got, "Using temporary") {
		t.Fatalf("Extra() = %q, want it to mention Using temporary — the dedup needed "+
			"scratch space because the index order did not match the dedup order", got)
	}
}

// C5: the two paths differ in what they cost, not in what they return.
func TestDistinct_BothPathsReturnTheSameValues(t *testing.T) {
	tbl := sampleTable()
	leading := NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, tbl.Rows)
	trailing := NewIndex("idx_created_staff", []string{"created_at", "staff_id"}, tbl.Rows)

	ordered, _ := DistinctOrdered(leading, "staff_id", nil)
	temped, _ := DistinctTempTable(trailing, "staff_id", nil)

	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	sort.Slice(temped, func(i, j int) bool { return temped[i] < temped[j] })

	if !reflect.DeepEqual(ordered, temped) {
		t.Fatalf("ordered path returned %v and temp-table path returned %v — "+
			"they must agree, they answer the same question", ordered, temped)
	}
}

// columnGroupsAreContiguous reports whether entries sharing a value in the
// named column form one unbroken run.
func columnGroupsAreContiguous(ix *Index, col string) bool {
	i := ix.ColIndex(col)
	if i < 0 {
		return false
	}
	seen := make(map[int64]bool)
	var prev int64
	started := false
	for _, e := range ix.Keys {
		v := e.Key[i]
		if started && v == prev {
			continue
		}
		if seen[v] {
			return false
		}
		seen[v] = true
		prev, started = v, true
	}
	return true
}
