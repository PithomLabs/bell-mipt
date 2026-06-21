package bellmipt

import (
	"math"
	"math/cmplx"
	"testing"
)

func TestHamiltonianDimension(t *testing.T) {
	basis, err := NewBasis(3)
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.Sites = 3
	cfg.Boundary = "open"
	H, err := BuildKitaevHamiltonian(cfg, basis)
	if err != nil {
		t.Fatal(err)
	}
	if H.Dim != 8 {
		t.Errorf("expected dim 8, got %d", H.Dim)
	}
	if len(H.Data) != 64 {
		t.Errorf("expected 64 elements, got %d", len(H.Data))
	}
}

func TestHamiltonianHermitianOpen(t *testing.T) {
	for _, sites := range []int{2, 3, 4, 5} {
		basis, err := NewBasis(sites)
		if err != nil {
			t.Fatal(err)
		}
		cfg := DefaultConfig()
		cfg.Sites = sites
		cfg.Boundary = "open"
		H, err := BuildKitaevHamiltonian(cfg, basis)
		if err != nil {
			t.Fatal(err)
		}
		herr := HermitianError(H)
		if herr > 1e-14 {
			t.Errorf("sites=%d open: Hermitian error %e exceeds tolerance", sites, herr)
		}
	}
}

func TestHamiltonianHermitianPeriodic(t *testing.T) {
	for _, sites := range []int{3, 4, 5, 6} {
		basis, err := NewBasis(sites)
		if err != nil {
			t.Fatal(err)
		}
		cfg := DefaultConfig()
		cfg.Sites = sites
		cfg.Boundary = "periodic"
		H, err := BuildKitaevHamiltonian(cfg, basis)
		if err != nil {
			t.Fatal(err)
		}
		herr := HermitianError(H)
		if herr > 1e-14 {
			t.Errorf("sites=%d periodic: Hermitian error %e exceeds tolerance", sites, herr)
		}
	}
}

func TestHamiltonianDiagonalWhenNoHoppingNoPairing(t *testing.T) {
	basis, err := NewBasis(3)
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.Sites = 3
	cfg.Boundary = "open"
	cfg.Parameters.T = 0
	cfg.Parameters.Delta = 0
	cfg.Parameters.Mu = 2.0
	H, err := BuildKitaevHamiltonian(cfg, basis)
	if err != nil {
		t.Fatal(err)
	}

	for n := 0; n < H.Dim; n++ {
		for m := 0; m < H.Dim; m++ {
			val := H.At(n, m)
			if n != m {
				if cmplx.Abs(val) > 1e-15 {
					t.Errorf("off-diagonal H[%d][%d] = %v, expected 0", n, m, val)
				}
			} else {
				// Diagonal should be -mu * number_of_occupied_sites
				nOcc := PopCount(uint64(n))
				expected := -2.0 * float64(nOcc)
				if math.Abs(real(val)-expected) > 1e-15 || math.Abs(imag(val)) > 1e-15 {
					t.Errorf("H[%d][%d] = %v, expected %f", n, m, val, expected)
				}
			}
		}
	}
}

