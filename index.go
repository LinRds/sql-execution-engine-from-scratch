package engine

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

func (e Entry) ToRow(ix *Index) Row {
	if ix == nil {
		return Row{}
	}
	row := Row{ID: e.RowID}
	if idx := ix.ColIndex("staff_id"); idx != -1 {
		row.StaffID = e.Key[idx]
	}
	if idx := ix.ColIndex("created_at"); idx != -1 {
		row.CreatedAt = e.Key[idx]
	}
	return row
}

// Index is a sorted index over a set of rows.
//
// Keys is sorted by Key, using the column order declared in Cols.
type Index struct {
	Name string
	Cols []string
	Keys []Entry
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
