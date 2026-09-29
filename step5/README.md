# Step 5 — Push down

Adds one capability: **judging part of the predicate before the row is fetched.**

A condition on an indexed column can be answered from the index entry itself.
Settling it there means the entries it rejects never cost a table lookup — and
that is the whole of `Index Condition Pushdown`.

## Concepts

**E1 — `Using index condition` is the report of a condition judged before the row.**
The entry carries the indexed columns, so a condition on them can be settled on
the entry, and what it rejects never reaches the table.

**E2 — `Using where` is the report of a condition judged after the row.**
The index decides where the walk starts and stops; every entry in the range is
fetched and the predicate runs on the fetched row. That is what MySQL names
`Using where`; the engine here reports only what the index did.

**E3 — ICP needs a table lookup and a pushable condition, and both are required.**
A covering index answers the query from the entries alone, so there is no fetch
for a pushed condition to save. A predicate with no indexed column gives the
entry nothing to judge. Either one missing and there is nothing to push — and
then the scan is the plain one, with the flag clear.

The primary key counts as indexed. Every entry points with it, so it is on every
index, and a condition on `id` is one the entry can judge.

**E4 — ICP reduces fetches; it does not eliminate them.**
What survives the pushed condition still has to be looked up, because the rest of
the predicate was never in the index.

**E5 — ICP can be reported and save nothing.**
When the range already pins the indexed column to the values the condition
accepts, the pushed condition has nothing left to reject: same entries read and
same rows fetched as the plain scan.

## What to implement

```go
func ScanWithoutICP(t *Table, ix *Index, lo, hi []int64, p Pred) ([]Row, Stats)
func ScanWithICP(t *Table, ix *Index, lo, hi []int64, p Pred) ([]Row, Stats)
```

`ScanWithoutICP` walks the range, fetches every row it points at, and applies the
predicate to the fetched row.

`ScanWithICP` splits the predicate in two: the conditions whose column the index
carries, and the rest. The first part is judged on each entry, only the rows
whose entry survived are fetched, and the second part is judged there. Hand the
first part to `p.evalEntry` — it judges the primary key from the entry's
`RowID`, and answers false for a condition on a column the index lacks, so the
split is what makes it usable.

When either part is empty there is nothing to push: the scan should behave
exactly like `ScanWithoutICP` and leave the flag clear.

Implement them in `icp.go`. Set `UsedICP` on the stats when there is a condition
worth pushing *and* a row still worth fetching, and render it in `Extra()` as
`Using index condition`.

## What you should see

```bash
go test ./step5/
```

Six tests. E3 checks the three ways the prerequisite can be missed, and E5
checks that a pushed condition can cost nothing at all — the same entries read
and the same rows fetched as the plain scan.
