# Platform Mesh Providers

A template for building Platform Mesh providers. This repo demonstrates how to create a provider that exposes APIs through kcp and integrates with the Platform Mesh UI.

## Start Platform Mesh locally

[Local Setup for Platform Mesh](https://github.com/platform-mesh/helm-charts/tree/main/local-setup)

## What This Repo Does

Each provider here exposes its own API (e.g. `materials.platform-mesh.io`) through kcp and registers
with Platform Mesh. It is based on [Platform Mesh - Provider Quickstart](https://github.com/platform-mesh/provider-quickstart).

Steps:

1. **Define and export APIs via kcp** - Using `APIExport` and `APIResourceSchema` resources
2. **Register as a Platform Mesh provider** - Using `ProviderMetadata` to describe your provider
3. **Configure UI integration** - Using `ContentConfiguration` to add navigation and views

## Key Resources

### ProviderMetadata

Registers your provider with Platform Mesh, including display name, description, contacts, and icons:

```yaml
apiVersion: ui.platform-mesh.io/v1alpha1
kind: ProviderMetadata
metadata:
  name: materials.platform-mesh.io  # Must match your APIExport name
spec:
  displayName: IANUS Simulation - Materials Provider
  description: ...
  contacts: [...]
  icon: {...}
```

### ContentConfiguration

Configures how your resources appear in the Platform Mesh UI. The key label links it to your APIExport:

```yaml
apiVersion: ui.platform-mesh.io/v1alpha1
kind: ContentConfiguration
metadata:
  labels:
    ui.platform-mesh.io/content-for: materials.platform-mesh.io  # Links to your APIExport
  name: models-ui
spec:
  inlineConfiguration:
    content: |
      { "luigiConfigFragment": { ... } }
```

The `ui.platform-mesh.io/content-for` label is critical - it associates your UI configuration with your APIExport.

This repo's `config/provider-<name>/contentconfiguration.yaml` + `providermetadata.yaml` (inline-JSON `ContentConfiguration`, `ui.platform-mesh.io/v1alpha1`) already follow this pattern and are validated against current Platform Mesh convention — no migration needed here.

## Project Structure

This repo hosts three providers (`ianus-injection-molding`, `skz`, `ianus-materials`), each
following the same file layout keyed by a `<name>` (`injectionmolding`, `skz`, `materials`). See
[creating-a-provider.md](./creating-a-provider.md) for the full file-by-file walkthrough of that pattern,
and [quick_start.md](./quick_start.md) for the actual end-to-end setup commands.

```
├── apis/
│   ├── injectionmolding/v1alpha1/   # InjectionMoldingService CRD
│   ├── skz/v1alpha1/                # Material CRD (dummy/demo provider)
│   └── materials/v1alpha1/          # MaterialCatalog, MaterialSubmission CRDs
├── cmd/
│   ├── injectionmolding-init/, injectionmolding/
│   ├── skz-init/                     # skz has no operator - bootstrap only
│   └── materials-init/, materials/
├── config/
│   ├── kcp-<name>/                   # kcp resources (APIExport, APIResourceSchema)
│   ├── provider-<name>/              # Provider resources (ProviderMetadata, ContentConfiguration, RBAC)
│   └── controller-<name>/            # Controller identity/RBAC (not present for skz)
├── operator/
│   ├── injectionmolding/, materials/   # Reconcilers (none for skz)
├── pkg/
│   ├── bootstrap/                    # Bootstrap logic for applying resources (one func per provider)
│   └── simod/                        # Shared SIMOD Platform API HTTP client
├── deploy/helm/
│   ├── ianus-portal/                 # Shared Luigi portal deployment, serves all three providers' panels
│   ├── injectionmolding-controller/
│   └── materials-controller/         # skz has no chart - no operator to deploy
└── portal/                           # Luigi microfrontend (Angular) - injection-molding, skz, materials pages
```

## Usage

> **Important:** Providers must live in a dedicated workspace type within a separate tree. This means platform administrators must configure providers using the **admin kubeconfig**. Regular user kubeconfigs will not have the necessary permissions to create provider workspaces. This is bound to change and improve in the future, but for now you must use the admin kubeconfig to set up your provider.

See [quick_start.md](./quick_start.md) for the actual, up-to-date end-to-end commands (admin/compute
kubeconfig setup, portal deployment, and each provider's registration steps) as run on this dev machine.
The sections below describe each provider's own CRDs/operator/chart and how to register it; the shared
mechanics (workspace creation, `make init-<name>`, image build/load, `helm upgrade --install`) are the
same pattern for every provider.

## Simulations Provider

The first provider in this repo ("IANUS Simulation - Simulations",
APIExport `injectionmolding.platform-mesh.io`) - own workspace, own CRD (`InjectionMoldingService`),
own operator binary/image/Helm chart. It
doesn't create simulations itself: the operator logs into the real SIMOD backend, lists the
account's existing Components/DigitalTwins/Simulations into `status`, and the portal shows a
link to the SIMOD web UI (`https://<env>.simod.de/models3D`, derived from `--simod-base-url`)
for actually creating one. See `apis/injectionmolding/v1alpha1/`,
`operator/injectionmolding/`, `pkg/simod/client.go`'s `List*` methods, and
`portal/src/app/injection-molding/`.

Steps, own workspace and binaries:

```bash
# 1. Create the provider workspace
KUBECONFIG=$PM_KUBECONFIG kubectl ws use :
KUBECONFIG=$PM_KUBECONFIG kubectl ws create providers --type=root:providers --enter --ignore-existing
KUBECONFIG=$PM_KUBECONFIG kubectl ws create ianus-injection-molding --type=root:provider --enter --ignore-existing

# 2. Bootstrap (own binary, own workspace)
KUBECONFIG=$PM_KUBECONFIG make init-injectionmolding HOST_OVERRIDE=https://frontproxy-front-proxy.platform-mesh-system:8443

# 3. Extract the operator kubeconfig
KUBECONFIG=$PM_KUBECONFIG kubectl get secret injectionmolding-controller-kubeconfig -n default -o jsonpath='{.data.kubeconfig}' | base64 -d > injectionmolding.kubeconfig

# 4. Run locally (optional)
TODO: Ensure that the `injectionmolding.kubeconfig` points to a reachable API endpoint for local development.

KUBECONFIG=./injectionmolding.kubeconfig go run ./cmd/injectionmolding --endpointslice=injectionmolding.platform-mesh.io --simod-base-url=https://backend.dev.simod.de

# 5. Build and load the image
export IMAGE_TAG=platform-mesh
make image-build-injectionmolding kind-load-injectionmolding IMAGE_TAG=$IMAGE_TAG

# 6. Deploy (devBackend.enabled only needed against this machine's local SIMOD backend - see
# "Local-sandbox-only gotchas" below)
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl create namespace provider-injection-molding
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl delete secret injectionmolding-controller-kubeconfig -n provider-injection-molding --ignore-not-found
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl create secret generic injectionmolding-controller-kubeconfig \
  --from-file=kubeconfig=./injectionmolding.kubeconfig -n provider-injection-molding

KUBECONFIG=$COMPUTE_KUBECONFIG helm upgrade --install injectionmolding-controller ./deploy/helm/injectionmolding-controller \
  --namespace provider-injection-molding \
  --set image.tag=$IMAGE_TAG \
  --set image.pullPolicy=IfNotPresent \
  --set devBackend.enabled=true
```

`--simod-base-url` (default `https://backend.dev.simod.de`, templated via the chart's
`controller.simodBaseURL` value) drives both the SIMOD API calls and the derived
`status.url` link - bump it to `backend.int.simod.de` then `backend.simod.de` as
`feat/platform-api` (on the `ianus-simod` side) moves from dev to int to prod. No code
change needed, just a different `--set controller.simodBaseURL=...` at install time.

The CR itself (`spec.secretRefs` -> a Secret with `email`/`password` from
`~/.sicle/ecotwin-sava.yaml`'s matching `simod_api.backends.<env>` entry, same pattern as
the SIMOD CRDs above) and `spec.refreshRequestedAt` (bumped by the portal's Refresh button,
or by hand for testing: `kubectl patch injectionmoldingservice <name> --type=merge -p
'{"spec":{"refreshRequestedAt":"'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"}}'`) live in a *consumer*
workspace that's bound the `injectionmolding.platform-mesh.io` APIExport - not in the
provider workspace itself.

### Local-sandbox-only gotchas (real dev/int/prod shouldn't hit these)

Verified live against this repo's local kind cluster, talking to the real (self-signed,
unmerged-branch) `backend.dev.simod.de`. A real deployment with proper DNS and a CA-signed
backend cert needs none of this - int/prod are never run on this machine and reach their real
DNS names/certs normally through standard cluster DNS.

Both workarounds are now **baked into `injectionmolding-controller`'s and `materials-controller`'s
charts** behind one flag, `--set devBackend.enabled=true` (default `false`, so a real deployment
is unaffected):
1. **DNS**: `backend.dev.simod.de` resolves to `127.0.0.1` from *inside* the kind cluster (a
   local-tunnel artifact), useless from a pod's own network namespace - nothing listens on the
   pod's loopback. `devBackend.enabled` adds a `hostAliases` entry mapping it to
   `devBackend.hostAliasIP` (default `172.18.0.1`, kind's default docker network gateway -
   override via `--set` if yours differs: `docker network inspect kind --format
   '{{(index .IPAM.Config 0).Gateway}}'`).
2. **TLS trust, in-container**: that backend serves a self-signed cert the operator's distroless
   image's own CA set doesn't trust ("x509: certificate signed by unknown authority").
   `devBackend.enabled` also mounts `files/dev-ca-bundle.crt` (committed in each chart - this
   image's own base `ca-certificates.crt`, captured once via `docker cp`, plus
   `backend.dev.simod.de`'s self-signed cert appended) over `/etc/ssl/certs`, replacing it
   wholesale - no live, not-in-chart ConfigMap to pre-create anymore. If the backend's cert is
   ever rotated, re-capture and re-commit both chart's `files/dev-ca-bundle.crt`:
   ```bash
   echo | openssl s_client -connect backend.dev.simod.de:443 -servername backend.dev.simod.de 2>/dev/null | openssl x509 > dev.simod.de.crt
   cid=$(docker create ghcr.io/platform-mesh/materials-controller:$IMAGE_TAG)
   docker cp "$cid:/etc/ssl/certs/ca-certificates.crt" base-ca-bundle.crt && docker rm "$cid"
   cat base-ca-bundle.crt dev.simod.de.crt > deploy/helm/materials-controller/files/dev-ca-bundle.crt
   cp deploy/helm/materials-controller/files/dev-ca-bundle.crt deploy/helm/injectionmolding-controller/files/dev-ca-bundle.crt
   ```
   (Both charts' base images share the same pinned distroless digest, so one capture covers both.)

## IANUS Simulation - Materials Provider

A second, independent provider ("IANUS Simulation - Materials", APIExport `materials.platform-mesh.io`)
lives alongside `ianus-injection-molding` in this same repo - own workspace, own two CRDs
(`MaterialCatalog`, `MaterialSubmission`), own operator binary/image/Helm chart. It demonstrates the
same multi-provider provisioning story the SKZ dummy provider's `ProviderMetadata` describes, carried
through for real: a customer characterizes a material in the SKZ Materials provider (`apis/skz/v1alpha1`'s
`Material` CRD, shaped 1:1 like `docs/assets/EastmanPlastics1000.json`), then provisions this provider
to submit that characterization to the real StrömungsRaum/SIMOD backend.

- `MaterialCatalog` is a **singleton per account** (the portal creates the first one it doesn't find,
  named `catalog`) - it only browses StrömungsRaum's existing materials into `status.availableMaterials`,
  refreshed via `spec.refreshRequestedAt` the same way `InjectionMoldingService.status.availableSimulations`
  works.
- `MaterialSubmission` is **HttpBin-style**, one per portal "+" click, create-once like the other SIMOD
  CRDs (`/api/platform/materials` has no update/delete route either). Its `spec.material` is a full
  copy of a chosen SKZ `Material`'s spec, embedded verbatim by the portal at creation time - the
  materials-controller never reads the SKZ CRD itself. Cross-provider composition happens in the portal
  (a single GraphQL client can query any APIExport the consumer workspace is bound to), not via a
  cross-APIExport kcp permission claim; see `apis/materials/v1alpha1/materialsubmission_types.go`'s
  package doc for why. The reconciler's `operator/materials/materialsubmission_controller.go` translates
  the CRD's physical-model fields (e.g. `tempModel.c1`) into the backend's generic
  `sections.{temperature,viscosity,heat,injection_molding,pvt}` `{method, parameters}` shape -
  `pkg/simod/client.go`'s `CreateMaterial`/`ListMaterials`.

See `apis/materials/v1alpha1/`, `apis/skz/v1alpha1/material_types.go`, `operator/materials/`,
`pkg/simod/client.go`'s `Material*` types, and `portal/src/app/materials/`.

The provider is known to users as **Material Provider A - Materials** (API group and APIExport
`genericmaterials.platform-mesh.io`, portal path `/generic-materials`); `skz` is only its internal name in
this repo (Go package, `config/*-skz`, `make *-skz`).

Its materials are a static catalog, not something a user uploads: `make init-skz` creates
one `Material` per `config/catalog-skz/materials/*.json` in the provider workspace and publishes them
read-only to every consumer through kcp cached resources (`ClusterCachedResource`, requires the `CacheAPIs`
feature gate on kcp). Run it against the workspace that hosts the `genericmaterials.platform-mesh.io` APIExport:

```bash
KUBECONFIG=$PM_KUBECONFIG make init-skz
KUBECONFIG=$PM_KUBECONFIG kubectl get clustercachedresource materials -o jsonpath='{.status.resourceCounts}'
# {"cache":3,"local":3} once replicated; consumers then see them with `kubectl get materials.genericmaterials.platform-mesh.io`
```

Steps mirror the `ianus-injection-molding` walkthrough above, with its own workspace and binaries:

```bash
# 1. Create the provider workspace (sibling of `ianus-injection-molding`)
KUBECONFIG=$PM_KUBECONFIG kubectl ws use :
KUBECONFIG=$PM_KUBECONFIG kubectl ws create providers --type=root:providers --enter --ignore-existing
KUBECONFIG=$PM_KUBECONFIG kubectl ws create ianus-materials --type=root:provider --enter --ignore-existing

# 2. Bootstrap (own binary, own workspace)
KUBECONFIG=$PM_KUBECONFIG make init-materials HOST_OVERRIDE=https://frontproxy-front-proxy.platform-mesh-system:8443

# 3. Extract the operator kubeconfig
KUBECONFIG=$PM_KUBECONFIG kubectl get secret materials-controller-kubeconfig -n default -o jsonpath='{.data.kubeconfig}' | base64 -d > materials.kubeconfig

# 4. Run locally (optional)
KUBECONFIG=./materials.kubeconfig go run ./cmd/materials --endpointslice=materials.platform-mesh.io --simod-base-url=https://backend.dev.simod.de

# 5. Build and load the image
export IMAGE_TAG=platform-mesh
make image-build-materials kind-load-materials IMAGE_TAG=$IMAGE_TAG

# 6. Deploy
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl create namespace provider-materials
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl delete secret materials-controller-kubeconfig -n provider-materials --ignore-not-found
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl create secret generic materials-controller-kubeconfig \
  --from-file=kubeconfig=./materials.kubeconfig -n provider-materials

KUBECONFIG=$COMPUTE_KUBECONFIG helm upgrade --install materials-controller ./deploy/helm/materials-controller \
  --namespace provider-materials \
  --set image.tag=$IMAGE_TAG \
  --set image.pullPolicy=IfNotPresent \
  --set devBackend.enabled=true
```

Like injection molding, `--simod-base-url` (default `https://backend.dev.simod.de`, chart value
`controller.simodBaseURL`, also overridable per-account via the credentials Secret's `baseUrl` key -
see `pkg/simod/auth.go`) drives the materials API calls - bump it the same way as dev moves to int to
prod, and leave `devBackend.enabled` off once it does (see "Local-sandbox-only gotchas" above - those
workarounds are specific to this machine's local dev SIMOD backend, not int/prod).

## Debugging

Assuming your provider workspace is `ianus-materials` under the `providers` tree:

### Check Marketplace Entries

View your provider's marketplace entry (combines APIExport + ProviderMetadata):

```bash
kubectl --server="https://kcp.api.pm.127-0-0-1.sslip.io:8443/services/marketplace/clusters/root:providers:ianus-materials" get marketplaceentries -A
kubectl --server="https://kcp.api.pm.127-0-0-1.sslip.io:8443/services/marketplace/clusters/root:providers:ianus-materials" get marketplaceentries -A -o yaml
```

### Check Content Configurations

View available API resources and content configurations:

```bash
kubectl --server="https://kcp.api.pm.127-0-0-1.sslip.io:8443/services/contentconfigurations/clusters/root:providers:ianus-materials" api-resources
kubectl --server="https://kcp.api.pm.127-0-0-1.sslip.io:8443/services/contentconfigurations/clusters/root:providers:ianus-materials" get contentconfigurations -A
kubectl --server="https://kcp.api.pm.127-0-0-1.sslip.io:8443/services/contentconfigurations/clusters/root:providers:ianus-materials" get contentconfigurations -A -o yaml
```

### URL Pattern

The server URL follows this pattern:
```
https://<host>/services/<virtual-workspace>/clusters/root:providers:<provider-workspace>
```

Where:
- `marketplace` - Virtual workspace for marketplace entries
- `contentconfigurations` - Virtual workspace for UI content configurations
- `<provider-workspace>` - Your provider workspace name (e.g., `ianus-materials`)

## Code Generation Tools

This project uses two key code generation tools:

### controller-gen

[controller-gen](https://github.com/kubernetes-sigs/controller-tools) is a Kubernetes code generator that:
- Generates **CRD manifests** from Go type definitions with `+kubebuilder` markers
- Generates **DeepCopy methods** (`zz_generated.deepcopy.go`) required for all Kubernetes API types
- Part of the controller-tools project from Kubernetes SIGs

### apigen

[apigen](https://github.com/kcp-dev/sdk) is a kcp-specific tool that:
- Converts standard Kubernetes **CRDs into APIResourceSchemas** for kcp
- APIResourceSchemas are kcp's way of defining API types that can be exported via `APIExport`
- Takes CRDs from `config/crds-<name>/` and outputs APIResourceSchemas to `config/kcp-<name>/`

**Generation flow:**
```
Go types (apis/<name>/) → controller-gen → CRDs (config/crds-<name>/) → apigen → APIResourceSchemas (config/kcp-<name>/)
```

## Make Targets

Every provider (`injectionmolding`, `skz`, `materials`) has the same targets, suffixed `-<name>`
(e.g. `build-materials`, `init-materials`, `generate-materials` — `skz` only has
`build-init-skz`/`init-skz`/`generate-skz`, no operator). **Run `make help` for the full, current list**
— this table only covers the shared shape, not every target; it drifts from `Makefile` otherwise.

| Target | Description |
|--------|-------------|
| `make build-materials` | Build both materials binaries (operator + init) |
| `make build-operator-materials` | Build the materials operator binary |
| `make build-init-materials` | Build the materials init/bootstrap binary |
| `make run-materials` | Run the materials operator locally |
| `make init-materials` | Bootstrap provider resources into workspace (requires KUBECONFIG, optional HOST_OVERRIDE) |
| `make generate-materials` | Generate code (deepcopy) and kcp resources |
| `make manifests-materials` | Generate CRD manifests from Go types |
| `make apiresourceschemas-materials` | Generate APIResourceSchemas from CRDs |
| `make image-build-materials` | Build controller container image |
| `make portal-image-build` | Build the shared portal container image |
| `make images` | Build all container images (all providers + portal) |
| `make kind-load-materials` | Load controller image into kind cluster |
| `make kind-load-portal` | Load portal image into kind cluster |
| `make kind-load-all` | Load all images into kind cluster (all providers + portal) |
| `make tools` | Install all required tools (controller-gen, apigen) |
| `make fmt` | Run go fmt |
| `make vet` | Run go vet |
| `make tidy` | Run go mod tidy |
| `make help` | Display help for all targets |

## Creating Your Own Provider

1. Fork this repo
2. Update the API group name (replace `materials.platform-mesh.io`)
3. Define your CRD schema in `config/kcp-<name>/`
4. Update `ProviderMetadata` with your provider details
5. Configure `ContentConfiguration` for your resource UI
6. Update RBAC to allow binding to your APIExport


## Custom Provider UI (Microfrontend)

Platform Mesh supports custom UIs for providers via microfrontends. This is useful when:

- Table views aren't sufficient for your resource representation
- You need custom wizards or multi-step flows (e.g., VM creation with SSH keys)
- You want to orchestrate multiple resources in a single view
- You need custom visualizations beyond standard lists

### How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                  Platform Mesh Portal                        │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                   Luigi Shell                        │    │
│  │                                                      │    │
│  │   Your MFE receives from Luigi context:             │    │
│  │   - token (Bearer auth for API calls)               │    │
│  │   - portalContext.crdGatewayApiUrl (GraphQL API)    │    │
│  │   - accountId (current account context)             │    │
│  │                                                      │    │
│  │   Your MFE can then:                                │    │
│  │   - Query/mutate K8s resources via GraphQL          │    │
│  │   - Use Luigi UX manager for alerts/dialogs         │    │
│  │   - Navigate within the portal                      │    │
│  │                                                      │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### Quick Start

1. **Generate a new microfrontend:**
   ```bash
   git clone https://github.com/openmfp/create-micro-frontend
   cd create-micro-frontend
   npm install && npm run build
   npx create-micro-frontend portal -y
   ```

2. **Run locally:**
   ```bash
   cd portal
   npm install
   npm start
   ```
   This serves your MFE at `http://localhost:4200`

3. **Enable local development mode** in the Portal to load your local MFE

4. **Deploy** by hosting the built files and creating a `ContentConfiguration` resource

### Example Implementation

This repo includes a complete working example in the `../portal/` directory (Luigi context integration, GraphQL queries/mutations for Kubernetes resources, SAP UI5 web components, local dev proxy). See [../portal/README.md](../portal/README.md) for the full Luigi microfrontend integration model (context/auth, GraphQL patterns, `ContentConfiguration` registration, navigation categories, local dev) — don't re-derive it here.

### Running as a Provider

As a provider, you are responsible for hosting your microfrontend (similar to running your operator). The MFE needs to be accessible to Portal users. Options include:

- Static hosting (S3, GCS, GitHub Pages, etc.)
- Container deployment alongside your operator
- Any web server that can serve static files

<p align="center"><img alt="Bundesministerium für Wirtschaft und Energie (BMWE)-EU funding logo" src="https://apeirora.eu/assets/img/BMWK-EU.png" width="400"/></p>
