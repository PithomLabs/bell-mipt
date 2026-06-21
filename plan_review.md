## prompt

evaluate each of the attached plans, answer their respective questions if any, then give your assessment on strengths of each plan, identify their strengths and synthesize to one cohesive plan, then identify which plan you would like to start as the base plan


## chatgpt

Yes — the plans are collectively strong enough to proceed. My recommendation is:

> **Use `plan_chatgpt.md` as the base plan, merge in Sonnet/Z52’s implementation discipline, Opus’s periodic-boundary correction, Kimi/Xiaomi’s minimal CLI clarity, and reject Qwen’s special periodic parity fix.**

The first toy should remain exactly:

```text
BELL-MIPT-0001:
Can we compute Bell rates and verify equivariance for a finite fermion model?
```

No MIPT, no holography, no conditional-wave-function bridge, no random trajectories.

## Direct answers to open questions

### 1. Go module path

Use the **existing repository module path** if this is being implemented inside an existing repo.

If this is a fresh standalone repo, use:

```go
module github.com/PithomLabs/bell-mipt
```

For a throwaway local scratch, `module bellmipt` is acceptable, but I would prefer the fully qualified PithomLabs path if this may become a real repo.

### 2. Dependencies: `gonum` or zero-dependency?

Use **zero external dependencies** for `BELL-MIPT-0001`.

The default system is tiny: 6 sites gives Hilbert dimension (2^6 = 64). Hand-rolled `complex128` matrix-vector math is simpler, auditable, and avoids dependency/API friction. Opus and Sonnet both explicitly favor standard-library-only implementation for this first ticket.  

### 3. Dense or sparse matrix?

Use **dense flat row-major slices** for v0:

```go
type Matrix struct {
    Dim  int
    Data []complex128
}
```

Not `[][]complex128`, because flat slices are simpler to serialize/test and avoid nested allocation overhead.

Use a hard guard:

```text
sites <= 10
```

Default remains 6. Dense is fine at 6 sites and still manageable at 10; 12 is already unnecessarily large for a first one-shot toy.

### 4. Periodic boundary Jordan-Wigner sign

Do **not** add a special extra parity sign for the periodic bond.

The safest rule is:

> Implement generic fermion operator composition and apply operators right-to-left. Then the wrap bond `(N-1, 0)` goes through the same operator machinery as every other bond.

Opus explicitly corrected this: in the full Fock-space basis, the `Create`/`Annihilate` Jordan-Wigner signs already handle the periodic wrap bond; no separate parity correction is needed. 

Qwen’s plan says to add an extra ((-1)^{\hat N}) factor for periodic terms. I would **not** use that in this v0 full-Fock implementation, because it risks double-counting the Jordan-Wigner boundary sign. 

### 5. Should rates be recomputed at every RK4 substage?

Yes. For the final cohesive plan, use **joint RK4 for ((\psi,\rho))** and recompute Bell currents/rates from the stage (\psi) at every RK4 substage.

Some plans freeze the rate matrix during a step; that is simpler and defensible, but this toy’s entire goal is an equivariance audit. The cleaner audit is to evolve the coupled pair together:

```text
stage ψ → compute J → compute rates → compute dρ
```

ChatGPT’s plan states this explicitly and uses Bell rates recomputed from (\psi) at each RK4 stage. 

## Evaluation of each plan

| Plan               | Assessment                                      | Strengths                                                                                                                                                                                                                                                                   | Weakness / repair                                                                                                                                                                                          |
| ------------------ | ----------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `plan_chatgpt.md`  | **Best base plan**                              | Most complete: scope lock, config, dense matrix, Bell currents/rates, joint RK4, audit rules, report schema, forbidden-language audit, non-goals, acceptance criteria, and coding-agent prompt. It clearly says passing the toy does **not** establish a Bell-MIPT bridge.  | Slightly long, but that is acceptable because it is implementation-ready. Use flat dense slices instead of `[][]` where possible.                                                                          |
| `plan_deepseek.md` | Solid technical checklist                       | Good step-by-step package plan; clear Bell current/rate formulas; good test list and performance notes.                                                                                                                                                                     | Some structure is less clean: operators appear under Hamiltonian/dynamics organization, and it allows simpler Euler/fixed-rate choices later in the plan. Use its tests, not its evolution simplification. |
| `plan_kimi.md`     | Good minimal orchestrator                       | Very practical CLI/run skeleton and report construction; strong test matrix; keeps the command non-bloated.                                                                                                                                                                 | Less detailed on subtle numerical/physics traps. Use its CLI skeleton, but not as the base physics plan.                                                                                                   |
| `plan_opus46.md`   | Best correction notes                           | Strong on the two open decisions: zero-dependency and no special periodic JW sign. It also clearly locks the toy scope and keeps the command single-purpose.                                                                                                                | Its “rates held fixed during RK4 substeps” choice is simpler but less ideal for an equivariance audit. Replace with joint RK4 stage recomputation.                                                         |
| `plan_qwen(1).md`  | Useful but contains a key physics/software trap | Good structure, clear joint RK4 idea, good test strategy.                                                                                                                                                                                                                   | Reject the extra periodic ((-1)^{\hat N}) fix and the over-aggressive sanitizer that strips/panics on terms like “MIPT” or “holography.” We need limitations to mention forbidden scope safely.            |
| `plan_sonnet46.md` | Best engineering discipline                     | Strongest on package containment, `run.go`, standard-library-only design, milestone gates, and explicitly naming implementation choices. It also warns that wrap-bond signs should be handled by generic operator composition, not special-cased shortcuts.                 | Chooses frozen-generator (\rho) evolution for simplicity. I would upgrade to joint RK4 stage recomputation for v0.                                                                                         |
| `plan_xiaomi.md`   | Best compact step order                         | Very readable build order and pass/fail/inconclusive logic; good short schema/report summary; strong test table.                                                                                                                                                            | Too thin for the tricky implementation details. Use it as the milestone checklist, not the full plan.                                                                                                      |
| `plan_z52.md`      | Best sparse/future-scaling design               | Strong report/audit structs, exit codes, and concise artifact design. It proposes sparse matrix structures that are better for later scaling.                                                                                                                               | Sparse is not needed for the first one-shot toy. Use its report structs and exit-code idea, but keep dense v0.                                                                                             |

