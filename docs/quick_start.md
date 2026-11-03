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

## 7. Run the StrömungsRaum frontend and backend locally

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

### 7a. Access

- Frontend: <https://dev.simod.de>
- Backend admin: <https://backend.dev.simod.de>
- phpMyAdmin: `127.0.0.1:8082`
- MailHog (dev mail capture): <http://localhost:8025>

### 7b. Wiring this provider repo to it

Point the `ecotwinRA` operators/CRDs at this local backend instead of `backend.dev.simod.de`
directly if you want an isolated loop:

```shell
KUBECONFIG=./operator.kubeconfig go run ./cmd/ianus --endpointslice=ianus.platform-mesh.io --simod-base-url=https://backend.dev.simod.de
KUBECONFIG=./injectionmolding.kubeconfig go run ./cmd/injectionmolding --endpointslice=injectionmolding.platform-mesh.io --simod-base-url=https://backend.dev.simod.de
```

Credentials go in a same-namespace `Secret` with `email`/`password` keys, referenced by each CR's
`spec.secretRefs[0]` — create a user via step 7d first.

If the provider's controller runs **inside the kind cluster** (not `go run` on the host) talking to
this same-machine backend, see the "Local-sandbox-only gotchas" section in
[platform-mesh.md](./platform-mesh.md#local-sandbox-only-gotchas-real-devintprod-shouldnt-hit-these)
for the DNS/TLS workarounds that entails (host aliases, trusting the self-signed cert on the host
and inside the controller's container image).

## 8. Install the Slurm scheduler

The backend's "act as cluster" feature (`Kunden Firmen / Benutzer` → `Cluster Computer` permission,
`Cluster / Clusterliste`) submits simulation jobs to a Slurm cluster. This section installs just the
Slurm scheduler itself, as a single-node cluster — matching how it's actually set up on this
machine (native systemd services, not containers). It does **not** cover the separate job-execution
layer (the private `runner`/`simodlib`/`simodclustercomponents` repos and the `sicle` config bundle)
that turns a submitted Slurm job into an actual running simulation — that's a different, private
piece of the pipeline, out of scope here.

### 8a. Install packages

```shell
sudo apt-get update
sudo apt-get install -y munge slurmd slurmctld slurmdbd mariadb-server
```

### 8b. Munge (inter-daemon authentication)

```shell
sudo systemctl enable --now munge
systemctl status munge --no-pager
```

A single-node setup works with the key `apt` generated for you at `/etc/munge/munge.key` — nothing
further to do unless you're joining more than one node to the cluster.

### 8c. Accounting database

Slurm's accounting (`AccountingStorageType=accounting_storage/slurmdbd`) needs its own MariaDB
database and user:

```shell
sudo mysql -e "CREATE DATABASE slurm_acct_db;"
sudo mysql -e "CREATE USER 'slurm'@'localhost' IDENTIFIED BY '<pick-a-password>';"
sudo mysql -e "GRANT ALL PRIVILEGES ON slurm_acct_db.* TO 'slurm'@'localhost';"
```

### 8d. `/etc/slurm/slurmdbd.conf`

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

### 8e. `/etc/slurm/slurm.conf`

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

### 8f. Start everything and register the cluster

```shell
sudo systemctl enable --now slurmdbd
sudo systemctl enable --now slurmctld
sudo systemctl enable --now slurmd

# slurmdbd only accepts accounting data for clusters it knows about
sacctmgr -i add cluster localcluster
sudo systemctl restart slurmctld
```

### 8g. Verify

```shell
sinfo            # expect: PARTITION LocalQ, STATE idle, your hostname listed
srun hostname    # runs `hostname` as a Slurm job on the local node
sacctmgr show cluster
```

The `partitions` block (`name: LocalQ`, `num_cores_per_node`, `max_nodes`, `memory_per_node`) that a
cluster-agent config (`cluster_config.scheduler_type: slurm`) points at should match the
`PartitionName=LocalQ` line above — adjust either side to keep them consistent.

## 9. Cleanup

```shell
kind delete cluster --name platform-mesh

# StrömungsRaum stack
cd ~/repo/ianus-simod && task dev:down

# Slurm (only if you want to tear the scheduler down too)
sudo systemctl disable --now slurmd slurmctld slurmdbd munge
```
