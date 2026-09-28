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

**B3 — `Using index` means "no row was fetched".**
It does not mean "an index was used". A scan that walks an index and then reads
every row is not covered, and must not claim the flag.

**B4 — A fetched row costs far more than an index entry.**
It is wider, and reaching it is a random seek instead of a walk. `FetchCost`
prices that in.

**B5 — Fetch count decides the total, and low cardinality inflates it.**
`Seek` finds a starting point, not a short tail. When the key repeats 500 times,
paying for 500 random fetches costs more than reading the table once,
sequentially. A unique key leaves one fetch, which wins easily.

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

Keeping them apart is the whole point — `Cost()` only means something if the
counters it adds up are honest about which kind of work they measured.

## What you should see

```bash
go test ./...
```

Five tests, one per concept. The last one is the interesting one: it runs the
same predicate three ways and checks that the cheap path is the one you would
pick by hand.
