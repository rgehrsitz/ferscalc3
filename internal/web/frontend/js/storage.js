/**
 * storage.js — localStorage persistence
 * Global namespace: FERSCalc.Storage
 */
window.FERSCalc = window.FERSCalc || {};

FERSCalc.Storage = (function() {
  'use strict';

  const CURRENT_KEY = 'ferscalc_current_config';
  const SAVED_KEY   = 'ferscalc_saved_configs';

  let _debounceTimer = null;

  /**
   * Save current config (debounced, called on form changes)
   */
  function autoSave(config) {
    clearTimeout(_debounceTimer);
    _debounceTimer = setTimeout(() => {
      try {
        localStorage.setItem(CURRENT_KEY, JSON.stringify(config));
      } catch (e) {
        console.warn('localStorage save failed:', e);
      }
    }, 500);
  }

  /**
   * Load last auto-saved config
   */
  function loadCurrent() {
    try {
      const raw = localStorage.getItem(CURRENT_KEY);
      return raw ? JSON.parse(raw) : null;
    } catch (e) {
      console.warn('localStorage load failed:', e);
      return null;
    }
  }

  /**
   * Clear auto-saved config
   */
  function clearCurrent() {
    localStorage.removeItem(CURRENT_KEY);
  }

  /**
   * Get all saved configs
   * @returns {Array<{name: string, timestamp: string, config: Object}>}
   */
  function getSavedConfigs() {
    try {
      const raw = localStorage.getItem(SAVED_KEY);
      return raw ? JSON.parse(raw) : [];
    } catch (e) {
      return [];
    }
  }

  /**
   * Save a named config
   */
  function saveConfig(name, config) {
    const saved = getSavedConfigs();
    // Replace if same name exists
    const idx = saved.findIndex(s => s.name === name);
    const entry = {
      name,
      timestamp: new Date().toISOString(),
      config: JSON.parse(JSON.stringify(config)),
    };
    if (idx >= 0) {
      saved[idx] = entry;
    } else {
      saved.push(entry);
    }
    localStorage.setItem(SAVED_KEY, JSON.stringify(saved));
  }

  /**
   * Delete a saved config by name
   */
  function deleteConfig(name) {
    const saved = getSavedConfigs().filter(s => s.name !== name);
    localStorage.setItem(SAVED_KEY, JSON.stringify(saved));
  }

  /**
   * Rename a saved config
   */
  function renameConfig(oldName, newName) {
    const saved = getSavedConfigs();
    const item = saved.find(s => s.name === oldName);
    if (item) {
      item.name = newName;
      item.timestamp = new Date().toISOString();
      localStorage.setItem(SAVED_KEY, JSON.stringify(saved));
    }
  }

  /**
   * Load a saved config by name
   */
  function loadConfig(name) {
    const saved = getSavedConfigs();
    const item = saved.find(s => s.name === name);
    return item ? item.config : null;
  }

  return {
    autoSave,
    loadCurrent,
    clearCurrent,
    getSavedConfigs,
    saveConfig,
    deleteConfig,
    renameConfig,
    loadConfig,
  };
})();
