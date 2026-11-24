# Quick Start: Platform Mesh + IANUS providers from scratch

End-to-end steps to stand up a local Platform Mesh (kcp + portal) and register this repo's three
providers (`ianus-injection-molding`, `ianus-materials`, `skz`) on a brand-new Linux
(amd64) machine with nothing installed. Mirrors what's actually running on this dev machine. See
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

What's actually on this machine is the `platform-mesh/platform-mesh` monorepo (not the older
`platform-mesh/helm-charts` + `task local-setup` flow some docs elsewhere describe) — it ships its
own `pmctl` bootstrapper:

```shell
mkdir -p ~/repo && cd ~/repo
git clone https://github.com/StromungsRaum/ecotwinRA.git
git clone https://github.com/platform-mesh/platform-mesh.git
```

## 2. Bootstrap Platform Mesh (kind cluster + kcp + portal)

```shell
cd ~/repo/platform-mesh
go run ./cmd/pmctl quickstart --dev
```

This creates the `platform-mesh` kind cluster, generates mkcert certs, and installs kcp,
Flux-managed core infra, Keycloak/OpenFGA, the portal, and the GraphQL CRD gateway via Flux
`HelmRelease`s. First run takes a while. **Known flaky bit**: Flux's `helm-controller` has, more
than once on this machine, silently stopped processing its reconcile queue mid-run (no crash, no
restart, just zero log output) while `HelmRelease`s like `infra`/`marketplace` sit `False` with
errors like `no matches for kind "CacheServer"` (a CRD another `HelmRelease` hasn't finished
installing yet). If reconciliation looks stuck for minutes with no log activity, it's the
controller, not a real ordering bug — `kubectl -n platform-mesh-system rollout restart
deploy/helm-controller` and give it another minute or two; repeat if needed.

Once it completes:

- Portal: <https://pm.127-0-0-1.sslip.io:8443>
- kcp API: <https://kcp.api.pm.127-0-0-1.sslip.io:8443>
- Admin kubeconfig: pulled live from the cluster (see step 3) rather than a checked-out file path.

## 3. Set up the admin + compute kubeconfigs

The admin kubeconfig isn't a static repo file — Platform Mesh writes a ready-made cluster-admin
kcp kubeconfig into a `kubeconfig-kcp-admin` Secret in `platform-mesh-system` on every install.
Pulling it fresh from there (rather than reusing an old copy) avoids chasing a stale `server:`
host after a cluster rebuild — kcp's own `kubectl ws create`/`kubectl ws use` fail with a generic
`404 NotFound` against a stale kubeconfig, not a DNS error, because `*.localhost`/`*.sslip.io`
addresses always resolve to *something* on loopback:

```shell
cd ~/repo/ecotwinRA
kubectl -n platform-mesh-system get secret kubeconfig-kcp-admin -o jsonpath='{.data.kubeconfig}' \
  | base64 -d > kcp-admin.kubeconfig
export PM_KUBECONFIG="$(realpath kcp-admin.kubeconfig)"
kind export kubeconfig --name platform-mesh --kubeconfig compute.kubeconfig
export COMPUTE_KUBECONFIG="$(realpath compute.kubeconfig)"
```

**kcp workspace-navigation gotcha**: the `kubectl-kcp` plugin keeps its "current workspace"
pointer in a special `workspace.kcp.io/current` cluster entry inside `$PM_KUBECONFIG` itself, and
on recent plugin versions bare `kubectl ws` (no args) means "go to my home workspace" (`cd`
semantics), not "print current" — use `kubectl ws .` for that. Running bare `kubectl ws` by habit
mid-script silently re-roots every following `kubectl ws use :`/`create --enter --ignore-existing`
call under your personal `user:<name>` home workspace instead of `root`, so `make init` ends up
creating a stray, duplicate provider workspace+APIExport+RBAC tree there instead of under
`root:providers:<name>`. Before trusting a `kubectl ws create ... --enter` chain (especially
inside a script), confirm with `KUBECONFIG=$PM_KUBECONFIG kubectl ws .` that you're actually under
`root:providers:...`, not a `<random-id>:providers:...` home-workspace path. If it did go sideways,
delete the stray workspace tree (`kubectl --server=".../clusters/<home-id>:providers" delete
workspace <name>`) and repoint the plugin's state directly rather than fighting `ws use :`:
```shell
KUBECONFIG=$PM_KUBECONFIG kubectl config set-cluster "workspace.kcp.io/current" \
  --server="https://kcp.api.pm.127-0-0-1.sslip.io:8443/clusters/root:providers:<name>"
```

