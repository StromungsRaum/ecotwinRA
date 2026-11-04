# Creating a Provider by Hand

Step-by-step, file-by-file walkthrough for adding a new Platform Mesh provider to this repo, generalized
from how the `materials` provider was actually built (alongside the pre-existing `ianus`,
`injectionmolding`, and `skz` providers). Each step says what the file does and why it exists, not just
what to paste — read [CLAUDE.md](../CLAUDE.md)'s "Four providers, one repo" section first for the
one-paragraph version of this same pattern.

Throughout, `<name>` is your provider's lowercase identifier used in package/binary/file names (e.g.
`materials`), `<Name>` is its PascalCase form for Go function names (e.g. `Materials`), and
`<name>.platform-mesh.io` is its APIExport/API group (pick something not already used — see `apis/*` for
what's taken). A provider with no operator (like `skz`, a dummy/demo provider) skips every step marked
**(operator only)**.

## 0. Decide your names up front

Changing these later means renaming across a dozen files, so settle them first:

- **API group**: `<name>.platform-mesh.io` (e.g. `materials.platform-mesh.io`)
- **Kind(s)**: PascalCase, plural-safe — **never end a Kind in `s`** (see step 1's gotcha)
- **Provider workspace name**: usually `ianus-<name>` for a sibling of `ianus`, or bare `<name>` for an
  unrelated provider like `skz` — this is a kcp workspace name, not a Go identifier, so it can have dashes
- **Binary names**: `<name>` (operator) and `<name>-init` (bootstrap CLI)
- **Namespace** the controller deploys into: `provider-<name>` (convention, not enforced)

## 1. CRD Go types — `apis/<name>/v1alpha1/`

The source of truth for your provider's API. Four files, copied from an existing provider and renamed:

| File | Role |
|---|---|
| `doc.go` | Package doc + `+groupName=<name>.platform-mesh.io` marker controller-gen reads to know the API group |
| `groupversion_info.go` | `GroupVersion`, `SchemeBuilder`, `AddToScheme` — boilerplate, copy verbatim and change the group string |
| `common_types.go` | Your own copy of `SecretReference` (one field, `Name string`) if the operator needs SIMOD credentials. **Don't import another provider's copy** — every provider duplicates this one tiny type instead of cross-importing API groups (confirmed convention: `ianus`, `injectionmolding`, and `materials` each have their own) |
| `<kind>_types.go` (one per Kind) | `<Kind>Spec`, `<Kind>Status` (if you have a status subresource), the `<Kind>`/`<Kind>List` structs, `+kubebuilder` markers, and the `init()` that registers them with `SchemeBuilder` |

**Gotcha, confirmed live against the real CRD Gateway:** a Kind name ending in `s` (e.g. `Materials`
instead of `Material`) breaks the gateway's pluralization — it generates no list/collection field, only a
singular getter. Pick a Kind that doesn't end in `s`.

**Gotcha:** if any field is a Go `float64`/`float32` (physical/measured values, not a Kubernetes resource
quantity), `controller-gen` refuses by default ("found float, the usage of which is highly discouraged").
Fix at the Makefile step (step 2), not here — see `apis/skz/v1alpha1/material_types.go` and
`apis/materials/v1alpha1/materialsubmission_types.go` for CRDs that need this.

Decide your CRD's data-flow shape now, since it drives everything downstream:

- **Create-once, no update/delete** (the pattern every SIMOD-backed CRD in this repo uses, because the
  backend's Platform API has no update/delete routes either): the reconciler no-ops once a status ID
  field is set. See `apis/ianus/v1alpha1/component_types.go` or
  `apis/materials/v1alpha1/materialsubmission_types.go`.
- **Browse/refresh singleton**: `spec.refreshRequestedAt` bumped by the UI, `status.availableX` +
  `status.lastRefreshedAt` + `status.lastProcessedRefreshRequestedAt` (the last field exists because a
  status-only update doesn't bump `metadata.generation`, so without it the reconciler can't tell "new
  refresh requested" from "reconciling because I just wrote status myself"). See
  `apis/injectionmolding/v1alpha1/injectionmoldingservice_types.go`'s `AvailableSimulations` or
  `apis/materials/v1alpha1/materialcatalog_types.go`.
- **Cross-provider composition happens in the portal, not the operator.** If your new CRD needs data from
  another provider's CRD (like `materials` needing a chosen `skz` `Material`'s spec), have the portal copy
  that data into your CRD's spec at creation time via a normal GraphQL query against the other provider's
  group — don't have your operator `Get` another provider's CRD. That would need a cross-APIExport kcp
  permission claim, which isn't proven anywhere in this repo (only core-group `Secrets` claims are — see
  step 4). See `apis/materials/v1alpha1/materialsubmission_types.go`'s package doc for the full reasoning.

## 2. Makefile targets

Add, mirroring an existing provider's block (search `Makefile` for `-injectionmolding` or `-materials` and
copy that whole set, renaming):

- `build-operator-<name>`, `build-init-<name>`, `build-<name>` **(skip build-operator for skz-style
  no-operator providers)**
- `run-<name>` **(operator only)**
- `init-<name>`
- `generate-<name>`, `manifests-<name>`, `apiresourceschemas-<name>`
- `image-build-<name>`, `image-push-<name>` **(operator only)**, and add `<name>` to the `images`/
  `images-push`/`kind-load-all` aggregate targets
- `kind-load-<name>` **(operator only)**

If step 1 has `float64`/`float32` fields, add `crd:allowDangerousTypes=true` to the `manifests-<name>`
target's `controller-gen crd` invocation (see `manifests-skz`/`manifests-materials` in `Makefile` for the
exact comment explaining why — this repo has exactly two CRDs needing it so far).

