// report.js — generador unificado de reportes para los escenarios k6.
//
// Produce 4 artefactos por corrida:
//   - stdout (resumen en consola)
//   - loadtest/results/<scenario>-<ts>.json      (datos crudos para CI)
//   - loadtest/results/<scenario>-<ts>.html      (reporte humano, self-contained)
//   - loadtest/results/<scenario>-<ts>.md        (resumen para PR/changelog)
//   - loadtest/results/<scenario>-latest.json    (alias al último, para tooling)
//
// SEGURIDAD:
//   - El nombre del escenario se valida contra /^[a-z0-9-]{1,40}$/ antes de
//     componer rutas (defense against path traversal — CWE-22).
//   - El HTML escapa todos los strings interpolados (defense against XSS —
//     CWE-79). Aunque los metric names son hardcoded, defense-in-depth.
//   - No se importan recursos externos (no CDN, no fetch). Todo inline.
//   - El JSON crudo no incluye headers ni respuestas del API server,
//     solo métricas agregadas y umbrales (sin riesgo de fuga de tokens).

function isoTimestamp() {
  // 2026-05-05T21-30-00 (sin caracteres conflictivos en filenames)
  return new Date()
    .toISOString()
    .replace(/:/g, '-')
    .replace(/\.[0-9]+Z?$/, '')
    .replace('Z', '');
}

function safeName(name) {
  if (typeof name !== 'string' || !/^[a-z0-9-]{1,40}$/.test(name)) {
    throw new Error(`scenario name inválido: ${name}`);
  }
  return name;
}

