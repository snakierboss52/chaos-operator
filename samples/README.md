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