## 3. Generate code

```shell
make generate-<name>
```

This runs, in order: `controller-gen object` (writes `apis/<name>/v1alpha1/zz_generated.deepcopy.go`) →
`controller-gen crd` (writes `config/crds-<name>/*.yaml`) → `apigen` (converts those CRDs into
`config/kcp-<name>/apiexport-<name>.platform-mesh.io.yaml` +
`config/kcp-<name>/apiresourceschema-<kind-plural>.<name>.platform-mesh.io.yaml`).

**Never hand-edit `config/crds-<name>/` or `zz_generated.deepcopy.go`** — change `apis/<name>/v1alpha1/`
and re-run this. The generated `APIExport` YAML is the one exception needing a hand edit, in step 4.

Verify: `go build ./...` and `go vet ./...` should both pass with zero other changes.

## 4. kcp resources — `config/kcp-<name>/`

| File | Role |
|---|---|
| `bootstrap.go` | `//go:embed *.yaml` wrapper exposing these files as an `embed.FS` named `FS`, in package `kcp<name>` |
| `apiexport-<name>.platform-mesh.io.yaml` | Generated by `apigen` in step 3 — **hand-edit it after every regen** if your operator needs core-group resources (see below) |
| `apiresourceschema-*.yaml` | Fully generated, never hand-edited |

**Manual edits apigen doesn't make, that you must redo after every `make generate-<name>`:**

1. `metadata.labels['ui.platform-mesh.io/content-for']: <name>.platform-mesh.io` on the `APIExport` — this
   is what links it to your `ProviderMetadata`/`ContentConfiguration` in step 5. Without it the provider
   won't show a marketplace entry.
2. If your operator reads a core-group resource (almost always `Secrets`, for SIMOD credentials) from the
   *consumer's* workspace, add to `spec`:
   ```yaml
   permissionClaims:
   - resource: secrets
     verbs: [get, list, watch]
   ```
   **This alone is not enough** — the *consumer's* `APIBinding` must separately accept the same claim
   (`spec.permissionClaims[].state: Accepted`, `selector: {matchAll: true}` works) before a `Secret` stops
   404ing as "no REST mapping" for the operator. That accept step happens on the consumer's side, outside
   this repo — see CLAUDE.md's "kcp gotcha" for the fuller account of this, verified live.

## 5. Provider resources — `config/provider-<name>/`

