# Quick Start: Platform Mesh + IANUS providers from scratch

End-to-end steps to stand up a local Platform Mesh (kcp + portal) and register this repo's two
providers (`ianus` and `ianus-injection-molding`) on a brand-new Linux (amd64) machine with
nothing installed. Mirrors what's actually running on this dev machine. See
[platform-mesh.md](./platform-mesh.md) for the per-provider deep dive this quick start summarizes.

## 0. Prerequisites

Install in order. Versions shown are what's verified working; newer patch versions are fine.

```shell
# git, build tools
sudo apt-get update && sudo apt-get install -y git curl build-essential libnss3-tools jq

# Docker Engine (container runtime kind/helm/images need)
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker "$USER"
# log out/in (or `newgrp docker`) so the group change takes effect

# Go 1.26+ (module requires go 1.26.3)
curl -LO https://go.dev/dl/go1.26.3.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.26.3.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc && export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin

# Node 22 + npm (portal is Angular 20)
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt-get install -y nodejs

# uv (Python package/runner for ecotwin/twinctl, requires Python >=3.11)
curl -LsSf https://astral.sh/uv/install.sh | sh

# kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl && sudo mv kubectl /usr/local/bin/

# Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.32.0/kind-linux-amd64
chmod +x ./kind && sudo mv ./kind /usr/local/bin/kind

# yq
sudo curl -L https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 -o /usr/local/bin/yq
sudo chmod +x /usr/local/bin/yq

# mkcert (local TLS certs for *.localhost)
curl -LO https://github.com/FiloSottile/mkcert/releases/latest/download/mkcert-v1.4.4-linux-amd64
chmod +x mkcert-v1.4.4-linux-amd64 && sudo mv mkcert-v1.4.4-linux-amd64 /usr/local/bin/mkcert
mkcert -install

# krew (kubectl plugin manager) -> kubectl-kcp plugin (needed for workspace commands)
( set -x; cd "$(mktemp -d)" && OS=linux ARCH=amd64 &&
  curl -fsSLO "https://github.com/kubernetes-sigs/krew/releases/latest/download/krew-${OS}_${ARCH}.tar.gz" &&
  tar zxvf "krew-${OS}_${ARCH}.tar.gz" && ./krew-"${OS}_${ARCH}" install krew )
echo 'export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH"' >> ~/.bashrc && export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH"
kubectl krew install kcp

# Task (optional, drives `task local-setup`)
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin
```

Open a fresh shell (or `source ~/.bashrc`) so `PATH` updates apply before continuing.

## 1. Clone the repos

```shell
mkdir -p ~/repo && cd ~/repo
git clone https://github.com/StromungsRaum/ecotwinRA.git
git clone https://github.com/platform-mesh/helm-charts.git
```

## 2. Bootstrap Platform Mesh (kind cluster + kcp + portal)

```shell
cd ~/repo/helm-charts
task local-setup
```

This creates the `platform-mesh` kind cluster, generates mkcert certs, and installs kcp,
Flux-managed core infra, Keycloak/OpenFGA, the portal, and the GraphQL CRD gateway. First run
takes a while (pulls/builds a lot of charts); default timeout is 1200s per readiness check
(`KUBECTL_WAIT_TIMEOUT` env var to raise it). Re-running `task local-setup` reuses the existing
cluster (`--iterate=true` default) — for a clean rebuild: `kind delete cluster --name
platform-mesh && task local-setup -- --iterate=false`.

Once it completes:

- Portal: <https://portal.localhost:8443>
- kcp API: <https://localhost:8443>
- Admin kubeconfig: `~/repo/helm-charts/.secret/kcp/admin.kubeconfig`

## 3. Set up the admin + compute kubeconfigs

```shell
cd ~/repo/ecotwinRA
cp ~/repo/helm-charts/.secret/kcp/admin.kubeconfig kcp-admin.kubeconfig
export PM_KUBECONFIG="$(realpath kcp-admin.kubeconfig)"
kind export kubeconfig --name platform-mesh --kubeconfig compute.kubeconfig
export COMPUTE_KUBECONFIG="$(realpath compute.kubeconfig)"
```

## 4. Register the `ianus` provider

