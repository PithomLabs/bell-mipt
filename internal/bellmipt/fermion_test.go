package bellmipt

import (
	"fmt"
	"testing"
)

func TestAnnihilationSigns(t *testing.T) {
	tests := []struct {
		name      string
		state     uint64
		site      int
		wantState uint64
		wantSign  int
	}{
		{
			name:      "c_0 |01>",
			state:     0b01, // site 0 occupied
			site:      0,
			wantState: 0b00,
			wantSign:  +1, // 0 bits below site 0
		},
		{
			name:      "c_1 |11>",
			state:     0b11, // sites 0,1 occupied
			site:      1,
			wantState: 0b01,
			wantSign:  -1, // 1 bit below site 1 (site 0)
		},
		{
			name:      "c_1 |10>",
			state:     0b10, // site 1 occupied
			site:      1,
			wantState: 0b00,
			wantSign:  +1, // 0 bits below site 1
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := ApplyAnnihilate(tc.state, tc.site)
			if !res.OK {
				t.Fatal("expected OK=true, got OK=false")
			}
			if res.State != tc.wantState {
				t.Errorf("state = 0b%b, want 0b%b", res.State, tc.wantState)
			}
			if res.Sign != tc.wantSign {
				t.Errorf("sign = %d, want %d", res.Sign, tc.wantSign)
			}
		})
	}
}

func TestCreationSigns(t *testing.T) {
	tests := []struct {
		name      string
		state     uint64
		site      int
		wantState uint64
		wantSign  int
	}{
		{
			name:      "c†_0 |00>",
			state:     0b00,
			site:      0,
			wantState: 0b01,
			wantSign:  +1, // 0 bits below site 0
		},
		{
			name:      "c†_1 |01>",
			state:     0b01, // site 0 occupied
			site:      1,
			wantState: 0b11,
			wantSign:  -1, // 1 bit below site 1 (site 0)
		},
		{
			name:      "c†_1 |00>",
			state:     0b00,
			site:      1,
			wantState: 0b10,
			wantSign:  +1, // 0 bits below site 1
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := ApplyCreate(tc.state, tc.site)
			if !res.OK {
				t.Fatal("expected OK=true, got OK=false")
			}
			if res.State != tc.wantState {
				t.Errorf("state = 0b%b, want 0b%b", res.State, tc.wantState)
			}
			if res.Sign != tc.wantSign {
				t.Errorf("sign = %d, want %d", res.Sign, tc.wantSign)
			}
		})
	}
}

func TestNilOnInvalidCreationOrAnnihilation(t *testing.T) {
	// Annihilate unoccupied site.
	res := ApplyAnnihilate(0b00, 0)
	if res.OK {
		t.Error("c_0 |00> should return OK=false")
	}

	// Create on already occupied site.
	res = ApplyCreate(0b01, 0)
	if res.OK {
		t.Error("c†_0 |01> should return OK=false")
	}
}

func TestApplyOpsRightToLeft(t *testing.T) {
	// Physics notation: c†_1 c_0, meaning "annihilate site 0, then create site 1".
	// ops in physics order: [{Create, 1}, {Annihilate, 0}]
	// Execution order (right-to-left): first Annihilate site 0, then Create site 1.
	//
	// state = 1 (0b01): c_0 gives state=0 sign=+1, then c†_1 gives state=2 sign=+1.
	// Total sign = +1, final state = 2 (0b10).
	ops := []FermionOp{
		{Kind: Create, Site: 1},
		{Kind: Annihilate, Site: 0},
	}
	res := ApplyOps(1, ops)
	if !res.OK {
		t.Fatal("expected OK=true, got OK=false")
	}
	if res.State != 2 {
		t.Errorf("state = %d (0b%b), want 2 (0b10)", res.State, res.State)
	}
	if res.Sign != +1 {
		t.Errorf("sign = %d, want +1", res.Sign)
	}
}

// TestFermionAnticommutationSmallBasis verifies {c_i, c†_j} = δ_{ij}
// on every basis state for small systems.
func TestFermionAnticommutationSmallBasis(t *testing.T) {
	for _, sites := range []int{2, 3} {
		t.Run(fmt.Sprintf("sites=%d", sites), func(t *testing.T) {
			b, err := NewBasis(sites)
			if err != nil {
				t.Fatalf("NewBasis(%d): %v", sites, err)
			}
			states := EnumerateStates(b)

			for i := 0; i < sites; i++ {
				for j := 0; j < sites; j++ {
					for _, n := range states {
						// Compute c_i c†_j |n> + c†_j c_i |n>
						// and check it equals δ_{ij} |n>.

						// Term 1: c_i c†_j |n>
						// First apply c†_j, then c_i.
						term1State := uint64(0)
						term1Sign := 0
						r1 := ApplyCreate(n, j)
						if r1.OK {
							r2 := ApplyAnnihilate(r1.State, i)
							if r2.OK {
								term1State = r2.State
								term1Sign = r1.Sign * r2.Sign
							}
						}

						// Term 2: c†_j c_i |n>
						// First apply c_i, then c†_j.
						term2State := uint64(0)
						term2Sign := 0
						r3 := ApplyAnnihilate(n, i)
						if r3.OK {
							r4 := ApplyCreate(r3.State, j)
							if r4.OK {
								term2State = r4.State
								term2Sign = r3.Sign * r4.Sign
							}
						}

						if i == j {
							// {c_i, c†_i} = 1, so result should be 1 * |n>.
							// Exactly one of the two terms should give |n> with sign +1,
							// or both should sum to coefficient +1 on |n>.
							coeff := 0
							if term1Sign != 0 && term1State == n {
								coeff += term1Sign
							}
							if term2Sign != 0 && term2State == n {
								coeff += term2Sign
							}
							if coeff != 1 {
								t.Errorf("{c_%d, c†_%d} |%d> = %d |%d>, want 1 |%d>",
									i, j, n, coeff, n, n)
							}
						} else {
							// {c_i, c†_j} = 0 for i != j.
							// Both terms should cancel or both be zero.
							// If neither term produced a result, anticommutator is 0. Good.
							if term1Sign == 0 && term2Sign == 0 {
								continue // both zero, OK
							}
							// If both produced results, they must be to the same state
							// with opposite signs.
							if term1Sign != 0 && term2Sign != 0 {
								if term1State != term2State {
									t.Errorf("{c_%d, c†_%d} |%d>: terms give different states %d and %d",
										i, j, n, term1State, term2State)
								} else if term1Sign+term2Sign != 0 {
									t.Errorf("{c_%d, c†_%d} |%d>: signs don't cancel: %d + %d = %d",
										i, j, n, term1Sign, term2Sign, term1Sign+term2Sign)
								}
							} else {
								// Only one term is nonzero — anticommutator is not zero.
								t.Errorf("{c_%d, c†_%d} |%d>: only one term nonzero, anticommutator != 0",
									i, j, n)
							}
						}
					}
				}
			}
		})
	}
}
