# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

ECOTWIN Reference Architecture: three **Platform Mesh / kcp providers** (`injectionmolding`, `skz`, `materials`) plus a Python client/API layer ("ecotwin") for the StrömungsRaum (SR) simulation platform, built for IANUS Simulation. Three independent codebases share one repo:

- **Go**: kcp-based Kubernetes operator (`cmd/`, `apis/`, `operator/`, `pkg/`, `config/`)
- **Python**: `twinctl` CLI + FastAPI "Digital Twin API" that talks to the external SR/SIMOD backend (`ecotwin/`)
- **Angular**: Platform Mesh Luigi microfrontend (`portal/`)

There is no CI (no `.github/workflows/`), no linter configs beyond `go fmt`/`go vet`, and no automated Python test suite — keep that in mind when asked to "run the tests" or "check CI".

See [agent_instructions.md](agent_instructions.md) for cross-repo context (this repo's `ecotwin/common/api_connector.py` is a client of the sibling `ianus-simod` Laravel backend) and agent working rules — read it too, it deliberately doesn't repeat what's here.

## Commands

### Go (operator)
```shell
make build-materials          # builds bin/materials and bin/materials-init (fmt + vet first)
make run-materials             # go run ./cmd/materials/main.go --endpointslice=materials.platform-mesh.io
make init-materials             # bootstrap provider resources; needs KUBECONFIG, optional HOST_OVERRIDE
make fmt vet tidy              # go fmt / go vet / go mod tidy
make generate-materials        # controller-gen deepcopy + manifests + apiresourceschemas (see Codegen below)
go test ./...                  # standard go test; no custom test tooling
```
Every provider (`injectionmolding`, `skz`, `materials`) has the same targets suffixed `-<name>`
(`build-materials`, `init-materials`, `generate-materials`, ...; `skz` has no operator, only
`build-init-skz`/`init-skz`/`generate-skz`). Bare `make build`/`generate`/`manifests`/`apiresourceschemas`
fan out across all providers (`init`/`run` don't - each needs its own kcp workspace context, so they stay
per-provider only). Run `make help` for the full, current list rather than trusting any doc's enumeration
of it.

### Images / local cluster
```shell
make image-build-materials / image-push-materials  # materials controller image (deploy/materials.Dockerfile)
make portal-image-build / portal-image-push         # shared portal image (deploy/portal.Dockerfile)
make api-image-build / api-image-push               # ecotwin API image (deploy/api.Dockerfile)
make kind-load-materials / kind-load-portal / kind-load-api  # load into kind cluster $(KIND_CLUSTER), default "platform-mesh"
make help                                            # list all Makefile targets with descriptions
```

### Python (ecotwin)
```shell
uv sync                                  # install deps (uv-managed, see pyproject.toml / uv.lock)
uv run ecotwin/scripts/twinctl.py --help  # the twinctl CLI — run directly, NOT via installed `twinctl` entry point (see gotcha below)
uv run uvicorn ecotwin.api_server:app --reload   # run the Digital Twin API locally
docker build -t ecotwin-api -f deploy/api.Dockerfile .   # build API image directly
```
There is no pytest suite and no lint config configured for this package.

### Portal (Angular, run from `portal/`)
```shell
npm install
npm start           # ng serve, http://localhost:4200
npm run build        # ng build
npm test             # ng test
```

## Architecture

### Go operator: kcp multi-cluster, not vanilla controller-runtime

`cmd/materials/main.go` (and each other provider's `cmd/<name>/main.go`) builds a **kcp-aware** manager (`sigs.k8s.io/multicluster-runtime` + `github.com/kcp-dev/multicluster-provider`), not a plain controller-runtime manager. It watches an `APIExportEndpointSlice` (name = `--endpointslice` flag, default `<name>.platform-mesh.io`) and reconciles per logical cluster via `mgr.GetCluster(ctx, req.ClusterName)`. Reconciler logic lives in **`operator/<name>/`** (one file per kind, using `mcbuilder`/`mcreconcile` types) — separate from `cmd/<name>` but same Go module, not a submodule.

There used to be a placeholder `Model` CRD (`Spec.Intent` string, canned-string reconciler) — removed
(2026-09-07) along with its reconciler and RBAC/APIExport entries. There was also, for a while, an `ianus`
provider with `Component`/`DigitalTwin`/`ExecutionParameter`/`Simulation` CRDs that replaced it — removed
in turn (2026-11-18) once it no longer played any role; `injectionmolding` and `materials` are the
providers that still drive the SIMOD backend today. Don't reintroduce a generic `Model`/`Intent` type.

### Three providers, one repo, one repeated file pattern

`injectionmolding`, `skz`, and `materials` each live
in the same repo as independent Platform Mesh providers — own kcp workspace, own `APIExport`, own binaries.
Each follows the same per-provider file set, keyed by a `<name>` (`injectionmolding`, `skz`, `materials`):

- `apis/<name>/v1alpha1/` — CRD Go types (`doc.go`, `groupversion_info.go`, `common_types.go`'s own copy of
  `SecretReference`, `<kind>_types.go`)
- `operator/<name>/` — reconciler(s) (skipped for `skz`, a dummy/demo provider with no operator)
- `cmd/<name>/main.go` + `cmd/<name>-init/main.go` — operator entrypoint + bootstrap CLI (skz has only
  `cmd/skz-init`)
- `config/{kcp,provider,controller}-<name>/` — the same three embed.FS config sets the "`config/` is Go
  embed.FS" section below describes, duplicated per provider (skz has no `config/controller-skz` - no
  operator to give an identity to)
- `pkg/bootstrap/bootstrap_<Name>.go` — wires those three sets, mirroring `bootstrap.go`'s `Bootstrap` func
- `deploy/<name>.Dockerfile` + `deploy/helm/<name>-controller/` — image + Helm chart (skz has neither)
- `Makefile` targets suffixed `-<name>` (`build-<name>`, `init-<name>`, `generate-<name>`, `manifests-<name>`,
  `apiresourceschemas-<name>`, `image-build-<name>`, `kind-load-<name>`) — run `make help` for the full,
  current list rather than trusting any doc's enumeration of it
- optionally `portal/src/app/<name>/` (Angular page) if the provider needs a custom UI — see "Portal" below

**`skz` is the odd one out, twice over.** First, the name: users and API consumers know it as the
"Material Provider A - Materials" - API group and APIExport `genericmaterials.platform-mesh.io`, portal path
`/generic-materials`. It began as a demo stand-in for the SKZ institute and only that outward-facing surface
was renamed; the Go package (`apis/skz`), `config/*-skz` dirs, `cmd/skz-init`, `portal/src/app/skz` and the
`make *-skz` targets deliberately keep the internal name `skz`. Second, it is a static, read-only catalog
served through kcp cached resources. Its
`Material`s are identical for every consumer, so consumers don't create them. `make init-skz`
(`pkg/bootstrap/bootstrap_skz.go`) installs the (cluster-scoped) `Material` CRD into the *provider*
workspace, creates one `Material` per `config/catalog-skz/materials/*.json` there (file name = object name;
add a material by dropping in a JSON file and re-running `make init-skz`), and publishes them with a
`ClusterCachedResource` + `ClusterCachedResourceEndpointSlice` (`config/catalog-skz/*.yaml`, needs kcp's
`CacheAPIs` feature gate). The `genericmaterials.platform-mesh.io` APIExport serves `materials` with `storage.virtual`
pointing at that endpoint slice - set by a mutator in `bootstrap_skz.go`, **not** in
`config/kcp-skz/apiexport-*.yaml`, because apigen rewrites that file with `storage: {crd: {}}` (and prunes
anything else in that directory) on every `make generate-skz`. Consumers therefore see the materials
read-only and cluster-scoped; the portal's SKZ page only lists them.

**Don't hand-write a new provider from scratch by pattern-matching this list alone** —
[docs/creating-a-provider.md](docs/creating-a-provider.md) is the step-by-step, file-by-file walkthrough
with templates and the gotchas (controller-gen's `allowDangerousTypes`, the kcp permission-claim dance,
icon-data splicing, the `_Input` GraphQL naming assumption) encountered building `materials`.

### SIMOD Platform API: shared native Go client

`pkg/simod/client.go` is a native Go HTTP client for the `ianus-simod` backend's token-based
`/api/platform/...` "Platform API" — no Python involved, the Go counterpart to
`ecotwin/common/api_connector.py`'s `JobHandler` (that Python client remains a prototype, untouched).
`Login`/`pkg/simod/auth.go`'s `LoginFromSecret` are shared by every provider operator that talks to SIMOD;
each provider's reconciler then uses only the calls its own CRDs need:
`operator/injectionmolding/injectionmoldingservice_controller.go` uses `ListSimulations`/`GetSimulation`
(read-only, it doesn't create simulations itself), and `operator/materials/*_controller.go` uses
`CreateMaterial`/`ListMaterials`. Credentials come from each CR's `Spec.SecretRefs[0]` → a same-namespace
`Secret` with `email`/`password` keys, logged in fresh per reconcile (no token caching yet). Base URL is
each operator's own `--simod-base-url` flag (default `https://backend.simod.de`).

**Two different failure classes, handled differently.** Every reconciler writes `status.phase = "Failed"`
+ `status.message` on any SIMOD error (already rendered by the portal as a visible banner - see
`injection-molding.component.html`/`materials.component.html`), but the two realistic causes are distinct
and shouldn't be treated the same:
- **Missing/invalid credentials Secret** - a static, human-fixable config problem, not a bug.
  `pkg/simod/auth.go`'s `LoginFromSecret` returns a `*simod.CredentialsError` for this (Secret not found,
  or found but missing a non-empty `email`/`password` key), detected via `simod.IsCredentialsError`. This
  is already fast (a k8s `Get`, not a network call to SIMOD), so it reaches `status.message` immediately.
  Each provider's `operator/<name>/simod_client.go` has a `loginResult(err)` helper reconcilers call
  instead of returning the raw error: a `CredentialsError` gets requeued directly at a fixed
  `credentialsRecheckInterval` (30s) so it self-heals soon after an admin creates the Secret, without
  going through controller-runtime's default escalating backoff or its "reconciler error" log level -
  that's reserved for the other class below.
- **SIMOD backend unreachable/slow** - `pkg/simod/client.go`'s `NewClient` sets a 15s HTTP client timeout
  (not `http.DefaultClient`, which has none) specifically so a black-holed connection can't hang the
  reconciler - and therefore `status` - forever. This path still returns the raw error from `loginResult`,
  so it gets controller-runtime's normal backoff.

**kcp gotcha, verified against a live local Platform Mesh (2026-09-07):** a bound consumer workspace's `Secret`s are invisible to the provider's operator by default — kcp's APIExport virtual workspace only exposes core resources the export explicitly requests via `spec.permissionClaims`, and the consumer's `APIBinding` must separately accept the same claim (`spec.permissionClaims[].state: Accepted`, with a `selector` — `{matchAll: true}` works) before the operator's `client.Get` on a `Secret` stops 404ing as "no REST mapping". `config/kcp-materials/apiexport-materials.platform-mesh.io.yaml` (and `injectionmolding`'s equivalent) claims `secrets` (`get`) for exactly this reason — any new CRD whose reconciler reads a core-group resource from the consumer workspace needs the same treatment, on both the APIExport (provider side, this repo) and every consumer's APIBinding (their side, out of this repo's control).