| File | Role |
|---|---|
| `bootstrap.go` | `//go:embed contentconfiguration.yaml providermetadata.yaml rbac.yaml`, package `provider<name>` |
| `kustomization.yaml` | Lists the three YAML files — not actually applied via kustomize anywhere in this repo (`config/` is embed.FS, not kustomize - see CLAUDE.md), kept mostly for IDE/tooling discovery |
| `providermetadata.yaml` | `ProviderMetadata`: `displayName`, `description`, `contacts`, `preferredSupportChannels`, `icon.{light,dark}.data` (base64 data URIs — **don't hand-type these**, splice an existing provider's icon bytes in with a shell command so you never load the giant base64 blob into an editor/chat context, e.g. `sed -n '/^  icon:/,$p' config/provider-injectionmolding/providermetadata.yaml \| tail -n +2 >> your-new-file.yaml`) |
| `contentconfiguration.yaml` | `ContentConfiguration` with `ui.platform-mesh.io/content-for: <name>.platform-mesh.io` label — the inline `luigiConfigFragment` JSON picks the nav `pathSegment`, `label`, `order`, and the portal page URL (`https://{context.organization}.portal.localhost:8443/<name>/`) |
| `rbac.yaml` | A `ClusterRole` granting `bind` on your `APIExport` (`resourceNames: ["<name>.platform-mesh.io"]`) + a `ClusterRoleBinding` to `system:anonymous` — this is what lets *any* consumer workspace bind your APIExport, matching every existing provider here |

## 6. Controller identity — `config/controller-<name>/` **(operator only)**

| File | Role |
|---|---|
| `bootstrap.go` | `//go:embed *.yaml`, package `controller<name>` |
| `namespace.yaml` | `Namespace: default` — yes, `default`, inside the *provider's own kcp workspace* (not the consumer's, and not the compute-cluster namespace the controller actually runs in later) |
| `serviceaccount-main.yaml` | `ServiceAccount` named `<name>-controller` |
| `serviceaccount-secret.yaml` | A `Secret` of type `kubernetes.io/service-account-token` annotated `kubernetes.io/service-account.name: <name>-controller` — this is what `pkg/bootstrap` polls until Kubernetes populates its `token`/`ca.crt` data, to mint the operator's kubeconfig |
| `rbac-clusterrole.yaml` | Full `get/list/watch` on your CRDs + their `/status` subresources, plus the same boilerplate `events`/`apiexports`/`apiexportendpointslices` rules every provider's controller needs |
| `rbac-clusterrolebinding.yaml` | Binds that ClusterRole to the ServiceAccount above |

## 7. Bootstrap wiring — `pkg/bootstrap/bootstrap_<Name>.go` **(skip only for truly operator-less, config-only providers)**

A `Bootstrap<Name>(ctx, config, hostOverride) error` function, structurally identical to
`Bootstrap`/`BootstrapInjectionMolding`/`BootstrapMaterials` in the same package: build discovery/dynamic/
kubernetes clients, then call the shared (unexported) `bootstrapFS` helper against your three `config/
*-<name>` embed.FS sets in order (kcp → provider → controller), then call the shared
`createControllerKubeconfigSecret` with your Secret names (`<name>-controller-token` →
`<name>-controller-kubeconfig`). Copy `bootstrap_materials.go` and rename.

## 8. Bootstrap CLI — `cmd/<name>-init/main.go`

Thin `pflag` wrapper around step 7's function: parses `--kubeconfig`/`--host-override`, builds a
`*rest.Config`, calls `bootstrap.Bootstrap<Name>(ctx, config, hostOverride)`. Copy
`cmd/materials-init/main.go` (or `cmd/skz-init/main.go` if there's no operator) and rename.

## 9. Reconciler(s) — `operator/<name>/` **(operator only)**

| File | Role |
|---|---|
| `simod_client.go` (if talking to SIMOD) | `simodBaseURL` package var + `SetSIMODBaseURL` (set once from the `--simod-base-url` flag before `SetupWithManager`), `phasePending`/`phaseReady`/`phaseFailed` constants, and a thin `loginSIMOD` wrapper around the shared `pkg/simod.LoginFromSecret` |
| `<kind>_controller.go` (one per Kind) | A `<Kind>Reconciler struct { Manager mcmanager.Manager }` with `SetupWithManager` (registers via `mcbuilder.ControllerManagedBy(mgr).Named(...).For(&<name>v1alpha1.<Kind>{}).Complete(mcreconcile.Func(r.Reconcile))`) and `Reconcile(ctx, req mcreconcile.Request) (ctrl.Result, error)` |

This is **kcp-aware multicluster-runtime**, not plain controller-runtime — `req.ClusterName` names the
logical cluster, and `r.Manager.GetCluster(ctx, req.ClusterName)` gets you a per-cluster client. Every
reconciler in this repo follows the same shape: `Get` the CR (ignore `NotFound`), check whether there's
actually work to do (refresh requested? not yet submitted? not yet complete?), call out to
`pkg/simod/client.go` if needed, write `status.phase`/`status.message` on any error, and return
`ctrl.Result{RequeueAfter: ...}` only when polling (not on every reconcile). See
`operator/materials/materialsubmission_controller.go` for a create-once example and
`operator/materials/materialcatalog_controller.go` for a refresh-loop example.

If your CRD's spec needs translating into a different shape before calling the backend (e.g. the
materials provider's CRD groups data by physical model, but the backend wants it grouped by generic
`method`/`parameters`), that translation function lives here, in the reconciler's package — not in
`pkg/simod` (which stays a dumb transport client) and not in `apis/<name>/v1alpha1` (which stays plain
data). See `operator/materials/materialsubmission_controller.go`'s `buildMaterialRequest`.

