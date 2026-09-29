package engine

import "sort"

// RangeCond is one condition on a column: Lo <= value <= Hi.
//
// Lo == Hi means the condition is an equality.
type RangeCond struct {
	Col string
	Lo  int64
	Hi  int64
}

// TightScan returns the distinct values of col by walking every index entry.
//
// Equal values sit next to each other, so comparing the current value with the
// previous one is enough to tell whether it is new.
//
// The conditions filter entries; pass nil to keep everything.
func TightScan(ix *Index, col string, conds []RangeCond) ([]int64, Stats) {
	return DistinctOrdered(ix, col, transRangeCondToPred(conds))
}

func transRangeCondToPred(conds []RangeCond) Pred {
	p := Pred{}
	for _, cond := range conds {
		p = append(p, Cond{
			Col: cond.Col,
			Ok:  func(i int64) bool { return i >= cond.Lo && i <= cond.Hi },
		})
	}
	return p
}

// LooseScan returns the distinct values of col by reading one entry per group
// and seeking past the rest of that group.
//
// Every condition on a column other than col is folded into the seek, so the
// scan lands on the first entry of a group that satisfies them and never looks
// at the rest of the group. A condition it cannot fold — a range on another
// column — is what CanLooseScan rules out.
//
// It sets UsedLooseScan, which Extra() reports as Using index for group-by.
func LooseScan(ix *Index, col string, conds []RangeCond) ([]int64, Stats) {
	stats := Stats{}
	if !CanLooseScan(ix, col, conds) {
		return nil, stats
	}
	stats.UsedLooseScan = true
	idx := ix.ColIndex(col)
	filtered := make([]int64, 0)
	var prefixCond RangeCond
	for _, cond := range conds {
		if cond.Col == col {
			prefixCond = cond
		}
	}

	st := ix.Keys[0].Key[idx]
	if prefixCond.Col != "" {
		st = prefixCond.Lo
	}
	suffix := buildSeekSuffix(ix, col, conds)
	for {
		keys := append([]int64{st}, suffix...)
		i := ix.Seek(keys)
		if i >= len(ix.Keys) {
			break
		}
		st = ix.Keys[i].Key[idx]
		if prefixCond.Col != "" && st > prefixCond.Hi {
			break
		}
		stats.IndexEntriesRead++
		if hasPrefix(ix.Keys[i].Key, keys) {
			filtered = append(filtered, st)
			stats.RowsReturned++
		}

		j := ix.SeekPrefixLast([]int64{st}) + 1
		if j >= len(ix.Keys) {
			break
		}
		st = ix.Keys[j].Key[idx]
	}
	return filtered, stats
}

func buildSeekSuffix(ix *Index, col string, conds []RangeCond) []int64 {
	if ix == nil {
		return nil
	}
	sorted := make([]RangeCond, len(conds))
	copy(sorted, conds)
	sort.Slice(sorted, func(i, j int) bool {
		return ix.ColIndex(sorted[i].Col) < ix.ColIndex(sorted[j].Col)
	})
	suffix := make([]int64, 0, len(sorted))
	for _, cond := range sorted {
		if cond.Col == col {
			continue
		}
		suffix = append(suffix, cond.Hi)
	}
	return suffix
}

// CanLooseScan reports whether a loose scan over col is allowed.
//
// The column has to lead the index, and every condition on a column other than
// col has to be an equality.
func CanLooseScan(ix *Index, col string, conds []RangeCond) bool {
	idx := ix.ColIndex(col)
	if idx != 0 {
		return false
	}
	for _, cond := range conds {
		if ix.ColIndex(cond.Col) == -1 {
			return false
		}
		if cond.Col != col && cond.Hi != cond.Lo {
			return false
		}
	}
	return true
}