## Synthesized cohesive plan

### Base

Start from `plan_chatgpt.md`.

It has the best complete acceptance criteria, EBP guardrails, report format, limitations, and final coding-agent prompt. It explicitly locks the boundary that this ticket only verifies finite Bell-rate algebra/equivariance and does not establish a bridge. 

### Merge these strengths

From **Sonnet**: use a single `internal/bellmipt` package with `run.go` orchestrating the toy, so `cmd/bellmipt/main.go` stays thin. Keep standard-library-only implementation. 

From **Opus**: preserve the periodic-boundary rule: no extra parity fix; generic operator composition handles it. 

From **Kimi**: keep the one-command skeleton and simple output flow: load config, build Hamiltonian, run audit, determine status, write `input.json`, `report.json`, and `report.md`. 

From **Xiaomi**: use the compact build order and test checklist. 

From **Z52**: adopt the explicit `Report`, `Checks`, `Metrics`, and `DebtStatus` structs; use exit codes `0 = passed`, `1 = failed`, `2 = inconclusive`. 

From **DeepSeek**: keep the anticommutation, Hermiticity, current antisymmetry, nonnegative-rate, deterministic-output, and integration tests. 

Reject or repair:

```text
- Do not add Qwen’s special periodic boundary parity fix.
- Do not strip all words like “MIPT” or “holography”; allow them only in limitation/non-goal text.
- Do not implement sparse matrix yet.
- Do not add random trajectories.
- Do not add agents.
- Do not add MIPT comparison.
```

## Final implementation plan to start from

```text
cmd/bellmipt/main.go
internal/bellmipt/
  config.go
  basis.go
  fermion.go
  hamiltonian.go
  bell.go
  evolve.go
  audit.go
  report.go
  forbidden.go
  run.go
  *_test.go
```

Run shape:

```bash
go run ./cmd/bellmipt --config bellmipt.json --out out/bellmipt-run
```

Default config:

```text
sites: 6
boundary: periodic
mu: 1.0
t: 1.0
delta: 0.5
dt: 0.001
steps: 1000
initial_state: random_normalized
seed: 12345
```

Core algorithm:

```text
1. Enumerate full Fock basis as uint64 bitstrings.
2. Implement Create/Annihilate with Jordan-Wigner signs.
3. Build dense finite Kitaev-chain Hamiltonian.
4. Check Hermiticity.
5. Generate deterministic normalized random ψ₀.
6. Set ρ₀ = |ψ₀|².
7. Evolve ψ and ρ together with RK4.
8. At every RK4 substage, recompute Bell current and Bell rates from stage ψ.
9. Audit:
   - norm preservation
   - rho sum preservation
   - rate nonnegativity
   - current antisymmetry
   - equivariance L1 error
   - NaN/Inf
   - forbidden promotion language
10. Write:
   - input.json
   - report.json
   - report.md
```

Pass condition:

```text
toy_goal_passed only if:
- Hamiltonian is Hermitian.
- ψ norm stays within tolerance.
- ρ stays valid within tolerance.
- Bell rates are nonnegative.
- ρ(t) tracks |ψ(t)|² within equivariance tolerance.
- No NaN/Inf.
- No forbidden promotion language.
```

Report must include:

```json
{
  "toy_analysis_only": true,
  "physics_claim": "none",
  "needMap": "unpaid",
  "needInvariant": "partially_paid_equivariance_only",
  "needToyCheck": "partially_paid_rate_algebra_only",
  "needNullModel": "unpaid",
  "needObstruction": "bell_jumps_are_not_measurements",
  "needFaithfulnessReview": "unpaid"
}
```

## My base-plan choice

Use **`plan_chatgpt.md` as the base plan**.

Why: it is the most complete and most aligned with EBP/PTW discipline. It has the clearest non-goals, report requirements, acceptance criteria, and implementation-agent prompt. Then patch it with:

```text
Sonnet discipline:
  single internal/bellmipt package + run.go + milestone tests

Opus correction:
  no extra periodic-boundary parity sign

Z52 structs:
  Report / Checks / Metrics / DebtStatus + exit code convention

Kimi/Xiaomi clarity:
  minimal CLI and build-order checklist
```

That gives us a cohesive, one-shot, non-bloated implementation plan ready to hand to a coding agent.