## 10. Operator entrypoint — `cmd/<name>/main.go` **(operator only)**

Copy `cmd/materials/main.go` (two reconcilers) or `cmd/injectionmolding/main.go` (one) and rename:
`init()` registers your `apis/<name>/v1alpha1` scheme alongside the kcp SDK schemes; `main()` parses
`--endpointslice` (default `<name>.platform-mesh.io`) and `--simod-base-url` if applicable, builds an
`apiexport.Provider` + `mcmanager.Manager`, then calls `SetupWithManager` on each reconciler before
`mgr.Start(ctx)`.

## 11. SIMOD client additions — `pkg/simod/client.go` **(only if calling a new backend endpoint)**

If your provider calls a `/api/platform/...` endpoint nothing else uses yet, add a `<Kind>Request`/
`<Kind>Response` pair plus `Create<Kind>`/`List<Kind>`/`Get<Kind>` methods on `*Client`, following the
existing methods' shape exactly (`c.post`/`c.get` helpers, response unwrapping — list endpoints wrap their
array in `{"data": [...]}`, confirmed no `withoutWrapping()` on the `ianus-simod` side). Use `any` instead
of `map[string]string` for any field whose backend-side shape isn't a flat map (see `MaterialSection`'s
doc comment for why — a flat map broke down once a section needed to hold a list of rows, not key/value
pairs).

## 12. Image + Helm chart — `deploy/<name>.Dockerfile`, `deploy/helm/<name>-controller/` **(operator only)**

- `deploy/<name>.Dockerfile`: copy any existing provider's Dockerfile, change the two `./cmd/<name>/...`
  build paths and the binary name. Multi-stage, distroless, non-root (`65532:65532`) — don't deviate.
- `deploy/helm/<name>-controller/`: copy an existing provider's chart wholesale and rename throughout
  (`Chart.yaml`'s `name`, every `{{ include "..." }}` helper name in `templates/_helpers.tpl`, `values.yaml`'s
  `image.repository`/`controller.endpointSlice`/`kubeconfig.secretName`). **Copy the `charts/common-X.Y.Z.tgz`
  dependency file with `cp`, not by re-fetching from the OCI registry** — it's a vendored binary blob, and
  every chart in this repo pins the identical version; a byte-for-byte copy (`sha256sum` both to confirm)
  avoids a network dependency and guarantees the same `Chart.lock` digest.
- Verify with `helm lint deploy/helm/<name>-controller` and `helm template test deploy/helm/<name>-controller`
  — both should be clean before moving on.

## 13. Portal page — `portal/src/app/<name>/` **(optional, only if the provider needs a custom UI)**

| File | Role |
|---|---|
| `<name>.service.ts` | GraphQL client: TypeScript interfaces mirroring your CRD's spec/status shape, `LIST_QUERY`/`CREATE_MUTATION`/etc as template literal GraphQL strings, methods wrapping `GraphQLClientService.post(query, variables)` |
| `<name>.component.ts` | Angular standalone component: `signal()`s for loaded state, `ngOnInit` hooked to `LuigiClient.addInitListener`, handlers for load/refresh/create/delete calling into the service |
| `<name>.component.html` | UI5 Web Components template (`ui5-dynamic-page`, card grid, `ui5-dialog` for the add form) |
| `<name>.component.scss` | Styling — copy an existing page's and rename the class prefix |

