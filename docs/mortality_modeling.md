# Mortality Modeling (Phase 1)

Phase 1 introduces deterministic death events and basic survivor income adjustments.

## Configuration Schema Additions

Within each scenario you may add an optional `mortality` block:

```yaml
scenarios:
  - name: "Mortality Shock: Person A dies 2034"
  person_a: { ... }
  person_b: { ... }
    mortality:
  person_a:
        death_date: "2034-06-30T00:00:00Z"  # OR use death_age: 79
  person_b:
        # optional
      assumptions:
        survivor_spending_factor: "0.90"    # Scale pensions & withdrawals post-first death (0.4-1.0 allowed)
        tsp_spousal_transfer: "merge"       # merge | separate (Phase 1 implements merge only)
        filing_status_switch: "next_year"   # next_year | immediate (tax impact not yet implemented in Phase 1)
```

You can specify either `death_date` (UTC timestamp) or `death_age` (integer) for each person, but not both.

## Modeling Behavior (Phase 1)

- At the start of the projection year at/after the death event, that person is marked deceased.
- Their FERS pension, FERS supplement, salary, and own Social Security cease.
- Survivor receives the higher of the two Social Security annual benefits (simple survivor rule).
- If `tsp_spousal_transfer: merge`, deceased TSP (traditional & Roth) balances are added to survivor balances at the first year of death; deceased balances reset to zero.
- `survivor_spending_factor` scales (multiplies) remaining pensions and both TSP withdrawals from the year of death onward (simplified proxy for reduced household spending).
- Filing status switch flag is stored but not yet applied to tax brackets in Phase 1 (future phase will alter standard deduction and SS taxation thresholds).

## Limitations / Roadmap

| Aspect | Current | Future Enhancement |
| ------ | ------- | ------------------ |
| Survivor pension | Not yet differentiated (pension simply stops) | Model elected survivor % and reduction factors |
| SS survivor reduction (age < FRA) | Not modeled | Apply widow(er) reduction schedule |
| Filing status change | Flag only | Adjust federal standard deduction & SS provisional thresholds |
| Separate inherited TSP account | Not modeled | Track inherited account, apply distinct withdrawal rules/RMDs |
| Multiple sequential deaths | Stops after first death event | Support both deaths with final projection termination |
| Mid-year proration | Year-level (start-of-year) | Month-level pro-rating if required |

## Validation

Golden snapshots updated to reflect structural changes. Add targeted mortality unit tests in `internal/calculation` or `internal/output` (planned next).

## Example Snippet (Age-Based Death)

```yaml
mortality:
  person_b:
    death_age: 88
  assumptions:
    survivor_spending_factor: "0.85"
    tsp_spousal_transfer: "merge"
```

## Backwards Compatibility

Scenarios without a `mortality` block behave exactly as before.

## Next Phases

1. Implement filing status transition affecting tax calculation.
2. Add survivor pension logic with explicit elected base reduction vs survivor %.
3. Accurate Social Security survivor benefit reductions (early claiming formula).
4. Optional stochastic mortality (Monte Carlo) using actuarial tables.