## 3b. Deploy the portal

The portal (`ianus-portal`) is one shared deployment serving all three providers' panels
(injection-molding, materials, generic-materials) - it doesn't belong to any single provider, so it
deploys once, independently of the per-provider registration steps below:

```shell
export IMAGE_TAG=platform-mesh
make portal-image-build kind-load-portal IMAGE_TAG=$IMAGE_TAG

KUBECONFIG=$COMPUTE_KUBECONFIG kubectl create namespace provider-models

KUBECONFIG=$COMPUTE_KUBECONFIG helm upgrade --install ianus-portal ./deploy/helm/ianus-portal \
  --namespace provider-models --set image.tag=$IMAGE_TAG --set image.pullPolicy=IfNotPresent \
  --set httpRoute.enabled=true --set middleware.enabled=true --set common.defaults.hostAliases.enabled=true
```

## 5. Register the `ianus-injection-molding` provider

Own workspace, own binaries/chart:

```shell
# 5a. Provider workspace
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

# root.kcp.pm.127-0-0-1.sslip.io also self-resolves to 127.0.0.1 inside a pod (sslip.io's own
# mechanism) - without this hostAlias the controller can never watch any consumer workspace at
# all (silent: no error visible outside its own logs, every refresh/create just hangs). See 7a.
# devBackend.enabled bakes in the matching fix for backend.dev.simod.de itself (DNS + the
# self-signed cert) - baked into the chart, not a live-only workaround; see 7a for both.
TRAEFIK_IP=$(KUBECONFIG=$COMPUTE_KUBECONFIG kubectl get svc platform-mesh-prerequisites-traefik \
  -n platform-mesh-system -o jsonpath='{.spec.clusterIP}')
KUBECONFIG=$COMPUTE_KUBECONFIG helm upgrade --install injectionmolding-controller ./deploy/helm/injectionmolding-controller \
  --namespace provider-injection-molding --set image.tag=$IMAGE_TAG --set image.pullPolicy=IfNotPresent \
  --set hostAliases.enabled=true --set hostAliases.entries[0].ip=$TRAEFIK_IP \
  --set hostAliases.entries[0].hostnames[0]=root.kcp.pm.127-0-0-1.sslip.io \
  --set devBackend.enabled=true
```

`--simod-base-url` (default `https://backend.dev.simod.de`, chart value `controller.simodBaseURL`)
is the real external SIMOD/StrömungsRaum backend this provider calls — see step 7.

## 6. Register the `ianus-materials` provider

Third sibling provider — two CRDs (`MaterialCatalog`, `MaterialSubmission`) instead of one, same pattern
otherwise:

