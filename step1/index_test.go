package step1

import "testing"

// sampleRows builds 1000 rows with two different cardinalities on purpose:
//
//   - id and created_at are unique      -> a Seek on them leaves a tail of 1
//   - staff_id has two values, 500 each -> a Seek on it leaves a tail of 500
//
// Every concept in this step is easier to see against that contrast.
func sampleRows() []Row {
	rows := make([]Row, 0, 1000)
	for i := 0; i < 1000; i++ {
		rows = append(rows, Row{
			ID:        int64(i + 1),
			StaffID:   int64(i%2 + 1),
			CreatedAt: 1700000000 + int64(i),
		})
	}
	return rows
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

// A1: a composite key is one key. (a, b) sorts by a first, then by b.
func TestCompareKeys_CompositeOrdersByFirstThenSecond(t *testing.T) {
	cases := []struct {
		name string
		a, b []int64
		want int
	}{
		{"the first element decides", []int64{1, 9}, []int64{2, 0}, -1},
		{"the second element decides when the first ties", []int64{1, 2}, []int64{1, 3}, -1},
		{"equal keys compare equal", []int64{1, 2}, []int64{1, 2}, 0},
		{"a prefix sorts before its extension", []int64{1}, []int64{1, 0}, -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sign(CompareKeys(c.a, c.b)); got != c.want {
				t.Fatalf("CompareKeys(%v, %v) = %d, want sign %d", c.a, c.b, got, c.want)
			}
			if got := sign(CompareKeys(c.b, c.a)); got != -c.want {
				t.Fatalf("CompareKeys(%v, %v) = %d but CompareKeys(%v, %v) = %d — comparison must be antisymmetric",
					c.a, c.b, c.want, c.b, c.a, got)
			}
		})
	}
}

// A2: Seek finds the first entry >= key. Both cases matter — the key exists,
// and the key is past the end. A range scan needs the second one as its stop.
func TestSeek_ReturnsFirstEntryNotLessThanKey(t *testing.T) {
	ix := NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, sampleRows())

	t.Run("finds an existing key", func(t *testing.T) {
		want := []int64{2, 0}
		i := ix.Seek(want)
		if i >= len(ix.Keys) {
			t.Fatalf("Seek(%v) = %d, past the end of a %d-entry index — staff_id 2 is in half the rows",
				want, i, len(ix.Keys))
		}
		if CompareKeys(ix.Keys[i].Key, want) < 0 {
			t.Fatalf("Seek(%v) landed on %v, which is smaller than the key it was looking for",
				want, ix.Keys[i].Key)
		}
		if i > 0 && CompareKeys(ix.Keys[i-1].Key, want) >= 0 {
			t.Fatalf("Seek(%v) = %d, but entry %d is %v — that one is already >= the key, so %d is not the first",
				want, i, i-1, ix.Keys[i-1].Key, i)
		}
	})

	t.Run("returns len when every key is smaller", func(t *testing.T) {
		want := []int64{99, 0}
		if i := ix.Seek(want); i != len(ix.Keys) {
			t.Fatalf("Seek(%v) = %d, want %d (len) — nothing in the index is >= a key past the end",
				want, i, len(ix.Keys))
		}
	})
}

// A3: a range scan is Seek(lo) plus reading forward until hi. No per-entry search.
func TestRangeScan_StopsAtTheUpperBound(t *testing.T) {
	ix := NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, sampleRows())

	lo, hi := []int64{1, 0}, []int64{2, 0}
	got := ix.RangeScan(lo, hi)

	if len(got) != 500 {
		t.Fatalf("RangeScan(%v, %v) returned %d entries, want 500 — staff_id 1 is in exactly half the rows",
			lo, hi, len(got))
	}
	for n, e := range got {
		if CompareKeys(e.Key, lo) < 0 || CompareKeys(e.Key, hi) >= 0 {
			t.Fatalf("entry %d is %v, outside [%v, %v)", n, e.Key, lo, hi)
		}
	}
}

