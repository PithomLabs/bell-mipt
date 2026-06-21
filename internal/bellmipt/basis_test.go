package bellmipt

import "testing"

func TestBasisEnumeration(t *testing.T) {
	b, err := NewBasis(3)
	if err != nil {
		t.Fatalf("NewBasis(3) returned error: %v", err)
	}
	if b.Sites != 3 {
		t.Errorf("Sites = %d, want 3", b.Sites)
	}
	if b.Dim != 8 {
		t.Errorf("Dim = %d, want 8", b.Dim)
	}

	states := EnumerateStates(b)
	if len(states) != 8 {
		t.Fatalf("EnumerateStates returned %d states, want 8", len(states))
	}
	for i, s := range states {
		if s != uint64(i) {
			t.Errorf("states[%d] = %d, want %d", i, s, i)
		}
	}
}

func TestNewBasisErrors(t *testing.T) {
	if _, err := NewBasis(0); err == nil {
		t.Error("NewBasis(0) should return an error")
	}
	if _, err := NewBasis(-1); err == nil {
		t.Error("NewBasis(-1) should return an error")
	}
	if _, err := NewBasis(64); err == nil {
		t.Error("NewBasis(64) should return an error")
	}
}

func TestOccupied(t *testing.T) {
	// state = 0b101 = 5: sites 0 and 2 occupied, site 1 not.
	state := uint64(0b101)
	if !Occupied(state, 0) {
		t.Error("site 0 should be occupied in 0b101")
	}
	if Occupied(state, 1) {
		t.Error("site 1 should NOT be occupied in 0b101")
	}
	if !Occupied(state, 2) {
		t.Error("site 2 should be occupied in 0b101")
	}
}

func TestPopCount(t *testing.T) {
	tests := []struct {
		state uint64
		want  int
	}{
		{0b000, 0},
		{0b001, 1},
		{0b010, 1},
		{0b011, 2},
		{0b101, 2},
		{0b111, 3},
		{0b1111, 4},
	}
	for _, tc := range tests {
		got := PopCount(tc.state)
		if got != tc.want {
			t.Errorf("PopCount(0b%b) = %d, want %d", tc.state, got, tc.want)
		}
	}
}

func TestCountBelow(t *testing.T) {
	// state = 0b110 = 6: sites 1 and 2 occupied.
	state := uint64(0b110)
	tests := []struct {
		site int
		want int
	}{
		{0, 0}, // no bits below site 0
		{1, 0}, // no bits below site 1 (site 0 is not occupied)
		{2, 1}, // site 1 is occupied below site 2
	}
	for _, tc := range tests {
		got := CountBelow(state, tc.site)
		if got != tc.want {
			t.Errorf("CountBelow(0b110, %d) = %d, want %d", tc.site, got, tc.want)
		}
	}

	// state = 0b111 = 7: sites 0, 1, 2 all occupied.
	state2 := uint64(0b111)
	if got := CountBelow(state2, 2); got != 2 {
		t.Errorf("CountBelow(0b111, 2) = %d, want 2", got)
	}
	if got := CountBelow(state2, 0); got != 0 {
		t.Errorf("CountBelow(0b111, 0) = %d, want 0", got)
	}
	if got := CountBelow(state2, 1); got != 1 {
		t.Errorf("CountBelow(0b111, 1) = %d, want 1", got)
	}
}
