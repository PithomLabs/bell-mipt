// Package bellmipt implements BELL-MIPT-0001: a one-shot toy check that computes
// Bell jump rates for a finite Kitaev-chain fermion lattice and numerically
// verifies equivariance.
//
// This is a toy analysis only. No MIPT claim, no holography claim,
// no Bell-jumps-equal-measurements claim.
package bellmipt

import (
	"math"
	"math/cmplx"
)

// Matrix is a dense complex matrix stored in row-major order.
// Convention: H[n][m] = ⟨n|H|m⟩, so row = destination state, col = source state.
type Matrix struct {
	Dim  int
	Data []complex128 // length Dim*Dim, row-major
}

// NewMatrix creates a zero-initialized dim×dim complex matrix.
func NewMatrix(dim int) Matrix {
	return Matrix{
		Dim:  dim,
		Data: make([]complex128, dim*dim),
	}
}

// At returns the element at (row, col).
func (m Matrix) At(row, col int) complex128 {
	return m.Data[row*m.Dim+col]
}

// Add adds v to the element at (row, col).
func (m Matrix) Add(row, col int, v complex128) {
	m.Data[row*m.Dim+col] += v
}

// Set sets the element at (row, col) to v.
func (m Matrix) Set(row, col int, v complex128) {
	m.Data[row*m.Dim+col] = v
}

// MatVec computes the matrix-vector product H*v and returns the result.
func (m Matrix) MatVec(v []complex128) []complex128 {
	result := make([]complex128, m.Dim)
	for i := 0; i < m.Dim; i++ {
		var sum complex128
		for j := 0; j < m.Dim; j++ {
			sum += m.Data[i*m.Dim+j] * v[j]
		}
		result[i] = sum
	}
	return result
}

// HermitianError returns the maximum |H[n,m] - conj(H[m,n])| over all n,m.
// For a Hermitian matrix, this should be zero (within floating-point tolerance).
func HermitianError(H Matrix) float64 {
	maxErr := 0.0
	for n := 0; n < H.Dim; n++ {
		for m := n; m < H.Dim; m++ {
			diff := H.At(n, m) - cmplx.Conj(H.At(m, n))
			err := cmplx.Abs(diff)
			if err > maxErr {
				maxErr = err
			}
		}
	}
	return maxErr
}

// BuildKitaevHamiltonian constructs the dense Hamiltonian matrix for a finite
// Kitaev chain in the occupation-number (Fock) basis.
//
// The Hamiltonian uses the convention from the prompt:
//
//	H = -μ Σ_i n_i  -  t Σ_<i,j> (c†_i c_j + c†_j c_i)  +  Δ Σ_<i,j> (c†_i c†_j + c_j c_i)
//
// Pairing sign convention: this toy uses the prompt-defined +Δ convention.
// This is a finite Bell-rate algebra/equivariance check, not a claim about
// a specific physical Kitaev-chain convention.
//
// For periodic boundary, the wrap bond (N-1, 0) is added using the same generic
// ApplyOps path. No extra boundary parity sign is applied — the Jordan-Wigner
// string handles it correctly in the full Fock space.
func BuildKitaevHamiltonian(cfg Config, basis Basis) (Matrix, error) {
	if cfg.Model != "finite_kitaev_chain" {
		return Matrix{}, &ValidationError{Field: "model", Message: "only finite_kitaev_chain is supported"}
	}

	dim := basis.Dim
	H := NewMatrix(dim)

	mu := cfg.Parameters.Mu
	t := cfg.Parameters.T
	delta := cfg.Parameters.Delta

	// Build list of bonds
	bonds := make([][2]int, 0, basis.Sites)
	for i := 0; i < basis.Sites-1; i++ {
		bonds = append(bonds, [2]int{i, i + 1})
	}
	if cfg.Boundary == "periodic" && basis.Sites > 2 {
		bonds = append(bonds, [2]int{basis.Sites - 1, 0})
	}

	// Iterate over all basis kets |m⟩
	for m := 0; m < dim; m++ {
		state := uint64(m)

		// 1. Chemical potential: -μ Σ_i n_i
		for i := 0; i < basis.Sites; i++ {
			if Occupied(state, i) {
				H.Add(m, m, complex(-mu, 0))
			}
		}

		// 2. For each bond (i, j)
		for _, bond := range bonds {
			bi, bj := bond[0], bond[1]

			// Hopping: -t * c†_i c_j
			res := ApplyOps(state, []FermionOp{{Create, bi}, {Annihilate, bj}})
			if res.OK {
				n := int(res.State)
				H.Add(n, m, complex(-t*float64(res.Sign), 0))
			}

			// Hopping: -t * c†_j c_i (Hermitian conjugate)
			res = ApplyOps(state, []FermionOp{{Create, bj}, {Annihilate, bi}})
			if res.OK {
				n := int(res.State)
				H.Add(n, m, complex(-t*float64(res.Sign), 0))
			}

			// Pair creation: +Δ * c†_i c†_j
			res = ApplyOps(state, []FermionOp{{Create, bi}, {Create, bj}})
			if res.OK {
				n := int(res.State)
				H.Add(n, m, complex(delta*float64(res.Sign), 0))
			}

			// Pair annihilation: +Δ * c_j c_i (Hermitian conjugate of c†_i c†_j)
			res = ApplyOps(state, []FermionOp{{Annihilate, bj}, {Annihilate, bi}})
			if res.OK {
				n := int(res.State)
				H.Add(n, m, complex(delta*float64(res.Sign), 0))
			}
		}
	}

	return H, nil
}

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// Norm computes the L2 norm of a complex vector: sqrt(Σ|ψ_i|²).
func Norm(psi []complex128) float64 {
	sum := 0.0
	for _, v := range psi {
		sum += real(v)*real(v) + imag(v)*imag(v)
	}
	return math.Sqrt(sum)
}

// Probabilities returns |ψ_i|² for each component.
func Probabilities(psi []complex128) []float64 {
	p := make([]float64, len(psi))
	for i, v := range psi {
		p[i] = real(v)*real(v) + imag(v)*imag(v)
	}
	return p
}
