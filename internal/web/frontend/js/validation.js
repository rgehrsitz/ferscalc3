/**
 * validation.js — Client-side validation (mirrors internal/config/input.go)
 * Global namespace: FERSCalc.Validation
 */
window.FERSCalc = window.FERSCalc || {};

FERSCalc.Validation = (function() {
  'use strict';

  /**
   * Validate an employee (person) form fields
   * @param {string} prefix — 'pa' or 'pb'
   * @returns {Array<{field: string, message: string}>}
   */
  function validatePerson(prefix) {
    const errors = [];
    const v = (id) => document.getElementById(id)?.value || '';
    const n = (id) => parseFloat(document.getElementById(id)?.value) || 0;

    // Required dates
    if (!v(prefix + '-birth-date')) {
      errors.push({ field: prefix + '-birth-date', message: 'Birth date is required' });
    }
    if (!v(prefix + '-hire-date')) {
      errors.push({ field: prefix + '-hire-date', message: 'Hire date is required' });
    }

    // Date logic
    if (v(prefix + '-birth-date') && v(prefix + '-hire-date')) {
      if (new Date(v(prefix + '-birth-date')) >= new Date(v(prefix + '-hire-date'))) {
        errors.push({ field: prefix + '-hire-date', message: 'Hire date must be after birth date' });
      }
    }

    // Salary
    if (n(prefix + '-current-salary') <= 0) {
      errors.push({ field: prefix + '-current-salary', message: 'Current salary must be positive' });
    }
    if (n(prefix + '-high3-salary') <= 0) {
      errors.push({ field: prefix + '-high3-salary', message: 'High-3 salary must be positive' });
    }

    // TSP balances
    if (n(prefix + '-tsp-traditional') < 0) {
      errors.push({ field: prefix + '-tsp-traditional', message: 'TSP Traditional cannot be negative' });
    }
    if (n(prefix + '-tsp-roth') < 0) {
      errors.push({ field: prefix + '-tsp-roth', message: 'TSP Roth cannot be negative' });
    }

    // TSP contribution percent
    const contrib = n(prefix + '-tsp-contrib');
    if (contrib < 0 || contrib > 1) {
      errors.push({ field: prefix + '-tsp-contrib', message: 'Must be between 0 and 1' });
    }

    // SS benefits — must be positive and ascending
    const ss62 = n(prefix + '-ss-62');
    const ssFRA = n(prefix + '-ss-fra');
    const ss70 = n(prefix + '-ss-70');
    if (ss62 <= 0) errors.push({ field: prefix + '-ss-62', message: 'Must be positive' });
    if (ssFRA <= 0) errors.push({ field: prefix + '-ss-fra', message: 'Must be positive' });
    if (ss70 <= 0) errors.push({ field: prefix + '-ss-70', message: 'Must be positive' });
    if (ss62 > 0 && ssFRA > 0 && ss62 > ssFRA) {
      errors.push({ field: prefix + '-ss-62', message: 'Age 62 benefit cannot exceed FRA benefit' });
    }
    if (ssFRA > 0 && ss70 > 0 && ssFRA > ss70) {
      errors.push({ field: prefix + '-ss-fra', message: 'FRA benefit cannot exceed age 70 benefit' });
    }

    // FEHB
    if (n(prefix + '-fehb') < 0) {
      errors.push({ field: prefix + '-fehb', message: 'Cannot be negative' });
    }

    // Survivor benefit
    const surv = n(prefix + '-survivor');
    if (surv < 0 || surv > 1) {
      errors.push({ field: prefix + '-survivor', message: 'Must be between 0 and 1' });
    }

    return errors;
  }

  /**
   * Validate global assumptions
   * @returns {Array<{field: string, message: string}>}
   */
  function validateAssumptions() {
    const errors = [];
    const n = (id) => parseFloat(document.getElementById(id)?.value);

    if (n('ga-inflation') < -0.10) {
      errors.push({ field: 'ga-inflation', message: 'Cannot be less than -10%' });
    }
    if (n('ga-fehb-inflation') < 0) {
      errors.push({ field: 'ga-fehb-inflation', message: 'Cannot be negative' });
    }
    if (n('ga-tsp-pre') < -1.0) {
      errors.push({ field: 'ga-tsp-pre', message: 'Cannot be less than -100%' });
    }
    if (n('ga-tsp-post') < -1.0) {
      errors.push({ field: 'ga-tsp-post', message: 'Cannot be less than -100%' });
    }
    if (n('ga-cola') < 0) {
      errors.push({ field: 'ga-cola', message: 'Cannot be negative' });
    }

    const years = n('ga-years');
    if (!years || years < 1 || years > 50) {
      errors.push({ field: 'ga-years', message: 'Must be between 1 and 50' });
    }

    return errors;
  }

  /**
   * Validate a scenario card
   * @param {number} idx — scenario index
   * @param {boolean} hasPersonB
   * @returns {Array<{field: string, message: string}>}
   */
  function validateScenario(idx, hasPersonB) {
    const errors = [];
    const v = (id) => document.getElementById(id)?.value || '';
    const n = (id) => parseFloat(document.getElementById(id)?.value);

    const prefix = 'sc-' + idx;

    if (!v(prefix + '-name').trim()) {
      errors.push({ field: prefix + '-name', message: 'Scenario name is required' });
    }

    // Validate each person in scenario
    const persons = ['pa', 'pb'];
    if (!hasPersonB) persons.pop();

    for (const p of persons) {
      const sp = prefix + '-' + p;

      if (!v(sp + '-retirement-date')) {
        errors.push({ field: sp + '-retirement-date', message: 'Retirement date is required' });
      }

      const ssAge = n(sp + '-ss-age');
      if (!ssAge || ssAge < 62 || ssAge > 70) {
        errors.push({ field: sp + '-ss-age', message: 'Must be between 62 and 70' });
      }

      const strategy = v(sp + '-strategy');
      if (!strategy) {
        errors.push({ field: sp + '-strategy', message: 'Strategy is required' });
      }

      // Strategy-specific fields
      if (strategy === 'need_based') {
        const target = n(sp + '-target-monthly');
        if (!target || target <= 0) {
          errors.push({ field: sp + '-target-monthly', message: 'Monthly target must be positive' });
        }
      }
      if (strategy === 'variable_percentage' || strategy === 'floor_ceiling') {
        const rate = n(sp + '-withdrawal-rate');
        if (rate === undefined || isNaN(rate) || rate < 0 || rate > 0.2) {
          errors.push({ field: sp + '-withdrawal-rate', message: 'Rate must be 0-0.20' });
        }
      }
      if (strategy === 'floor_ceiling') {
        const floor = n(sp + '-floor');
        const ceiling = n(sp + '-ceiling');
        if (floor && ceiling && floor >= ceiling) {
          errors.push({ field: sp + '-floor', message: 'Floor must be less than ceiling' });
        }
      }
      if (strategy === 'fixed_annuity') {
        const payoutRate = n(sp + '-annuity-payout-rate');
        if (!payoutRate || payoutRate <= 0 || payoutRate > 0.20) {
          errors.push({ field: sp + '-annuity-payout-rate', message: 'Payout rate must be 0-20%' });
        }
      }
    }

    return errors;
  }

  /**
   * Show errors on the form (inline messages + field highlights)
   * @param {Array<{field: string, message: string}>} errors
   */
  function showErrors(errors) {
    // Clear previous errors
    clearErrors();

    for (const err of errors) {
      const el = document.getElementById(err.field);
      if (el) {
        el.classList.add('error');
        // Add error message after the input
        const msg = document.createElement('div');
        msg.className = 'error-msg';
        msg.textContent = err.message;
        msg.dataset.errorFor = err.field;
        el.parentNode.appendChild(msg);
      }
    }
  }

  /**
   * Clear all error states
   */
  function clearErrors() {
    document.querySelectorAll('.error').forEach(el => el.classList.remove('error'));
    document.querySelectorAll('.error-msg').forEach(el => el.remove());
  }

  return {
    validatePerson,
    validateAssumptions,
    validateScenario,
    showErrors,
    clearErrors,
  };
})();
