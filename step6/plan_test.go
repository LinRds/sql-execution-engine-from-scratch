package step6

import "testing"

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

func estimatedRows(st TableStats) int {
	if st.Groups <= 0 {
		return st.Rows
	}
	return st.Rows / st.Groups
}

func fixedEstPlan(name string, usable bool, est int) Plan {
	return Plan{
		Name:   name,
		Usable: func(Pred, *Index) bool { return usable },
		Est:    func(TableStats, *Index, Pred) int { return est },
	}
}

func fullScanPlan() Plan {
	return Plan{
		Name:   "full scan",
		Usable: func(Pred, *Index) bool { return true },
		Est:    func(st TableStats, ix *Index, p Pred) int { return st.Rows * ScanCost },
	}
}

func indexScanPlan() Plan {
	return Plan{
		Name:   "index scan",
		Usable: func(Pred, *Index) bool { return true },
		Est: func(st TableStats, ix *Index, p Pred) int {
			matched := estimatedRows(st)
			return matched + matched*FetchCost
		},
	}
}

func coveringScanPlan() Plan {
	return Plan{
		Name:   "covering scan",
		Usable: func(p Pred, ix *Index) bool { return ix.Covers(p) },
		Est:    func(st TableStats, ix *Index, p Pred) int { return estimatedRows(st) },
	}
}

func TestChoosePlan_PicksTheCheapestEstimatedCost(t *testing.T) {
	tbl := sampleTable()
	ix := NewIndex("idx_staff", []string{"staff_id"}, tbl.Rows)
	p := Pred{condEq("staff_id", 1)}

	plans := []Plan{
		fixedEstPlan("full scan", true, 1),
		fixedEstPlan("covering scan", true, 500),
	}

	got, ok := ChoosePlan(plans, p, ix, TableStats{Rows: 1000, Groups: 2})
	if !ok {
		t.Fatal("no plan came back, but both candidates are usable — ChoosePlan has to return the " +
			"cheapest of the plans it was given")
	}
	if got.Name != "full scan" {
		t.Fatalf("chose %q, want full scan — the choice follows the estimated cost, not the real "+
			"one: the two estimates are 1 and 500, and nothing was executed to check either",
			got.Name)
	}

	_, full := FullScan(tbl, p)
	_, covering := CoveringScan(tbl, ix, []int64{1}, []int64{2}, p)
	if full.Cost() <= covering.Cost() {
		t.Fatalf("full scan really cost %d and covering scan really cost %d — the fixture no longer "+
			"shows a plan winning on an estimate that the real cost contradicts, so this test no "+
			"longer demonstrates anything", full.Cost(), covering.Cost())
	}
}

func TestCost_IsEntriesPlusFetchesTimesFetchCost(t *testing.T) {
	tbl := sampleTable()
	ix := NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, tbl.Rows)
	p := Pred{condEq("staff_id", 1)}
	st := TableStats{Rows: 1000, Groups: 2}

	full, index, covering := fullScanPlan(), indexScanPlan(), coveringScanPlan()
	plans := []Plan{full, index, covering}

	matched := estimatedRows(st)
	fullEst, indexEst, coverEst := full.Est(st, ix, p), index.Est(st, ix, p), covering.Est(st, ix, p)

	if indexEst != matched+matched*FetchCost {
		t.Fatalf("index scan estimated %d, want %d — %d index entries read, plus %d rows fetched at "+
			"FetchCost=%d each", indexEst, matched+matched*FetchCost, matched, matched, FetchCost)
	}
	if indexEst <= fullEst {
		t.Fatalf("index scan estimated %d and full scan estimated %d — %d fetches at FetchCost=%d "+
			"cost more than scanning all %d rows, so the index path has to lose here",
			indexEst, fullEst, matched, FetchCost, st.Rows)
	}
	if coverEst != matched {
		t.Fatalf("covering scan estimated %d, want %d — it reads the same entries as the index "+
			"scan but fetches no rows, so only the entry term is left", coverEst, matched)
	}

	got, ok := ChoosePlan(plans, p, ix, st)
	if !ok {
		t.Fatal("no plan came back, but all three candidates are usable")
	}
	if got.Name != "covering scan" {
		t.Fatalf("chose %q, want covering scan — %d entries and no fetches beats %d, and it beats "+
			"the index scan's %d by exactly the fetch term", got.Name, coverEst, fullEst, indexEst)
	}
}

