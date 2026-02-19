/**
 * wizard.js — 4-step wizard navigation
 * Global namespace: FERSCalc.Wizard
 */
window.FERSCalc = window.FERSCalc || {};

FERSCalc.Wizard = (function() {
  'use strict';

  let _currentStep = 1;
  const TOTAL_STEPS = 4;

  /**
   * Go to a specific step
   */
  function goToStep(step) {
    if (step < 1 || step > TOTAL_STEPS) return;

    // Hide all panels
    for (let i = 1; i <= TOTAL_STEPS; i++) {
      const panel = document.getElementById('step-' + i);
      if (panel) panel.style.display = 'none';
    }

    // Show target panel
    const target = document.getElementById('step-' + step);
    if (target) target.style.display = '';

    _currentStep = step;
    updateIndicators();

    // If going to review step, update preview
    if (step === 4) {
      updateReviewStep();
    }
  }

  /**
   * Go to next step (with validation)
   */
  function next() {
    FERSCalc.Validation.clearErrors();

    const errors = validateCurrentStep();
    if (errors.length > 0) {
      FERSCalc.Validation.showErrors(errors);
      // Scroll to first error
      const firstField = document.getElementById(errors[0].field);
      if (firstField) firstField.scrollIntoView({ behavior: 'smooth', block: 'center' });
      return;
    }

    if (_currentStep < TOTAL_STEPS) {
      goToStep(_currentStep + 1);
      window.scrollTo({ top: 0, behavior: 'smooth' });
    }
  }

  /**
   * Go to previous step
   */
  function prev() {
    FERSCalc.Validation.clearErrors();
    if (_currentStep > 1) {
      goToStep(_currentStep - 1);
      window.scrollTo({ top: 0, behavior: 'smooth' });
    }
  }

  /**
   * Navigate to a specific step via the step indicators.
   * Going backward: no validation needed.
   * Going forward: validates every intermediate step in order.
   */
  function navigateToStep(targetStep) {
    if (targetStep < 1 || targetStep > TOTAL_STEPS || targetStep === _currentStep) return;

    FERSCalc.Validation.clearErrors();

    // Going backward — always allowed
    if (targetStep < _currentStep) {
      goToStep(targetStep);
      window.scrollTo({ top: 0, behavior: 'smooth' });
      return;
    }

    // Going forward — validate each step from current up to (but not including) target
    const savedStep = _currentStep;
    for (let s = _currentStep; s < targetStep; s++) {
      _currentStep = s; // temporarily set so validateCurrentStep() checks the right step
      const errors = validateCurrentStep();
      if (errors.length > 0) {
        // Show the failing step and its errors
        goToStep(s);
        FERSCalc.Validation.showErrors(errors);
        const firstField = document.getElementById(errors[0].field);
        if (firstField) firstField.scrollIntoView({ behavior: 'smooth', block: 'center' });
        return;
      }
    }

    // All intermediate steps passed — go to target
    goToStep(targetStep);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  /**
   * Validate the current step
   * @returns {Array<{field, message}>}
   */
  function validateCurrentStep() {
    switch (_currentStep) {
      case 1: {
        let errors = FERSCalc.Validation.validatePerson('pa');
        if (document.getElementById('include-person-b')?.checked) {
          errors = errors.concat(FERSCalc.Validation.validatePerson('pb'));
        }
        return errors;
      }
      case 2:
        return FERSCalc.Validation.validateAssumptions();
      case 3: {
        const hasPersonB = document.getElementById('include-person-b')?.checked;
        let errors = [];
        document.querySelectorAll('.scenario-card').forEach(card => {
          const idx = parseInt(card.id.replace('scenario-card-', ''));
          errors = errors.concat(FERSCalc.Validation.validateScenario(idx, hasPersonB));
        });
        return errors;
      }
      case 4:
        return []; // Review step — no validation needed
      default:
        return [];
    }
  }

  /**
   * Update step indicators (active / completed)
   */
  function updateIndicators() {
    const indicators = document.querySelectorAll('.wizard-step');
    indicators.forEach(ind => {
      const step = parseInt(ind.dataset.step);
      ind.classList.remove('active', 'completed');
      if (step === _currentStep) {
        ind.classList.add('active');
      } else if (step < _currentStep) {
        ind.classList.add('completed');
      }
    });
  }

  /**
   * Update the review step with current config
   */
  function updateReviewStep() {
    const config = FERSCalc.Forms.buildConfig();

    // Summary
    const summaryEl = document.getElementById('review-summary');
    if (summaryEl) {
      summaryEl.innerHTML = FERSCalc.Forms.buildReviewSummary(config);
    }

    // JSON preview
    const jsonEl = document.getElementById('json-preview');
    if (jsonEl) {
      jsonEl.textContent = JSON.stringify(config, null, 2);
    }

    // Auto-save
    FERSCalc.Storage.autoSave(config);
  }

  /**
   * Get current step number
   */
  function getCurrentStep() {
    return _currentStep;
  }

  return {
    goToStep,
    next,
    prev,
    navigateToStep,
    getCurrentStep,
    updateReviewStep,
  };
})();
