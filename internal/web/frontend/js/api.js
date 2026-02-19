/**
 * api.js — Fetch wrappers for all API endpoints
 * Global namespace: FERSCalc.API
 */
window.FERSCalc = window.FERSCalc || {};

FERSCalc.API = (function() {
  'use strict';

  const BASE = '/api/v1';

  // Cached metadata
  let _meta = null;

  /**
   * Fetch all metadata endpoints once and cache
   */
  async function loadMetadata() {
    if (_meta) return _meta;
    const [states, strategies, funds, employmentTypes, annuityOptions] = await Promise.all([
      fetchJSON(BASE + '/meta/states'),
      fetchJSON(BASE + '/meta/tsp-strategies'),
      fetchJSON(BASE + '/meta/tsp-funds'),
      fetchJSON(BASE + '/meta/employment-types'),
      fetchJSON(BASE + '/meta/annuity-options'),
    ]);
    _meta = { states, strategies, funds, employmentTypes, annuityOptions };
    return _meta;
  }

  function getMetadata() {
    return _meta;
  }

  /**
   * Run a scenario calculation
   * @param {Object} config — full Configuration JSON
   * @returns {Object} ScenarioComparison result
   */
  async function runScenario(config) {
    const resp = await fetch(BASE + '/scenarios/run', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    });
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: resp.statusText }));
      throw new Error(err.error || 'Calculation failed');
    }
    return resp.json();
  }

  /**
   * Get example configuration
   */
  async function getExampleConfig() {
    return fetchJSON(BASE + '/configurations/example');
  }

  /**
   * Export config as YAML
   * @param {Object} config — JSON configuration
   * @returns {string} YAML string
   */
  async function exportYAML(config) {
    const resp = await fetch(BASE + '/configurations/export-yaml', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    });
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: resp.statusText }));
      throw new Error(err.error || 'YAML export failed');
    }
    return resp.text();
  }

  /**
   * Parse YAML to JSON
   * @param {string} yamlStr — raw YAML string
   * @returns {Object} parsed configuration
   */
  async function parseYAML(yamlStr) {
    const resp = await fetch(BASE + '/configurations/parse-yaml', {
      method: 'POST',
      headers: { 'Content-Type': 'text/plain' },
      body: yamlStr,
    });
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: resp.statusText }));
      throw new Error(err.error || 'YAML parse failed');
    }
    return resp.json();
  }

  // ── Heartbeat ────────────────────────────────────────────────
  // Keeps the server alive while the browser tab is open.
  // The server auto-shuts down ~30s after the last heartbeat.
  let _heartbeatTimer = null;

  function startHeartbeat() {
    if (_heartbeatTimer) return;
    sendHeartbeat(); // send immediately
    _heartbeatTimer = setInterval(sendHeartbeat, 10000); // then every 10s

    // Also fire when tab becomes visible again (e.g. user switches back)
    document.addEventListener('visibilitychange', function() {
      if (!document.hidden) sendHeartbeat();
    });
  }

  function sendHeartbeat() {
    fetch(BASE + '/heartbeat', { method: 'POST' }).catch(function() {
      // Server is gone — stop pinging
      if (_heartbeatTimer) {
        clearInterval(_heartbeatTimer);
        _heartbeatTimer = null;
      }
    });
  }

  // Internal helper
  async function fetchJSON(url) {
    const resp = await fetch(url);
    if (!resp.ok) throw new Error(`GET ${url} failed: ${resp.statusText}`);
    return resp.json();
  }

  return {
    loadMetadata,
    getMetadata,
    runScenario,
    getExampleConfig,
    exportYAML,
    parseYAML,
    startHeartbeat,
  };
})();
