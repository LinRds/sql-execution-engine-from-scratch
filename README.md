# sql-execution-engine-from-scratch

A minimal SQL execution engine, built one mechanism at a time.

## Why

Reading about B+ trees, covering indexes, or index condition pushdown doesn't stick.
Writing them does.

This lab implements the mechanisms that decide **how a SQL query reads its data** —
small enough to finish in an evening, faithful enough that the numbers match what
real `EXPLAIN` tells you.

Every step leaves the core functions empty. Read that step's `README.md` for the
contract, fill them in until the tests pass, and move on.

## The steps

| Step | Capability | The question it answers |
|------|-----------|-------------------------|
| [1](step1/) | **Locate** | How does the engine find a starting point in a composite index? |
| [2](step2/) | **Pick an access path** | When does it read the full row, and when can the index answer alone? |
| [3](step3/) | **Deduplicate** | Why does `DISTINCT` sometimes need a temporary table? |
| [4](step4/) | **Skip groups** | Why can a loose index scan read 2 entries where a tight scan reads 1000? |
| [5](step5/) | **Push down** | When does filtering happen before the row is fetched? |
| [6](step6/) | **Choose a plan** | How does the optimizer pick, and why does it sometimes pick wrong? |

Each step is the previous one plus a single new capability, so the engine grows
with you. Steps share a base of `index.go`, `pred.go` and `scan.go`; each step adds
one file.

## Concepts covered

Composite key ordering · `Seek` and range scans · leftmost prefix · covering index ·
table lookup · `Using index` · `Using index condition` · index condition pushdown ·
`Using temporary` · `Using filesort` · tight vs loose index scan ·
`Using index for group-by` · cardinality · row width · cost model ·
optimizer statistics · plan selection

## Running

```bash
cd step1
go test ./...
```

Every step is self-contained — `go test` in its directory is the whole workflow.
All green means that step is done.

## Reading the tests

The tests are the spec. Each step has **one test per concept**, and the test names
read as that step's table of contents — if you want to know what a step teaches,
read the function names.
