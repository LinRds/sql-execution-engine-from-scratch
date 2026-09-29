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
	for i := range 1000 {
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

// D1: a tight scan only learns that a group has ended by reaching an entry with
// a different value, so it walks the whole group. A loose scan has the group's
// value already and can leave the rest of the group unread.
func TestLooseScan_SkipsWholeGroupsInsteadOfWalkingThem(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, tbl.Rows)
	filter := []engine.RangeCond{{Col: "staff_id", Lo: 2, Hi: 2}}

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
	for i := range 1000 {
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

// D4: a condition on another column is folded into the seek, so a group whose
// entries never satisfy it is skipped instead of reported.
func TestLooseScan_SkipsGroupsTheOtherConditionsExclude(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, tbl.Rows)

	// staff_id 1 owns the even created_at values, staff_id 2 the odd ones.
	odd := []engine.RangeCond{
		{Col: "staff_id", Lo: 1, Hi: 2},
		{Col: "created_at", Lo: 1700000501, Hi: 1700000501},
	}
	if got, _ := engine.LooseScan(ix, "staff_id", odd); !reflect.DeepEqual(got, []int64{2}) {
		t.Fatalf("got %v for created_at = 1700000501, want [2] — only staff_id 2 has a row with "+
			"that value, and the scan has to land on it rather than on the group's first entry", got)
	}

	even := []engine.RangeCond{
		{Col: "staff_id", Lo: 1, Hi: 2},
		{Col: "created_at", Lo: 1700000500, Hi: 1700000500},
	}
	if got, _ := engine.LooseScan(ix, "staff_id", even); !reflect.DeepEqual(got, []int64{1}) {
		t.Fatalf("got %v for created_at = 1700000500, want [1] — only staff_id 1 has a row with "+
			"that value", got)
	}
}

// D6: the grouped column has to lead the index. The jump to the next group is
// a seek on that column's next value, and a column that does not start the
// index has no such seek — its groups are scattered.
func TestLooseScan_RefusesANonLeadingColumn(t *testing.T) {
	tbl := sampleTable()
	trailing := engine.NewIndex("idx_created_staff", []string{"created_at", "staff_id"}, tbl.Rows)

	if engine.CanLooseScan(trailing, "staff_id", nil) {
		t.Fatal("staff_id trails created_at in this index — its groups are scattered, so there " +
			"is no single seek that lands on the next one")
	}
}

// D4: a range on the grouped column narrows the walk — the scan starts at the
// lower bound and stops at the upper one.
func TestLooseScan_HonoursTheGroupedColumnsRange(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, tbl.Rows)

	only, _ := engine.LooseScan(ix, "staff_id", []engine.RangeCond{{Col: "staff_id", Lo: 1, Hi: 1}})
	if !reflect.DeepEqual(only, []int64{1}) {
		t.Fatalf("got %v for staff_id in [1,1], want [1] — the scan stops at the range's upper "+
			"bound instead of running to the end of the index", only)
	}

	both, _ := engine.LooseScan(ix, "staff_id", []engine.RangeCond{{Col: "staff_id", Lo: 1, Hi: 2}})
	if !reflect.DeepEqual(both, []int64{1, 2}) {
		t.Fatalf("got %v for staff_id in [1,2], want [1 2] — the scan starts at the range's lower "+
			"bound, not at its upper one", both)
	}
}

// D4: the conditions belong to the caller. Folding them into a seek means
// reading them, not rewriting them.
func TestLooseScan_LeavesTheConditionsAlone(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, tbl.Rows)
	conds := []engine.RangeCond{
		{Col: "created_at", Lo: 1700000500, Hi: 1700000500},
		{Col: "staff_id", Lo: 1, Hi: 1},
	}

	engine.LooseScan(ix, "staff_id", conds)

	if conds[0].Col != "created_at" || conds[1].Col != "staff_id" {
		t.Fatalf("the caller's conditions came back as [%s, %s] — the scan reads them to build a "+
			"seek, it does not get to rewrite them", conds[0].Col, conds[1].Col)
	}
}

// D4: a condition on a column the index does not carry is worse than a range —
// there is nothing to fold it into, and nothing to look at either.
func TestLooseScan_RefusesAColumnTheIndexDoesNotCarry(t *testing.T) {
	tbl := sampleTable()
	ix := engine.NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, tbl.Rows)

	offIndex := []engine.RangeCond{{Col: "id", Lo: 1, Hi: 1}}
	if engine.CanLooseScan(ix, "staff_id", offIndex) {
		t.Fatal("an equality on id allowed a loose scan — id is not in this index, so the scan " +
			"can neither fold the condition into a seek nor look at the entries it skipped")
	}
}
