/**
 * scenarios.js — Save/load/delete named configurations, export/import
 * Global namespace: FERSCalc.Scenarios
 */
window.FERSCalc = window.FERSCalc || {};

FERSCalc.Scenarios = (function() {
  'use strict';

  /**
   * Render the saved configurations list
   */
  function renderSavedList() {
    const container = document.getElementById('saved-configs-list');
    if (!container) return;

    const saved = FERSCalc.Storage.getSavedConfigs();
    if (saved.length === 0) {
      container.innerHTML = '<p class="text-muted">No saved configurations yet. Use "Save Configuration" from the Review step.</p>';
      return;
    }

    let html = '<ul class="saved-list">';
    for (const item of saved) {
      const date = new Date(item.timestamp).toLocaleDateString(undefined, {
        year: 'numeric', month: 'short', day: 'numeric',
        hour: '2-digit', minute: '2-digit',
      });
      html += `<li>
        <div>
          <span class="config-name">${escapeHTML(item.name)}</span>
          <span class="config-date">${date}</span>
        </div>
        <div class="btn-group">
          <button class="btn btn-primary btn-sm" onclick="FERSCalc.Scenarios.loadSaved('${escapeAttr(item.name)}')">Load</button>
          <button class="btn btn-danger btn-sm" onclick="FERSCalc.Scenarios.deleteSaved('${escapeAttr(item.name)}')">Delete</button>
        </div>
      </li>`;
    }
    html += '</ul>';
    container.innerHTML = html;
  }

  /**
   * Save current config with a name
   */
  function saveCurrentConfig() {
    const name = prompt('Enter a name for this configuration:');
    if (!name || !name.trim()) return;

    const config = FERSCalc.Forms.buildConfig();
    FERSCalc.Storage.saveConfig(name.trim(), config);
    FERSCalc.App.showToast('Configuration saved: ' + name.trim(), 'success');
    renderSavedList();
  }

  /**
   * Load a saved config by name
   */
  function loadSaved(name) {
    const config = FERSCalc.Storage.loadConfig(name);
    if (!config) {
      FERSCalc.App.showToast('Configuration not found', 'error');
      return;
    }

    FERSCalc.Forms.populateFromConfig(config);
    FERSCalc.App.showView('wizard');
    FERSCalc.Wizard.goToStep(1);
    FERSCalc.App.showToast('Loaded: ' + name, 'info');
  }

  /**
   * Delete a saved config
   */
  function deleteSaved(name) {
    if (!confirm('Delete "' + name + '"?')) return;
    FERSCalc.Storage.deleteConfig(name);
    FERSCalc.App.showToast('Deleted: ' + name, 'info');
    renderSavedList();
  }

  /**
   * Export current config as JSON file
   */
  function exportJSON() {
    const config = FERSCalc.Forms.buildConfig();
    const blob = new Blob([JSON.stringify(config, null, 2)], { type: 'application/json' });
    downloadBlob(blob, 'ferscalc_config.json');
  }

  /**
   * Export current config as YAML file
   */
  async function exportYAML() {
    try {
      const config = FERSCalc.Forms.buildConfig();
      const yamlStr = await FERSCalc.API.exportYAML(config);
      const blob = new Blob([yamlStr], { type: 'text/yaml' });
      downloadBlob(blob, 'ferscalc_config.yaml');
    } catch (err) {
      FERSCalc.App.showToast('YAML export failed: ' + err.message, 'error');
    }
  }

  /**
   * Import a JSON config file
   */
  function importJSON() {
    triggerFileInput('.json', async (file) => {
      try {
        const text = await file.text();
        const config = JSON.parse(text);
        FERSCalc.Forms.populateFromConfig(config);
        FERSCalc.App.showView('wizard');
        FERSCalc.Wizard.goToStep(1);
        FERSCalc.App.showToast('Imported: ' + file.name, 'success');
      } catch (err) {
        FERSCalc.App.showToast('Import failed: ' + err.message, 'error');
      }
    });
  }

  /**
   * Import a YAML config file
   */
  function importYAML() {
    triggerFileInput('.yaml,.yml', async (file) => {
      try {
        const text = await file.text();
        const config = await FERSCalc.API.parseYAML(text);
        FERSCalc.Forms.populateFromConfig(config);
        FERSCalc.App.showView('wizard');
        FERSCalc.Wizard.goToStep(1);
        FERSCalc.App.showToast('Imported: ' + file.name, 'success');
      } catch (err) {
        FERSCalc.App.showToast('YAML import failed: ' + err.message, 'error');
      }
    });
  }

  // Helpers
  function downloadBlob(blob, filename) {
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }

  function triggerFileInput(accept, callback) {
    const input = document.getElementById('file-import');
    if (!input) return;
    input.accept = accept;
    input.value = '';
    input.onchange = () => {
      if (input.files.length > 0) {
        callback(input.files[0]);
      }
    };
    input.click();
  }

  function escapeHTML(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  function escapeAttr(str) {
    return str.replace(/'/g, "\\'").replace(/"/g, '&quot;');
  }

  return {
    renderSavedList,
    saveCurrentConfig,
    loadSaved,
    deleteSaved,
    exportJSON,
    exportYAML,
    importJSON,
    importYAML,
  };
})();