Then wire it into `portal/src/app/app.component.ts`'s `window.location.pathname`-based branch (see
CLAUDE.md's "Portal" section for why this isn't a Router route) and its `ContentConfiguration`'s `url`
field in step 5 to match.

**GraphQL mutation gotcha, not fully verified:** nested object fields in your spec need their own
generated `<GoTypeName>_Input` GraphQL input type to pass as a typed `$variable` — confirmed live for
`StringMap_Input` (the old flat-map `skz` schema), *assumed* (not independently re-verified) for the
newer per-field input types this repo's `skz`/`materials` pages rely on. Verify via introspection against
a live gateway before trusting it blindly. A field with no further nesting can be passed as scalar
`$variables` and assembled as an inline object literal in the mutation text instead, which sidesteps this
risk entirely — see `injection-molding.service.ts`'s `secretRefs: [{ name: $secretName }]` for that
pattern.

**Verify with a real build, not just `tsc --noEmit`:** `npx tsc -p tsconfig.app.json --noEmit` does not
run Angular's template type-checker and will pass even with a broken template binding (a real instance of
this: `rows="16"` on a `ui5-textarea` needs to be `[rows]="16"`, a type error `tsc --noEmit` missed
entirely). Run `npx ng build --configuration development` from `portal/` instead — it must produce zero
errors and zero warnings.

## 14. Docs

- [platform-mesh.md](./platform-mesh.md): add a `## <Provider Display Name> Provider` section mirroring
  the `Injection Molding Service Provider`/`IANUS Simulation - Materials Provider` ones — what the CRDs
  are, the cross-provider story if any, then the numbered `kubectl ws create` → `make init-<name>` →
  extract kubeconfig → build/load image → `helm upgrade --install` walkthrough.
- [quick_start.md](./quick_start.md): add a new numbered section for registering the provider, in the
  same slot as the existing provider-registration sections, and **renumber every section after it** —
  including each section's own lettered sub-steps (`Na`, `Nb`, ...), which is easy to miss and was missed
  once already while writing this repo's own quick start.
- This file: if your provider's pattern deviates from the steps above in some reusable way, update the
  step here, don't just note the deviation in your provider's own section elsewhere.

## 15. Deploy to a live cluster and verify

```shell
# Provider workspace (sibling of the others, under root:providers)
KUBECONFIG=$PM_KUBECONFIG kubectl ws use :
KUBECONFIG=$PM_KUBECONFIG kubectl ws create providers --type=root:providers --enter --ignore-existing
KUBECONFIG=$PM_KUBECONFIG kubectl ws create <workspace-name> --type=root:provider --enter --ignore-existing

KUBECONFIG=$PM_KUBECONFIG make init-<name> HOST_OVERRIDE=https://frontproxy-front-proxy.platform-mesh-system:8443

KUBECONFIG=$PM_KUBECONFIG kubectl get secret <name>-controller-kubeconfig -n default -o jsonpath='{.data.kubeconfig}' | base64 -d > <name>.kubeconfig

# operator only, from here down:
export IMAGE_TAG=platform-mesh
make image-build-<name> kind-load-<name> IMAGE_TAG=$IMAGE_TAG

KUBECONFIG=$COMPUTE_KUBECONFIG kubectl create namespace provider-<name>
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl create secret generic <name>-controller-kubeconfig \
  --from-file=kubeconfig=./<name>.kubeconfig -n provider-<name>

KUBECONFIG=$COMPUTE_KUBECONFIG helm upgrade --install <name>-controller ./deploy/helm/<name>-controller \
  --namespace provider-<name> --set image.tag=$IMAGE_TAG --set image.pullPolicy=IfNotPresent
```

Verify the marketplace entry actually exists before trusting anything in the portal:

```shell
kubectl --server="https://localhost:8443/services/marketplace/clusters/root:providers:<workspace-name>" get marketplaceentries -A
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl get pods -n provider-<name>
```

**Nothing above happens automatically** — writing the files in steps 1–13 and even running
`go build`/`go vet`/`helm lint`/`ng build` locally verifies the code compiles, not that it's deployed. A
provider only shows up in a running Platform Mesh's marketplace after `make init-<name>` actually runs
against that cluster's kcp, and only runs the new code after the image is rebuilt, reloaded, and the
Helm release is upgraded.
