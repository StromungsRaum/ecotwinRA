# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

ECOTWIN Reference Architecture: a **Platform Mesh / kcp provider** called "IANUS" plus a Python client/API layer ("ecotwin") for the StrömungsRaum (SR) simulation platform. Three independent codebases share one repo:

- **Go**: kcp-based Kubernetes operator (`cmd/`, `apis/`, `operator/`, `pkg/`, `config/`)
- **Python**: `twinctl` CLI + FastAPI "Digital Twin API" that talks to the external SR/SIMOD backend (`ecotwin/`)
- **Angular**: Platform Mesh Luigi microfrontend (`portal/`)

There is no CI (no `.github/workflows/`), no linter configs beyond `go fmt`/`go vet`, and no automated Python test suite — keep that in mind when asked to "run the tests" or "check CI".

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

`cmd/ianus/main.go` builds a **kcp-aware** manager (`sigs.k8s.io/multicluster-runtime` + `github.com/kcp-dev/multicluster-provider`), not a plain controller-runtime manager. It watches an `APIExportEndpointSlice` (name = `--endpointslice` flag, default `ianus.platform-mesh.io`) and reconciles per logical cluster via `mgr.GetCluster(ctx, req.ClusterName)`. Reconciler logic lives in **`operator/ianus/controller.go`** (`ModelReconciler`, using `mcbuilder`/`mcreconcile` types) — separate from `cmd/ianus` but same Go module, not a submodule.

The single CRD is `Model` (group `ianus.platform-mesh.io/v1alpha1`, defined in `apis/ianus/v1alpha1/model_types.go`): `Spec.Intent` + `Spec.SecretRefs`, `Status.Result`. The current reconciler is intentionally a placeholder — it just sets `Status.Result` to a canned string when `Intent` is set and `Result` is empty. There is no real simulation/twin orchestration wired in yet.

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

- `ecotwin/scripts/twinctl.py` is the real `twinctl` Typer CLI (PEP 723 inline-script, run via `uv run`). Subcommands lazily import `ecotwin/info/group.py` (`info system`, `info models`) and `ecotwin/login/store.py` (`login store`, persists SR credentials to `~/.config/ecotwin/.env`).
- **Gotcha**: `pyproject.toml` declares the console-script entry point as `twinctl = "ecotwin.twinctl:twinctl"`, but no `ecotwin/twinctl.py` module exists — this entry point is dead. Always invoke via `uv run ecotwin/scripts/twinctl.py`, not the installed `twinctl` shim.
- `ecotwin/api_server.py` is the FastAPI "Digital Twin API" from the README. `GET /models` is currently hardcoded to `System.INT` via `ecotwin.common.api_connector.create_api_connector`.
- `ecotwin/common/api_connector.py` is the real integration with the external SIMOD/StrömungsRaum backend (`backend.{prod,int,test,dev}.simod.de`): form-login + CSRF scraping (`BackendConnector`), token auth (`JobHandler`), and model-list scraping (`AjaxApiHandler`/`ApiHandler`). Credentials come from `EMAIL`/`PASSWD` (or `EMAIL_<SYS>`/`PASSWD_<SYS>`) env vars, loaded from root `.env` or `~/.config/ecotwin/.env`.
- `service/` is dead scaffolding (untracked, only a stray `.pyc`) — ignore it; the real API is `ecotwin/api_server.py`.

### Portal: mocked data, GraphQL wiring exists but is disabled

`portal/src/app/models/models.service.ts` has a real GraphQL implementation against the Platform Mesh CRD Gateway (query pattern `{apiGroup}_{version}_{Kind}`, e.g. `ianus_platform_mesh_io.v1alpha1.Models`) written but **commented out**; `listModels()` currently returns a hardcoded `MOCK_MODELS` array. If asked to "wire up the portal to real data," this is the file to re-enable, not rewrite. See `portal/README.md` for the full Luigi microfrontend integration model (context, `ContentConfiguration` registration, local dev proxy).

### Deploy: two chart trees, only one is live

- `deploy/helm/ianus-controller/` and `deploy/helm/ianus-portal/` are the real charts used to deploy this repo's images (built from `deploy/Dockerfile`, `deploy/portal.Dockerfile`, `deploy/api.Dockerfile`).
- `charts/ianus-operator/` is stale/unrelated scaffolding — its CRDs are generic `httpbin` examples from a chart template generator, not the `ianus.platform-mesh.io` `Model` CRD. Don't treat it as the deployment path.

### Docs

Docs are published as an mdBook (`book.toml`, built output in `book/`, source in `docs/`, TOC in `docs/SUMMARY.md`). Some README links to docs are stale (e.g. links to `./docs/twinctl.md`, actual path is `docs/twinctl/twinctl.md`) — prefer `docs/SUMMARY.md` to find the current path. `docs/00-readme-simod-platform-mesh-yamls.md` + `manifests/phase1`/`phase2` are an illustrative walkthrough (contain hardcoded personal paths), separate from the actual codegen'd `config/` used by `pkg/bootstrap`.
