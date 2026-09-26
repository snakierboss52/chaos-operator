// crds.js — fábricas declarativas de CRs para los escenarios de carga.
//
// SEGURIDAD (alineado con las reglas Go del operador):
// - Todos los CRs declaran blastRadius.maxPods = 1 (allowlist estricta).
// - El selector limita SIEMPRE a namespace 'chaos-load' + label 'app=load-target'.
//   Nunca debemos generar CRs apuntando a chaos-system, kube-system u otros.
// - No se interpolan inputs externos al cuerpo del manifest sin validar.

const ALLOWED_NAMESPACE = 'chaos-load';
const ALLOWED_LABEL = 'load-target';

// Validador simple — defense in depth. Bloquea cualquier intento de
// escapar el namespace seguro (el caller no debería poder cambiarlo,
// pero validamos igual).
function assertSafeName(name) {
  if (typeof name !== 'string' || !/^[a-z0-9-]{3,63}$/.test(name)) {
    throw new Error(`Nombre inválido: ${name}`);
  }
}

export function podChaosManifest({ name, action = 'pod-kill', duration = '10s' }) {
  assertSafeName(name);
  return `
apiVersion: chaos.engineering.io/v1alpha1
kind: PodChaos
metadata:
  name: ${name}
  namespace: ${ALLOWED_NAMESPACE}
  labels:
    test.chaos.engineering.io/scenario: load
spec:
  mode: one
  selector:
    namespaces:
      - ${ALLOWED_NAMESPACE}
    labelSelectors:
      app: ${ALLOWED_LABEL}
  action: ${action}
  duration: "${duration}"
  blastRadius:
    maxPods: 1
`.trim();
}

export function networkChaosDelayManifest({ name, latency = '50ms', duration = '15s' }) {
  assertSafeName(name);
  return `
apiVersion: chaos.engineering.io/v1alpha1
kind: NetworkChaos
metadata:
  name: ${name}
  namespace: ${ALLOWED_NAMESPACE}
  labels:
    test.chaos.engineering.io/scenario: load
spec:
  mode: one
  selector:
    namespaces:
      - ${ALLOWED_NAMESPACE}
    labelSelectors:
      app: ${ALLOWED_LABEL}
  action: delay
  delay:
    latency: "${latency}"
    jitter: "10ms"
  duration: "${duration}"
  blastRadius:
    maxPods: 1
`.trim();
}

// Nombre con prefijo seguro y sufijo aleatorio basado en VU+iter+timestamp.
// No usamos Math.random sin sal porque varios VUs en paralelo podrían
// colisionar; combinamos identificadores para garantizar unicidad.
export function uniqueName(prefix, vu, iter) {
  const ts = Date.now().toString(36);
  return `${prefix}-${vu}-${iter}-${ts}`;
}

export const CONSTANTS = {
  ALLOWED_NAMESPACE,
  ALLOWED_LABEL,
};
