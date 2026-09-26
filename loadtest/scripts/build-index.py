#!/usr/bin/env python3
"""
build-index.py — genera un index.html agregador de todos los reportes de carga.

Lee loadtest/results/*.json, extrae métricas clave (duración, checks rate,
thresholds OK/fail), y construye un index.html con cards ordenadas por fecha
descendente.

Defensivo por diseño:
  - Filenames de reports validados con regex antes de embeber en HTML
    (defense contra path-traversal y XSS).
  - Todo string interpolado pasa por html.escape().
  - JSON parser tolera campos faltantes sin romper.
  - Si un JSON está malformado, se omite con un warning (no aborta el build).
"""

from __future__ import annotations

import argparse
import html
import json
import re
import sys
from pathlib import Path
from datetime import datetime

# Allowlist con grupos: <scenario>-<YYYY-MM-DDThh-mm-ss>.json
# El scenario no puede empezar con dígito ni contener guiones que confundan
# con la fecha; la fecha tiene 6 grupos separados por guiones (4-2-2T2-2-2).
SAFE_FILENAME = re.compile(
    r'^([a-z][a-z0-9]{0,39})-(\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2})\.json$'
)
SAFE_LATEST = re.compile(r'^[a-z][a-z0-9]{0,39}-latest\.json$')


def load_report(path: Path) -> dict | None:
    if SAFE_LATEST.match(path.name):
        # Los aliases -latest.json son duplicados; solo procesamos los timestamped.
        return None
    m = SAFE_FILENAME.match(path.name)
    if not m:
        print(f"⚠  saltado (filename no válido): {path.name}", file=sys.stderr)
        return None
    scenario = m.group(1)
    ts = m.group(2)
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as e:
        print(f"⚠  saltado (JSON inválido): {path.name} — {e}", file=sys.stderr)
        return None

    metrics = data.get('metrics', {})
    checks = (metrics.get('checks') or {}).get('values', {}) or {}
    duration_ms = (data.get('state') or {}).get('testRunDurationMs') or 0

    # Thresholds
    th_total = 0
    th_ok = 0
    for m in metrics.values():
        ths = (m or {}).get('thresholds') or {}
        for info in ths.values():
            th_total += 1
            if info and info.get('ok') is True:
                th_ok += 1

    return {
        'scenario': scenario,
        'timestamp': ts,
        'datetime': parse_ts(ts),
        'duration_s': duration_ms / 1000.0 if duration_ms else 0.0,
        'checks_rate': checks.get('rate'),
        'checks_passes': checks.get('passes', 0),
        'checks_fails': checks.get('fails', 0),
        'thresholds_total': th_total,
        'thresholds_ok': th_ok,
        'all_passed': th_total > 0 and th_ok == th_total,
        'html_filename': path.with_suffix('.html').name,
        'json_filename': path.name,
        'md_filename': path.with_suffix('.md').name,
    }


def parse_ts(ts: str) -> str:
    # 2026-05-05T21-30-00 -> 2026-05-05 21:30:00
    try:
        d = datetime.strptime(ts, '%Y-%m-%dT%H-%M-%S')
        return d.strftime('%Y-%m-%d %H:%M:%S')
    except ValueError:
        return ts


def render_card(r: dict) -> str:
    badge_class = 'badge-pass' if r['all_passed'] else (
        'badge-warn' if r['thresholds_total'] == 0 else 'badge-fail'
    )
    badge_text = 'PASS' if r['all_passed'] else (
        'NO THRESHOLDS' if r['thresholds_total'] == 0 else 'FAIL'
    )
    cr = r.get('checks_rate')
    cr_str = f"{cr * 100:.1f} %" if cr is not None else '—'

    # html.escape sobre cada interpolación (defense in depth, ya validamos arriba)
    return f"""
    <a class="card" href="{html.escape(r['html_filename'])}">
      <div class="card-head">
        <span class="scenario">{html.escape(r['scenario'])}</span>
        <span class="{badge_class}">{badge_text}</span>
      </div>
      <div class="card-meta">{html.escape(r['datetime'])}</div>
      <div class="card-body">
        <div class="kpi"><span class="lbl">Duración</span><span class="val">{r['duration_s']:.1f} s</span></div>
        <div class="kpi"><span class="lbl">Checks</span><span class="val">{html.escape(cr_str)}</span></div>
        <div class="kpi"><span class="lbl">Thresholds</span><span class="val">{r['thresholds_ok']}/{r['thresholds_total']}</span></div>
      </div>
      <div class="card-foot">
        <span class="link-hint">HTML</span>
        <a class="alt" href="{html.escape(r['json_filename'])}">JSON</a>
        <a class="alt" href="{html.escape(r['md_filename'])}">MD</a>
      </div>
    </a>"""