func TestHamiltonianTwoSiteHandCalculation(t *testing.T) {
	// 2-site Kitaev chain, open boundary
	// Basis states: |00>=0, |01>=1, |10>=2, |11>=3
	// H = -μ(n_0 + n_1) - t(c†_0 c_1 + c†_1 c_0) + Δ(c†_0 c†_1 + c_1 c_0)
	//
	// Chemical potential contributions:
	//   |01>: -μ (site 0 occupied)
	//   |10>: -μ (site 1 occupied)
	//   |11>: -2μ (both occupied)
	//
	// Hopping c†_0 c_1: acts on states with site 1 occupied, site 0 empty
	//   c†_0 c_1 |10> = c†_0 |00> = |01> with sign: c_1 on |10> has CountBelow(10,1)=0 -> sign +1
	//                                                  c†_0 on |00> has CountBelow(00,0)=0 -> sign +1
	//                                                  total sign = +1
	//   So H[1][2] += -t * 1 = -t
	//
	// Hopping c†_1 c_0: acts on states with site 0 occupied, site 1 empty
	//   c†_1 c_0 |01> = c†_1 |00> = |10> with sign: c_0 on |01> has CountBelow(01,0)=0 -> sign +1
	//                                                  c†_1 on |00> has CountBelow(00,1)=0 -> sign +1
	//                                                  total sign = +1
	//   So H[2][1] += -t * 1 = -t
	//
	// Pair creation c†_0 c†_1: acts on |00>
	//   c†_1 on |00>: CountBelow(00,1)=0 -> sign +1, state -> |10>
	//   c†_0 on |10>: CountBelow(10,0)=0 -> sign +1, state -> |11>
	//   total sign = +1
	//   So H[3][0] += Δ * 1 = Δ
	//
	// Pair annihilation c_1 c_0: acts on |11>
	//   c_0 on |11>: CountBelow(11,0)=0 -> sign +1, state -> |10>
	//   c_1 on |10>: CountBelow(10,1)=0 -> sign +1, state -> |00>
	//   total sign = +1
	//   So H[0][3] += Δ * 1 = Δ

	basis, err := NewBasis(2)
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.Sites = 2
	cfg.Boundary = "open"
	cfg.Parameters.Mu = 1.0
	cfg.Parameters.T = 1.0
	cfg.Parameters.Delta = 0.5

	H, err := BuildKitaevHamiltonian(cfg, basis)
	if err != nil {
		t.Fatal(err)
	}

	// Expected 4x4 matrix:
	//        |00>  |01>  |10>  |11>
	// <00|   0     0     0     Δ
	// <01|   0    -μ    -t     0
	// <10|   0    -t    -μ     0
	// <11|   Δ     0     0    -2μ
	mu := 1.0
	tHop := 1.0
	delta := 0.5

	expected := [4][4]complex128{
		{0, 0, 0, complex(delta, 0)},
		{0, complex(-mu, 0), complex(-tHop, 0), 0},
		{0, complex(-tHop, 0), complex(-mu, 0), 0},
		{complex(delta, 0), 0, 0, complex(-2*mu, 0)},
	}

	for n := 0; n < 4; n++ {
		for m := 0; m < 4; m++ {
			got := H.At(n, m)
			exp := expected[n][m]
			diff := cmplx.Abs(got - exp)
			if diff > 1e-14 {
				t.Errorf("H[%d][%d] = %v, expected %v (diff=%e)", n, m, got, exp, diff)
			}
		}
	}
}

func TestPeriodicBoundaryUsesGenericJWNoSpecialParity(t *testing.T) {
	// Verify that the periodic boundary Hamiltonian is Hermitian
	// and has different entries than the open boundary version,
	// confirming the wrap bond is included.
	basis, err := NewBasis(4)
	if err != nil {
		t.Fatal(err)
	}

	cfgOpen := DefaultConfig()
	cfgOpen.Sites = 4
	cfgOpen.Boundary = "open"
	HOpen, err := BuildKitaevHamiltonian(cfgOpen, basis)
	if err != nil {
		t.Fatal(err)
	}

	cfgPeriodic := DefaultConfig()
	cfgPeriodic.Sites = 4
	cfgPeriodic.Boundary = "periodic"
	HPeriodic, err := BuildKitaevHamiltonian(cfgPeriodic, basis)
	if err != nil {
		t.Fatal(err)
	}

	// Periodic should be different from open
	different := false
	for i := range HOpen.Data {
		if cmplx.Abs(HOpen.Data[i]-HPeriodic.Data[i]) > 1e-15 {
			different = true
			break
		}
	}
	if !different {
		t.Error("periodic and open Hamiltonians are identical; wrap bond not included")
	}

	// Both should be Hermitian
	herrOpen := HermitianError(HOpen)
	herrPeriodic := HermitianError(HPeriodic)
	if herrOpen > 1e-14 {
		t.Errorf("open Hermitian error: %e", herrOpen)
	}
	if herrPeriodic > 1e-14 {
		t.Errorf("periodic Hermitian error: %e", herrPeriodic)
	}
}

func TestUnsupportedModel(t *testing.T) {
	basis, _ := NewBasis(2)
	cfg := DefaultConfig()
	cfg.Sites = 2
	cfg.Model = "hubbard"
	_, err := BuildKitaevHamiltonian(cfg, basis)
	if err == nil {
		t.Error("expected error for unsupported model")
	}
}
