#!/usr/bin/env bash
set -euo pipefail

# Chaos Experiment Results Reporter
# Usage: ./scripts/chaos-report.sh [--namespace NS] [--json]

NAMESPACE="${CHAOS_NAMESPACE:-chaos-demo}"
JSON_OUTPUT=false

while [[ $# -gt 0 ]]; do
  case $1 in
    --namespace|-n) NAMESPACE="$2"; shift 2 ;;
    --json)         JSON_OUTPUT=true; shift ;;
    --help|-h)
      echo "Usage: $0 [--namespace NS] [--json]"
      echo "  --namespace, -n   Namespace to query (default: chaos-demo)"
      echo "  --json            Output full JSON results"
      exit 0 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

# Fetch all chaos-report ConfigMaps
CMS=$(kubectl get configmap -n "$NAMESPACE" \
  -l "app.kubernetes.io/managed-by=chaos-operator" \
  -o json 2>/dev/null)

COUNT=$(echo "$CMS" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('items',[])))" 2>/dev/null || echo "0")

if [[ "$COUNT" == "0" ]]; then
  echo "No experiment reports found in namespace '$NAMESPACE'."
  echo "Run experiments first, then check again after they complete."
  exit 0
fi

if [[ "$JSON_OUTPUT" == "true" ]]; then
  echo "$CMS" | python3 -c "
import sys, json

data = json.load(sys.stdin)
reports = []
for item in data.get('items', []):
    d = item.get('data', {})
    result = json.loads(d.get('result.json', '{}'))
    # Collect all detail keys (exclude known metadata keys)
    meta_keys = {'result.json','summary','type','phase','message','startTime','completionTime','duration','affectedPods'}
    details = {k: v for k, v in d.items() if k not in meta_keys}
    pods = json.loads(d.get('affectedPods', '[]')) if d.get('affectedPods') else []
    reports.append({
        'experiment': item['metadata']['labels'].get('chaos.engineering.io/experiment', ''),
        'type': d.get('type', ''),
        'phase': d.get('phase', ''),
        'duration': d.get('duration', ''),
        'startTime': d.get('startTime', ''),
        'completionTime': d.get('completionTime', ''),
        'message': d.get('message', ''),
        'affectedPods': pods,
        'result': result,
        'details': details,
    })
print(json.dumps(reports, indent=2))
"
  exit 0
fi

# Detailed output
echo "$CMS" | python3 -c "
import sys, json

data = json.load(sys.stdin)
items = data.get('items', [])

# Group by type
groups = {}
for item in items:
    d = item.get('data', {})
    ctype = d.get('type', 'Unknown')
    groups.setdefault(ctype, []).append((item, d))

# Known metadata keys to exclude from details
meta_keys = {'result.json','summary','type','phase','message','startTime','completionTime','duration','affectedPods'}

# Type descriptions
type_desc = {
    'PodChaos': 'Pod lifecycle disruption (kill, restart)',
    'NetworkChaos': 'Network fault injection (delay, loss, partition)',
    'StressChaos': 'Resource stress (CPU, memory, disk)',
    'HTTPChaos': 'HTTP fault injection (abort, delay, replace)',
}

total_exp = len(items)
total_completed = 0
total_failed = 0

print()
print('=' * 80)
print('  CHAOS EXPERIMENT RESULTS REPORT')
print(f'  Namespace: {items[0][\"metadata\"][\"namespace\"]}' if items else '')
print('=' * 80)

for ctype in sorted(groups.keys()):
    experiments = groups[ctype]
    desc = type_desc.get(ctype, '')

    print()
    print(f'--- {ctype} ({len(experiments)} experiments) ---')
    if desc:
        print(f'    {desc}')
    print()

    for item, d in sorted(experiments, key=lambda x: x[1].get('startTime', '')):
        name = item['metadata']['labels'].get('chaos.engineering.io/experiment', '?')
        phase = d.get('phase', '?')
        duration = d.get('duration', '-')
        message = d.get('message', '')
        start = d.get('startTime', '-')
        end = d.get('completionTime', '-')
        result = json.loads(d.get('result.json', '{}'))
        pods = json.loads(d.get('affectedPods', '[]')) if d.get('affectedPods') else []

        if phase == 'Completed':
            total_completed += 1
            phase_icon = '[OK]'
        elif phase == 'Failed':
            total_failed += 1
            phase_icon = '[FAIL]'
        else:
            phase_icon = f'[{phase.upper()}]'

        print(f'  {phase_icon} {name}')
        print(f'      Phase: {phase}  |  Duration: {duration}  |  Start: {start}')

        # Targets
        targets = result.get('totalTargets', 0)
        success = result.get('successfulTargets', 0)
        failed = result.get('failedTargets', 0)
        if targets > 0:
            print(f'      Targets: {targets} total, {success} success, {failed} failed')

        if pods:
            print(f'      Affected pods: {', '.join(pods)}')

        if message:
            print(f'      Message: {message}')

        # Type-specific details
        details = {k: v for k, v in d.items() if k not in meta_keys and v}
        if details:
            detail_parts = []
            # Order by relevance per type
            order = ['action','mode','value','direction','target','port','method','path',
                     'duration','gracePeriod',
                     'delay.latency','delay.jitter','delay.percentage',
                     'loss.percent',
                     'abort.statusCode','abort.percentage',
                     'replace.statusCode',
                     'cpu.workers','cpu.load','memory.workers','memory.size',
                     'injectionCount']
            shown = set()
            for key in order:
                if key in details:
                    detail_parts.append(f'{key}={details[key]}')
                    shown.add(key)
            for key in sorted(details.keys()):
                if key not in shown:
                    detail_parts.append(f'{key}={details[key]}')
            print(f'      Config: {', '.join(detail_parts)}')
        print()

print('-' * 80)
print(f'  Total: {total_exp} experiments | {total_completed} completed | {total_failed} failed')
print('-' * 80)
print()
"
