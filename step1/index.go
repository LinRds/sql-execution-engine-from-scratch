package step1

import "sort"

// Row is one row of the table. Step 1 only needs these columns.
type Row struct {
	ID        int64
	StaffID   int64
	CreatedAt int64
}

// Entry is one entry in an index.
//
// Key holds one value per indexed column, in the index's column order.
// RowID is the primary key of the row this entry points at — the handle used
// to fetch the full row in later steps.
type Entry struct {
	Key   []int64
	RowID int64
}

// Index is a sorted index over a set of rows.
//
// Keys is sorted by Key, using the column order declared in Cols.
type Index struct {
	Name string
	Cols []string
	Keys []Entry
}

// CompareKeys compares two composite keys lexicographically.
//
// The first differing element decides. If one key is a prefix of the other,
// the shorter one sorts first.
func CompareKeys(a, b []int64) int {
	if len(a) > len(b) {
		return -CompareKeys(b, a)
	}
	for i := range a {
		if a[i] < b[i] {
			return -1
		} else if a[i] > b[i] {
			return 1
		}
	}
	if len(a) != len(b) {
		return -1
	}
	return 0
}

// Seek returns the index of the first entry whose Key is >= key.
//
// It returns len(ix.Keys) when every key in the index is smaller than the
// one being searched for.
func (ix *Index) Seek(key []int64) int {
	return sort.Search(len(ix.Keys), func(i int) bool {
		return CompareKeys(key, ix.Keys[i].Key) <= 0
	})
}

// RangeScan returns every entry with lo <= Key < hi, in index order.
//
// The result is a contiguous slice of ix.Keys.
func (ix *Index) RangeScan(lo, hi []int64) []Entry {
	entries := make([]Entry, 0) // 没法估计一个合适的 cap，是不是需要有预估数据分布的能力
	for i := ix.Seek(lo); i < ix.Seek(hi); i++ {
		entries = append(entries, ix.Keys[i])
	}
	return entries
}

// NewIndex builds an index over rows, sorted by the given columns.
func NewIndex(name string, cols []string, rows []Row) *Index {
	keys := make([]Entry, 0, len(rows))
	for _, r := range rows {
		key := make([]int64, len(cols))
		for i, c := range cols {
			key[i] = columnValue(r, c)
		}
		keys = append(keys, Entry{Key: key, RowID: r.ID})
	}
	sort.Slice(keys, func(i, j int) bool {
		return CompareKeys(keys[i].Key, keys[j].Key) < 0
	})
	return &Index{Name: name, Cols: cols, Keys: keys}
}

func columnValue(r Row, col string) int64 {
	switch col {
	case "id":
		return r.ID
	case "staff_id":
		return r.StaffID
	case "created_at":
		return r.CreatedAt
	}
	panic("unknown column: " + col)
}
