# Cluster local kind

```bash
make kind-up       # crea chaos-testing-v2 si falta y espera nodos Ready
make kind-status   # consulta nodos del cluster
make kind-down     # elimina el cluster
```

Se puede cambiar el nombre con `KIND_CLUSTER=nombre make kind-up`. La creación
usa `kind-cluster.yaml`. `make load-up` reutiliza el mismo script.
