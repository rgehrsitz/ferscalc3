/**
 * app.js — Entry point, view routing, initialization
 * Global namespace: FERSCalc.App
 */
window.FERSCalc = window.FERSCalc || {};

FERSCalc.App = (function() {
  'use strict';

  /**
   * Initialize the application
   */
  async function init() {
    // Start heartbeat so the server stays alive while the browser is open
    FERSCalc.API.startHeartbeat();

    try {
      // Load metadata from API
      await FERSCalc.API.loadMetadata();
    } catch (err) {
      console.warn('Failed to load metadata:', err);
      showToast('Warning: Could not load server metadata. Some features may be limited.', 'error');
    }

    // Initialize forms
    FERSCalc.Forms.init();

    // Try to restore from localStorage
    const saved = FERSCalc.Storage.loadCurrent();
    if (saved) {
      try {
        FERSCalc.Forms.populateFromConfig(saved);
        showToast('Restored previous session', 'info');
      } catch (e) {
        console.warn('Failed to restore saved config:', e);
      }
    }

    // Render saved configs list
    FERSCalc.Scenarios.renderSavedList();

    // Auto-save on form changes
    document.addEventListener('change', debounce(() => {
      try {
        const config = FERSCalc.Forms.buildConfig();
        FERSCalc.Storage.autoSave(config);
      } catch (e) {
        // Ignore auto-save errors
      }
    }, 500));

    console.log('FERS Retirement Calculator initialized');
  }

  /**
   * Switch to a view
   * @param {string} viewName — 'wizard', 'results', or 'saved'
   */
  function showView(viewName) {
    // Hide all views
    document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));

    // Show target
    const target = document.getElementById('view-' + viewName);
    if (target) target.classList.add('active');

    // Update nav buttons
    document.querySelectorAll('.nav-btn').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.view === viewName);
    });

    // Refresh saved list when showing saved view
    if (viewName === 'saved') {
      FERSCalc.Scenarios.renderSavedList();
    }
  }

  /**
   * Run the calculation
   */
  async function runCalculation() {
    const config = FERSCalc.Forms.buildConfig();

    showSpinner(true);
    try {
      const results = await FERSCalc.API.runScenario(config);
      FERSCalc.Results.render(results);
      showView('results');
      showToast('Calculation complete!', 'success');
    } catch (err) {
      showToast('Calculation failed: ' + err.message, 'error');
      console.error('Calculation error:', err);
    } finally {
      showSpinner(false);
    }
  }

  /**
   * Switch back to wizard for editing
   */
  function editConfig() {
    showView('wizard');
  }

  /**
   * Show/hide loading spinner
   */
  function showSpinner(show) {
    const el = document.getElementById('spinner');
    if (el) el.classList.toggle('hidden', !show);
  }

  /**
   * Show a toast notification
   * @param {string} message
   * @param {string} type — 'success', 'error', 'info'
   */
  function showToast(message, type) {
    const container = document.getElementById('toasts');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = 'toast ' + (type || 'info');
    toast.textContent = message;
    container.appendChild(toast);

    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transition = 'opacity 0.3s';
      setTimeout(() => toast.remove(), 300);
    }, 4000);
  }

  // Utility: debounce
  function debounce(fn, delay) {
    let timer;
    return function(...args) {
      clearTimeout(timer);
      timer = setTimeout(() => fn.apply(this, args), delay);
    };
  }

  // Initialize on DOM ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  return {
    showView,
    runCalculation,
    editConfig,
    showSpinner,
    showToast,
  };
})();
