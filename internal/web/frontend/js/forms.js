/**
 * forms.js — Form rendering, dynamic show/hide, config assembly
 * Global namespace: FERSCalc.Forms
 */
window.FERSCalc = window.FERSCalc || {};

FERSCalc.Forms = (function() {
  'use strict';

  let _scenarioCount = 0;

  /**
   * Initialize the forms module
   */
  function init() {
    // Person B toggle
    const toggle = document.getElementById('include-person-b');
    if (toggle) {
      toggle.addEventListener('change', () => {
        const col = document.getElementById('person-b-column');
        if (col) col.classList.toggle('hidden', !toggle.checked);
        // Also update scenario cards
        updateScenarioPersonBVisibility();
      });
    }

    // Add first scenario card
    if (_scenarioCount === 0) {
      addScenario();
    }
  }

  /**
   * Add a new scenario card
   */
  function addScenario() {
    const container = document.getElementById('scenarios-container');
    if (!container) return;

    const idx = _scenarioCount++;
    const hasPersonB = document.getElementById('include-person-b')?.checked;
    const html = renderScenarioCard(idx, hasPersonB);
    container.insertAdjacentHTML('beforeend', html);

    // Wire up strategy change listeners
    wireStrategyListeners(idx, 'pa');
    if (hasPersonB) {
      wireStrategyListeners(idx, 'pb');
    }
  }

  /**
   * Remove a scenario card
   */
  function removeScenario(idx) {
    const card = document.getElementById('scenario-card-' + idx);
    if (card) card.remove();
  }

  /**
   * Render a scenario card HTML
   */
  function renderScenarioCard(idx, hasPersonB) {
    const prefix = 'sc-' + idx;
    const strategies = FERSCalc.API.getMetadata()?.strategies || [
      '4_percent_rule', 'need_based', 'variable_percentage', 'fixed_annuity', 'floor_ceiling'
    ];

    const strategyOptions = strategies.map(s => {
      const labels = {
        '4_percent_rule': '4% Rule',
        'need_based': 'Need-Based',
        'variable_percentage': 'Variable Percentage',
        'fixed_annuity': 'Fixed Annuity',
        'floor_ceiling': 'Floor/Ceiling',
      };
      return `<option value="${s}">${labels[s] || s}</option>`;
    }).join('');

    const renderPersonSection = (person, personLabel) => {
      const sp = prefix + '-' + person;
      return `
        <div class="scenario-person-section" id="${sp}-section">
          <h4>${personLabel}</h4>
          <div class="form-group">
            <label>Retirement Date</label>
            <input type="date" id="${sp}-retirement-date">
          </div>
          <div class="form-group">
            <label>SS Start Age <span class="hint">(62-70)</span></label>
            <input type="number" id="${sp}-ss-age" min="62" max="70" value="67">
          </div>
          <div class="form-group">
            <label>TSP Withdrawal Strategy</label>
            <select id="${sp}-strategy" onchange="FERSCalc.Forms.onStrategyChange('${sp}')">
              ${strategyOptions}
            </select>
          </div>
          <div class="form-group">
            <label>TSP Withdrawal Ordering</label>
            <select id="${sp}-ordering">
              <option value="">Default (Traditional First)</option>
              <option value="traditional_first">Traditional First</option>
              <option value="roth_first">Roth First</option>
            </select>
          </div>

          <!-- Strategy-dependent fields -->
          <div id="${sp}-fields-need_based" class="strategy-fields hidden">
            <div class="form-group">
              <label>Monthly Target <span class="hint">($)</span></label>
              <input type="number" id="${sp}-target-monthly" step="100" min="0">
            </div>
          </div>
          <div id="${sp}-fields-variable_percentage" class="strategy-fields hidden">
            <div class="form-group">
              <label>Withdrawal Rate <span class="hint">(decimal, e.g. 0.04)</span></label>
              <input type="number" id="${sp}-withdrawal-rate" step="0.001" min="0" max="0.2">
            </div>
          </div>
          <div id="${sp}-fields-floor_ceiling" class="strategy-fields hidden">
            <div class="form-group">
              <label>Withdrawal Rate <span class="hint">(decimal)</span></label>
              <input type="number" id="${sp}-fc-withdrawal-rate" step="0.001" min="0" max="0.2">
            </div>
            <div class="form-row">
              <div class="form-group">
                <label>Floor <span class="hint">($/yr)</span></label>
                <input type="number" id="${sp}-floor" step="1000" min="0">
              </div>
              <div class="form-group">
                <label>Ceiling <span class="hint">($/yr)</span></label>
                <input type="number" id="${sp}-ceiling" step="1000" min="0">
              </div>
            </div>
          </div>
          <div id="${sp}-fields-fixed_annuity" class="strategy-fields hidden">
            <div class="form-group">
              <label>Payout Rate <span class="hint">(decimal, e.g. 0.055)</span></label>
              <input type="number" id="${sp}-annuity-payout-rate" step="0.001" min="0" max="0.2">
            </div>
            <div class="form-group">
              <label>Premium % of TSP <span class="hint">(0-1, e.g. 0.5 = 50%)</span></label>
              <input type="number" id="${sp}-annuity-premium" step="0.01" min="0" max="1" value="1.0">
            </div>
            <div class="form-row">
              <div class="form-group">
                <label>Annuity COLA <span class="hint">(decimal)</span></label>
                <input type="number" id="${sp}-annuity-cola" step="0.01" min="0" max="0.1" value="0.02">
              </div>
              <div class="form-group">
                <label>Guaranteed Years</label>
                <input type="number" id="${sp}-annuity-guaranteed" min="0" max="20" value="0">
              </div>
            </div>
          </div>
        </div>
      `;
    };

    return `
      <div class="scenario-card" id="scenario-card-${idx}">
        <div class="scenario-header">
          <input type="text" id="${prefix}-name" value="Scenario ${idx + 1}" placeholder="Scenario name">
          ${idx > 0 ? `<button class="btn btn-danger btn-sm" onclick="FERSCalc.Forms.removeScenario(${idx})">Remove</button>` : ''}
        </div>
        <div class="scenario-person-grid">
          ${renderPersonSection('pa', 'Person A')}
          ${hasPersonB ? renderPersonSection('pb', 'Person B') : ''}
        </div>
      </div>
    `;
  }

  /**
   * Handle strategy dropdown change
   */
  function onStrategyChange(sp) {
    const strategy = document.getElementById(sp + '-strategy')?.value;
    // Hide all strategy-specific fields
    const allFields = document.querySelectorAll(`[id^="${sp}-fields-"]`);
    allFields.forEach(el => el.classList.add('hidden'));

    // Show the relevant one
    if (strategy) {
      const target = document.getElementById(sp + '-fields-' + strategy);
      if (target) target.classList.remove('hidden');
    }
  }

  function wireStrategyListeners(idx, person) {
    const sp = 'sc-' + idx + '-' + person;
    // Trigger initial state
    onStrategyChange(sp);
  }

  function updateScenarioPersonBVisibility() {
    const hasPersonB = document.getElementById('include-person-b')?.checked;
    document.querySelectorAll('[id$="-pb-section"]').forEach(el => {
      el.style.display = hasPersonB ? '' : 'none';
    });
  }

  /**
   * Build the full Configuration JSON from form state
   * @returns {Object} — matches domain.Configuration struct
   */
  function buildConfig() {
    const hasPersonB = document.getElementById('include-person-b')?.checked;
    const v = (id) => document.getElementById(id)?.value || '';
    const n = (id) => document.getElementById(id)?.value || '0';

    // Build personal_details
    const personalDetails = {};
    personalDetails['person_a'] = buildEmployee('pa');
    if (hasPersonB) {
      personalDetails['person_b'] = buildEmployee('pb');
    }

    // Build global_assumptions
    const globalAssumptions = {
      inflation_rate: parseFloat(n('ga-inflation')) || 0.025,
      fehb_premium_inflation: parseFloat(n('ga-fehb-inflation')) || 0.06,
      tsp_return_pre_retirement: parseFloat(n('ga-tsp-pre')) || 0.07,
      tsp_return_post_retirement: parseFloat(n('ga-tsp-post')) || 0.05,
      cola_general_rate: parseFloat(n('ga-cola')) || 0.025,
      projection_years: parseInt(v('ga-years')) || 25,
      current_location: {
        state: v('ga-state') || 'PA',
        county: v('ga-county') || '',
        municipality: v('ga-municipality') || '',
      },
    };

    // Build scenarios
    const scenarios = [];
    document.querySelectorAll('.scenario-card').forEach((card) => {
      const idx = parseInt(card.id.replace('scenario-card-', ''));
      const prefix = 'sc-' + idx;

      const scenario = {
        name: v(prefix + '-name') || 'Scenario ' + (idx + 1),
        person_a: buildScenarioPerson(prefix + '-pa', 'person_a'),
      };

      if (hasPersonB) {
        scenario.person_b = buildScenarioPerson(prefix + '-pb', 'person_b');
      } else {
        // Empty person_b to match expected format
        scenario.person_b = {
          employee_name: '',
          retirement_date: '0001-01-01T00:00:00Z',
          ss_start_age: 0,
          tsp_withdrawal_strategy: '',
        };
      }

      scenarios.push(scenario);
    });

    return {
      personal_details: personalDetails,
      global_assumptions: globalAssumptions,
      scenarios: scenarios,
    };
  }

  function buildEmployee(prefix) {
    const v = (id) => document.getElementById(id)?.value || '';

    const employee = {
      name: v(prefix + '-name') || 'Person',
      birth_date: toISO(v(prefix + '-birth-date')),
      hire_date: toISO(v(prefix + '-hire-date')),
      employment_type: v(prefix + '-employment-type') || 'federal',
      current_salary: v(prefix + '-current-salary') || '0',
      high_3_salary: v(prefix + '-high3-salary') || '0',
      tsp_balance_traditional: v(prefix + '-tsp-traditional') || '0',
      tsp_balance_roth: v(prefix + '-tsp-roth') || '0',
      tsp_contribution_percent: v(prefix + '-tsp-contrib') || '0',
      ss_benefit_62: v(prefix + '-ss-62') || '0',
      ss_benefit_fra: v(prefix + '-ss-fra') || '0',
      ss_benefit_70: v(prefix + '-ss-70') || '0',
      fehb_premium_per_pay_period: v(prefix + '-fehb') || '0',
      survivor_benefit_election_percent: v(prefix + '-survivor') || '0',
    };

    const sickLeave = v(prefix + '-sick-leave');
    if (sickLeave && parseFloat(sickLeave) > 0) {
      employee.sick_leave_hours = sickLeave;
    }

    return employee;
  }

  function buildScenarioPerson(sp, employeeName) {
    const v = (id) => document.getElementById(id)?.value || '';

    const strategy = v(sp + '-strategy') || '4_percent_rule';
    const result = {
      employee_name: employeeName,
      retirement_date: toISO(v(sp + '-retirement-date')),
      ss_start_age: parseInt(v(sp + '-ss-age')) || 67,
      tsp_withdrawal_strategy: strategy,
    };

    // Withdrawal ordering
    const ordering = v(sp + '-ordering');
    if (ordering) {
      result.tsp_withdrawal_ordering = ordering;
    }

    // Strategy-specific fields
    if (strategy === 'need_based') {
      const target = v(sp + '-target-monthly');
      if (target) result.tsp_withdrawal_target_monthly = target;
    }

    if (strategy === 'variable_percentage') {
      const rate = v(sp + '-withdrawal-rate');
      if (rate) result.tsp_withdrawal_rate = rate;
    }

    if (strategy === 'floor_ceiling') {
      const rate = v(sp + '-fc-withdrawal-rate');
      if (rate) result.tsp_withdrawal_rate = rate;
      const floor = v(sp + '-floor');
      if (floor) result.tsp_withdrawal_floor = floor;
      const ceiling = v(sp + '-ceiling');
      if (ceiling) result.tsp_withdrawal_ceiling = ceiling;
    }

    if (strategy === 'fixed_annuity') {
      const payoutRate = v(sp + '-annuity-payout-rate');
      if (payoutRate) result.annuity_payout_rate = payoutRate;
      const premium = v(sp + '-annuity-premium');
      if (premium) result.annuity_premium_percent = premium;
      const cola = v(sp + '-annuity-cola');
      if (cola) result.annuity_cola_rate = cola;
      const guaranteed = v(sp + '-annuity-guaranteed');
      if (guaranteed) result.annuity_guaranteed_years = parseInt(guaranteed);
    }

    return result;
  }

  /**
   * Populate forms from a config object (for loading saved configs)
   */
  function populateFromConfig(config) {
    if (!config) return;

    // Personal details
    const pd = config.personal_details || {};
    if (pd.person_a) populateEmployee('pa', pd.person_a);
    if (pd.person_b) {
      document.getElementById('include-person-b').checked = true;
      document.getElementById('person-b-column')?.classList.remove('hidden');
      populateEmployee('pb', pd.person_b);
    } else {
      document.getElementById('include-person-b').checked = false;
      document.getElementById('person-b-column')?.classList.add('hidden');
    }

    // Global assumptions
    const ga = config.global_assumptions || {};
    setVal('ga-inflation', ga.inflation_rate);
    setVal('ga-fehb-inflation', ga.fehb_premium_inflation);
    setVal('ga-tsp-pre', ga.tsp_return_pre_retirement);
    setVal('ga-tsp-post', ga.tsp_return_post_retirement);
    setVal('ga-cola', ga.cola_general_rate);
    setVal('ga-years', ga.projection_years);
    if (ga.current_location) {
      setVal('ga-state', ga.current_location.state);
      setVal('ga-county', ga.current_location.county);
      setVal('ga-municipality', ga.current_location.municipality);
    }

    // Scenarios — clear existing and re-add
    const container = document.getElementById('scenarios-container');
    if (container) container.innerHTML = '';
    _scenarioCount = 0;

    const scenarios = config.scenarios || [];
    if (scenarios.length === 0) {
      addScenario();
    } else {
      for (const sc of scenarios) {
        addScenario();
        const idx = _scenarioCount - 1;
        const prefix = 'sc-' + idx;
        setVal(prefix + '-name', sc.name);
        if (sc.person_a) populateScenarioPerson(prefix + '-pa', sc.person_a);
        if (sc.person_b && document.getElementById('include-person-b')?.checked) {
          populateScenarioPerson(prefix + '-pb', sc.person_b);
        }
      }
    }
  }

  function populateEmployee(prefix, emp) {
    setVal(prefix + '-name', emp.name);
    setVal(prefix + '-birth-date', fromISO(emp.birth_date));
    setVal(prefix + '-hire-date', fromISO(emp.hire_date));
    setVal(prefix + '-employment-type', emp.employment_type);
    setVal(prefix + '-current-salary', emp.current_salary);
    setVal(prefix + '-high3-salary', emp.high_3_salary);
    setVal(prefix + '-tsp-traditional', emp.tsp_balance_traditional);
    setVal(prefix + '-tsp-roth', emp.tsp_balance_roth);
    setVal(prefix + '-tsp-contrib', emp.tsp_contribution_percent);
    setVal(prefix + '-ss-62', emp.ss_benefit_62);
    setVal(prefix + '-ss-fra', emp.ss_benefit_fra);
    setVal(prefix + '-ss-70', emp.ss_benefit_70);
    setVal(prefix + '-fehb', emp.fehb_premium_per_pay_period);
    setVal(prefix + '-survivor', emp.survivor_benefit_election_percent);
    setVal(prefix + '-sick-leave', emp.sick_leave_hours);
  }

  function populateScenarioPerson(sp, scenario) {
    setVal(sp + '-retirement-date', fromISO(scenario.retirement_date));
    setVal(sp + '-ss-age', scenario.ss_start_age);
    setVal(sp + '-strategy', scenario.tsp_withdrawal_strategy);
    setVal(sp + '-ordering', scenario.tsp_withdrawal_ordering || '');

    // Trigger strategy change to show correct fields
    onStrategyChange(sp);

    // Populate strategy-specific fields
    if (scenario.tsp_withdrawal_target_monthly) {
      setVal(sp + '-target-monthly', scenario.tsp_withdrawal_target_monthly);
    }
    if (scenario.tsp_withdrawal_rate) {
      setVal(sp + '-withdrawal-rate', scenario.tsp_withdrawal_rate);
      setVal(sp + '-fc-withdrawal-rate', scenario.tsp_withdrawal_rate);
    }
    if (scenario.tsp_withdrawal_floor) {
      setVal(sp + '-floor', scenario.tsp_withdrawal_floor);
    }
    if (scenario.tsp_withdrawal_ceiling) {
      setVal(sp + '-ceiling', scenario.tsp_withdrawal_ceiling);
    }
    if (scenario.annuity_payout_rate) {
      setVal(sp + '-annuity-payout-rate', scenario.annuity_payout_rate);
    }
    if (scenario.annuity_premium_percent) {
      setVal(sp + '-annuity-premium', scenario.annuity_premium_percent);
    }
    if (scenario.annuity_cola_rate) {
      setVal(sp + '-annuity-cola', scenario.annuity_cola_rate);
    }
    if (scenario.annuity_guaranteed_years) {
      setVal(sp + '-annuity-guaranteed', scenario.annuity_guaranteed_years);
    }
  }

  // Helpers
  function toISO(dateStr) {
    if (!dateStr) return '0001-01-01T00:00:00Z';
    return dateStr + 'T00:00:00Z';
  }

  function fromISO(isoStr) {
    if (!isoStr || isoStr.startsWith('0001')) return '';
    return isoStr.substring(0, 10); // YYYY-MM-DD
  }

  function setVal(id, val) {
    const el = document.getElementById(id);
    if (el && val !== undefined && val !== null) {
      el.value = val;
    }
  }

  /**
   * Get current scenario count
   */
  function getScenarioCount() {
    return document.querySelectorAll('.scenario-card').length;
  }

  /**
   * Build review summary HTML
   */
  function buildReviewSummary(config) {
    const pd = config.personal_details || {};
    const ga = config.global_assumptions || {};
    const sc = config.scenarios || [];

    let html = '<div class="form-grid">';

    // Personal details summary
    for (const [key, emp] of Object.entries(pd)) {
      html += `<div class="card" style="margin-bottom:12px;">
        <h3>${emp.name || key}</h3>
        <p><strong>Employment:</strong> ${emp.employment_type || 'federal'}</p>
        <p><strong>Birth:</strong> ${fromISO(emp.birth_date)}</p>
        <p><strong>Salary:</strong> $${Number(emp.current_salary).toLocaleString()}</p>
        <p><strong>High-3:</strong> $${Number(emp.high_3_salary).toLocaleString()}</p>
        <p><strong>TSP:</strong> $${Number(emp.tsp_balance_traditional).toLocaleString()} traditional + $${Number(emp.tsp_balance_roth || 0).toLocaleString()} Roth</p>
        <p><strong>SS at FRA:</strong> $${Number(emp.ss_benefit_fra).toLocaleString()}/mo</p>
      </div>`;
    }
    html += '</div>';

    // Assumptions summary
    html += `<div class="card" style="margin-bottom:12px;">
      <h3>Assumptions</h3>
      <p>COLA: ${(ga.cola_general_rate * 100).toFixed(1)}% | TSP Pre: ${(ga.tsp_return_pre_retirement * 100).toFixed(1)}% | TSP Post: ${(ga.tsp_return_post_retirement * 100).toFixed(1)}% | Projection: ${ga.projection_years} years</p>
    </div>`;

    // Scenarios summary
    html += '<h3>Scenarios</h3>';
    for (const s of sc) {
      html += `<div class="card" style="margin-bottom:8px;">
        <strong>${s.name}</strong>
        <p>Person A: retire ${fromISO(s.person_a.retirement_date)}, SS at ${s.person_a.ss_start_age}, strategy: ${s.person_a.tsp_withdrawal_strategy}</p>`;
      if (s.person_b && s.person_b.employee_name) {
        html += `<p>Person B: retire ${fromISO(s.person_b.retirement_date)}, SS at ${s.person_b.ss_start_age}, strategy: ${s.person_b.tsp_withdrawal_strategy}</p>`;
      }
      html += '</div>';
    }

    return html;
  }

  return {
    init,
    addScenario,
    removeScenario,
    onStrategyChange,
    buildConfig,
    populateFromConfig,
    getScenarioCount,
    buildReviewSummary,
  };
})();
