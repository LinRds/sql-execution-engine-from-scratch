# Step 3 — Deduplicate

Adds one capability: **collapsing repeated values into one.**

`DISTINCT` looks like a trivial operation. It is not — the whole cost depends on
whether the data already arrives in the order you need, and that is decided by
the index, not by the query.

## Concepts

**C1 — "Adjacent" turns set-deduplication into neighbour-comparison.**
When equal values sit next to each other, remembering the previous value is
enough. One variable, no matter how many rows.

**C2 — When they are scattered, you have to remember everything.**
There is no previous value to lean on, so every value seen so far has to be
kept. That is the temporary table.

**C3 — The deciding question is whether the column leads the index.**
The same column is contiguous in `(staff_id, created_at)` and scattered in
`(created_at, staff_id)`. Only the leading position gives you runs.

**C4 — `Using temporary` means "the data was not in the order this step needed".**
It is a statement about ordering, not about size.

**C5 — The two paths differ in cost, not in result.**
Both return the same values. One needs scratch space that grows with the row
count; the other needs one variable.

## What to implement

```go
func DistinctOrdered(ix *Index, col string, p Pred) ([]int64, Stats)
func DistinctTempTable(ix *Index, col string, p Pred) ([]int64, Stats)
```

The two functions are the same loop with one line different — that difference is
the entire concept. Read them side by side once you have written both.

Set `UsedTempTable` on the stats so the caller can tell which path ran, and so
`Extra()` can report `Using temporary`.

## What you should see

```bash
go test ./...
```

Five tests. The last one runs both paths over the same data and checks they
agree — the point being that the expensive path is not wrong, just expensive.
