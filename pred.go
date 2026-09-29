package engine

// Cond is one condition: a column has to satisfy a predicate.
type Cond struct {
	Col string
	Ok  func(int64) bool
}

// Pred is a list of conditions, all of which must hold.
type Pred []Cond

// Covers reports whether every column the predicate touches is carried by
// this index — that is, whether the predicate can be judged from an entry
// instead of from the row.
//
// Every index carries the primary key, because that is what an entry points
// with, so a condition on id is always covered.
func (ix *Index) Covers(p Pred) bool {
	for _, c := range p {
		if c.Col != "id" && ix.ColIndex(c.Col) < 0 {
			return false
		}
	}
	return true
}

// eval reports whether a full row satisfies every condition.
func (p Pred) eval(r Row) bool {
	for _, c := range p {
		if !c.Ok(columnValue(r, c.Col)) {
			return false
		}
	}
	return true
}

// evalEntry reports whether an index entry satisfies every condition.
//
// It returns false for a condition on a column the index does not carry —
// an entry cannot answer a question about a column it never stored. The primary
// key is the exception: it is what the entry points with, so it is always there.
func (p Pred) evalEntry(ix *Index, e Entry) bool {
	for _, c := range p {
		if c.Col == "id" {
			if !c.Ok(e.RowID) {
				return false
			}
			continue
		}
		i := ix.ColIndex(c.Col)
		if i < 0 || !c.Ok(e.Key[i]) {
			return false
		}
	}
	return true
}