```shell
# 4a. Provider workspace
KUBECONFIG=$PM_KUBECONFIG kubectl ws use :
KUBECONFIG=$PM_KUBECONFIG kubectl ws create providers --type=root:providers --enter --ignore-existing
KUBECONFIG=$PM_KUBECONFIG kubectl ws create ianus --type=root:provider --enter --ignore-existing

# 4b. Bootstrap (APIExport/APIResourceSchema, ProviderMetadata, ContentConfiguration, controller RBAC)
KUBECONFIG=$PM_KUBECONFIG make init HOST_OVERRIDE=https://frontproxy-front-proxy.platform-mesh-system:8443

# 4c. Extract the operator kubeconfig the init step generated
KUBECONFIG=$PM_KUBECONFIG kubectl get secret ianus-controller-kubeconfig -n default -o jsonpath='{.data.kubeconfig}' | base64 -d > operator.kubeconfig

# 4d. Build + load the controller and portal images into kind
export IMAGE_TAG=platform-mesh
make images kind-load-all IMAGE_TAG=$IMAGE_TAG

# 4e. Deploy
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl create namespace provider-models
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl create secret generic ianus-controller-kubeconfig \
  --from-file=kubeconfig=./operator.kubeconfig -n provider-models

KUBECONFIG=$COMPUTE_KUBECONFIG helm upgrade --install ianus-controller ./deploy/helm/ianus-controller \
  --namespace provider-models --set image.tag=$IMAGE_TAG --set image.pullPolicy=IfNotPresent \
  --set common.defaults.hostAliases.enabled=true

KUBECONFIG=$COMPUTE_KUBECONFIG helm upgrade --install ianus-portal ./deploy/helm/ianus-portal \
  --namespace provider-models --set image.tag=$IMAGE_TAG --set image.pullPolicy=IfNotPresent \
  --set httpRoute.enabled=true --set middleware.enabled=true --set common.defaults.hostAliases.enabled=true
```

## 5. Register the `ianus-injection-molding` provider

Sibling provider, own workspace, own binaries/chart — same pattern:

```shell
# 5a. Provider workspace (sibling of `ianus`, not nested under it)
KUBECONFIG=$PM_KUBECONFIG kubectl ws use :
KUBECONFIG=$PM_KUBECONFIG kubectl ws create providers --type=root:providers --enter --ignore-existing
KUBECONFIG=$PM_KUBECONFIG kubectl ws create ianus-injection-molding --type=root:provider --enter --ignore-existing

# 5b. Bootstrap
KUBECONFIG=$PM_KUBECONFIG make init-injectionmolding HOST_OVERRIDE=https://frontproxy-front-proxy.platform-mesh-system:8443

# 5c. Extract the operator kubeconfig
KUBECONFIG=$PM_KUBECONFIG kubectl get secret injectionmolding-controller-kubeconfig -n default -o jsonpath='{.data.kubeconfig}' | base64 -d > injectionmolding.kubeconfig

# 5d. Build + load the image
make image-build-injectionmolding kind-load-injectionmolding IMAGE_TAG=$IMAGE_TAG

# 5e. Deploy
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl create namespace provider-injection-molding
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl create secret generic injectionmolding-controller-kubeconfig \
  --from-file=kubeconfig=./injectionmolding.kubeconfig -n provider-injection-molding

KUBECONFIG=$COMPUTE_KUBECONFIG helm upgrade --install injectionmolding-controller ./deploy/helm/injectionmolding-controller \
  --namespace provider-injection-molding --set image.tag=$IMAGE_TAG --set image.pullPolicy=IfNotPresent
```

`--simod-base-url` (default `https://backend.dev.simod.de`, chart value `controller.simodBaseURL`)
is the real external SIMOD/StrömungsRaum backend this provider calls — see step 6.

## 6. Verify

```shell
# Marketplace entries (APIExport + ProviderMetadata) for each provider
kubectl --server="https://localhost:8443/services/marketplace/clusters/root:providers:ianus" get marketplaceentries -A
kubectl --server="https://localhost:8443/services/marketplace/clusters/root:providers:ianus-injection-molding" get marketplaceentries -A

KUBECONFIG=$COMPUTE_KUBECONFIG kubectl get pods -n provider-models
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl get pods -n provider-injection-molding
```

Open <https://portal.localhost:8443>, log in, and the `ianus` and injection-molding provider
UIs should appear once an account workspace binds their APIExports.

## 7. Optional: the real SIMOD backend

The providers' controllers and CRDs (`Component`, `DigitalTwin`, `ExecutionParameter`, `Simulation`,
`InjectionMoldingService`) talk to the external StrömungsRaum/SIMOD backend
(`backend.{dev,test,int,prod}.simod.de`) for anything beyond bootstrap/registration — login,
creating components/twins/simulations, listing existing ones. That backend is a separate Laravel
app (`ianus-simod` repo, not part of this one) with its own local setup; without it reachable, the
CRDs will register fine but their reconcilers will fail to authenticate. Credentials go in a
same-namespace `Secret` with `email`/`password` keys, referenced by each CR's `spec.secretRefs[0]`.

If you're pointing at a self-signed/local instance of that backend rather than a real
`*.simod.de` environment, see the "Local-sandbox-only gotchas" section in
[platform-mesh.md](./platform-mesh.md#local-sandbox-only-gotchas-real-devintprod-shouldnt-hit-these)
for the DNS/TLS workarounds that entails (host aliases, trusting a self-signed cert on the host
and inside the controller's container image).

## 8. Cleanup

```shell
kind delete cluster --name platform-mesh
```
