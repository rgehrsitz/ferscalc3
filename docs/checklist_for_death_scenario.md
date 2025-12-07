Checklist
Identify financial components impacted by a spouse’s death.
Define calculation logic changes for each (SS, FERS, TSP, taxes, FEHB, withdrawals).
Specify event timing + sequencing (what happens in death year vs following years).
Propose config schema extensions (YAML) to encode deterministic death dates or ages.
Cover survivor Social Security & pension rules (simplified vs detailed options).
Address tax filing status change & thresholds.
Include TSP inheritance mechanics and options.
Note FEHB continuation assumptions.
Provide edge cases & phased implementation roadmap.
1. Impacted Components
Social Security
One benefit ceases; survivor receives the higher of (their own at current claim status) or deceased’s benefit (with COLAs to date).
Widow(er) early claiming reduction if survivor < FRA (simplified optional modeling).
FERS Pension
If annuitant dies: survivor annuity equals elected percentage (e.g. 50% or 25%) of unreduced base (or of reduced—decide explicitly).
Ongoing COLAs continue on survivor portion.
TSP / Investments
Deceased’s balance transfers to spouse (spousal beneficiary) without immediate taxation.
Future RMD / withdrawal strategy may change (optional: merge vs keep separate).
Taxes
Filing status changes from MFJ to Single starting the year after death (or configurable: same-year switch vs next-year).
Standard deduction and SS taxation thresholds adjust.
Social Security Taxation
Provisional income formula switches to single thresholds after status change.
FEHB
Survivor can generally continue coverage if eligibility conditions met.
Premium may shift from “self & family” to “self only” (configurable).
Cash Flow / Withdrawal Strategy
Target withdrawals may adjust (e.g., reduce spending need after one spouse dies).
Option to specify survivor spending adjustment factor.
COLA & Inflation
Continue normally; just applied to remaining streams.
Medicare / IRMAA
Single filer MAGI thresholds lower → potential higher Part B premiums (optional phase 2).
2. Event Modeling Concepts
Define a timeline with discrete “events”:

Retirement events
Social Security start
Death events Each yearly (or monthly if you later refine) projection checks for events at start-of-year; death effects apply immediately or mid-year using a rule (simplify: apply from next year, or allow pro‑rating if desired).
Modes:

Deterministic (explicit death date or age)
Life expectancy (assumed age; treat as deterministic)
Future: stochastic mortality (actuarial table in Monte Carlo).
3. Sequencing on Death
Mark employee as deceased.
Stop their earned income, future COLAs on their own benefit streams, cease their SS benefit.
Compute survivor pension: survivor_annuity = base_pension * elected_survivor_percent (base vs reduced—store both explicitly to avoid ambiguity).
Merge or transfer TSP balance per strategy.
Recompute annual withdrawal plan (option: apply survivor_spending_factor to planned withdrawals).
Update filing status effective next tax year (or same year if config says).
Recalculate tax liabilities using new status and income mix.
Recalculate SS taxation using single thresholds if filing status switched.
Adjust FEHB premiums (switch plan type or keep same as config).
Continue projections with new state.
4. Config Schema Extensions (Proposed)
Add to RetirementScenario or better: introduce a new Events or Mortality block at scenario level:

If you prefer embedding into each person’s scenario:

Employee-level (optional if constant across scenarios):

Clarify FERS survivor election detail (split existing field):

(Deprecate or reinterpret current survivor_benefit_election_percent to avoid confusion.)

5. Domain Model Additions (Conceptual)
Structures to add:

Add to Scenario:

6. Social Security Survivor Logic (Simplified)
After death:

survivorAnnualSS = max( survivor_own_current, deceased_current )
Apply early survivor reduction if survivor < FRA: simplified factor = linear from 71.5% at age 60 to 100% at FRA
Replace combined two-benefit sum with survivorAnnualSS.
7. FERS Survivor Logic
Store base pension (before election reduction).
Retiree receives: base * (1 - election_reduction_percent)
Survivor receives after death: base * survivor_benefit_percent_of_base
Continue COLAs on survivor amount.
8. TSP Transfer Logic
If merge:

On death event: survivorBalance += deceasedBalance (no tax event)
Future withdrawals computed on merged balance. If separate:
Track inherited balance with possibly different withdrawal rules (phase 2).
9. Tax Filing Status Change
Add a status variable in projection state. Switch year determined by config. Use MFJ thresholds before switch, single after. Update standard deduction and SS provisional thresholds.

10. FEHB Handling
If deceased was policyholder:

If action = switch_to_self_only: reduce annual FEHB cost using single premium assumption (need param or derive: maybe 0.55 of family premium).
If terminate: remove FEHB cost (risky, maybe not realistic). Add config field for single_premium_ratio if you want explicit.
11. Edge Cases
Death before retirement date (handle: no pension yet; maybe deferred; immediate survivor benefit not applicable).
Death before claiming SS (survivor may claim survivor benefit earlier than their own—future enhancement).
Both deaths within projection (stop all flows).
Death occurs mid-year (simplify: effects apply next projection year unless “immediate” mode).
Missing both death_date and death_age → assume no death within horizon.
12. Implementation Phases
Phase 1 (Deterministic):

Add config parsing & validation for mortality.
Inject death check into annual loop; adjust SS, pension, filing status, FEHB, withdrawals.
Add survivor spending factor.
Phase 2:

Refine survivor SS reductions accurately.
Add FERS 25% vs 50% canonical options with correct reduction amounts (10% / 5%).
FEHB premium differentiation self-only vs family.
Phase 3:

Monte Carlo mortality (life table per person; random death year).
Separate inherited TSP account & different RMD logic.
Phase 4:

Medicare IRMAA recalculation post-filing status change.
Estate / beneficiary tax nuances (outside TSP).
13. Validation & Tests (Plan)
Add tests:

death_event_stops_second_ss_and_keeps_higher.go
survivor_pension_correct_percentage.go
tax_status_switch_reduces_std_deduction.go
tsp_merge_adds_balances.go
survivor_spending_factor_applied.go
14. Minimal YAML Example Snippet
15. Suggested Renames / Clarifications
Current SurvivorBenefitElectionPercent is ambiguous (is it payment percent or reduction?). Replace with two explicit fields to avoid miscalculation.

16. Roadmap Order
Data model + config parse/validate.
Core projection engine: inject mortality state.
Adjust SS + pension + TSP + tax status.
Add tests.
Enhance FEHB and survivor reductions.