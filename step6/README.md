# Step 6 — Choose a plan

Adds one capability: **picking the access path without being told which one.**

Steps 2 through 5 each ran one fixed path, and the test named the one to use.
Nothing chose. Here the engine is handed a set of candidate plans and has to
pick — which is the entire job of an optimizer.

## Concepts

**F1 — The choice is made on estimated cost, before anything runs.**
A plan wins because its estimate is the smallest, not because it turned out to be
fastest. The optimizer never executes the candidates to find out; it reads a
number each plan reports about itself.

**F2 — A cost is entries read, plus rows fetched times `FetchCost`.**
Reading an index entry is cheap. Fetching the row it points at is `FetchCost`
times dearer. That one constant is why a covering scan beats an index scan over
the same range, and why an index scan can lose to a full table scan.

**F3 — The estimate is built from statistics, not from data.**
`TableStats` carries a row count and a number of distinct values (cardinality),
and estimated rows matched is `Rows / Groups`. Every plan's estimate is a
function of that number and of nothing else.

**F4 — A plan that cannot run never becomes a candidate.**
Covering scan needs every column the predicate touches to be carried by the
index. A plan that fails its precondition is not ranked last — it is not there,
however cheap it claims to be.

**F5 — Stale statistics pick the wrong plan, and nothing notices.**
The estimate is only as good as the statistics behind it. When they say a column
has 1000 distinct values and it really has 2, the optimizer believes a predicate
matches one row, picks the index path, and then reads 500 entries and fetches 500
rows to get them.

## What to implement

```go
func ChoosePlan(plans []Plan, p Pred, ix *Index, st TableStats) (Plan, bool)
```

Implement it in `plan.go`. Each `Plan` carries its own `Usable` precondition and
its own `Est`. `ChoosePlan` drops the candidates whose precondition fails and
returns the cheapest of what is left. When nothing is left it returns the zero
`Plan` and `false` — an empty candidate set is a real answer, not an error to hide.

## What you should see

```bash
go test ./step6/
```

Five tests. Each one hands the engine plans and statistics and checks which plan
came back; no data is read to make the decision, which is the point. The last
test runs the plan that was chosen and shows it was the expensive one.
