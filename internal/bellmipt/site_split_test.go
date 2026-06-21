package bellmipt

import (
	"testing"
)

func TestSplitCombineRoundTripAllConfigs(t *testing.T) {
	sites := 6
	split, err := NewSiteSplit([]int{0, 2, 5}, []int{1, 3, 4}, sites)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for q := uint64(0); q < (1 << sites); q++ {
		subQ, envQ := SplitConfig(q, split)
		qRecombined := CombineConfig(subQ, envQ, split)
		if q != qRecombined {
			t.Errorf("round trip failed for %d: got %d", q, qRecombined)
		}
	}
}

func TestNonContiguousPartition(t *testing.T) {
	sites := 4
	split, err := NewSiteSplit([]int{3, 1}, []int{0, 2}, sites)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// sub bits at 1 and 3, env bits at 0 and 2
	// For q = 10 (binary 1010): bit 1 is 1, bit 3 is 1 -> subQ = 11 (binary 3)
	// bit 0 is 0, bit 2 is 0 -> envQ = 0
	q := uint64(10)
	subQ, envQ := SplitConfig(q, split)
	if subQ != 3 {
		t.Errorf("expected subQ 3, got %d", subQ)
	}
	if envQ != 0 {
		t.Errorf("expected envQ 0, got %d", envQ)
	}

	qRecombined := CombineConfig(subQ, envQ, split)
	if qRecombined != q {
		t.Errorf("expected %d, got %d", q, qRecombined)
	}
}
