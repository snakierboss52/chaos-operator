# Samples - Chaos Engineering Operator

Este directorio contiene ejemplos de manifiestos YAML para diferentes tipos de experimentos de chaos que puedes usar como punto de partida para tus propios experimentos.

## Estructura

```
samples/
├── podchaos/          # Ejemplos de PodChaos
├── networkchaos/      # Ejemplos de NetworkChaos
├── stresschaos/       # Ejemplos de StressChaos
└── httpchaos/         # Ejemplos de HTTPChaos
```

## Workloads del laboratorio local

Los manifiestos `workloads/nginx-target.yaml` y `workloads/apache-target.yaml`
despliegan los objetivos de experimentos en `chaos-demo`: seis réplicas de
Nginx y dos de Apache. Cada pod solicita 50m CPU y 64Mi de memoria, con límites
de 200m CPU y 128Mi; esto mantiene un tamaño moderado y deja suficiente
replicación para observar fallos y recuperación.

Desde la raíz del repositorio, `make lab-deploy` construye y carga la imagen
local de Apache (incluye `tc` e `iptables` para NetworkChaos), aplica ambos
deployments y espera a que estén listos. Requiere tener Docker, kind y kubectl
instalados y usar el cluster kind del proyecto. `make lab-clean` elimina solo
estos workloads y conserva el cluster, el operator y el monitoring.

Estos targets son independientes de `make addons-install` y del escenario de
carga `make load-deploy`. Los experimentos de ejemplo pueden apuntar a
`namespace: chaos-demo` con `app: nginx` o `app: apache-target`.

## PodChaos

### basic-kill-one.yaml
Ejemplo básico que elimina exactamente un pod.
```bash
kubectl apply -f podchaos/basic-kill-one.yaml
```

### kill-percentage.yaml
Elimina un porcentaje de pods con control de blast radius.
```bash
kubectl apply -f podchaos/kill-percentage.yaml
```

### container-kill.yaml
Elimina contenedores específicos (útil para sidecars).
```bash
kubectl apply -f podchaos/container-kill.yaml
```

## NetworkChaos

### delay.yaml
Añade latencia de red entre servicios.
```bash
kubectl apply -f networkchaos/delay.yaml
```

### packet-loss.yaml
Simula pérdida de paquetes.
```bash
kubectl apply -f networkchaos/packet-loss.yaml
```

### partition.yaml
Crea particiones de red (split-brain scenarios).
```bash
kubectl apply -f networkchaos/partition.yaml
```

## StressChaos

### cpu-stress.yaml
Genera estrés de CPU.
```bash
kubectl apply -f stresschaos/cpu-stress.yaml
```

### memory-stress.yaml
Genera estrés de memoria.
```bash
kubectl apply -f stresschaos/memory-stress.yaml
```

### combined-stress.yaml
Estrés combinado de CPU y memoria.
```bash
kubectl apply -f stresschaos/combined-stress.yaml
```

## HTTPChaos

### abort-503.yaml
Aborta requests HTTP con código 503.
```bash
kubectl apply -f httpchaos/abort-503.yaml
```

### delay.yaml
Añade latencia a requests HTTP.
```bash
kubectl apply -f httpchaos/delay.yaml
```

### replace-response.yaml
Reemplaza responses HTTP.
```bash
kubectl apply -f httpchaos/replace-response.yaml
```

## Uso Rápido

Para probar un experimento:

1. **Preparar aplicación de prueba**:
```bash
kubectl create namespace demo
kubectl create deployment nginx --image=nginx --replicas=3 -n demo
```

2. **Aplicar experimento**:
```bash
kubectl apply -f podchaos/basic-kill-one.yaml
```

3. **Observar resultados**:
```bash
kubectl get pods -n demo -w
kubectl get podchaos -n demo
kubectl describe podchaos basic-kill-one -n demo
```

4. **Limpiar**:
```bash
kubectl delete -f podchaos/basic-kill-one.yaml
```

## Personalización

Modifica los ejemplos según tus necesidades:

- **Selector**: Cambia labels y namespaces
- **Mode**: Ajusta el modo de selección (one, all, fixed, fixed-percent)
- **Duration**: Modifica la duración del experimento
- **BlastRadius**: Configura límites de seguridad

## Ver También

- [Documentación Completa](../docs/)
- [Getting Started](../docs/GETTING_STARTED.md)
- [Tipos de Chaos](../docs/CHAOS_TYPES.md)