func TestChoosePlan_UsesTheStatisticsItIsGiven(t *testing.T) {
	tbl := sampleTable()
	ix := NewIndex("idx_staff", []string{"staff_id"}, tbl.Rows)
	p := Pred{condEq("staff_id", 1)}
	plans := []Plan{fullScanPlan(), indexScanPlan()}

	manyValues := TableStats{Rows: 1000, Groups: 1000}
	fewValues := TableStats{Rows: 1000, Groups: 2}

	first, ok := ChoosePlan(plans, p, ix, manyValues)
	if !ok {
		t.Fatal("no plan came back with Groups=1000, but both candidates are usable")
	}
	second, ok := ChoosePlan(plans, p, ix, fewValues)
	if !ok {
		t.Fatal("no plan came back with Groups=2, but both candidates are usable")
	}

	if first.Name == second.Name {
		t.Fatalf("both sets of statistics chose %q — same plans, same predicate, same index; only "+
			"the statistics differed, so the choice has to differ too", first.Name)
	}
	if first.Name != "index scan" {
		t.Fatalf("with Groups=1000 the estimate says the predicate matches %d row, so the index "+
			"scan costs %d against full scan's %d — want index scan, got %q",
			estimatedRows(manyValues), indexScanPlan().Est(manyValues, ix, p),
			fullScanPlan().Est(manyValues, ix, p), first.Name)
	}
	if second.Name != "full scan" {
		t.Fatalf("with Groups=2 the estimate says the predicate matches %d rows, so the index scan "+
			"costs %d against full scan's %d — want full scan, got %q",
			estimatedRows(fewValues), indexScanPlan().Est(fewValues, ix, p),
			fullScanPlan().Est(fewValues, ix, p), second.Name)
	}
}

func TestChoosePlan_SkipsPlansWhosePreconditionsFail(t *testing.T) {
	tbl := sampleTable()
	p := Pred{condEq("staff_id", 1)}
	st := TableStats{Rows: 1000, Groups: 2}

	narrow := NewIndex("idx_created", []string{"created_at"}, tbl.Rows)
	wide := NewIndex("idx_staff", []string{"staff_id"}, tbl.Rows)

	if narrow.Covers(p) {
		t.Fatal("idx_created does not carry staff_id, so it cannot answer a predicate on staff_id " +
			"without the row — Covers must say so")
	}
	if !wide.Covers(p) {
		t.Fatal("idx_staff carries staff_id, so it can answer a predicate on staff_id straight " +
			"from the index — Covers must say so")
	}

	plans := []Plan{fullScanPlan(), coveringScanPlan()}

	got, ok := ChoosePlan(plans, p, narrow, st)
	if !ok {
		t.Fatal("no plan came back, but full scan is usable under any index")
	}
	if got.Name != "full scan" {
		t.Fatalf("chose %q over full scan — the covering scan estimated %d against full scan's %d, "+
			"but idx_created does not carry staff_id, so it never entered the candidate set",
			got.Name, coveringScanPlan().Est(st, narrow, p), fullScanPlan().Est(st, narrow, p))
	}

	got, ok = ChoosePlan(plans, p, wide, st)
	if !ok {
		t.Fatal("no plan came back, but both candidates are usable under idx_staff")
	}
	if got.Name != "covering scan" {
		t.Fatalf("chose %q — under idx_staff the covering scan's precondition holds and it estimates "+
			"%d against full scan's %d, so it has to win", got.Name,
			coveringScanPlan().Est(st, wide, p), fullScanPlan().Est(st, wide, p))
	}

	if dead, ok := ChoosePlan([]Plan{fixedEstPlan("covering scan", false, 1)}, p, wide, st); ok {
		t.Fatalf("got plan %q when every candidate failed its precondition — an empty candidate "+
			"set returns the zero Plan and false, not the cheapest of nothing", dead.Name)
	}
}

func TestChoosePlan_PicksWrongWhenTheStatisticsAreStale(t *testing.T) {
	tbl := sampleTable()
	ix := NewIndex("idx_staff", []string{"staff_id"}, tbl.Rows)
	p := Pred{condEq("staff_id", 1)}
	plans := []Plan{fullScanPlan(), indexScanPlan()}

	believed := TableStats{Rows: 1000, Groups: 1000}

	got, ok := ChoosePlan(plans, p, ix, believed)
	if !ok {
		t.Fatal("no plan came back, but both candidates are usable")
	}
	if got.Name != "index scan" {
		t.Fatalf("chose %q, want index scan — Groups=1000 tells the optimizer staff_id is nearly "+
			"unique, so it expects %d matching row and estimates %d against full scan's %d",
			got.Name, estimatedRows(believed), indexScanPlan().Est(believed, ix, p),
			fullScanPlan().Est(believed, ix, p))
	}

	_, full := FullScan(tbl, p)
	_, index := IndexScan(tbl, ix, []int64{1}, []int64{2}, p)
	if index.Cost() <= full.Cost() {
		t.Fatalf("index scan really cost %d and full scan really cost %d — the fixture no longer "+
			"shows the estimate leading the engine into the dearer plan, so this test no longer "+
			"demonstrates anything", index.Cost(), full.Cost())
	}

	t.Logf("staff_id really has 2 values, %d rows each, so the index scan really costs %d against "+
		"full scan's %d — the estimate of %d said otherwise",
		len(tbl.Rows)/2, index.Cost(), full.Cost(), indexScanPlan().Est(believed, ix, p))
}
