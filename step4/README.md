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

**D2 — The jump is `SeekPrefixLast(group value) + 1`.**
One B+ tree lookup, landing on the first entry past the group — however far away
the next value happens to be. It is not a step to the neighbouring entry, and it
is not the group's value plus one: that only means anything for a type whose
values can be counted, and it does not even work for a composite key — every
entry in the group already sorts after the group's own value.

**D3 — So the cost is the number of groups, not the number of entries.**
1000 entries in 2 groups: a tight scan reads 1000, a loose scan reads 2. Same
answer, and only the cheap path is worth the name.

**D4 — The grouped column can carry a range; the others can only carry equalities.**
The scan reads the grouped column once per group anyway, so narrowing it costs
nothing — the seek starts at the range's lower bound and stops at its upper one.
A condition on another column is folded into the seek as well, but only an
equality: that names a single value the seek can land on. A range there asks
about entries the scan never visits, and `CanLooseScan` refuses it. A condition
on a column the index does not carry is refused as well — there is nothing to
fold it into, and nothing to look at either.

Two lookups, two jobs. **Stepping past** a group asks only where the prefix ends,
so it names the grouped column and nothing else. **Landing on** a group asks
whether any of its entries satisfies the whole predicate, so it names every
condition — a seek that leaves them out lands on an entry that does not match and
reports the group anyway. And `Seek` answers "at or after", not "equal", so the
landing is only trustworthy once the entry it found is checked to start with the
key that was asked for.

The conditions belong to the caller: the scan reads them to build a seek, it does
not get to rewrite them.

**D5 — `Using index for group-by` is the signature of the loose scan.**
That line in `EXPLAIN` is the engine telling you it skipped whole groups.

**D6 — The grouped column has to lead the index.**
The jump to the next group is a seek on that column's next value. A column that
does not start the index has no such seek — its groups are scattered, and one
lookup cannot land past the group. This is why `GROUP BY` only gets the loose
treatment when it names a leftmost prefix of the index.

> **A simplification this lab makes.** The grouping column is a single column, so
> "leftmost prefix" always means the first one. A real `GROUP BY c1, c2` over an
> index on `(c1,c2,c3)` groups by the pair and jumps with `Seek([c1, c2+1])` — the
> same trick applied to the last grouping column. A range condition on a grouping
> column stays fine either way: that column is what is being grouped, not a filter
> on it.

## What to implement

```go
func TightScan(ix *Index, col string, conds []RangeCond) ([]int64, Stats)
func LooseScan(ix *Index, col string, conds []RangeCond) ([]int64, Stats)
func CanLooseScan(ix *Index, col string, conds []RangeCond) bool
```

`Cond` is a closure: it can say whether a value passes, but not what value it
was looking for, so there is nothing to fold into a seek. This step adds
`RangeCond{Col, Lo, Hi}` — `Lo == Hi` is an equality — and both scans take those
instead. `CanLooseScan` reads the same form, which is what lets `LooseScan`
call it.

The two scans answer the same question and differ in how they move to the next
value: one steps to the neighbouring entry, the other seeks past the whole
group. Read them side by side once you have written both.

Implement them in `loose.go`. Set `UsedLooseScan` on the stats so the caller can
tell which path ran, and so `Extra()` can report `Using index for group-by`.

## What you should see

```bash
go test ./step4/
```

Ten tests. One of them runs both scans over the same data and checks the entry
counts: 1000 for the tight scan, 2 for the loose one, same values out.