```shell
# 6a. Provider workspace (sibling of `ianus-injection-molding`)
KUBECONFIG=$PM_KUBECONFIG kubectl ws use :
KUBECONFIG=$PM_KUBECONFIG kubectl ws create providers --type=root:providers --enter --ignore-existing
KUBECONFIG=$PM_KUBECONFIG kubectl ws create ianus-materials --type=root:provider --enter --ignore-existing

# 6b. Bootstrap
KUBECONFIG=$PM_KUBECONFIG make init-materials HOST_OVERRIDE=https://frontproxy-front-proxy.platform-mesh-system:8443

# 6c. Extract the operator kubeconfig
KUBECONFIG=$PM_KUBECONFIG kubectl get secret materials-controller-kubeconfig -n default -o jsonpath='{.data.kubeconfig}' | base64 -d > materials.kubeconfig

# 6d. Build + load the image
make image-build-materials kind-load-materials IMAGE_TAG=$IMAGE_TAG

# 6e. Deploy
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl create namespace provider-materials
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl create secret generic materials-controller-kubeconfig \
  --from-file=kubeconfig=./materials.kubeconfig -n provider-materials

# Same root.kcp.pm.127-0-0-1.sslip.io hostAlias as step 5e, plus devBackend.enabled - see 7a.
TRAEFIK_IP=$(KUBECONFIG=$COMPUTE_KUBECONFIG kubectl get svc platform-mesh-prerequisites-traefik \
  -n platform-mesh-system -o jsonpath='{.spec.clusterIP}')
KUBECONFIG=$COMPUTE_KUBECONFIG helm upgrade --install materials-controller ./deploy/helm/materials-controller \
  --namespace provider-materials --set image.tag=$IMAGE_TAG --set image.pullPolicy=IfNotPresent \
  --set hostAliases.enabled=true --set hostAliases.entries[0].ip=$TRAEFIK_IP \
  --set hostAliases.entries[0].hostnames[0]=root.kcp.pm.127-0-0-1.sslip.io \
  --set devBackend.enabled=true
```

