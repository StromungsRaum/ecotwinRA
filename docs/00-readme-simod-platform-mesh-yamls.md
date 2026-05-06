# SIMOD Platform Mesh YAML starter set

These files define a minimal declarative Platform Mesh integration for the SIMOD portfolio element `injectmold`.

## Flow

1. KCP/Platform Mesh exposes `simod.de` via `APIExport`.
2. Consumer creates a `Model` custom resource for `injectmold`.
3. `api-syncagent` syncs the spec from KCP to the MSP cluster.
4. `ecotwin-operator` reconciles the resource and calls SIMOD at `https://www.simod.de`.
5. Operator updates `status` and creates a result `Secret`.
6. `api-syncagent` syncs status and result secret back to the user's workspace.

## Files

| File | Purpose |
|---|---|
| `01-api-model-crd-declare-simod-injectmold-api.yaml` | Defines the public Kubernetes API for SIMOD InjectMold runs. |
| `02-kcp-apiexport-expose-simod-api.yaml` | Exposes the SIMOD API group in kcp / Platform Mesh. |
| `03-syncagent-values-connect-simod-api-to-msp.yaml` | Helm values template for the api-syncagent. |
| `04-syncagent-publishedresource-sync-simod-model-and-result-secret.yaml` | Publishes `Model` and syncs related result `Secret`. |
| `05-operator-simod-provider-config-runtime-endpoint.yaml` | Configures the operator with SIMOD base URL and InjectMold route. |
| `06-operator-simod-api-credentials-template.yaml` | Secret template for SIMOD API credentials. |
| `07-example-model-injectmold-run-user-order.yaml` | User-facing example order for an InjectMold run. |
| `08-example-result-secret-https-results-template.yaml` | Result secret template using normal HTTPS links. |
| `09-example-status-injectmold-completed.yaml` | Example completed status payload for docs/tests. |

## Minimal apply order

```bash
kubectl apply -f 01-api-model-crd-declare-simod-injectmold-api.yaml
kubectl apply -f 05-operator-simod-provider-config-runtime-endpoint.yaml
kubectl apply -f 06-operator-simod-api-credentials-template.yaml
kubectl apply -f 07-example-model-injectmold-run-user-order.yaml
```

KCP/APIExport and syncagent setup depend on your actual workspace and kubeconfig wiring.
