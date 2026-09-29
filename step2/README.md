# Step 2 — Pick an access path

Adds one capability: **deciding whether a row has to be read, or the index alone
can answer.**

Step 1 could only find a starting point. This step gives the engine three ways to
read a table, and a way to say what each one cost.

## Concepts

**B1 — A fetch is what happens when the index is not enough.**
An entry carries the key and the row id. Anything else the predicate asks about
has to come from the row — so every entry in the range costs one fetch.

**B2 — A covering index needs no fetch at all.**
When the index carries every column the predicate touches, the entries alone
answer it. `Index.Covers` reports whether that holds.

**B3 — `Using index` needs two things: an index was walked, and no row was fetched.**
Either half alone is not enough. A scan that walks an index and then reads every
row is not covered. A full scan fetches no row either — but it walks no index, so
the index cannot have answered it.

**B4 — A fetched row costs far more than an index entry.**
It is wider, and reaching it is a random seek instead of a walk. `FetchCost`
prices that in.

**B5 — Fetch count decides the total, and low cardinality inflates it.**
`Seek` finds a starting point, not a short tail. When the key repeats 500 times,
paying for 500 random fetches costs more than reading the table once,
sequentially. A unique key leaves one fetch, which wins easily.

**B6 — The counters are a report of what the scan did, so they have to match it.**
An entry read is an entry counted; a row handed back is a row counted. A scan
that reports zero entries walked costs nothing, and an optimizer reading that
number would pick it every time.

**B7 — The three paths differ in what they cost, not in what they return.**
`FullScan`, `IndexScan` and `CoveringScan` answer the same predicate with the
same rows. Only the bill changes.

> **A simplification this lab makes.** There is no `SELECT`, so every scan
> returns the whole row and the caller is responsible for having checked that
> the index can serve the query before choosing this path. A real optimizer
> takes the union of the selected and the filtered columns — which is what lets
> an index on `(staff_id)` alone answer
> `SELECT staff_id FROM t WHERE staff_id = 1`.

## What to implement

```go
func FullScan(t *Table, p Pred) ([]Row, Stats)
func IndexScan(t *Table, ix *Index, lo, hi []int64, p Pred) ([]Row, Stats)
func CoveringScan(t *Table, ix *Index, lo, hi []int64, p Pred) ([]Row, Stats)
func (s Stats) Extra() string
```

`Stats` has three counters, and they are not interchangeable:

| Counter | Counts | Priced at |
|---|---|---|
| `IndexEntriesRead` | entries walked | 1 |
| `RowsFetched` | random row fetches | `FetchCost` |
| `RowsScanned` | rows read in a sequential scan | `ScanCost` |

Implement them in `scan.go`. Keeping the counters apart is the whole point —
`Cost()` only means something if the counters it adds up are honest about which
kind of work they measured.

## What you should see

```bash
go test ./step2/
```

Eight tests. Three of them check that the numbers you report are the numbers you
earned: a covering scan that walks 500 entries cannot bill zero, a full scan
cannot claim `Using index`, and the three paths have to agree on the rows they
hand back.
