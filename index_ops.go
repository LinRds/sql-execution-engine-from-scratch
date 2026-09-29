package engine

import "sort"

// CompareKeys compares two composite keys lexicographically.
//
// The first differing element decides. If one key is a prefix of the other,
// the shorter one sorts first.
func CompareKeys(a, b []int64) int {
	if len(a) > len(b) {
		return -CompareKeys(b, a)
	}
	for i := range a {
		if a[i] < b[i] {
			return -1
		} else if a[i] > b[i] {
			return 1
		}
	}
	if len(a) != len(b) {
		return -1
	}
	return 0
}

// Seek returns the index of the first entry whose Key is >= key.
//
// It returns len(ix.Keys) when every key in the index is smaller than the
// one being searched for.
func (ix *Index) Seek(key []int64) int {
	return sort.Search(len(ix.Keys), func(i int) bool {
		return CompareKeys(key, ix.Keys[i].Key) <= 0
	})
}

// SeekPrefixLast returns the index of the last entry whose Key starts with
// prefix, or -1 when no entry does.
//
// This is what a scan uses to step past a whole group. Asking for the first key
// greater than the group's value does not work: every entry in the group is
// already greater than that value once the key has more columns, so the lookup
// lands back inside the group it was meant to leave.
func (ix *Index) SeekPrefixLast(prefix []int64) int {
	i := sort.Search(len(ix.Keys), func(i int) bool {
		return comparePrefix(ix.Keys[i].Key, prefix) > 0
	})
	if i > 0 && hasPrefix(ix.Keys[i-1].Key, prefix) {
		return i - 1
	}
	return -1
}

// comparePrefix orders two keys by the first len(prefix) elements only. A key
// too short to hold the prefix sorts before it.
func comparePrefix(key, prefix []int64) int {
	if len(key) < len(prefix) {
		return -1
	}
	return CompareKeys(key[:len(prefix)], prefix)
}

func hasPrefix(key, prefix []int64) bool {
	return comparePrefix(key, prefix) == 0
}

// RangeScan returns every entry with lo <= Key < hi, in index order.
//
// The result is a contiguous slice of ix.Keys.
func (ix *Index) RangeScan(lo, hi []int64) []Entry {
	return ix.Keys[ix.Seek(lo):ix.Seek(hi)]
}