function escapeHtml(s) {
  if (s === null || s === undefined) return '';
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function fmt(n, digits = 2) {
  if (n === null || n === undefined || !Number.isFinite(n)) return '—';
  if (Math.abs(n) >= 1000) return n.toFixed(0);
  return n.toFixed(digits);
}

function pickMetric(data, name) {
  const m = data && data.metrics && data.metrics[name];
  if (!m || !m.values) return {};
  return m.values;
}

function thresholdState(data) {
  // Devuelve [{name, ok}] para todos los thresholds declarados.
  const out = [];
  const metrics = (data && data.metrics) || {};
  for (const [metricName, m] of Object.entries(metrics)) {
    const ths = m.thresholds || {};
    for (const [expr, info] of Object.entries(ths)) {
      out.push({
        metric: metricName,
        expr: expr,
        ok: info && info.ok === true,
      });
    }
  }
  return out;
}

// ----------------------------------------------------
// stdout (texto plano para la terminal)
// ----------------------------------------------------
function textSummary(data, scenarioName, ts) {
  const lines = [];
  lines.push('');
  lines.push('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  lines.push(`  Reporte: ${scenarioName}`);
  lines.push(`  Timestamp: ${ts}`);
  lines.push('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

  const dur = (data.state && data.state.testRunDurationMs) || 0;
  lines.push(`  Duración total: ${(dur / 1000).toFixed(1)} s`);

  const checks = pickMetric(data, 'checks');
  if (checks.rate !== undefined) {
    lines.push(`  Checks rate: ${(checks.rate * 100).toFixed(2)} %  ` +
      `(${checks.passes || 0} pass / ${checks.fails || 0} fail)`);
  }

  // Métricas custom de carga
  const customNames = [
    'chaos_load_crs_created',
    'chaos_load_crs_reconciled',
    'chaos_load_crs_completed',
    'chaos_load_crs_failed',
    'chaos_load_crs_timeout',
    'chaos_load_crs_apply_errors',
  ];
  const haveCustom = customNames.some((n) => pickMetric(data, n).count !== undefined);
  if (haveCustom) {
    lines.push('  ─── Métricas de carga ───');
    for (const n of customNames) {
      const v = pickMetric(data, n);
      if (v.count !== undefined) {
        lines.push(`    ${n.padEnd(36)} ${v.count}`);
      }
    }
  }

  // Trends (latencias)
  const trendNames = ['chaos_load_reconcile_wait_ms', 'chaos_load_apply_duration_ms'];
  const haveTrends = trendNames.some((n) => pickMetric(data, n).avg !== undefined);
  if (haveTrends) {
    lines.push('  ─── Latencias (ms) ───');
    for (const n of trendNames) {
      const v = pickMetric(data, n);
      if (v.avg !== undefined) {
        lines.push(
          `    ${n.padEnd(36)} avg=${fmt(v.avg)} p95=${fmt(v['p(95)'])} p99=${fmt(v['p(99)'])} max=${fmt(v.max)}`
        );
      }
    }
  }

  // Thresholds
  const ths = thresholdState(data);
  if (ths.length > 0) {
    lines.push('  ─── Thresholds ───');
    for (const t of ths) {
      const mark = t.ok ? '✓' : '✗';
      lines.push(`    ${mark} ${t.metric}: ${t.expr}`);
    }
  }

  lines.push('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  lines.push('');
  return lines.join('\n');
}

// ----------------------------------------------------
// Markdown
// ----------------------------------------------------
function mdReport(data, scenarioName, ts) {
  const lines = [];
  lines.push(`# Reporte de carga — ${scenarioName}`);
  lines.push('');
  lines.push(`**Timestamp:** ${ts}  `);
  const dur = (data.state && data.state.testRunDurationMs) || 0;
  lines.push(`**Duración:** ${(dur / 1000).toFixed(1)} s  `);

  const checks = pickMetric(data, 'checks');
  if (checks.rate !== undefined) {
    lines.push(
      `**Checks:** ${(checks.rate * 100).toFixed(2)} % (${checks.passes || 0}/${(checks.passes || 0) + (checks.fails || 0)})`
    );
  }
  lines.push('');

  // Métricas custom
  const customNames = [
    'chaos_load_crs_created',
    'chaos_load_crs_reconciled',
    'chaos_load_crs_completed',
    'chaos_load_crs_failed',
    'chaos_load_crs_timeout',
    'chaos_load_crs_apply_errors',
  ];
  lines.push('## Métricas de carga');
  lines.push('');
  lines.push('| Métrica | Valor |');
  lines.push('|---------|------:|');
  for (const n of customNames) {
    const v = pickMetric(data, n);
    if (v.count !== undefined) lines.push(`| \`${n}\` | ${v.count} |`);
  }
  lines.push('');

  // Latencias
  lines.push('## Latencias');
  lines.push('');
  lines.push('| Métrica (ms) | avg | p95 | p99 | max |');
  lines.push('|--------------|----:|----:|----:|----:|');
  for (const n of ['chaos_load_reconcile_wait_ms', 'chaos_load_apply_duration_ms']) {
    const v = pickMetric(data, n);
    if (v.avg !== undefined) {
      lines.push(`| \`${n}\` | ${fmt(v.avg)} | ${fmt(v['p(95)'])} | ${fmt(v['p(99)'])} | ${fmt(v.max)} |`);
    }
  }
  lines.push('');

  // Thresholds
  const ths = thresholdState(data);
  if (ths.length > 0) {
    lines.push('## Thresholds');
    lines.push('');
    lines.push('| Estado | Métrica | Condición |');
    lines.push('|:------:|---------|-----------|');
    for (const t of ths) {
      const mark = t.ok ? '✅' : '❌';
      lines.push(`| ${mark} | \`${t.metric}\` | \`${t.expr}\` |`);
    }
    lines.push('');
  }

  lines.push('---');
  lines.push('');
  lines.push('Reporte generado por `loadtest/scripts/lib/report.js`.');
  return lines.join('\n');
}

// ----------------------------------------------------
// HTML (self-contained, sin CDN ni fetch)
// ----------------------------------------------------
function htmlReport(data, scenarioName, ts) {
  const dur = (data.state && data.state.testRunDurationMs) || 0;
  const checks = pickMetric(data, 'checks');
  const ths = thresholdState(data);
  const allPassed = ths.length > 0 && ths.every((t) => t.ok);

  const customRows = [];
  for (const n of [
    'chaos_load_crs_created',
    'chaos_load_crs_reconciled',
    'chaos_load_crs_completed',
    'chaos_load_crs_failed',
    'chaos_load_crs_timeout',
    'chaos_load_crs_apply_errors',
  ]) {
    const v = pickMetric(data, n);
    if (v.count !== undefined) {
      customRows.push(
        `<tr><td><code>${escapeHtml(n)}</code></td><td class="num">${v.count}</td></tr>`
      );
    }
  }

  const trendRows = [];
  for (const n of ['chaos_load_reconcile_wait_ms', 'chaos_load_apply_duration_ms']) {
    const v = pickMetric(data, n);
    if (v.avg !== undefined) {
      trendRows.push(
        `<tr><td><code>${escapeHtml(n)}</code></td>` +
          `<td class="num">${fmt(v.avg)}</td>` +
          `<td class="num">${fmt(v['p(95)'])}</td>` +
          `<td class="num">${fmt(v['p(99)'])}</td>` +
          `<td class="num">${fmt(v.max)}</td></tr>`
      );
    }
  }

  const thRows = ths
    .map(
      (t) =>
        `<tr><td>${t.ok ? '<span class="ok">✓</span>' : '<span class="err">✗</span>'}</td>` +
        `<td><code>${escapeHtml(t.metric)}</code></td>` +
        `<td><code>${escapeHtml(t.expr)}</code></td></tr>`
    )
    .join('');

  return `<!doctype html>
<html lang="es">
<head>
<meta charset="utf-8">
<title>Reporte de carga — ${escapeHtml(scenarioName)} (${escapeHtml(ts)})</title>
<style>
  :root { --navy:#1F3864; --blue:#2E75B6; --gray:#6B7280; --bg:#F8FAFC; --ok:#16A34A; --err:#DC2626; }
  body { font-family: -apple-system, "Segoe UI", Arial, sans-serif; margin: 0; padding: 24px 40px; background: var(--bg); color: #1F2937; }
  h1 { color: var(--navy); margin-bottom: 4px; }
  h2 { color: var(--blue); border-bottom: 2px solid #E5E7EB; padding-bottom: 4px; margin-top: 32px; }
  .meta { color: var(--gray); font-size: 14px; margin-bottom: 24px; }
  .badge { display: inline-block; padding: 4px 12px; border-radius: 12px; font-weight: 600; font-size: 13px; }
  .badge.ok { background: #DCFCE7; color: var(--ok); }
  .badge.err { background: #FEE2E2; color: var(--err); }
  table { width: 100%; border-collapse: collapse; margin-top: 12px; background: white; border-radius: 6px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.05); }
  th { background: var(--navy); color: white; text-align: left; padding: 8px 12px; font-size: 13px; }
  td { padding: 8px 12px; border-bottom: 1px solid #F1F5F9; font-size: 14px; }
  td.num { text-align: right; font-variant-numeric: tabular-nums; }
  tr:last-child td { border-bottom: none; }
  tr:nth-child(even) td { background: #FAFAFA; }
  code { background: #F1F5F9; padding: 2px 6px; border-radius: 4px; font-size: 12.5px; }
  .ok { color: var(--ok); font-weight: bold; }
  .err { color: var(--err); font-weight: bold; }
  .summary-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 12px; margin-top: 12px; }
  .card { background: white; padding: 12px 16px; border-radius: 6px; box-shadow: 0 1px 3px rgba(0,0,0,0.05); }
  .card .label { color: var(--gray); font-size: 12px; text-transform: uppercase; letter-spacing: 0.5px; }
  .card .value { color: var(--navy); font-size: 22px; font-weight: 600; margin-top: 4px; }
  footer { color: var(--gray); font-size: 12px; margin-top: 40px; text-align: center; }
</style>
</head>
<body>
<h1>Reporte de carga — ${escapeHtml(scenarioName)}</h1>
<div class="meta">
  Timestamp: <code>${escapeHtml(ts)}</code>
  &nbsp;&middot;&nbsp;
  Estado:
  ${
    ths.length === 0
      ? '<span class="badge">sin thresholds</span>'
      : allPassed
        ? '<span class="badge ok">PASS</span>'
        : '<span class="badge err">FAIL</span>'
  }
</div>

<div class="summary-grid">
  <div class="card"><div class="label">Duración</div><div class="value">${fmt(dur / 1000, 1)} s</div></div>
  ${
    checks.rate !== undefined
      ? `<div class="card"><div class="label">Checks rate</div><div class="value">${(checks.rate * 100).toFixed(2)} %</div></div>` +
        `<div class="card"><div class="label">Checks pass / fail</div><div class="value">${checks.passes || 0} / ${checks.fails || 0}</div></div>`
      : ''
  }
  <div class="card"><div class="label">Thresholds</div><div class="value">${ths.filter((t) => t.ok).length} / ${ths.length}</div></div>
</div>

${
  customRows.length > 0
    ? `<h2>Métricas de carga</h2>
<table><thead><tr><th>Métrica</th><th class="num">Valor</th></tr></thead><tbody>${customRows.join('')}</tbody></table>`
    : ''
}

${
  trendRows.length > 0
    ? `<h2>Latencias (ms)</h2>
<table><thead><tr><th>Métrica</th><th class="num">avg</th><th class="num">p95</th><th class="num">p99</th><th class="num">max</th></tr></thead><tbody>${trendRows.join('')}</tbody></table>`
    : ''
}

${
  thRows
    ? `<h2>Thresholds</h2>
<table><thead><tr><th></th><th>Métrica</th><th>Condición</th></tr></thead><tbody>${thRows}</tbody></table>`
    : ''
}

<footer>Generado por loadtest/scripts/lib/report.js — sin recursos externos.</footer>
</body>
</html>`;
}

// ----------------------------------------------------
// Public API
// ----------------------------------------------------
export function buildSummary(data, scenarioName) {
  const safe = safeName(scenarioName);
  const ts = isoTimestamp();
  const base = `loadtest/results/${safe}-${ts}`;
  const stdoutText = textSummary(data, safe, ts);
  const json = JSON.stringify(data, null, 2);
  return {
    'stdout': stdoutText,
    [`${base}.json`]: json,
    [`${base}.html`]: htmlReport(data, safe, ts),
    [`${base}.md`]: mdReport(data, safe, ts),
    [`loadtest/results/${safe}-latest.json`]: json,
    [`loadtest/results/${safe}-latest.html`]: htmlReport(data, safe, ts),
  };
}
