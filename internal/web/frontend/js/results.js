/**
 * results.js — Results rendering, comparison table, drill-down
 * Global namespace: FERSCalc.Results
 */
window.FERSCalc = window.FERSCalc || {};

FERSCalc.Results = (function() {
  'use strict';

  let _lastResults = null;
  let _selectedScenario = 0;

  function getLastResults() { return _lastResults; }

  /**
   * Render results from API response
   * @param {Object} comparison — ScenarioComparison from the API
   */
  function render(comparison) {
    _lastResults = comparison;
    _selectedScenario = 0;

    const container = document.getElementById('results-content');
    if (!container) return;

    const scenarios = comparison.scenarios || comparison.Scenarios || [];
    if (scenarios.length === 0) {
      container.innerHTML = '<div class="card"><p class="text-muted">No scenario results returned.</p></div>';
      return;
    }

    let html = '';

    // Comparison Table
    html += renderComparisonTable(scenarios);

    // Drill-down area
    html += '<div id="drill-down-area"></div>';

    // Chart canvases
    html += renderChartCanvases(scenarios);

    container.innerHTML = html;

    // Render charts
    FERSCalc.Charts.renderAll(scenarios);

    // Show first scenario detail by default
    showDrillDown(0, scenarios);
  }

  function renderComparisonTable(scenarios) {
    let html = '<div class="card"><h2>Scenario Comparison</h2>';
    html += '<table class="data-table"><thead><tr>';
    html += '<th>Scenario</th><th>Year 1 Net</th><th>Year 5 Net</th><th>Year 10 Net</th>';
    html += '<th>TSP Longevity</th><th>Final TSP</th>';
    html += '</tr></thead><tbody>';

    scenarios.forEach((s, i) => {
      const summary = s.summary || s;
      html += `<tr class="clickable ${i === 0 ? 'selected' : ''}" onclick="FERSCalc.Results.selectScenario(${i})">`;
      html += `<td><strong>${s.name || s.Name || 'Scenario ' + (i + 1)}</strong></td>`;
      html += `<td>${dollar(summary.first_year_net_income)}</td>`;
      html += `<td>${dollar(summary.year_5_net_income)}</td>`;
      html += `<td>${dollar(summary.year_10_net_income)}</td>`;
      html += `<td>${summary.tsp_longevity || 'N/A'} yrs</td>`;
      html += `<td>${dollar(summary.final_tsp_balance)}</td>`;
      html += '</tr>';
    });

    html += '</tbody></table></div>';
    return html;
  }

  function renderChartCanvases(scenarios) {
    let html = '<div class="card"><h2>Visual Analysis</h2>';

    html += '<div class="chart-grid">';
    html += '<div><h3>TSP Balance</h3><div class="chart-container"><canvas id="chart-tsp"></canvas></div></div>';
    html += '<div><h3>Net Income</h3><div class="chart-container"><canvas id="chart-income"></canvas></div></div>';
    html += '</div>';

    html += '<div><h3>Federal Tax Breakdown</h3><div class="chart-container"><canvas id="chart-federal-tax"></canvas></div></div>';

    html += '<div class="chart-grid">';
    html += '<div><h3>FEHB Premiums</h3><div class="chart-container"><canvas id="chart-fehb"></canvas></div></div>';
    html += '<div><h3>Medicare Premiums</h3><div class="chart-container"><canvas id="chart-medicare"></canvas></div></div>';
    html += '</div>';

    html += '<div><h3>Cumulative Net Income</h3><div class="chart-container"><canvas id="chart-cumulative"></canvas></div></div>';

    html += '<div class="chart-grid">';
    scenarios.forEach((s, i) => {
      const name = s.name || s.Name || 'Scenario ' + (i + 1);
      html += `<div><h3>${name} - Income Sources</h3><div class="chart-container"><canvas id="chart-sources-${i}"></canvas></div></div>`;
    });
    html += '</div>';

    html += '</div>';
    return html;
  }

  /**
   * Select a scenario for drill-down
   */
  function selectScenario(idx) {
    _selectedScenario = idx;
    const scenarios = _lastResults?.scenarios || _lastResults?.Scenarios || [];

    // Update table selection
    document.querySelectorAll('.data-table tbody tr').forEach((tr, i) => {
      tr.classList.toggle('selected', i === idx);
    });

    showDrillDown(idx, scenarios);
  }

  function showDrillDown(idx, scenarios) {
    const container = document.getElementById('drill-down-area');
    if (!container || !scenarios[idx]) return;

    const scenario = scenarios[idx];
    const projection = scenario.projection || scenario.Projection || [];

    let html = '<div class="card drill-down">';
    html += `<h2>Detail: ${scenario.name || scenario.Name}</h2>`;

    // Before & After (if available)
    if (projection.length >= 2) {
      html += renderBeforeAfter(projection);
    }

    // Year-by-year table
    html += '<h3 class="mt-16">Year-by-Year Projection</h3>';
    html += '<div class="year-table-wrap">';
    html += renderProjectionTable(projection);
    html += '</div>';

    html += '</div>';
    container.innerHTML = html;
  }

  function renderBeforeAfter(projection) {
    // Find transition: last year with salary > 0 vs first year without
    let lastWorking = null;
    let firstRetired = null;

    for (let i = 0; i < projection.length; i++) {
      const y = projection[i];
      const hasSalary = (num(y.salary_person_a) + num(y.salary_person_b)) > 0;
      if (hasSalary) {
        lastWorking = y;
      } else if (lastWorking && !firstRetired) {
        firstRetired = y;
        break;
      }
    }

    if (!lastWorking || !firstRetired) return '';

    const yearW = yearOf(lastWorking);
    const yearR = yearOf(firstRetired);

    let html = '<h3>Before &amp; After Retirement</h3>';
    html += '<table class="data-table"><thead><tr>';
    html += `<th>Component</th><th>Working (${yearW})</th><th>Retired (${yearR})</th><th>Change</th>`;
    html += '</tr></thead><tbody>';

    const rows = [
      ['Total Salary', num(lastWorking.salary_person_a) + num(lastWorking.salary_person_b), num(firstRetired.salary_person_a) + num(firstRetired.salary_person_b)],
      ['FERS Pension', num(lastWorking.pension_person_a) + num(lastWorking.pension_person_b), num(firstRetired.pension_person_a) + num(firstRetired.pension_person_b)],
      ['TSP Withdrawals', num(lastWorking.tsp_withdrawal_person_a) + num(lastWorking.tsp_withdrawal_person_b), num(firstRetired.tsp_withdrawal_person_a) + num(firstRetired.tsp_withdrawal_person_b)],
      ['Social Security', num(lastWorking.ss_benefit_person_a) + num(lastWorking.ss_benefit_person_b), num(firstRetired.ss_benefit_person_a) + num(firstRetired.ss_benefit_person_b)],
      ['Gross Income', num(lastWorking.total_gross_income), num(firstRetired.total_gross_income)],
      ['Federal Tax', num(lastWorking.federal_tax), num(firstRetired.federal_tax)],
      ['State Tax', num(lastWorking.state_tax), num(firstRetired.state_tax)],
      ['FICA', num(lastWorking.fica_tax), num(firstRetired.fica_tax)],
      ['FEHB Premium', num(lastWorking.fehb_premium), num(firstRetired.fehb_premium)],
      ['Net Income', num(lastWorking.net_income), num(firstRetired.net_income)],
    ];

    for (const [label, before, after] of rows) {
      const change = after - before;
      const changeClass = change > 0 ? 'color:#2ecc71' : (change < 0 ? 'color:#e74c3c' : '');
      const isBold = label === 'Gross Income' || label === 'Net Income';
      const wrap = isBold ? '<strong>' : '';
      const wrapEnd = isBold ? '</strong>' : '';
      html += `<tr${isBold ? ' class="total-row"' : ''}>`;
      html += `<td>${wrap}${label}${wrapEnd}</td>`;
      html += `<td>${wrap}${dollar(before)}${wrapEnd}</td>`;
      html += `<td>${wrap}${dollar(after)}${wrapEnd}</td>`;
      html += `<td style="${changeClass}">${wrap}${change >= 0 ? '+' : ''}${dollar(change)}${wrapEnd}</td>`;
      html += '</tr>';
    }

    html += '</tbody></table>';
    return html;
  }

  function renderProjectionTable(projection) {
    if (!projection || projection.length === 0) return '<p class="text-muted">No projection data.</p>';

    let html = '<table class="data-table"><thead><tr>';
    html += '<th>Year</th><th>Net Income</th><th>Gross Income</th>';
    html += '<th>Pension</th><th>SS Benefits</th><th>TSP Withdrawal</th>';
    html += '<th>Federal Tax</th><th>FEHB</th><th>TSP Balance</th>';
    html += '</tr></thead><tbody>';

    for (const y of projection) {
      const year = yearOf(y);
      html += '<tr>';
      html += `<td>${year}</td>`;
      html += `<td><strong>${dollar(y.net_income)}</strong></td>`;
      html += `<td>${dollar(y.total_gross_income)}</td>`;
      html += `<td>${dollar(num(y.pension_person_a) + num(y.pension_person_b))}</td>`;
      html += `<td>${dollar(num(y.ss_benefit_person_a) + num(y.ss_benefit_person_b))}</td>`;
      html += `<td>${dollar(num(y.tsp_withdrawal_person_a) + num(y.tsp_withdrawal_person_b))}</td>`;
      html += `<td>${dollar(y.federal_tax)}</td>`;
      html += `<td>${dollar(y.fehb_premium)}</td>`;
      html += `<td>${dollar(num(y.tsp_balance_person_a) + num(y.tsp_balance_person_b))}</td>`;
      html += '</tr>';
    }

    html += '</tbody></table>';
    return html;
  }

  // Helpers
  function num(v) {
    if (v === undefined || v === null) return 0;
    if (typeof v === 'string') return parseFloat(v) || 0;
    return v;
  }

  function dollar(v) {
    const n = num(v);
    if (n === 0) return '$0';
    return '$' + Math.round(n).toLocaleString();
  }

  function yearOf(y) {
    if (y.date) return new Date(y.date).getFullYear();
    if (y.Date) return new Date(y.Date).getFullYear();
    return y.year || y.Year || '?';
  }

  return {
    render,
    selectScenario,
    getLastResults,
  };
})();
