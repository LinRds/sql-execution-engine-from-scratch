package step6

import "sort"

// Row is one row of the table.
type Row struct {
	ID        int64
	StaffID   int64
	CreatedAt int64
}

// Entry is one entry in an index.
//
// Key holds one value per indexed column, in the index's column order.
// RowID is the primary key of the row this entry points at.
type Entry struct {
	Key   []int64
	RowID int64
}

// Index is a sorted index over a set of rows.
type Index struct {
	Name string
	Cols []string
	Keys []Entry
}

// CompareKeys compares two composite keys lexicographically.
func CompareKeys(a, b []int64) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	}
	return 0
}

// Seek returns the index of the first entry whose Key is >= key, or
// len(ix.Keys) when every key is smaller.
func (ix *Index) Seek(key []int64) int {
	return sort.Search(len(ix.Keys), func(i int) bool {
		return CompareKeys(ix.Keys[i].Key, key) >= 0
	})
}

// RangeScan returns every entry with lo <= Key < hi, in index order.
func (ix *Index) RangeScan(lo, hi []int64) []Entry {
	return ix.Keys[ix.Seek(lo):ix.Seek(hi)]
}

// ColIndex returns the position of a column in this index, or -1 when the
// index does not carry it.
func (ix *Index) ColIndex(col string) int {
	for i, c := range ix.Cols {
		if c == col {
			return i
		}
	}
	return -1
}

// Covers reports whether every column the predicate touches is carried by
// this index. When it is, the predicate can be answered without the row.
func (ix *Index) Covers(p Pred) bool {
	for _, c := range p {
		if ix.ColIndex(c.Col) < 0 {
			return false
		}
	}
	return true
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
