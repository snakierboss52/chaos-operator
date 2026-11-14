# Inicio rápido

Esta guía histórica se consolidó en
[Primeros pasos](GETTING_STARTED.md). Para levantar el laboratorio kind usa:

```bash
make kind-up
make docker-build
make kind-load-image
make install-crds
make deploy
make addons-install
make lab-deploy
```

Después aplica un manifiesto de `samples/` y consulta su estado con kubectl.
No uses los antiguos scripts `install.sh`/`uninstall.sh` ni los targets
`make install`/`make manifests`: no forman parte del flujo actual. Para conocer
los comandos y la limpieza, consulta [COMMANDS](COMMANDS.md) e
[INSTALLATION](INSTALLATION.md).
