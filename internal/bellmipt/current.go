package bellmipt

// BellCurrent computes the Bell probability current matrix J.
//
// For configurations n and m:
//
//	J[n][m] = 2 * Im(conj(ψ[n]) * H[n,m] * ψ[m])
//
// Properties:
//   - J[n][m] = -J[m][n] (antisymmetric, when H is Hermitian)
//   - J[n][n] = 0 (diagonal is zero for Hermitian H with real diagonal)
//
// Returns a dim×dim slice of float64.
func BellCurrent(H Matrix, psi []complex128) [][]float64 {
	dim := H.Dim
	J := make([][]float64, dim)
	for i := range J {
		J[i] = make([]float64, dim)
	}

	for n := 0; n < dim; n++ {
		for m := 0; m < dim; m++ {
			// J_nm = 2 * Im(conj(ψ_n) * H_nm * ψ_m)
			psiNConj := complex(real(psi[n]), -imag(psi[n]))
			product := psiNConj * H.At(n, m) * psi[m]
			J[n][m] = 2 * imag(product)
		}
	}

	return J
}

// CurrentAntisymmetryError returns the maximum |J[n][m] + J[m][n]| over all n,m.
// For a Hermitian Hamiltonian, this should be zero within floating-point tolerance.
func CurrentAntisymmetryError(J [][]float64) float64 {
	dim := len(J)
	maxErr := 0.0
	for n := 0; n < dim; n++ {
		for m := n + 1; m < dim; m++ {
			err := abs64(J[n][m] + J[m][n])
			if err > maxErr {
				maxErr = err
			}
		}
	}
	return maxErr
}

func abs64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
