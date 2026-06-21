## prompt

answer clarification below:
Evolution coupling: The prompt specifies joint RK4 with rate recomputation at each RK4 sub-stage (§11, line 642). This means 4 full Bell-current + rate-matrix computations per time step (one per RK4 stage). For a 64×64 system this is cheap, but I want to confirm: are you okay with this being the only supported strategy, or do you also want a simpler "rates-at-start-of-step" mode as a fallback/comparison?
1) Joint RK4 with rate recomputation at each stage only (as specified in the prompt)
2) Also add an optional simpler "rates-at-start" mode behind a config flag for comparison
3) other, give your answer



answer:

Choose **1) Joint RK4 with rate recomputation at each stage only** for `BELL-MIPT-0001`.

Reason: this first toy is an **equivariance audit**, not a performance benchmark. The cleanest implementation is the one where (\psi) and (\rho) are evolved as one coupled system, and the Bell rates are recomputed from the stage wavefunction at every RK4 sub-stage.

Do **not** add a `rates-at-start` mode in v0. It adds config complexity, doubles the audit surface, and creates two possible interpretations of the report. For a 64×64 default system, the extra cost is negligible.

Coding-agent answer:

```text
Use joint RK4 with Bell-current and rate recomputation at every RK4 sub-stage as the only supported v0 strategy.

Do not add a rates-at-start fallback/comparison mode in BELL-MIPT-0001.

Rationale:
- BELL-MIPT-0001 is an equivariance audit, so correctness/clarity matters more than speed.
- Rates depend on ψ(t), so stage-level recomputation is the cleanest coupled evolution.
- The default 6-site system is small enough that 4 current/rate computations per step are cheap.
- A rates-at-start mode can be added later as BELL-MIPT-0001.1 if we want a numerical-method comparison, but it should not complicate the first one-shot toy.
```

EBP status: `needToyCheck = partially_paid_rate_algebra_only`; no MIPT/bridge claim.



## prompt

Exit codes: The prompt specifies exit 0 for passed, 1 for failed, 2 for inconclusive (§16, lines 1010-1012). go test typically uses exit 1 for test failures. Should the integration test (TestDefaultRunWritesInputReportJSONAndMarkdown) invoke bellmipt as a subprocess and check exit codes, or just call bellmipt.Run() as a library function and check the returned report?



1) Call bellmipt.Run() as a library function (simpler, faster tests)
2) Invoke as subprocess and check exit codes (tests the full binary, but slower)
3) give answer


answer:

Choose **1) Call `bellmipt.Run()` as a library function** for `TestDefaultRunWritesInputReportJSONAndMarkdown`.

Reason: the core scientific/audit correctness should be tested at the library level. It is faster, deterministic, easier to debug, and avoids `go test` confusion around subprocess exit behavior.

Use this structure:

```text
Primary integration test:
  Call bellmipt.Run(cfg, outDir)
  Assert:
    input.json exists
    report.json exists
    report.md exists
    report.goal_status is valid
    report.toy_analysis_only == true
    report.physics_claim == "none"
    required debt fields exist
    no forbidden promotion language
```

For exit codes, do **not** make the main default integration test depend on subprocess execution. Instead, factor the exit-code mapping into a small pure function:

```go
func ExitCodeForGoalStatus(status string) int {
	switch status {
	case "toy_goal_passed":
		return 0
	case "toy_goal_failed":
		return 1
	case "toy_goal_inconclusive":
		return 2
	default:
		return 1
	}
}
```

Then test that function directly.

Coding-agent answer:

```text
Use option 1.

Implement the integration test by calling bellmipt.Run() directly, not by invoking the binary as a subprocess.

Also add a small unit test for ExitCodeForGoalStatus so the CLI mapping is covered without slow subprocess tests.

Do not add subprocess-based tests in BELL-MIPT-0001 unless there is already an established repo convention for CLI smoke tests.
```

This keeps v0 focused: **library-level toy audit correctness first, CLI exit behavior as a thin tested mapping.**


## prompt

Pairing term sign convention: The Hamiltonian in §7 uses +Δ Σ (c†_i c†_j + c_j c_i). Some references use -Δ for the pairing term. The prompt is explicit about +Δ. Should I follow the prompt exactly, or do you have a preference for the conventional Kitaev chain sign -Δ?

1) Use +Δ exactly as written in the prompt
2) Use -Δ (standard Kitaev chain convention) and note the deviation

answer:

Choose **1) Use `+Δ` exactly as written in the prompt**.

For `BELL-MIPT-0001`, the sign convention is not the physics target. The target is whether the implementation correctly builds a Hermitian finite fermion Hamiltonian, computes Bell currents/rates, and verifies equivariance. Changing the pairing sign just to match one common Kitaev convention adds unnecessary ambiguity.

Coding-agent answer:

```text
Use +Δ exactly as specified:

H =
  - μ Σ_i n_i
  - t Σ_<i,j> (c†_i c_j + c†_j c_i)
  + Δ Σ_<i,j> (c†_i c†_j + c_j c_i)

Do not switch to -Δ in BELL-MIPT-0001.

Rationale:
- The prompt explicitly fixes +Δ.
- Equivariance should hold for any Hermitian Hamiltonian, so the sign convention is not the key audit target.
- Keeping the prompt convention avoids hidden deviations between plan, code, and report.
- Later tickets can add a convention field if we need to compare against standard Kitaev-chain literature.
```

Add this to the code comment/report:

```text
Pairing sign convention: this toy uses the prompt-defined +Δ convention. This is a finite Bell-rate algebra/equivariance check, not a claim about a specific physical Kitaev-chain convention.
```






