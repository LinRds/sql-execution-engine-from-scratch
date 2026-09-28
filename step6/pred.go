package step6

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
// an entry cannot answer a question about a column it never stored.
func (p Pred) evalEntry(ix *Index, e Entry) bool {
	for _, c := range p {
		i := ix.ColIndex(c.Col)
		if i < 0 || !c.Ok(e.Key[i]) {
			return false
		}
	}
	return true
}
