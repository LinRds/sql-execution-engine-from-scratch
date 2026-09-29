# Step 4 — Skip Whole Groups

Adds one capability: **answering a grouped query without reading every entry in each group.**

Step 3 deduplicated by walking the index. That works, but it pays for rows it does
not need: once a tight scan has the group's value, the remaining 499 entries of
that group are already decided, and it still reads them. A loose index scan reads
one entry per group and jumps over the rest.

## Concepts

**D1 — A tight scan walks every entry in the group; a loose scan skips the whole group.**
The tight scan only notices the group ended when it reaches an entry with a
different value. The loose scan has the group's value already.

**D2 — The jump is `Seek(current value + 1)`.**
One B+ tree lookup, landing on the first entry past the group — however far away
the next value happens to be. It is not a step to the neighbouring entry.

**D3 — So the cost is the number of groups, not the number of entries.**
1000 entries in 2 groups: a tight scan reads 1000, a loose scan reads 2. Same
answer, and only the cheap path is worth the name.

**D4 — The other conditions have to be equalities.**
A range on another column asks about rows the scan jumps over, and it never looks
at them. A range on the grouped column is fine — the scan reads that column once
per group anyway.

**D5 — `Using index for group-by` is the signature of the loose scan.**
That line in `EXPLAIN` is the engine telling you it skipped whole groups.

## What to implement

```go
func TightScan(ix *Index, col string, p Pred) ([]int64, Stats)
func LooseScan(ix *Index, col string, p Pred) ([]int64, Stats)
func CanLooseScan(ix *Index, col string, conds []RangeCond) bool
```

`Cond` cannot say whether a condition is an equality or a range, so this step adds
`RangeCond{Col, Lo, Hi}` — `Lo == Hi` is an equality. That is what `CanLooseScan`
reads, and it is the whole of D4.

The two scans are the same loop with one line different — how they move to the
next value. Read them side by side once you have written both.

Implement them in `loose.go`. Set `UsedLooseScan` on the stats so the caller can
tell which path ran, and so `Extra()` can report `Using index for group-by`.

## What you should see

```bash
go test ./step4/
```

Five tests. One of them runs both scans over the same data and checks the entry
counts: 1000 for the tight scan, 2 for the loose one, same values out.
