package bellmipt

// OpKind distinguishes creation and annihilation operators.
type OpKind int

const (
	Create     OpKind = iota // c†
	Annihilate               // c
)

// FermionOp represents a single fermionic creation or annihilation operator
// acting on a specific site.
type FermionOp struct {
	Kind OpKind
	Site int
}

// OpResult is the result of applying one or more fermion operators to a Fock state.
// If OK is false, the operator annihilated the state (e.g., annihilating an
// unoccupied site or creating on an already occupied site).
type OpResult struct {
	OK    bool
	State uint64
	Sign  int // +1 or -1
}

// ApplyAnnihilate applies the annihilation operator c_site to the given Fock state.
//
// Jordan-Wigner convention: right-to-left application order.
// sign = (-1)^(number of occupied sites below 'site').
//
// If the site is unoccupied, returns {OK: false}.
func ApplyAnnihilate(state uint64, site int) OpResult {
	if !Occupied(state, site) {
		return OpResult{OK: false}
	}
	below := CountBelow(state, site)
	sign := 1
	if below%2 != 0 {
		sign = -1
	}
	newState := state ^ (1 << uint(site)) // clear bit
	return OpResult{OK: true, State: newState, Sign: sign}
}

// ApplyCreate applies the creation operator c†_site to the given Fock state.
//
// Jordan-Wigner convention: right-to-left application order.
// sign = (-1)^(number of occupied sites below 'site').
//
// If the site is already occupied, returns {OK: false}.
func ApplyCreate(state uint64, site int) OpResult {
	if Occupied(state, site) {
		return OpResult{OK: false}
	}
	below := CountBelow(state, site)
	sign := 1
	if below%2 != 0 {
		sign = -1
	}
	newState := state | (1 << uint(site)) // set bit
	return OpResult{OK: true, State: newState, Sign: sign}
}

// ApplyOps applies a sequence of fermion operators to a Fock state.
//
// The ops slice is given in PHYSICS NOTATION ORDER (left-to-right, e.g.,
// c†_i c_j means [{Create, i}, {Annihilate, j}]) but operators are
// EXECUTED RIGHT-TO-LEFT, so the last operator in the slice acts first.
//
// Signs compose multiplicatively. If any intermediate step kills the state
// (OK == false), the entire result is {OK: false}.
func ApplyOps(state uint64, ops []FermionOp) OpResult {
	currentState := state
	totalSign := 1

	// Execute right-to-left: iterate from the last operator to the first.
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		var res OpResult
		switch op.Kind {
		case Create:
			res = ApplyCreate(currentState, op.Site)
		case Annihilate:
			res = ApplyAnnihilate(currentState, op.Site)
		default:
			return OpResult{OK: false}
		}
		if !res.OK {
			return OpResult{OK: false}
		}
		currentState = res.State
		totalSign *= res.Sign
	}

	return OpResult{OK: true, State: currentState, Sign: totalSign}
}
