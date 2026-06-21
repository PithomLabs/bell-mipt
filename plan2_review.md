Verdict: **approve with minor repairs before coding.** The plan is now coherent, one-shot, and aligned with the clarified decisions: Go module path, zero deps, joint RK4 only, library-level integration testing, prompt-defined `+Δ`, no extra periodic-boundary parity sign, and `sites <= 10` are all locked in. 

## What is strong

The plan preserves the correct scope boundary: **toy analysis only**, with `physics_claim: "none"` and `toy_analysis_only: true`. It explicitly excludes MIPT, holography, Bell-jumps-equal-measurements, and conditional-wave-function bridge claims. 

The implementation structure is good: a thin `cmd/bellmipt/main.go`, all real logic in `internal/bellmipt`, and a single `Run()` orchestrator. That keeps the CLI non-bloated and makes testing straightforward. 

The physics-sign choices are now correct for this v0: `+Δ` follows the prompt, and the periodic boundary uses the same generic `ApplyOps` path with **no extra parity patch**. That avoids the most dangerous double-counting bug. 

The evolution choice is also correct: joint RK4 for ((\psi,\rho)), with rates recomputed at every RK4 sub-stage. That is the cleanest choice for an equivariance audit. 

## Required minor repairs

### 1. Fix the forbidden-language scanner

Right now the plan says the forbidden phrases include things like:

```text
physics promotion
Bell jumps are measurements
```

But the required limitations include:

```text
This is not a physics promotion.
This does not show Bell jumps are measurements.
```

That can create false positives.

Repair it this way:

```text
The forbidden scanner should catch promotional positive claims, not negated limitation statements.
```

Use exact bad phrases such as:

```text
proves the Bell-MIPT bridge
establishes MIPT
confirms holography
Bell jumps equal measurements
Bell jumps are measurements
explains black holes
explains holography
```

But explicitly allow negated scope sentences:

```text
This is not a physics promotion.
This does not show Bell jumps are measurements.
No MIPT claim.
No holography claim.
No physics promotion.
```

Do **not** ban raw words like `MIPT`, `holography`, or `measurement` globally, because the report must mention them in limitations.

### 2. Define “significant probability-floor hits”

The plan says significant probability-floor hits should cause `toy_goal_inconclusive`, but it does not define “significant.” Add a precise v0 rule.

Simplest acceptable rule:

```text
If ProbabilityFloorHits > 0, mark toy_goal_inconclusive.
```

Better rule:

```text
Track probability_floor_current_mass = Σ positive current from floored source states.
If ProbabilityFloorHits > 0 and probability_floor_current_mass > 1e-12, mark inconclusive.
Otherwise record the hits but do not fail.
```

For v0, I would use the better rule if easy; otherwise use the simple rule and document it.

### 3. Add `RhoNegativeTolerance`

The plan mentions “no serious rho negativity” but does not give a separate tolerance. Do not overload `norm_tolerance` silently.

Add either:

```go
RhoNegativeTolerance float64 `json:"rho_negative_tolerance,omitempty"`
```

or define a constant:

```go
const RhoNegativeTolerance = 1e-10
```

For v0, a constant is fine. Then:

```text
max_rho_negative_violation <= RhoNegativeTolerance passes.
above that fails or becomes inconclusive depending on severity.
```

I would classify severe negativity as inconclusive only if it makes the probability distribution unusable; otherwise failed.

### 4. Add explicit current-antisymmetry tolerance

The report tracks `MaxCurrentAntisymmetryError`, but config has no tolerance for it. Use:

```text
current antisymmetry tolerance = max(hermitian_tolerance, 1e-12)
```

Document that in `audit.go`.

### 5. Make `RunResult.Error` mean internal execution error only

Do not return `Error` for `toy_goal_failed` or `toy_goal_inconclusive`.

Use:

```text
RunResult.Error != nil only for:
- invalid config
- file I/O failure
- impossible internal state
```

A completed scientific/audit failure should still return a report with `goal_status = toy_goal_failed`.

### 6. Add `go vet ./...` to verification

The plan includes `go test` and `go test -race`. Add:

```bash
go vet ./...
```

before declaring implementation complete.

## Approval status

```json
{
  "review_status": "accept_with_minor_repairs",
  "base_plan": "plan2.md",
  "critical_blockers": [],
  "minor_repairs_required": [
    "repair forbidden-language scanner to allow negated limitation statements",
    "define significant probability-floor hit rule",
    "define rho negativity tolerance",
    "define current antisymmetry tolerance",
    "ensure RunResult.Error is reserved for internal/config/I/O errors",
    "add go vet ./... to validation"
  ],
  "promotion_status": "unpromoted_implementation_plan_only"
}
```

## EBP status

| Item                     | Status                                           |
| ------------------------ | ------------------------------------------------ |
| `needMap`                | unpaid                                           |
| `needInvariant`          | planned partial payment: equivariance audit only |
| `needToyCheck`           | ready to implement                               |
| `needNullModel`          | unpaid                                           |
| `needObstruction`        | `bell_jumps_are_not_measurements` preserved      |
| `needFaithfulnessReview` | unpaid                                           |
| Promotion                | none                                             |

You can proceed to implementation after those small text/code-contract repairs.

