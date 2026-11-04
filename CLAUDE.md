# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

ECOTWIN Reference Architecture: a **Platform Mesh / kcp provider** called "IANUS" plus a Python client/API layer ("ecotwin") for the StrömungsRaum (SR) simulation platform. Three independent codebases share one repo:

- **Go**: kcp-based Kubernetes operator (`cmd/`, `apis/`, `operator/`, `pkg/`, `config/`)
- **Python**: `twinctl` CLI + FastAPI "Digital Twin API" that talks to the external SR/SIMOD backend (`ecotwin/`)
- **Angular**: Platform Mesh Luigi microfrontend (`portal/`)

There is no CI (no `.github/workflows/`), no linter configs beyond `go fmt`/`go vet`, and no automated Python test suite — keep that in mind when asked to "run the tests" or "check CI".

See [agent_instructions.md](agent_instructions.md) for cross-repo context (this repo's `ecotwin/common/api_connector.py` is a client of the sibling `ianus-simod` Laravel backend) and agent working rules — read it too, it deliberately doesn't repeat what's here.

## Commands

### Go (operator)
```shell
make build              # builds bin/ianus and bin/ianus-init
make build-operator      # bin/ianus only (fmt + vet first)
make build-init           # bin/ianus-init only
make run                  # go run ./cmd/ianus/main.go --endpointslice=ianus.platform-mesh.io
make init                 # bootstrap provider resources; needs KUBECONFIG, optional HOST_OVERRIDE
make fmt vet tidy         # go fmt / go vet / go mod tidy
make generate             # controller-gen deepcopy + manifests + apiresourceschemas (see Codegen below)
go test ./...             # standard go test; no custom test tooling
```
Every other provider (`injectionmolding`, `skz`, `materials`) has the same targets suffixed `-<name>`
(`build-materials`, `init-materials`, `generate-materials`, ...) - run `make help` for the full, current
list rather than trusting any doc's enumeration of it.

### Images / local cluster
```shell
make image-build / image-push               # operator image (deploy/Dockerfile)
make portal-image-build / portal-image-push  # portal image (deploy/portal.Dockerfile)
make api-image-build / api-image-push        # ecotwin API image (deploy/api.Dockerfile)
make kind-load / kind-load-portal / kind-load-api  # load into kind cluster $(KIND_CLUSTER), default "platform-mesh"
make help                                    # list all Makefile targets with descriptions
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

`cmd/ianus/main.go` builds a **kcp-aware** manager (`sigs.k8s.io/multicluster-runtime` + `github.com/kcp-dev/multicluster-provider`), not a plain controller-runtime manager. It watches an `APIExportEndpointSlice` (name = `--endpointslice` flag, default `ianus.platform-mesh.io`) and reconciles per logical cluster via `mgr.GetCluster(ctx, req.ClusterName)`. Reconciler logic lives in **`operator/ianus/`** (one file per kind, using `mcbuilder`/`mcreconcile` types) — separate from `cmd/ianus` but same Go module, not a submodule.

There used to be a placeholder `Model` CRD (`Spec.Intent` string, canned-string reconciler) — removed (2026-09-07, no longer used once the real SIMOD CRDs below existed) along with its reconciler and RBAC/APIExport entries. Don't reintroduce a generic `Model`/`Intent` type; extend the SIMOD CRDs or add a new typed one instead.

### Four providers, one repo, one repeated file pattern

`ianus` (above) is no longer the only provider here: `injectionmolding`, `skz`, and `materials` each live
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

**Don't hand-write a new provider from scratch by pattern-matching this list alone** —
[docs/creating-a-provider.md](docs/creating-a-provider.md) is the step-by-step, file-by-file walkthrough
with templates and the gotchas (controller-gen's `allowDangerousTypes`, the kcp permission-claim dance,
icon-data splicing, the `_Input` GraphQL naming assumption) encountered building `materials`.

### SIMOD CRDs: native Go client, one CRD per backend resource

`Component`, `DigitalTwin`, `ExecutionParameter`, `Simulation` (same group/version, `apis/ianus/v1alpha1/{component,digitaltwin,executionparameter,simulation}_types.go`) each have a real reconciler (`operator/ianus/{component,digitaltwin,executionparameter,simulation}_controller.go`) that drives the `ianus-simod` backend's token-based `/api/platform/...` "Platform API" — no Python involved. `pkg/simod/client.go` is the native Go HTTP client (`Login`, `CreateComponent`, `CreateDigitalTwin`, `CreateExecutionParameter`, `CreateSimulation`), the Go counterpart to `ecotwin/common/api_connector.py`'s `JobHandler` (that Python client remains a prototype, untouched). Base URL is `--simod-base-url` on `cmd/ianus` (default `https://backend.simod.de`).

These form a dependency chain via same-namespace cross-refs, resolved by `Get`-ing the referenced CR and requeuing (`RequeueAfter`) until its `Status.Phase == "Ready"`: `Component` (optionally referencing child `Component`s via `Spec.Components`) → `DigitalTwin` (`Spec.ComponentRef`) → `Simulation` (`Spec.DigitalTwinRef` + `Spec.ExecutionParameterRef`); `ExecutionParameter` has no local dependency (`Spec.FormID` is a raw backend uuid — `Form`/`ComponentType`/`SimulationFile` are backend-only reference data, not modeled as CRDs here). None of the backend Platform API endpoints support update/delete, so reconciliation is create-once: a reconciler no-ops once its ID/token status field is set, and there are no finalizers. Credentials come from `Spec.SecretRefs[0]` → a same-namespace `Secret` with `email`/`password` keys, logged in fresh per reconcile (no token caching yet).

`Material` (`/api/platform/materials`) *is* now modeled as a CRD, just not here — the `materials` provider
(its own APIExport, not `ianus`'s) has `MaterialCatalog`/`MaterialSubmission`, and `pkg/simod/client.go`'s
`CreateMaterial`/`ListMaterials` are the client calls, same as `CreateComponent`/`ListSimulations` above.
See "Four providers, one repo" above.

**kcp gotcha, verified against a live local Platform Mesh (2026-09-07):** a bound consumer workspace's `Secret`s are invisible to the provider's operator by default — kcp's APIExport virtual workspace only exposes core resources the export explicitly requests via `spec.permissionClaims`, and the consumer's `APIBinding` must separately accept the same claim (`spec.permissionClaims[].state: Accepted`, with a `selector` — `{matchAll: true}` works) before the operator's `client.Get` on a `Secret` stops 404ing as "no REST mapping". `config/kcp/apiexport-ianus.platform-mesh.io.yaml` claims `secrets` (`get`/`list`/`watch`) alongside the pre-existing `events` claim for exactly this reason — any new CRD whose reconciler reads a core-group resource from the consumer workspace needs the same treatment, on both the APIExport (provider side, this repo) and every consumer's APIBinding (their side, out of this repo's control).

### Codegen pipeline — edit apis/, never hand-edit generated YAML

`apis/` is the source of truth. `make generate` runs, in order: `controller-gen object` (deepcopy) → `make manifests` (CRD → `config/crds/*.yaml` via controller-gen) → `make apiresourceschemas` (runs the `apigen` tool to convert those CRDs into kcp-native `APIResourceSchema`s under `config/kcp/`). Never hand-edit `config/crds/` or `config/kcp/` or `zz_generated.deepcopy.go` — change `apis/` and regenerate.

### `config/` is Go embed.FS, not kustomize

`config/{controller,kcp,provider}/` are each a small Go package wrapping an `embed.FS` of static YAML. `pkg/bootstrap/bootstrap.go` (invoked by the `ianus-init` binary / `make init`) applies these three sets in order via a discovery/dynamic client with create-or-update + retry semantics:
1. `config/kcp` — the kcp `APIExport`/`APIResourceSchema`
2. `config/provider` — Platform Mesh provider registration (`ContentConfiguration`, `ProviderMetadata`, RBAC)
3. `config/controller` — controller identity/RBAC (namespace, service accounts, clusterrole/binding)

It then waits for the `ianus-controller-token` ServiceAccount token Secret and writes a standalone `ianus-controller-kubeconfig` Secret (optionally rewriting the server host via `--host-override`/`HOST_OVERRIDE`) — this is the kubeconfig the running operator (`cmd/ianus`) later uses.

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
`GraphQLClientService` (`portal/src/app/shared/graphql-client.service.ts`). `ianus` has no page.

There is no router (removed deliberately - see the git history around the `InjectionMoldingService` CRD
rework for why Angular's Router doesn't fit this deployment's per-prefix relative `<base href>`):
`AppComponent` instead picks a component directly from `window.location.pathname` (`includes("/skz")`,
`includes("/materials")`, else injection-molding) - each page is deployed under its own Gateway path
prefix (`/injection-molding`, `/skz`, `/materials`), so Angular's own Router would see zero remaining path
segments regardless of which prefix loaded the page. Adding a fourth page means adding a fourth branch
here, not a route. The materials page additionally queries the `skz` provider's `Material` CRD directly
(cross-provider GraphQL query, not a cross-APIExport kcp permission claim - see
`apis/materials/v1alpha1/materialsubmission_types.go`'s package doc) to let the user pick a source
material to submit.

An earlier `models/` page (mocked `Model` CRD data, matching the now-removed generic `Model`/`Intent` type)
was deleted once it was confirmed to have no real backing CRD. See `portal/README.md` for the general
Luigi microfrontend integration model (context, `ContentConfiguration` registration, local dev proxy) -
some of its code samples still reference the old `models` page as an illustrative example, not as
something present in this repo.

### Deploy: two chart trees, only one is live

- `deploy/helm/ianus-controller/`, `deploy/helm/ianus-portal/`, `deploy/helm/injectionmolding-controller/`,
  and `deploy/helm/materials-controller/` are the real charts used to deploy this repo's images (built
  from `deploy/Dockerfile`, `deploy/portal.Dockerfile`, `deploy/api.Dockerfile`,
  `deploy/injectionmolding.Dockerfile`, `deploy/materials.Dockerfile`). `skz` has no chart/image - it's a
  dummy provider with no operator (bootstrap-only, `cmd/skz-init`). Every `*-controller` chart shares the
  same `common` chart dependency (`charts/common-X.Y.Z.tgz`, vendored per chart - copy the `.tgz` byte-for-byte
  into a new provider's chart rather than re-fetching it, see `docs/creating-a-provider.md`).
- `charts/ianus-operator/` is stale/unrelated scaffolding — its CRDs are generic `httpbin` examples from a chart template generator, not this repo's real `ianus.platform-mesh.io` CRDs. Don't treat it as the deployment path.

### Docs

Docs are published as an mdBook (`book.toml`, built output in `book/`, source in `docs/`, TOC in `docs/SUMMARY.md`). Some README links to docs are stale (e.g. links to `./docs/twinctl.md`, actual path is `docs/twinctl/twinctl.md`) — prefer `docs/SUMMARY.md` to find the current path. `docs/00-readme-simod-platform-mesh-yamls.md` + `manifests/phase1`/`phase2` are an illustrative walkthrough (contain hardcoded personal paths), separate from the actual codegen'd `config/` used by `pkg/bootstrap`.
