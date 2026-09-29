package engine

// Table holds the rows an index points into.
type Table struct {
	Rows []Row
	ByID map[int64]Row
}