// A4: an entry carries RowID — the handle used to fetch the full row later.
// Without it, an index could never return anything but its own columns.
func TestEntry_CarriesRowIDForLaterLookup(t *testing.T) {
	rows := sampleRows()
	ix := NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, rows)

	byID := make(map[int64]Row, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}

	for n, e := range ix.Keys {
		r, ok := byID[e.RowID]
		if !ok {
			t.Fatalf("entry %d is %v with RowID %d, which matches no row", n, e.Key, e.RowID)
		}
		if r.StaffID != e.Key[0] || r.CreatedAt != e.Key[1] {
			t.Fatalf("entry %d is %v but RowID %d points at %+v — the key and the row disagree",
				n, e.Key, e.RowID, r)
		}
	}
}

// A5: leftmost prefix. Equality on the first column is one contiguous range,
// so a single Seek plus a forward scan covers every matching entry.
func TestLeftmostPrefix_EqualityOnFirstColCoversOneContiguousRange(t *testing.T) {
	ix := NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, sampleRows())

	got := ix.RangeScan([]int64{1, 0}, []int64{2, 0})

	if len(got) != 500 {
		t.Fatalf("one RangeScan returned %d entries, want 500 — staff_id 1 should be one unbroken run",
			len(got))
	}
	for n, e := range got {
		if e.Key[0] != 1 {
			t.Fatalf("entry %d is %v — the run broke at position %d", n, e.Key, n)
		}
	}
}

// A6: column order decides what can be located. The same column leads one index
// and trails another, and only the leading position can be jumped to.
func TestLeftmostPrefix_ColumnOrderDecidesWhatCanBeLocated(t *testing.T) {
	rows := sampleRows()
	byStaff := NewIndex("idx_staff_created", []string{"staff_id", "created_at"}, rows)
	byCreated := NewIndex("idx_created_staff", []string{"created_at", "staff_id"}, rows)

	if !columnGroupsAreContiguous(byStaff, 0) {
		t.Fatal("in (staff_id, created_at), all staff_id 1 entries should form one run — staff_id leads this index")
	}
	if columnGroupsAreContiguous(byCreated, 1) {
		t.Fatal("in (created_at, staff_id), staff_id 1 entries are scattered across the index — " +
			"staff_id trails here, so WHERE staff_id = 1 cannot jump to them")
	}
}

// A7: Seek finds a starting point, not a short tail. How far the end is from
// that point depends on how many times the key repeats.
func TestSeek_DoesNotGuaranteeAShortTail(t *testing.T) {
	rows := sampleRows()
	unique := NewIndex("idx_id", []string{"id"}, rows)
	twoValued := NewIndex("idx_staff", []string{"staff_id"}, rows)

	if n := runLength(unique, unique.Seek([]int64{500})); n != 1 {
		t.Fatalf("on a unique column, the tail after Seek is %d entries, want 1", n)
	}
	if n := runLength(twoValued, twoValued.Seek([]int64{1})); n != 500 {
		t.Fatalf("on a two-valued column, the tail after Seek is %d entries, want 500 — "+
			"Seek located the start, but the end is half the index away", n)
	}
}

// columnGroupsAreContiguous reports whether entries sharing a value in the given
// column position form one unbroken run. A value that reappears after another
// value has been seen means the groups are scattered.
func columnGroupsAreContiguous(ix *Index, col int) bool {
	seen := make(map[int64]bool)
	var prev int64
	started := false
	for _, e := range ix.Keys {
		v := e.Key[col]
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

// runLength counts how many consecutive entries share the first key element with
// the entry at index from.
func runLength(ix *Index, from int) int {
	if from >= len(ix.Keys) {
		return 0
	}
	n := 0
	for i := from; i < len(ix.Keys) && ix.Keys[i].Key[0] == ix.Keys[from].Key[0]; i++ {
		n++
	}
	return n
}
