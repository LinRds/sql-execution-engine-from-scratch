# Step 1 — Locate

Adds one capability to the engine: **finding a starting point in a composite index.**

Everything later (range scans, covering indexes, loose scans, pushdown) is built on
top of this. If `Seek` is wrong, every number downstream is wrong.

## Concepts

**A1 — A composite key is one key.**
`(a, b)` sorts by `a` first, then by `b`. Looking up a composite key is a single
descent, not two separate lookups.

**A2 — `Seek(key)` finds the first entry `>= key`.**
Two cases, and both matter: the key exists, or it does not. When nothing is `>= key`,
the answer is `len(Keys)` — one past the end, which is exactly what a range scan
needs as a stopping point.

**A3 — A range scan is `Seek(lo)` plus reading forward until `hi`.**
No searching per entry. The scan walks the already-sorted array.

**A4 — An entry carries `Key` and `RowID`.**
`Key` is the indexed columns. `RowID` is the primary key — the handle used to fetch
the full row later. An index that did not carry `RowID` could never be used to read
anything but its own columns.

**A5 — Leftmost prefix: equality on the first column is one contiguous range.**
`WHERE a = 1` over index `(a, b)` is the range `[(1, -∞), (2, -∞))`. One `Seek` plus
one forward scan covers every matching entry.

**A6 — Column order decides what can be located.**
The same column can be a locating column in one index and a scattered one in another.
`staff_id` leads in `(staff_id, created_at)` and trails in `(created_at, staff_id)` —
so `WHERE staff_id = 1` can use the first index to jump straight to its entries, and
cannot use the second.

**A7 — `Seek` does not guarantee a short tail.**
`Seek` finds a starting point. How far the end is from that point depends on how many
repeated values the key has. A unique column leaves a tail of one; a two-valued column
leaves half the index.

## What to implement

```go
// CompareKeys compares two composite keys lexicographically:
// first element first, then the second, and so on.
func CompareKeys(a, b []int64) int

// Seek returns the index of the first entry whose Key is >= key.
// It returns len(ix.Keys) when every key is smaller.
func (ix *Index) Seek(key []int64) int

// RangeScan returns every entry with lo <= Key < hi, in index order.
func (ix *Index) RangeScan(lo, hi []int64) []Entry
```

`NewIndex` and `columnValue` are already written — they build the sorted `Keys` slice
you will be searching. Read them first: they define the shape you are working with.

## What you should see

```bash
go test ./...
```

Eight tests, one per concept above. The names read as this step's table of contents.
