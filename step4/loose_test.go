package step4

import (
	"reflect"
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

// D1: a tight scan only learns that a group has ended by reaching an entry with
// a different value, so it walks the whole group. A loose scan has the group's
// value already and can leave the rest of the group unread.
func TestLooseScan_SkipsWholeGroupsInsteadOfWalkingThem(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, tbl.Rows)
	filter := engine.Pred{condEq("staff_id", 2)}

	tight, tightStats := engine.TightScan(ix, "staff_id", filter)
	loose, looseStats := engine.LooseScan(ix, "staff_id", filter)

	if tightStats.IndexEntriesRead != len(ix.Keys) {
		t.Fatalf("tight scan read %d of %d entries — it has no way to tell that a group has ended "+
			"until it looks at the next entry, so a filter does not let it skip anything",
			tightStats.IndexEntriesRead, len(ix.Keys))
	}
	rowsPerGroup := len(ix.Keys) / 2
	if looseStats.IndexEntriesRead >= rowsPerGroup {
		t.Fatalf("loose scan read %d entries while each staff_id owns %d of them — anything in that "+
			"range means it walked through a group instead of jumping over it",
			looseStats.IndexEntriesRead, rowsPerGroup)
	}
	if !reflect.DeepEqual(tight, loose) {
		t.Fatalf("tight scan returned %v, loose scan returned %v — the filter decides which values "+
			"survive and the scan decides what they cost, so both must agree", tight, loose)
	}
}

// D2: the jump is Seek(current value + 1) — one tree lookup that lands on the
// first entry past the group, however far away the next value happens to be.
func TestLooseScan_SeeksToTheNextValue(t *testing.T) {
	rows := make([]engine.Row, 0, 1000)
	for i := 0; i < 1000; i++ {
		staff := int64(1)
		if i >= 500 {
			staff = 1000000
		}
		rows = append(rows, engine.Row{
			ID:        int64(i + 1),
			StaffID:   staff,
			CreatedAt: 1700000000 + int64(i),
		})
	}
	ix := engine.NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, rows)

	got, s := engine.LooseScan(ix, "staff_id", nil)

	if !reflect.DeepEqual(got, []int64{1, 1000000}) {
		t.Fatalf("got %v, want [1 1000000] — the jump lands on the first entry past the current "+
			"group, so the next value does not have to be the current value + 1", got)
	}
	if s.IndexEntriesRead != 2 {
		t.Fatalf("loose scan read %d entries for 2 groups — the jump is one Seek, a tree lookup, "+
			"not a step to the neighbouring entry: 500 rows with staff_id 1 sit between the two "+
			"values and none of them is touched", s.IndexEntriesRead)
	}
}

// D3: the cost is the number of groups, not the number of entries. Same query,
// same answer, two very different bills.
func TestLooseScan_ReadsOneEntryPerGroup(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, tbl.Rows)

	tight, tightStats := engine.TightScan(ix, "staff_id", nil)
	loose, looseStats := engine.LooseScan(ix, "staff_id", nil)

	if tightStats.IndexEntriesRead != 1000 {
		t.Fatalf("tight scan read %d entries, want 1000 — one per index entry, O(entries)",
			tightStats.IndexEntriesRead)
	}
	if looseStats.IndexEntriesRead != 2 {
		t.Fatalf("loose scan read %d entries, want 2 — one per distinct staff_id, O(groups); the "+
			"500 entries inside each group are jumped over, not counted", looseStats.IndexEntriesRead)
	}
	if !reflect.DeepEqual(tight, loose) {
		t.Fatalf("tight scan returned %v and loose scan returned %v — the cheap path is not a "+
			"different answer, it is the same answer for less work", tight, loose)
	}
}

// D4: a range on the grouped column is fine, the scan reads that column for
// every group anyway. A range on any other column is not — the scan jumps over
// those values without ever looking at them.
func TestLooseScan_RefusesRangeCondsOnOtherColumns(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, tbl.Rows)

	if !engine.CanLooseScan(ix, "staff_id", nil) {
		t.Fatal("no conditions at all blocked the loose scan — with nothing to check, nothing can " +
			"force the scan to look inside a group")
	}

	onGrouped := []engine.RangeCond{{Col: "staff_id", Lo: 1, Hi: 2}}
	if !engine.CanLooseScan(ix, "staff_id", onGrouped) {
		t.Fatal("a range on the grouped column blocked the loose scan — the scan reads staff_id " +
			"once per group, so narrowing it costs nothing extra")
	}

	equalityElsewhere := []engine.RangeCond{{Col: "created_at", Lo: 1700000500, Hi: 1700000500}}
	if !engine.CanLooseScan(ix, "staff_id", equalityElsewhere) {
		t.Fatal("an equality on created_at blocked the loose scan — an equality is a single value, " +
			"so it can be folded into the seek; only a range asks about rows the scan never visits")
	}

	rangeElsewhere := []engine.RangeCond{{Col: "created_at", Lo: 1700000500, Hi: 1700000600}}
	if engine.CanLooseScan(ix, "staff_id", rangeElsewhere) {
		t.Fatal("a range on created_at allowed a loose scan — the scan jumps from one staff_id to " +
			"the next without visiting the entries it skipped, so it cannot tell which groups hold " +
			"a created_at in that range")
	}
}

// D5: Using index for group-by is EXPLAIN's way of saying the engine skipped
// whole groups instead of walking them.
func TestExtra_SaysUsingIndexForGroupBy(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, tbl.Rows)

	_, tightStats := engine.TightScan(ix, "staff_id", nil)
	_, looseStats := engine.LooseScan(ix, "staff_id", nil)

	if !strings.Contains(looseStats.Extra(), "Using index for group-by") {
		t.Fatalf("Extra() = %q, want it to mention Using index for group-by — that line is how "+
			"EXPLAIN reports a loose scan", looseStats.Extra())
	}
	if strings.Contains(tightStats.Extra(), "Using index for group-by") {
		t.Fatalf("Extra() = %q — a tight scan walks every entry, so it must not claim the group-by "+
			"shortcut", tightStats.Extra())
	}
}