**`devBackend` gotcha**: `devBackend.enabled` (both charts) bakes in the dev-SIMOD-backend DNS
hostAlias and a committed CA bundle (`files/dev-ca-bundle.crt` - this image's own base trust store
plus `backend.dev.simod.de`'s self-signed cert, captured once) that replaces
`/etc/ssl/certs/ca-certificates.crt` wholesale — no live, not-in-chart ConfigMap to pre-create
anymore. Defaults to `false`: leave it off if you're only registering the provider and not yet
talking to the real SIMOD backend, or if you're pointing `simod-credentials`' `baseUrl` at
int/prod instead (those run elsewhere, with real DNS and real CA-signed certs — pods reach them
over standard cluster DNS, no override needed, and `devBackend` should stay off).

This provider submits a material characterized in the `skz` provider to the real StrömungsRaum
backend — see [platform-mesh.md](./platform-mesh.md#ianus-simulation---materials-provider) for the
full cross-provider data flow.

## 6b. Register the `skz` ("Material Provider A - Materials") provider

Bootstrap-only — no operator, no image, no chart (see [CLAUDE.md](../CLAUDE.md)'s "`skz` is the
odd one out" section). Still needs its own workspace + `make init-skz` to publish the static
`Material` catalog and the `genericmaterials.platform-mesh.io` APIExport, so run this too on a
fresh cluster:

```shell
KUBECONFIG=$PM_KUBECONFIG kubectl ws use :
KUBECONFIG=$PM_KUBECONFIG kubectl ws create providers --type=root:providers --enter --ignore-existing
KUBECONFIG=$PM_KUBECONFIG kubectl ws create skz --type=root:provider --enter --ignore-existing

KUBECONFIG=$PM_KUBECONFIG make init-skz HOST_OVERRIDE=https://frontproxy-front-proxy.platform-mesh-system:8443
```

## 7. Verify

```shell
# Marketplace entries (APIExport + ProviderMetadata) for each provider - the list is global
# (same result from any provider's cluster path), one entry expected per provider:
# injectionmolding.platform-mesh.io, materials.platform-mesh.io, genericmaterials.platform-mesh.io (skz)
KUBECONFIG=$PM_KUBECONFIG kubectl --server="https://kcp.api.pm.127-0-0-1.sslip.io:8443/services/marketplace/clusters/root:providers:ianus-injection-molding" get marketplaceentries -A

KUBECONFIG=$COMPUTE_KUBECONFIG kubectl get pods -n provider-models
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl get pods -n provider-injection-molding
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl get pods -n provider-materials
```

Open <https://pm.127-0-0-1.sslip.io:8443>, log in, and the injection-molding, materials,
and generic-materials provider UIs should appear once an account workspace binds their
APIExports.

## 7a. Troubleshooting

**Provider shows up in the marketplace, but enabling it just spins on the loading dots
forever.** This has happened more than once on this machine, always from the same cause: the
injection-molding / materials / generic-materials panels aren't generic-list-view webcomponents -
they're Luigi iframes pointing at a per-org wildcard subdomain
(`https://{org}.pm.127-0-0-1.sslip.io:8443/injection-molding/` etc., see
`config/provider-injectionmolding/contentconfiguration.yaml`'s `url` field), all served by the one
`ianus-portal` deployment. That only works if its `HTTPRoute` + Traefik `Middleware` (CORS) +
cross-namespace `ReferenceGrant` exist - and they're **disabled by default** in
`deploy/helm/ianus-portal/values.yaml` (`httpRoute.enabled: false`, `middleware.enabled: false`).
If you deploy/upgrade `ianus-portal` without explicitly re-enabling them (easy to do if you copy
just the `image.tag`/`pullPolicy` bit of the step 3b command and drop the rest), the iframe has no
route into the cluster at all, the Luigi handshake with it never completes, and the panel spins
forever - with nothing useful in any pod's logs, because the request never reaches a pod.

Check: `KUBECONFIG=$COMPUTE_KUBECONFIG kubectl get httproute,middleware -n platform-mesh-system |
grep ianus-portal` - if empty, redeploy with the full step 3b command:
```shell
KUBECONFIG=$COMPUTE_KUBECONFIG helm upgrade --install ianus-portal ./deploy/helm/ianus-portal \
  --namespace provider-models --set image.tag=$IMAGE_TAG --set image.pullPolicy=IfNotPresent \
  --set httpRoute.enabled=true --set middleware.enabled=true --set common.defaults.hostAliases.enabled=true
```
Verify directly (bypasses needing a portal login): `curl -sk -o /dev/null -w '%{http_code}\n'
"https://<org>.pm.127-0-0-1.sslip.io:8443/injection-molding/"` should be `200`, not a connection
error/timeout.

Before chasing this, it's worth ruling out the *other* thing that can make panels misbehave on a
freshly-registered cluster: the `kubernetes-graphql-gateway`'s client to kcp runs at client-go's
bare defaults (QPS=5, Burst=10, see `--kubernetes-qps`/`--kubernetes-burst` in that service's
`options.go`) unless its `HelmRelease` sets `gateway.extraArgs` higher. Under load from several
providers + the marketplace list at once this shows up as `ERROR Unable to list objects ...
"client rate limiter Wait returned an error: context canceled"` in the gateway's logs - a real
bug, but a *different* one (intermittent list failures, not a permanently-dead route), and on its
own it did not explain a panel that never loads at all.

**Running a command against the wrong cluster (`$PM_KUBECONFIG` vs. `$COMPUTE_KUBECONFIG`).**
The two kubeconfigs set up in step 3 point at different API surfaces, and mixing them up produces
confusing not-found/wrong-type errors rather than an obvious "wrong cluster" message:

- `$PM_KUBECONFIG` talks to kcp's front-proxy (`kcp.api.pm.127-0-0-1.sslip.io:8443`) - the **logical
  control plane**: workspaces (`root:providers:*`, `root:orgs:*`), `APIExport`/`APIResourceSchema`,
  `ProviderMetadata`, `ContentConfiguration`, the marketplace virtual workspace. kcp itself has no
  kubelet/scheduler - it's API machinery only, no pods run here. Used for `kubectl ws ...`,
  `make init-<name>`, checking marketplace entries/APIBindings.
- `$COMPUTE_KUBECONFIG` (from `kind export kubeconfig --name platform-mesh`) is the kind cluster's own
  native admin kubeconfig, no kcp layering - the **compute plane**: actual Deployments/Pods/Services/
  Secrets/Namespaces (`provider-models`, `provider-injection-molding`, `provider-materials`,
  `platform-mesh-system`), Gateway API `HTTPRoute`/`Middleware`/`ReferenceGrant`, Flux `HelmRelease`s.
  Used for `helm upgrade --install`, creating controller-kubeconfig secrets, checking pod health.

In this `pmctl quickstart --dev` setup both happen to be the *same* physical kind cluster, but each
kubeconfig addresses it through a completely different API surface - that's why `kubectl get pods`
against `$PM_KUBECONFIG` doesn't show the controller pods, and `kubectl get apiexport` against
`$COMPUTE_KUBECONFIG` doesn't work at all. If a command gives a surprising "not found" or "no kind
registered" error, check which `$...KUBECONFIG` it used before assuming the resource is actually
missing.

**Refresh/create on any provider silently does nothing - no error, no update, status never changes.**
Confirmed live, 2026-11-18: the resource's `status` is *completely absent*, not even the initial empty
one a fresh reconcile would set - meaning the operator never reconciled it at all. Check
`injectionmolding-controller`'s/`materials-controller`'s logs for a repeating
`error starting endpoint watcher ... dial tcp 127.0.0.1:8443: connect: connection refused` against
`https://root.kcp.pm.127-0-0-1.sslip.io:8443/services/apiexport/...`. This looks like a routing gap but
isn't one - `kubectl get gateway k8sapi-gateway -n platform-mesh-system -o yaml` already has a
`passthrough-root` listener for exactly `root.kcp.pm.127-0-0-1.sslip.io`, backed by a `TLSRoute`
(`kcp-root-shard-tlsroute` → `root-kcp:6443`), both `Accepted`/`ResolvedRefs: True`. The real cause is the
same class of gotcha as the SIMOD backend DNS one below: every `*.pm.127-0-0-1.sslip.io` hostname
resolves to `127.0.0.1` by sslip.io's own design, which *inside a pod's own network namespace* is the pod
itself - nothing listens on 8443 there, hence "connection refused", regardless of how correctly Traefik's
routing is configured. Fix: hostAlias that one hostname to Traefik's ClusterIP (not the kcp Service
directly - the TLS routing above is SNI-based passthrough through Traefik itself) on both controllers:
```shell
TRAEFIK_IP=$(KUBECONFIG=$COMPUTE_KUBECONFIG kubectl get svc platform-mesh-prerequisites-traefik \
  -n platform-mesh-system -o jsonpath='{.spec.clusterIP}')
cat > /tmp/kcp-hostalias-override.yaml <<EOF
hostAliases:
  enabled: true
  entries:
    - ip: "$TRAEFIK_IP"
      hostnames:
        - "root.kcp.pm.127-0-0-1.sslip.io"
EOF
KUBECONFIG=$COMPUTE_KUBECONFIG helm upgrade injectionmolding-controller ./deploy/helm/injectionmolding-controller \
  -n provider-injection-molding --reuse-values -f /tmp/kcp-hostalias-override.yaml
KUBECONFIG=$COMPUTE_KUBECONFIG helm upgrade materials-controller ./deploy/helm/materials-controller \
  -n provider-materials --reuse-values -f /tmp/kcp-hostalias-override.yaml
```
Not baked into the charts' checked-in values - the ClusterIP is allocated fresh per cluster install, so a
hardcoded one would just go stale on the next `pmctl quickstart --dev` rebuild (same reason the `common`
chart's own default `hostAliases` entry, `10.96.188.4` for `root.kcp.localhost`, is already wrong for this
cluster - it's a `.localhost`-domain example left over from wherever the chart's defaults were authored,
not this cluster's actual ClusterIP/domain). Verify: the endpoint-watcher error stops, and a refresh
actually reaches `status` - confirm with `kubectl get <kind> -o yaml` directly on the resource; if it's
still stuck on the old `backend.dev.simod.de` DNS gotcha next, that's a *different*, already-documented
problem (see "Local-sandbox-only gotchas" below), not this one.

## 8. Run the StrömungsRaum frontend and backend locally

The providers' controllers and CRDs (`Component`, `DigitalTwin`, `ExecutionParameter`, `Simulation`,
`InjectionMoldingService`) talk to the external StrömungsRaum/SIMOD backend
(`backend.{dev,test,int,prod}.simod.de`) for anything beyond bootstrap/registration — login,
creating components/twins/simulations, listing existing ones. That backend (+ its frontend) is a
separate Laravel/Vue app, the `ianus-simod` repo (not part of this one), with its own docker-compose
stack. Without it running, the CRDs above still register fine but their reconcilers fail to
authenticate. This section reproduces the `dev` stack the way it's actually configured on this
machine — full details and troubleshooting live in that repo's own
[docs/SETUP.md](https://github.com/StromungsRaum/ianus-simod/blob/main/docs/SETUP.md), don't
duplicate them here.

The optional **cadbuilder** and **stlbuilder** container overlays need to be enabled
(visible as the `dev-simod-cadbuilder-1`/`dev-simod-stlbuilder-1` containers).

### 8a. Access

- Frontend: <https://dev.simod.de>
- Backend admin: <https://backend.dev.simod.de>
- phpMyAdmin: `127.0.0.1:8082`
- MailHog (dev mail capture): <http://localhost:8025>

### 8b. Wiring this provider repo to it

Point the `ecotwinRA` operators/CRDs at this local backend instead of `backend.dev.simod.de`
directly if you want an isolated loop:

```shell
KUBECONFIG=./injectionmolding.kubeconfig go run ./cmd/injectionmolding --endpointslice=injectionmolding.platform-mesh.io --simod-base-url=https://backend.dev.simod.de
```

Set connection details to simod backend by creating a **simod-credentials** secret CRD.

```shell
export KUBECONFIG=$PM_PLATFORM
kubectl --server https://kcp.api.pm.127-0-0-1.sslip.io:8443/clusters/root:orgs:<org-name>:<org-user> create secret generic simod-credentials  -from-literal=email='<simod-email>' --from-literal=password='<simod-password>' -n default --insecure-skip-tls-verify
```

where <org-name>, <org-user>, <simod-email> and <simod-password> need to be replaced with adequate values. 

Credentials go in a same-namespace `Secret` with `email`/`password` keys, referenced by each CR's
`spec.secretRefs[0]` — create a user on the backend admin UI (step 8a) first.

If the provider's controller runs **inside the kind cluster** (not `go run` on the host) talking to
this same-machine backend, see the "Local-sandbox-only gotchas" section in
[platform-mesh.md](./platform-mesh.md#local-sandbox-only-gotchas-real-devintprod-shouldnt-hit-these)
for the DNS/TLS workarounds that entails (host aliases, trusting the self-signed cert on the host
and inside the controller's container image).

## 9. Install the Slurm scheduler

The backend's "act as cluster" feature (`Kunden Firmen / Benutzer` → `Cluster Computer` permission,
`Cluster / Clusterliste`) submits simulation jobs to a Slurm cluster. This section installs just the
Slurm scheduler itself, as a single-node cluster — matching how it's actually set up on this
machine (native systemd services, not containers). It does **not** cover the separate job-execution
layer (the private `runner`/`simodlib`/`simodclustercomponents` repos and the `sicle` config bundle)
that turns a submitted Slurm job into an actual running simulation — that's a different, private
piece of the pipeline, out of scope here.

### 9a. Install packages

```shell
sudo apt-get update
sudo apt-get install -y munge slurmd slurmctld slurmdbd mariadb-server
```

### 9b. Munge (inter-daemon authentication)

```shell
sudo systemctl enable --now munge
systemctl status munge --no-pager
```

A single-node setup works with the key `apt` generated for you at `/etc/munge/munge.key` — nothing
further to do unless you're joining more than one node to the cluster.

### 9c. Accounting database

Slurm's accounting (`AccountingStorageType=accounting_storage/slurmdbd`) needs its own MariaDB
database and user:

```shell
sudo mysql -e "CREATE DATABASE slurm_acct_db;"
sudo mysql -e "CREATE USER 'slurm'@'localhost' IDENTIFIED BY '<pick-a-password>';"
sudo mysql -e "GRANT ALL PRIVILEGES ON slurm_acct_db.* TO 'slurm'@'localhost';"
```

### 9d. `/etc/slurm/slurmdbd.conf`

```ini
AuthType=auth/munge
DbdHost=localhost
DbdPort=6819
SlurmUser=slurm
StorageType=accounting_storage/mysql
StorageHost=localhost
StorageUser=slurm
StoragePass=<same-password-as-above>
StorageLoc=slurm_acct_db
PidFile=/var/run/slurmdbd.pid
LogFile=/var/log/slurm-llnl/slurmdbd.log
```

```shell
sudo chown slurm:slurm /etc/slurm/slurmdbd.conf
sudo chmod 600 /etc/slurm/slurmdbd.conf
```

### 9e. `/etc/slurm/slurm.conf`

Single-node cluster where the controller and the (only) compute node are the same machine — this
is exactly this machine's config, generalized to detect your own hostname/CPU/memory:

```ini
ClusterName=localcluster
SlurmctldHost=<run: hostname>

MpiDefault=none
ProctrackType=proctrack/linuxproc
ReturnToService=2

SlurmctldPidFile=/var/run/slurm/slurmctld.pid
SlurmctldPort=6817
SlurmdPidFile=/var/run/slurm/slurmd.pid
SlurmdPort=6818
SlurmdSpoolDir=/var/lib/slurm-llnl/slurmd
SlurmUser=slurm
StateSaveLocation=/var/lib/slurm-llnl/slurmctld
SwitchType=switch/none
TaskPlugin=task/none

InactiveLimit=0
KillWait=30
MinJobAge=300
SlurmctldTimeout=120
SlurmdTimeout=300
Waittime=0

SchedulerType=sched/backfill
SelectType=select/cons_tres
SelectTypeParameters=CR_Core

AccountingStorageType=accounting_storage/slurmdbd
AccountingStorageHost=<run: hostname>
AccountingStoragePort=6819
JobCompType=jobcomp/none
JobAcctGatherFrequency=30
JobAcctGatherType=jobacct_gather/linux
SlurmctldDebug=info
SlurmctldLogFile=/var/log/slurm-llnl/slurmctld.log
SlurmdDebug=info
SlurmdLogFile=/var/log/slurm-llnl/slurmd.log

# Replace with: hostname, `nproc`, sockets/cores-per-socket/threads-per-core from `lscpu`,
# and RealMemory in MB from `free -m` (leave a little headroom below the physical total).
NodeName=<hostname> CPUs=<nproc> Sockets=1 CoresPerSocket=<cores> ThreadsPerCore=<threads> RealMemory=<MB> State=UNKNOWN
PartitionName=LocalQ Nodes=<hostname> Default=YES MaxTime=INFINITE State=UP
```

```shell
sudo mkdir -p /var/lib/slurm-llnl/slurmctld /var/lib/slurm-llnl/slurmd /var/log/slurm-llnl /var/run/slurm
sudo chown slurm:slurm /var/lib/slurm-llnl/slurmctld /var/lib/slurm-llnl/slurmd /var/log/slurm-llnl /var/run/slurm
```

### 9f. Start everything and register the cluster

```shell
sudo systemctl enable --now slurmdbd
sudo systemctl enable --now slurmctld
sudo systemctl enable --now slurmd

# slurmdbd only accepts accounting data for clusters it knows about
sacctmgr -i add cluster localcluster
sudo systemctl restart slurmctld
```

### 9g. Verify

```shell
sinfo            # expect: PARTITION LocalQ, STATE idle, your hostname listed
srun hostname    # runs `hostname` as a Slurm job on the local node
sacctmgr show cluster
```

The `partitions` block (`name: LocalQ`, `num_cores_per_node`, `max_nodes`, `memory_per_node`) that a
cluster-agent config (`cluster_config.scheduler_type: slurm`) points at should match the
`PartitionName=LocalQ` line above — adjust either side to keep them consistent.

## 10. Cleanup

```shell
kind delete cluster --name platform-mesh

# StrömungsRaum stack
cd ~/repo/ianus-simod && task dev:down

# Slurm (only if you want to tear the scheduler down too)
sudo systemctl disable --now slurmd slurmctld slurmdbd munge
```
