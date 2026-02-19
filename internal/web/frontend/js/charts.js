/**
 * charts.js — Chart.js integration (ported from report.html.tmpl)
 * Global namespace: FERSCalc.Charts
 */
window.FERSCalc = window.FERSCalc || {};

FERSCalc.Charts = (function() {
  'use strict';

  const COLORS = ['#3498db', '#e74c3c', '#2ecc71', '#f39c12', '#9b59b6'];
  const SOURCE_COLORS = {
    salary: '#3498db',
    pension: '#2ecc71',
    tspWithdrawal: '#f39c12',
    socialSecurity: '#9b59b6',
  };
  const BRACKET_COLORS = {
    '10': '#d4edda',
    '12': '#a8d5ba',
    '22': '#ffc107',
    '24': '#ff9800',
    '32': '#ff5722',
    '35': '#f44336',
    '37': '#c62828',
  };

  let _charts = [];

  /**
   * Destroy all existing charts
   */
  function destroyAll() {
    _charts.forEach(c => c.destroy());
    _charts = [];
  }

  /**
   * Render all charts from scenario data
   * @param {Array} scenarioResults — array of scenario objects with projection data
   */
  function renderAll(scenarioResults) {
    destroyAll();

    if (!scenarioResults || scenarioResults.length === 0) return;

    // Extract chart data from results
    const scenarioData = scenarioResults.map(extractChartData);

    renderTSPChart(scenarioData);
    renderIncomeChart(scenarioData);
    renderFederalTaxChart(scenarioData);
    renderFEHBChart(scenarioData);
    renderMedicareChart(scenarioData);
    renderCumulativeChart(scenarioData);
    renderIncomeSourcesCharts(scenarioData);
  }

  function extractChartData(scenario) {
    const projection = scenario.projection || scenario.Projection || [];
    return {
      name: scenario.name || scenario.Name || 'Scenario',
      years: projection.map(y => y.date ? new Date(y.date).getFullYear() : (y.Date ? new Date(y.Date).getFullYear() : y.year || y.Year)),
      tspBalances: projection.map(y => num(y.tsp_balance_person_a) + num(y.tsp_balance_person_b)),
      netIncomes: projection.map(y => num(y.net_income)),
      salaries: projection.map(y => num(y.salary_person_a) + num(y.salary_person_b)),
      pensions: projection.map(y => num(y.pension_person_a) + num(y.pension_person_b)),
      tspWithdrawals: projection.map(y => num(y.tsp_withdrawal_person_a) + num(y.tsp_withdrawal_person_b)),
      socialSecurity: projection.map(y => num(y.ss_benefit_person_a) + num(y.ss_benefit_person_b)),
      fehbPremiums: projection.map(y => num(y.fehb_premium)),
      medicarePremiumsA: projection.map(y => num(y.medicare_premium_person_a)),
      medicarePremiumsB: projection.map(y => num(y.medicare_premium_person_b)),
      federalTaxes: projection.map(y => num(y.federal_tax)),
      effectiveTaxRates: projection.map(y => {
        const gross = num(y.total_gross_income);
        return gross > 0 ? num(y.federal_tax) / gross : 0;
      }),
      taxBrackets: projection.map(y => y.federal_tax_brackets || []),
    };
  }

  function num(v) {
    if (v === undefined || v === null) return 0;
    if (typeof v === 'string') return parseFloat(v) || 0;
    return v;
  }

  function getYearRange(scenarioData) {
    const allYears = scenarioData.flatMap(s => s.years);
    return {
      min: Math.min(...allYears) - 1,
      max: Math.max(...allYears) + 1,
    };
  }

  function dollarTick(value) {
    return '$' + Math.round(value).toLocaleString();
  }

  function yearAxis(range) {
    return {
      type: 'linear', position: 'bottom',
      title: { display: true, text: 'Year' },
      min: range.min, max: range.max,
      ticks: { stepSize: 5, callback: v => Math.round(v) },
    };
  }

  function createChart(canvasId, config) {
    const canvas = document.getElementById(canvasId);
    if (!canvas) return null;
    const chart = new Chart(canvas.getContext('2d'), config);
    _charts.push(chart);
    return chart;
  }

  // ── Individual Charts ──────────────────────────

  function renderTSPChart(scenarioData) {
    const range = getYearRange(scenarioData);
    createChart('chart-tsp', {
      type: 'line',
      data: {
        datasets: scenarioData.map((s, i) => ({
          label: s.name,
          data: s.years.map((y, j) => ({ x: y, y: s.tspBalances[j] })),
          borderColor: COLORS[i % COLORS.length],
          fill: false, tension: 0.1,
        })),
      },
      options: {
        responsive: true, maintainAspectRatio: false,
        plugins: { title: { display: true, text: 'TSP Balance Over Time' } },
        scales: { x: yearAxis(range), y: { title: { display: true, text: 'TSP Balance ($)' }, ticks: { callback: dollarTick } } },
      },
    });
  }

  function renderIncomeChart(scenarioData) {
    const range = getYearRange(scenarioData);
    createChart('chart-income', {
      type: 'line',
      data: {
        datasets: scenarioData.map((s, i) => ({
          label: s.name,
          data: s.years.map((y, j) => ({ x: y, y: s.netIncomes[j] })),
          borderColor: COLORS[i % COLORS.length],
          fill: false, tension: 0.1,
        })),
      },
      options: {
        responsive: true, maintainAspectRatio: false,
        plugins: { title: { display: true, text: 'Net Income Over Time' } },
        scales: { x: yearAxis(range), y: { title: { display: true, text: 'Net Income ($)' }, ticks: { callback: dollarTick } } },
      },
    });
  }

  function renderFederalTaxChart(scenarioData) {
    const range = getYearRange(scenarioData);
    const datasets = [];

    scenarioData.forEach((scenario, index) => {
      const allBrackets = scenario.taxBrackets.flatMap(yb => (yb || []).map(b => Math.round(num(b.rate) * 100)));
      const uniqueRates = [...new Set(allBrackets)].sort((a, b) => a - b);

      uniqueRates.forEach(rate => {
        const rateStr = rate.toString();
        datasets.push({
          type: 'bar',
          label: `${scenario.name} - ${rate}% Bracket`,
          data: scenario.years.map((year, yi) => {
            const brackets = scenario.taxBrackets[yi] || [];
            const bracket = brackets.find(b => Math.round(num(b.rate) * 100) === rate);
            return { x: year, y: bracket ? num(bracket.tax_from_bracket) : 0 };
          }),
          backgroundColor: BRACKET_COLORS[rateStr] || '#888',
          stack: `stack-${index}`,
          yAxisID: 'yTaxes',
        });
      });

      datasets.push({
        type: 'line',
        label: `${scenario.name} - Effective Rate`,
        data: scenario.years.map((y, i) => ({ x: y, y: scenario.effectiveTaxRates[i] * 100 })),
        borderColor: COLORS[index % COLORS.length],
        borderWidth: 2, fill: false, tension: 0.1,
        yAxisID: 'yRate', pointRadius: 3,
      });
    });

    createChart('chart-federal-tax', {
      type: 'bar',
      data: { datasets },
      options: {
        responsive: true, maintainAspectRatio: false,
        plugins: { title: { display: true, text: 'Federal Taxes by Bracket and Effective Rates' } },
        scales: {
          x: yearAxis(range),
          yTaxes: { type: 'linear', position: 'left', stacked: true, title: { display: true, text: 'Federal Tax ($)' }, ticks: { callback: dollarTick } },
          yRate: { type: 'linear', position: 'right', title: { display: true, text: 'Effective Rate (%)' }, min: 0, max: 30, ticks: { callback: v => v + '%' }, grid: { drawOnChartArea: false } },
        },
      },
    });
  }

  function renderFEHBChart(scenarioData) {
    const range = getYearRange(scenarioData);
    createChart('chart-fehb', {
      type: 'line',
      data: {
        datasets: scenarioData.map((s, i) => ({
          label: s.name,
          data: s.years.map((y, j) => ({ x: y, y: s.fehbPremiums[j] })),
          borderColor: COLORS[i % COLORS.length],
          fill: false, tension: 0.1,
        })),
      },
      options: {
        responsive: true, maintainAspectRatio: false,
        plugins: { title: { display: true, text: 'FEHB Premiums Over Time' } },
        scales: { x: yearAxis(range), y: { title: { display: true, text: 'Annual FEHB Premium ($)' }, ticks: { callback: dollarTick } } },
      },
    });
  }

  function renderMedicareChart(scenarioData) {
    const range = getYearRange(scenarioData);
    const datasets = [];
    scenarioData.forEach((s, i) => {
      const color = COLORS[i % COLORS.length];
      datasets.push({
        label: s.name + ' - Person A',
        data: s.years.map((y, j) => ({ x: y, y: s.medicarePremiumsA[j] })),
        borderColor: color, fill: false, tension: 0.1,
      });
      datasets.push({
        label: s.name + ' - Person B',
        data: s.years.map((y, j) => ({ x: y, y: s.medicarePremiumsB[j] })),
        borderColor: color, borderDash: [6, 4], fill: false, tension: 0.1,
      });
    });

    createChart('chart-medicare', {
      type: 'line',
      data: { datasets },
      options: {
        responsive: true, maintainAspectRatio: false,
        plugins: { title: { display: true, text: 'Medicare Premiums Over Time' } },
        scales: { x: yearAxis(range), y: { title: { display: true, text: 'Annual Medicare Premium ($)' }, ticks: { callback: dollarTick } } },
      },
    });
  }

  function renderCumulativeChart(scenarioData) {
    const range = getYearRange(scenarioData);
    const cumulativeData = scenarioData.map(s => {
      let sum = 0;
      return s.netIncomes.map(n => { sum += n; return sum; });
    });

    const datasets = scenarioData.map((s, i) => ({
      label: s.name + ' (cumulative)',
      data: s.years.map((y, j) => ({ x: y, y: cumulativeData[i][j] })),
      borderColor: COLORS[i % COLORS.length],
      fill: false, tension: 0.1,
    }));

    if (scenarioData.length >= 2) {
      const deltaData = scenarioData[0].years.map((y, i) => ({
        x: y,
        y: (cumulativeData[1]?.[i] || 0) - (cumulativeData[0]?.[i] || 0),
      }));
      datasets.push({
        label: 'Delta (Scenario 2 - 1)',
        data: deltaData,
        borderColor: '#333', borderDash: [6, 4],
        backgroundColor: 'rgba(50,50,50,0.08)',
        fill: true, tension: 0.1,
      });
    }

    createChart('chart-cumulative', {
      type: 'line',
      data: { datasets },
      options: {
        responsive: true, maintainAspectRatio: false,
        plugins: { title: { display: true, text: 'Cumulative Net Income Comparison' } },
        scales: { x: yearAxis(range), y: { title: { display: true, text: 'Cumulative ($)' }, ticks: { callback: dollarTick } } },
      },
    });
  }

  function renderIncomeSourcesCharts(scenarioData) {
    scenarioData.forEach((s, i) => {
      const canvasId = 'chart-sources-' + i;
      const range = {
        min: Math.min(...s.years) - 1,
        max: Math.max(...s.years) + 1,
      };
      createChart(canvasId, {
        type: 'line',
        data: {
          datasets: [
            { label: 'Salary', data: s.years.map((y, j) => ({ x: y, y: s.salaries[j] })), backgroundColor: SOURCE_COLORS.salary + '80', borderColor: SOURCE_COLORS.salary, fill: 'origin' },
            { label: 'FERS Pension', data: s.years.map((y, j) => ({ x: y, y: s.pensions[j] })), backgroundColor: SOURCE_COLORS.pension + '80', borderColor: SOURCE_COLORS.pension, fill: '-1' },
            { label: 'TSP Withdrawal', data: s.years.map((y, j) => ({ x: y, y: s.tspWithdrawals[j] })), backgroundColor: SOURCE_COLORS.tspWithdrawal + '80', borderColor: SOURCE_COLORS.tspWithdrawal, fill: '-1' },
            { label: 'Social Security', data: s.years.map((y, j) => ({ x: y, y: s.socialSecurity[j] })), backgroundColor: SOURCE_COLORS.socialSecurity + '80', borderColor: SOURCE_COLORS.socialSecurity, fill: '-1' },
          ],
        },
        options: {
          responsive: true, maintainAspectRatio: false,
          plugins: { title: { display: true, text: s.name + ' - Income Sources' } },
          scales: {
            x: yearAxis(range),
            y: { title: { display: true, text: 'Annual ($)' }, stacked: true, ticks: { callback: dollarTick } },
          },
        },
      });
    });
  }

  return {
    renderAll,
    destroyAll,
  };
})();