### Codegen pipeline — edit apis/, never hand-edit generated YAML

`apis/` is the source of truth. `make generate-<name>` runs, in order: `controller-gen object` (deepcopy) → `make manifests-<name>` (CRD → `config/crds-<name>/*.yaml` via controller-gen) → `make apiresourceschemas-<name>` (runs the `apigen` tool to convert those CRDs into kcp-native `APIResourceSchema`s under `config/kcp-<name>/`). Never hand-edit `config/crds-<name>/` or `config/kcp-<name>/` or `zz_generated.deepcopy.go` — change `apis/` and regenerate.

### `config/` is Go embed.FS, not kustomize

`config/{controller,kcp,provider}-<name>/` are each a small Go package wrapping an `embed.FS` of static YAML. `pkg/bootstrap/bootstrap_<Name>.go` (invoked by the `<name>-init` binary / `make init-<name>`) applies these three sets in order via a discovery/dynamic client with create-or-update + retry semantics (the create/update/retry helpers themselves live in `pkg/bootstrap/bootstrap.go`, shared by every provider's `bootstrap_<Name>.go`):
1. `config/kcp-<name>` — the kcp `APIExport`/`APIResourceSchema`
2. `config/provider-<name>` — Platform Mesh provider registration (`ContentConfiguration`, `ProviderMetadata`, RBAC)
3. `config/controller-<name>` — controller identity/RBAC (namespace, service accounts, clusterrole/binding)

It then waits for the `<name>-controller-token` ServiceAccount token Secret and writes a standalone `<name>-controller-kubeconfig` Secret (optionally rewriting the server host via `--host-override`/`HOST_OVERRIDE`) — this is the kubeconfig the running operator (`cmd/<name>`) later uses.

`hack/go-install.sh` is the versioned-tool-install pattern used for both `controller-gen` and `apigen` (installs into `hack/tools/`, gitignored).

### Python: two entry points, one broken

- `ecotwin/scripts/twinctl.py` is the real `twinctl` Typer CLI (PEP 723 inline-script, run via `uv run`). Subcommands lazily import `ecotwin/info/group.py` (`info system`, `info models`), `ecotwin/login/store.py` (`login store`, persists SR credentials to `~/.config/ecotwin/.env`), and `ecotwin/submission/group.py` (`submission submit`, the token-based "full submission" API call — builds geometry, freezes a twin, creates a simulation in one call).
- **Gotcha**: `pyproject.toml` declares the console-script entry point as `twinctl = "ecotwin.twinctl:twinctl"`, but no `ecotwin/twinctl.py` module exists — this entry point is dead. Always invoke via `uv run ecotwin/scripts/twinctl.py`, not the installed `twinctl` shim.
- `ecotwin/api_server.py` is the FastAPI "Digital Twin API" from the README. `GET /models` is currently hardcoded to `System.INT` via `ecotwin.common.api_connector.create_api_connector`.
- `ecotwin/common/api_connector.py` is the real integration with the external SIMOD/StrömungsRaum backend (`backend.{prod,int,test,dev}.simod.de`): form-login + CSRF scraping (`BackendConnector`), token auth (`JobHandler`), and model-list scraping (`AjaxApiHandler`/`ApiHandler`). Credentials come from `EMAIL`/`PASSWD` (or `EMAIL_<SYS>`/`PASSWD_<SYS>`) env vars, loaded from root `.env` or `~/.config/ecotwin/.env`. The legacy `BackendConnector`/`AjaxApiHandler` scraping path is kept working deliberately (pass `legacy=True` to `create_api_handler()`) alongside the newer token-based API — don't remove it without an explicit ask.

### Portal: three pages, one per provider with a UI, hitting real data

`portal/` serves three pages, one per provider that has one: `portal/src/app/injection-molding/`
(`InjectionMoldingComponent`/`InjectionMoldingServiceClient`), `portal/src/app/skz/` (`SkzComponent`/
`SkzService`), and `portal/src/app/materials/` (`MaterialsComponent`/`MaterialsService`) - all live GraphQL
calls against the Platform Mesh CRD Gateway (query pattern `{apiGroup}_{version}_{Kind}`, e.g.
`injectionmolding_platform_mesh_io.v1alpha1.InjectionMoldingServices`), no mock data, sharing one
`GraphQLClientService` (`portal/src/app/shared/graphql-client.service.ts`).

There is no router (removed deliberately - see the git history around the `InjectionMoldingService` CRD
rework for why Angular's Router doesn't fit this deployment's per-prefix relative `<base href>`):
`AppComponent` instead picks a component directly from `window.location.pathname` (`includes("/generic-materials")`,
`includes("/materials")`, else injection-molding) - each page is deployed under its own Gateway path
prefix (`/injection-molding`, `/generic-materials`, `/materials`), so Angular's own Router would see zero remaining path
segments regardless of which prefix loaded the page. Adding a fourth page means adding a fourth branch
here, not a route. `AppComponent` re-reads the path on every `popstate`, and that listener is load-bearing:
the three pages share one origin, so Luigi reuses the iframe when the user clicks from one provider's nav
entry to another's and only tells LuigiClient to `history.replaceState` + dispatch `popstate` instead of
loading the new URL. The materials page additionally queries the `skz` provider's `Material` CRD directly
(cross-provider GraphQL query, not a cross-APIExport kcp permission claim - see
`apis/materials/v1alpha1/materialsubmission_types.go`'s package doc) to let the user pick a source
material to submit.

An earlier `models/` page (mocked `Model` CRD data, matching the now-removed generic `Model`/`Intent` type)
was deleted once it was confirmed to have no real backing CRD. See `portal/README.md` for the general
Luigi microfrontend integration model (context, `ContentConfiguration` registration, local dev proxy) -
some of its code samples still reference the old `models` page as an illustrative example, not as
something present in this repo.

### Deploy

`deploy/helm/ianus-portal/` (the shared Luigi portal, serving all three providers' panels),
`deploy/helm/injectionmolding-controller/`, and `deploy/helm/materials-controller/` are the real charts
used to deploy this repo's images (built from `deploy/portal.Dockerfile`, `deploy/api.Dockerfile`,
`deploy/injectionmolding.Dockerfile`, `deploy/materials.Dockerfile`). `skz` has no chart/image - it's a
dummy provider with no operator (bootstrap-only, `cmd/skz-init`). Every `*-controller` chart shares the
same `common` chart dependency (`charts/common-X.Y.Z.tgz`, vendored per chart - copy the `.tgz` byte-for-byte
into a new provider's chart rather than re-fetching it, see `docs/creating-a-provider.md`).

### Docs

Docs are published as an mdBook (`book.toml`, built output in `book/`, source in `docs/`, TOC in `docs/SUMMARY.md`). Some README links to docs are stale (e.g. links to `./docs/twinctl.md`, actual path is `docs/twinctl/twinctl.md`) — prefer `docs/SUMMARY.md` to find the current path. `docs/00-readme-simod-platform-mesh-yamls.md` + `manifests/phase1`/`phase2` are an illustrative walkthrough (contain hardcoded personal paths), separate from the actual codegen'd `config/` used by `pkg/bootstrap`.