HTML_TEMPLATE = """<!doctype html>
<html lang="es">
<head>
<meta charset="utf-8">
<title>Reportes de carga — Chaos Engineering Operator</title>
<style>
  :root {{ --navy:#1F3864; --blue:#2E75B6; --gray:#6B7280; --bg:#F8FAFC; --ok:#16A34A; --err:#DC2626; --warn:#D97706; }}
  * {{ box-sizing: border-box; }}
  body {{ font-family: -apple-system, "Segoe UI", Arial, sans-serif; margin: 0; padding: 24px 40px; background: var(--bg); color: #1F2937; }}
  h1 {{ color: var(--navy); margin: 0 0 4px; }}
  .subtitle {{ color: var(--gray); margin: 0 0 24px; font-size: 14px; }}
  .grid {{ display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 16px; }}
  .card {{ background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.06); padding: 16px; text-decoration: none; color: inherit; transition: transform .1s, box-shadow .1s; }}
  .card:hover {{ transform: translateY(-2px); box-shadow: 0 4px 12px rgba(0,0,0,0.08); }}
  .card-head {{ display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }}
  .scenario {{ font-weight: 600; color: var(--navy); font-size: 16px; }}
  .badge-pass, .badge-fail, .badge-warn {{ font-size: 11px; padding: 3px 10px; border-radius: 12px; font-weight: 700; letter-spacing: .3px; }}
  .badge-pass {{ background: #DCFCE7; color: var(--ok); }}
  .badge-fail {{ background: #FEE2E2; color: var(--err); }}
  .badge-warn {{ background: #FEF3C7; color: var(--warn); }}
  .card-meta {{ color: var(--gray); font-size: 12px; margin-bottom: 12px; font-variant-numeric: tabular-nums; }}
  .card-body {{ display: flex; gap: 14px; padding: 8px 0; border-top: 1px solid #F1F5F9; border-bottom: 1px solid #F1F5F9; }}
  .kpi {{ display: flex; flex-direction: column; flex: 1; }}
  .lbl {{ font-size: 10px; color: var(--gray); text-transform: uppercase; letter-spacing: .5px; }}
  .val {{ font-size: 14px; color: var(--navy); font-weight: 600; font-variant-numeric: tabular-nums; }}
  .card-foot {{ margin-top: 10px; display: flex; gap: 12px; font-size: 12px; }}
  .link-hint {{ color: var(--blue); font-weight: 600; }}
  .alt {{ color: var(--gray); text-decoration: none; }}
  .alt:hover {{ color: var(--blue); }}
  footer {{ color: var(--gray); font-size: 12px; margin-top: 40px; text-align: center; }}
  .empty {{ background: white; padding: 40px; text-align: center; border-radius: 8px; color: var(--gray); }}
</style>
</head>
<body>
<h1>Reportes de carga — Chaos Engineering Operator</h1>
<p class="subtitle">Generado: {generated} · {count} reportes</p>
{body}
<footer>Generado por loadtest/scripts/build-index.py — sin recursos externos.</footer>
</body>
</html>"""


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--results-dir', default='loadtest/results',
                        help='Directorio con los reports (default: loadtest/results)')
    parser.add_argument('--output', default='loadtest/results/index.html',
                        help='Archivo de salida (default: loadtest/results/index.html)')
    args = parser.parse_args()

    results_dir = Path(args.results_dir).resolve()
    output_path = Path(args.output).resolve()

    # Defense contra path traversal en el output
    if not output_path.parent.exists():
        print(f"❌ El directorio del output no existe: {output_path.parent}", file=sys.stderr)
        sys.exit(1)

    if not results_dir.is_dir():
        print(f"❌ {results_dir} no existe.", file=sys.stderr)
        sys.exit(1)

    reports = []
    for p in results_dir.glob('*.json'):
        r = load_report(p)
        if r is not None:
            reports.append(r)

    reports.sort(key=lambda r: r['timestamp'], reverse=True)

    if not reports:
        body = '<div class="empty">No hay reportes todavía. Corré <code>make load-smoke</code>.</div>'
    else:
        body = '<div class="grid">' + ''.join(render_card(r) for r in reports) + '</div>'

    out = HTML_TEMPLATE.format(
        generated=html.escape(datetime.now().strftime('%Y-%m-%d %H:%M:%S')),
        count=len(reports),
        body=body,
    )
    output_path.write_text(out, encoding='utf-8')
    print(f"✓ index.html generado: {output_path} ({len(reports)} reportes)")


if __name__ == '__main__':
    main()
