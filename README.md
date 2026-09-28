# sql-execution-engine-from-scratch

A minimal SQL execution engine, built one mechanism at a time.

## Why

Reading about B+ trees, covering indexes, or index condition pushdown doesn't stick.
Writing them does.

This lab implements the mechanisms that decide **how a SQL query reads its data** —
small enough to finish in an evening, faithful enough that the numbers match what
real `EXPLAIN` tells you.

## What's here

Six steps. Each one adds a single capability to the same engine, and each step is a
copy of the previous one plus the new piece.

| Step | Capability | What it answers |
|------|-----------|-----------------|
| 1 | **Locate** | How does the engine find a starting point in a composite index? |
| 2 | **Pick an access path** | When does it read the full row, and when can it skip that? |
| 3 | **Deduplicate** | Why does `DISTINCT` sometimes need a temporary table? |
| 4 | **Skip groups** | Why can a loose index scan read 51 entries instead of 728,505? |
| 5 | **Push down** | When does filtering happen before the row is fetched? |
| 6 | **Choose a plan** | How does the optimizer pick, and why does it sometimes pick wrong? |

Steps 2–6 land as the lab grows. Step 1 is complete.

## Running

```bash
cd step1
go test ./...        # all green = this step is done
```

Every step is self-contained — `go test` in its directory is the whole workflow.

## How to use it

Each step leaves the core functions empty. Read that step's `README.md` for the
contract, then fill them in until the tests pass.

The tests are the spec: **one test per concept**, and the test names read as the
table of contents for that step.
