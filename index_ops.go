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

// RangeScan returns every entry with lo <= Key < hi, in index order.
//
// The result is a contiguous slice of ix.Keys.
func (ix *Index) RangeScan(lo, hi []int64) []Entry {
	return ix.Keys[ix.Seek(lo):ix.Seek(hi)]
}
