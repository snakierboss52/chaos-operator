SHELL := /bin/bash

APP_NAME := goland-operator
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*')
IMG ?= goland-operator:latest

# Pruebas de carga (k6 + xk6-kubernetes). Targets: load-up, load-deploy,
# load-smoke, load-steady, load-spike, load-clean, load-down. Ver loadtest/runbook.md.
include loadtest/load.mk

.PHONY: help
help:
	@echo "Common targets:"
	@echo "  make build             - Build binary"
	@echo "  make test              - Run unit tests"
	@echo "  make lint              - Run linters"
	@echo "  make tidy              - Go mod tidy"
	@echo "  make run               - Run the manager locally"
	@echo "  make generate          - Generate deepcopy code"
	@echo ""
	@echo "Kubernetes deployment targets:"
	@echo "  make generate-crds     - Generate CRD manifests"
	@echo "  make docker-build      - Build Docker image"
	@echo "  make docker-push       - Push Docker image"
	@echo "  make install-crds      - Install CRDs to cluster"
	@echo "  make uninstall-crds    - Uninstall CRDs from cluster"
	@echo "  make deploy            - Deploy operator to cluster"
	@echo "  make undeploy          - Remove operator from cluster"
	@echo "  make install-all       - Generate CRDs, build image and deploy"
	@echo "  make kind-up           - Create the local kind cluster"
	@echo "  make kind-status       - Show local kind cluster nodes"
	@echo "  make kind-down         - Delete the local kind cluster"
	@echo "  make addons-install    - Install local monitoring addons with Helm"
	@echo "  make addons-uninstall  - Remove local monitoring Helm releases"
	@echo ""
	@echo "Load testing targets (run 'make load-help' for details):"
	@echo "  make load-up           - Create kind cluster (idempotent)"
	@echo "  make load-deploy       - Build + deploy operator + monitoring + workloads"
	@echo "  make load-smoke        - Smoke test (30s)"
	@echo "  make load-steady       - Steady load (5min, 5 VUs)"
	@echo "  make load-spike        - Spike test (ramp 0→100 VUs)"
	@echo "  make load-down         - Clean up load workloads"

.PHONY: build
build:
	@mkdir -p bin
	go build -o bin/$(APP_NAME) ./cmd/manager

.PHONY: test
test:
	@mkdir -p coverage
	go test ./... -coverprofile=coverage/cover.out
	go tool cover -func=coverage/cover.out | tail -n 1 || true

.PHONY: lint
lint:
	@golangci-lint run ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: run
run:
	go run ./cmd/manager

.PHONY: generate
generate:
	@echo "Generating deepcopy code..."
	@if ! command -v controller-gen &> /dev/null; then \
		echo "Installing controller-gen..."; \
		go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest; \
	fi
	@CONTROLLER_GEN=$$(which controller-gen 2>/dev/null || echo "$$HOME/go/bin/controller-gen"); \
	$$CONTROLLER_GEN object:headerFile="hack/boilerplate.go.txt" paths="./api/..."

.PHONY: generate-crds
generate-crds:
	@chmod +x hack/generate-crds.sh
	@hack/generate-crds.sh

.PHONY: docker-build
docker-build:
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push:
	docker push ${IMG}

.PHONY: install-crds
install-crds: generate-crds
	@echo "Installing CRDs..."
	kubectl apply -f config/crd/

.PHONY: uninstall-crds
uninstall-crds:
	@echo "Uninstalling CRDs..."
	kubectl delete -f config/crd/ --ignore-not-found=true

.PHONY: deploy
deploy:
	@echo "Deploying operator to cluster..."
	kubectl apply -f config/manager/namespace.yaml
	kubectl apply -f config/rbac/
	kubectl apply -f config/manager/deployment.yaml
	kubectl apply -f config/manager/service.yaml
	@echo "Waiting for deployment to be ready..."
	kubectl wait --for=condition=available --timeout=300s deployment/chaos-operator -n chaos-system

.PHONY: undeploy
undeploy:
	@echo "Removing operator from cluster..."
	kubectl delete -f config/manager/deployment.yaml --ignore-not-found=true
	kubectl delete -f config/manager/service.yaml --ignore-not-found=true
	kubectl delete -f config/rbac/ --ignore-not-found=true
	kubectl delete -f config/manager/namespace.yaml --ignore-not-found=true

.PHONY: report
report:
	@scripts/chaos-report.sh $(ARGS)

.PHONY: install-all
install-all: generate-crds docker-build install-crds deploy
	@echo "✅ Installation complete!"
	@echo "Check operator status with: kubectl get pods -n chaos-system"

.PHONY: kind-up kind-down kind-status addons-install addons-uninstall
kind-up:
	@KIND_CLUSTER=$(KIND_CLUSTER) scripts/kind/up.sh

kind-down:
	@KIND_CLUSTER=$(KIND_CLUSTER) scripts/kind/down.sh

kind-status:
	@KIND_CLUSTER=$(KIND_CLUSTER) scripts/kind/status.sh

addons-install: kind-up
	@KIND_CLUSTER=$(KIND_CLUSTER) scripts/helms/install-local.sh

addons-uninstall:
	@KIND_CLUSTER=$(KIND_CLUSTER) scripts/helms/uninstall-local.sh
