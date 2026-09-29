# sql-execution-engine-from-scratch

A minimal SQL execution engine, built one mechanism at a time.

## Why

Reading about B+ trees, covering indexes, or index condition pushdown doesn't stick.
Writing them does.

This lab implements the mechanisms that decide **how a SQL query reads its data** —
small enough to finish in an evening, faithful enough that the numbers match what
real `EXPLAIN` tells you.

The engine is one package at the repository root. Each step is a directory holding
its `README.md` and the tests for the capability it adds — the tests are the spec,
and the code they exercise lives in the root package.

Each step leaves its functions empty, so its tests fail until you write them. Read
that step's `README.md` for the contract, fill in the bodies, and move on.

## The steps

| Step | Capability | The question it answers |
|------|-----------|-------------------------|
| [1](step1/) | **Locate** | How does the engine find a starting point in a composite index? |
| [2](step2/) | **Pick an access path** | When does it read the full row, and when can the index answer alone? |
| [3](step3/) | **Deduplicate** | Why does `DISTINCT` sometimes need a temporary table? |
| [4](step4/) | **Skip groups** | Why can a loose index scan read 2 entries where a tight scan reads 1000? |
| [5](step5/) | **Push down** | When does filtering happen before the row is fetched? |
| [6](step6/) | **Choose a plan** | How does the optimizer pick, and why does it sometimes pick wrong? |

Steps 3 to 6 each add an independent capability on top of the same engine; they
are siblings, not a chain. Only step 1 is a prerequisite for all of them.

## Concepts covered

Composite key ordering · `Seek` and range scans · leftmost prefix · covering index ·
table lookup · `Using index` · `Using index condition` · index condition pushdown ·
`Using temporary` · `Using filesort` · tight vs loose index scan ·
`Using index for group-by` · cardinality · row width · cost model ·
optimizer statistics · plan selection

## Layout

```
├── index.go        Row, Entry, Index, NewIndex, columnValue, ColIndex
├── table.go        Table
├── pred.go         Cond, Pred, Covers, eval, evalEntry
├── stats.go        Stats, Cost, FetchCost, ScanCost, TempInsertCost
│
├── index_ops.go    CompareKeys, Seek, RangeScan          ← step 1
├── scan.go         FullScan, IndexScan, CoveringScan, Extra   ← step 2
├── distinct.go     DistinctOrdered, DistinctTempTable    ← step 3
├── loose.go        RangeCond, TightScan, LooseScan, CanLooseScan  ← step 4
├── icp.go          ScanWithICP, ScanWithoutICP           ← step 5
├── plan.go         TableStats, Plan, ChoosePlan          ← step 6
│
├── step1/ … step6/ one README and one test file each
```

The first four files are written. Everything below them is yours.

## Running

```bash
go test ./step1/     # one step
go test ./...        # everything
```

Every step is a separate test package, so you can run them one at a time. All green
means that step is done.

## Reading the tests

The tests are the spec. Each step has **one test per concept**, and the test names
read as that step's table of contents — if you want to know what a step teaches,
read the function names.
