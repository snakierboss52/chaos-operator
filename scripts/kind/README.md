# Cluster local kind

```bash
make kind-up       # crea chaos-testing-v2 si falta y espera nodos Ready
make kind-status   # consulta nodos del cluster
make kind-down     # elimina el cluster
```

La creación usa el archivo `kind-cluster.yaml`. Puedes seleccionar otro archivo
de configuración (ruta relativa a la raíz del repo o absoluta) y un nombre:

```bash
make kind-up KIND_CONFIG=config/kind/dev.yaml KIND_CLUSTER=chaos-dev
```

kind solo consume ese archivo cuando crea el cluster; `kind-up` no modifica la
configuración de un cluster existente.

`make load-up` reutiliza el mismo script y respeta ambas variables.

## Compilar y cargar la imagen del operador

Los pasos se pueden ejecutar por separado:

```bash
make build                                      # binario local en bin/goland-operator
make docker-build IMG=goland-operator:dev       # imagen del contenedor
make kind-load-image IMG=goland-operator:dev    # carga imagen existente al cluster
make install-crds
make deploy IMG=goland-operator:dev             # aplica esa misma imagen
```

`docker-build` construye la imagen mediante el Dockerfile (compila el binario
Linux dentro del build de Docker). `make build` genera el binario host para
ejecutarlo localmente; son salidas independientes. `make load-deploy` sigue
disponible como flujo integrado y usa esos mismos pasos de imagen.
