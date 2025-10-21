# Introduction: What is StrömungsRaum?

## StrömungsRaum
**StrömungsRaum (SR)** is an **industry cloud platform** for **high‑fidelity simulation**, **optimization**, and **AI models**—built for **HPC at scale** (cloud / on‑prem / hybrid). For the Platform‑Mesh team, SR acts as a **service provider (black box)** in the **application layer**: capabilities are exposed via **clear contracts** (order, lifecycle, status/events, telemetry, egress).

## ECOTWIN
**ECOTWIN** is a cloud-edge extension of **StrömungsRaum (SR)** that turns SR’s simulation/optimization/AI capabilities into a **distributed “cloud ↔ edge” digital-twin fabric**. It lets you run **inference and monitoring close to machines at the edge**, while **training, DoE, and heavy simulation** continue in SR’s cloud/HPC core—connected by a governed data/metadata plane.

**Key ideas**
- **Twin continuum:** Edge twins (fast inference, anomaly/drift detection) pair with cloud twins (hi-fi sims, retraining, optimization).  
- **Policy-first data flow:** Classified, encrypted **ingress/egress** between edge sites and SR’s enterprise data fabric.  
- **Lifecycle sync:** Model registry, versioning, and **promotion** propagate from cloud to edge with staged rollouts and rollback.  
- **Resilience & locality:** Works offline/intermittent; buffers events and reconciles when connectivity returns.  
- **Agentic operations (optional):** Edge agents can trigger **retraining** or **parameter updates** upstream based on drift/quality KPIs.

**What it adds to SR**
- Edge runtimes & connectors (OPC UA/MQTT)  
- Deployment policies for **geo/residency** and **SLOs** per site  
- Managed **over-the-air** updates for models and monitoring rules  
- Unified observability & FinOps across cloud and edge scopes

---
## Value Proposition (at a glance)
- **Time‑to‑Value:** Turn input data into trustworthy results in hours instead of weeks.
- **Scale:** Thousands of parallel runs (DoE/optimization/training) across orchestrated compute on supercomputers.
- **Quality & Reproducibility:** Versioned process chains, data/model governance, SLO‑backed.
- **Secure Data Paths:** Policy‑driven **egress** into the enterprise data fabric, OIDC federation, auditability.

---
## Relevant Core Capabilities
**Provisioning (Control Plane):** Tenant/project creation, quotas/policies, application (process‑chain) CRUD, secret/connection bindings, observability & FinOps plans.

Provisioning defines **who can do what, where, and under which guardrails**. It creates and configures the *context* in which workloads will later run: tenants, projects, quotas, policies, process-chains (applications), secrets/bindings, and telemetry/FinOps plans.  
- **Audience:** Platform/Mesh admins, security, FinOps, app publishers  
- **Artifacts:** Tenants, projects, app definitions, policy attachments, bindings  
- **Outcomes:** Governed, auditable environments ready for use; no payload data processed yet  
- **Failure domain:** Misconfiguration & policy drift (fixed via validation, idempotent updates)


**Operation on a Tenant (Runtime):** Simulation runs, DoE/optimization, surrogate/model training, model‑registry ops, inference/queries, sensor monitoring, data curation, retraining, artifact egress, SLO/FinOps reports.

Operations execute **work inside an existing provisioned context**. This is where value is produced: start/manage simulations and DoE batches, train/promote models, run inference, curate datasets, monitor sensors, export artifacts, and report SLO/FinOps.  
- **Audience:** Engineers, data scientists, operators, downstream apps  
- **Artifacts:** Runs/batches, datasets, models, results, reports, usage records  
- **Outcomes:** Computation performed, artifacts produced and egressed, telemetry emitted  
- **Failure domain:** Workload/runtime errors (handled via retries, checkpoints, graceful control)

The open reference architecture for SR’s cloud–edge twin continuum is available here:
- **GitHub:** https://github.com/StromungsRaum/ecotwinRA
---
## Architecture (simplified)
- **Application Layer (Black‑box Interface):** SR **control‑plane adapter** speaks mesh contracts (discovery / order / lifecycle / events). Optional micro‑frontend for UI integration.
> One task in this initiative on the IANUS side is to finalize this adapter.
- **Execution Layer:** Execution of Hi‑Fi sims, optimizers, pipelines, training jobs; schedulers/queues; artifact store.
- **Data/Model Layer:** Datasets, model registry, metadata/catalog; egress into the enterprise data fabric (copy or link; push or pull).



---
## Interfaces & Contracts (for the mesh)
- **Order/Lifecycle:** Declarative orders for capabilities (inputs, SLOs, resources), idempotent state machine.
- **Status & Events:** `Queued|Running|Succeeded|Failed`, `ArtifactReady`, `ModelPromoted`, `AnomalyDetected`, etc.
- **Cost & SLA:** Metering keys, pricing/allowance models, SLA tiers (TTFR, success rate, support window).

---
## Typical End‑to‑End Workflows (examples)
1. **DoE → Surrogate → Inference:** Order design space → scale batch runs → dataset & Pareto → train surrogate → promote model in registry → serve inference in target systems.
2. **Hi‑Fi Simulation → Report/Egress:** Parameterize geometry/material/BCs → start run → live status/TTFR → artifacts & KPIs egressed to the enterprise lake → auto‑report & SLO report.

We want to start with 2 and here is a little more detail:


### Hi-Fi Simulation → Report/Egress (simulate–inspect–publish)

**Goal:** Run high-fidelity simulations with strong reproducibility and publish results safely.

1. **Order Hi-Fi Run**  
   Choose *“Start Simulation Run”*; provide geometry/materials/BCs/params, resource hints, and SLOs. Adapter creates an idempotent run.

2. **Execute with live status**  
   SR provisions compute (queue → start), streams logs/progress and early KPIs (TTFR). Mesh UI exposes pause/cancel.

3. **Produce artifacts & KPIs**  
   On completion, SR emits field data, KPIs, and a metadata manifest (versions, seeds, solver settings). Optional post-processing (reports/plots/CSV summaries).

4. **Policy-checked egress**  
   Adapter validates data class/residency/retention; pushes to the results lake (or issues signed links). Catalog entry updated for discovery/lineage.

5. **Consumption & reporting**  
   BI/notebooks consume results; stakeholders get a concise report (SLO attainment, key plots, decisions). FinOps shows cost breakdown (cpu/gpu hours, bytes-out).

6. **Failure handling:** checkpoint resume, graceful cancel, “last good” report; RCA hints (top errors).  


---
## Service Levels & Key Metrics
- **SLIs/SLOs:** Queue time, time‑to‑first‑result (TTFR), success rate, throughput, cost per run, egress time, error‑budget burn.
- **Compliance Evidence:** Audit trails, policy‑enforcement rate, data‑residency conformance.

---
## Glossary (brief)
- **Capability:** An orderable ability (e.g., *Start Simulation Run*).
- **Process Chain / Application:** Versioned, executable pipeline (sim/doe/train/etl).
- **Egress:** Controlled outflow of data/artifacts from SR to enterprise targets.
- **Adapter:** Component mapping mesh contracts to SR APIs (order ⇄ run, events, metering).

> **Takeaway:** For the Platform‑Mesh team, SR is **not a monolith** but a **clearly bounded provider** with **declared capabilities**, **policies**, and **telemetry**—integrated like any other enterprise capability in the mesh.

# SR Capabilities — Detailed Descriptions
## Provisioning (Control Plane)
### P-01 — Create Tenant (simplified)
**Purpose**: Provision a new tenant/account with quotas, policies, and billing linkage.  
**Consumers**: Platform Mesh (entitlement/marketplace), central ops.  
**Inputs**: `tenantSpec{name, billingRef, quota{cpu,gpu,mem,storage}, dataClasses, residency, retention, contacts}`  
**Outputs/State**: `tenantId`, `status{Pending,Ready,Error}`, events `TenantCreated`  
**Contract (order)**: `kind: Tenant`, `action: Create`, `spec: tenantSpec`  
**Lifecycle**: Create → Validate → Apply quotas & bindings → Ready  
**SLIs/SLOs**: Provisioning time (P50/P95), policy-compliance pass, error rate  
**AuthZ**: `tenant.admin:create`  
**Telemetry/Metering**: `provision.duration`, `policy.violations=0`  
**Policy checks**: Residency, allowed data classes, quota limits  
**Failure Modes**: Policy violation, billing linkage error; retries with backoff  

### P-02 — Update Tenant / Quotas / Policies (simplified)
**Purpose**: Adjust quotas, RBAC, KMS keys, and policies.  
**Consumers**: Platform Mesh, SecOps, FinOps.  
**Inputs**: `patch{quota, rbac, kmsKeyRefs, policyRefs}`  
**Outputs/State**: `status{Pending,Updating,Ready,Error}`, `TenantPolicyChanged`  
**Contract (order)**: `kind: Tenant`, `action: Update`, `spec: patch`  
**Lifecycle**: Validate patch → Apply → Conformity check  
**SLIs/SLOs**: Time-to-effect ≤ 5m (P95), violations=0  
**AuthZ**: `tenant.admin:update`  
**Telemetry/Metering**: `tenant.update.duration`, `policy.violations`  
**Policy checks**: Residency EU-only, data-class allowlist, quota ceilings  
**Failure Modes**: Policy violation, quota conflict, key-rotation error  

### P-03 — Delete/Suspend Tenant (simplified)
**Purpose**: Decommission or freeze a tenant.  
**Consumers**: Central ops, legal/compliance.  
**Inputs**: `tenantId`, `retentionAction{archive|purge}`, `effectiveAt`  
**Outputs/State**: Finalizers done, `TenantDeleted|TenantSuspended`  
**Contract (order)**: `kind: Tenant`, `action: Delete|Suspend`  
**Lifecycle**: Quiesce → Egress/Archive → Disable → Delete  
**SLIs/SLOs**: Deletion window, zero orphaned resources  
**AuthZ**: `tenant.admin:delete`  
**Telemetry/Metering**: `tenant.delete.duration`  
**Policy checks**: Legal hold, retention, export controls  
**Failure Modes**: Active runs, legal hold blocks, dangling bindings  

### P-04 — Create Project (Workspace) (simplified)
**Purpose**: Create an isolated workspace under a tenant.  
**Consumers**: Project admins, operators.  
**Inputs**: `projectSpec{name, labels, dataDomains, artifactBuckets}`  
**Outputs/State**: `projectId`, `status{Pending,Ready,Error}`  
**Contract (order)**: `kind: Project`, `action: Create`, `spec: projectSpec`  
**Lifecycle**: Create → Bind roles/buckets → Ready  
**SLIs/SLOs**: Provision ≤ 2m (P95)  
**AuthZ**: `project.admin:create`  
**Telemetry/Metering**: `project.provision.duration`  
**Policy checks**: Namespace isolation, bucket residency  
**Failure Modes**: Bucket binding failure, RBAC misconfig  

### P-05 — Application / Process-Chain CRUD (simplified)
**Purpose**: Publish and manage executable chains (sim/doe/train/etl).  
**Consumers**: App publishers, platform engineers.  
**Inputs**: `appSpec{type, version, graph, resources, inputsSchema, outputsSchema, SLO}`  
**Outputs/State**: `appId@version`, validation report  
**Contract (order)**: `kind: Application`, `action: Publish|Update|Delete`  
**Lifecycle**: Draft → Validate → Publish → Deprecate  
**SLIs/SLOs**: Validation pass rate, time-to-publish  
**AuthZ**: `app.publisher:*`  
**Telemetry/Metering**: `app.publish.duration`, `schema.validation.errors`  
**Policy checks**: Input/output schema, resource ceilings  
**Failure Modes**: Schema invalid, incompatible graph, SLO unsatisfied  

### P-06 — Capability Template Registry (simplified)
**Purpose**: Provide reusable order templates (e.g., Hi-Fi Run, DoE Batch).  
**Consumers**: Catalog curators, end users.  
**Inputs**: `templateSpec{params, defaults, SLOs, costModel}`  
**Outputs/State**: `templateId`, catalog entry  
**Contract (order)**: `kind: Template`, `action: Publish|Update|Delete`  
**Lifecycle**: Author → Validate → Publish  
**SLIs/SLOs**: Catalog availability, lint pass  
**AuthZ**: `catalog.curator:publish`  
**Telemetry/Metering**: `template.publish.duration`  
**Policy checks**: Param bounds, SLO sanity  
**Failure Modes**: Invalid defaults, cost model missing  

### P-07 — Secret & Connection Binding (simplified)
**Purpose**: Bind secure connections to sources/sinks (S3/DB/MQTT/OPC UA).  
**Consumers**: Project admins, data engineers.  
**Inputs**: `binding{type, endpoint, role, secretRef, test?}`  
**Outputs/State**: `bindingId`, connection test result  
**Contract (order)**: `kind: Binding`, `action: Create|Update|Delete`  
**Lifecycle**: Create → Test → Activate  
**SLIs/SLOs**: Bind success rate, rotation latency  
**AuthZ**: `binding.admin:*`  
**Telemetry/Metering**: `binding.test.result`, `secret.rotate.count`  
**Policy checks**: Least privilege, TTL, residency  
**Failure Modes**: Credential invalid, CA mismatch, scope too broad  

### P-08 — Policy-as-Code Attach (simplified)
**Purpose**: Attach DLP/geo/residency/retention/export controls.  
**Consumers**: SecOps, compliance.  
**Inputs**: `policyRefs`, `rego/k8s-policy`, `mode{enforce|warn}`  
**Outputs/State**: `policyAttachmentId`, conformance report  
**Contract (order)**: `kind: PolicyAttachment`, `action: Attach|Detach`  
**Lifecycle**: Validate → Attach → Monitor  
**SLIs/SLOs**: Enforcement rate, violations=0  
**AuthZ**: `policy.admin:attach`  
**Telemetry/Metering**: `policy.violations`, `enforcement.latency`  
**Policy checks**: Data class, residency, export, retention  
**Failure Modes**: Conflicting policies, false positives  

### P-09 — Observability & FinOps Plan Attach (simplified)
**Purpose**: Declare telemetry and usage metering for scope.  
**Consumers**: SRE, FinOps.  
**Inputs**: `telemetryPlan{metrics,traces,logs}`, `usagePlan{meteringKeys}`  
**Outputs/State**: `planId`, activation status  
**Contract (order)**: `kind: TelemetryPlan`, `action: Attach|Update|Detach`  
**Lifecycle**: Validate → Attach → Emit  
**SLIs/SLOs**: Meter coverage, attribution accuracy  
**AuthZ**: `telemetry.admin:configure`  
**Telemetry/Metering**: Emitted per plan; coverage %  
**Policy checks**: PII redaction, log retention  
**Failure Modes**: Missing meters, incorrect labels


## Operation on a Tenant (Runtime)

### O-01 — Start Simulation Run (simplified)
**Purpose**: Execute a reproducible high-fidelity simulation.  
**Consumers**: Engineers, analysts, automated pipelines.  
**Inputs**: `appId@version|templateId`, `runSpec{geometryRef, materials, BCs, params, resources}`, optional `SLO`  
**Outputs/State**: `runId`, streams `RunStatus|Logs|Artifacts`  
**Contract (order)**: `kind: Run`, `action: Start`, `spec: runSpec`  
**Lifecycle**: Queued → Running → Succeeded|Failed → Archived  
**SLIs/SLOs**: Queue time, TTFR, success rate, cost/run  
**AuthZ**: `runs:start`  
**Telemetry/Metering**: cpu/gpu/mem/io, bytes-out  
**Policy checks**: Data class & residency at egress  
**Failure Modes**: Solver fail, preemption; checkpoint/retry  

### O-02 — Start DoE / Optimization Batch (simplified)
**Purpose**: Explore/optimize a design space via many runs.  
**Consumers**: Optimization engineers, data scientists.  
**Inputs**: `designSpace`, `sampler|optimizer`, `budget`, `constraints`  
**Outputs/State**: `batchId`, `datasetRef`, `paretoRef`  
**Contract (order)**: `kind: DoEBatch`, `action: Start`  
**Lifecycle**: Plan → Evaluate → Consolidate → Complete  
**SLIs/SLOs**: eval/sec, feasible rate, hypervolume  
**AuthZ**: `runs:start`  
**Telemetry/Metering**: evaluations, cost by eval  
**Policy checks**: Max evaluations, param bounds  
**Failure Modes**: Infeasible space, convergence stall  

### O-03 — Manage Run (Pause/Resume/Cancel) (simplified)
**Purpose**: Control lifecycle of running jobs.  
**Consumers**: Operators, support.  
**Inputs**: `runId`, `action{pause|resume|cancel}`  
**Outputs/State**: `status`, events `RunPaused|RunCancelled`  
**Contract (order)**: `kind: Run`, `action: Manage`  
**Lifecycle**: Live control → Confirmed state  
**SLIs/SLOs**: Control latency ≤ 30s (P95)  
**AuthZ**: `runs:manage`  
**Telemetry/Metering**: control events  
**Policy checks**: Graceful termination windows  
**Failure Modes**: Non-cooperative job, lost heartbeat  

### O-04 — Fetch Status / Logs / Events (simplified)
**Purpose**: Observe progress and diagnostics.  
**Consumers**: Users, SRE.  
**Inputs**: `runId|batchId`, `since`, `follow?`  
**Outputs/State**: `status`, `progress`, `events`, `metrics`  
**Contract (order)**: `kind: RunStatus`, `action: Get`  
**Lifecycle**: Read-only  
**SLIs/SLOs**: API P99 ≤ 500ms, freshness  
**AuthZ**: `runs:view`  
**Telemetry/Metering**: request/latency counters  
**Policy checks**: Redaction of sensitive logs  
**Failure Modes**: Over-fetch, throttling  

### O-05 — Artifact Export / Data Egress (simplified)
**Purpose**: Publish artifacts into enterprise lake/catalog.  
**Consumers**: Data platform, BI, downstream apps.  
**Inputs**: `artifactQuery{runId,type}`, `egressTarget{lake/cat/topic}`, `mode{copy|link}`  
**Outputs/State**: `artifactUris`, `catalogEntry`, `ArtifactReady`  
**Contract (order)**: `kind: Egress`, `action: Export`  
**Lifecycle**: Validate → Transfer → Register  
**SLIs/SLOs**: Egress time, violations=0  
**AuthZ**: `artifacts:egress`  
**Telemetry/Metering**: bytes-out, transfer duration  
**Policy checks**: Residency, DLP, retention  
**Failure Modes**: Expired creds, size limits, timeout  

### O-06 — Train Surrogate / Foundation Model (simplified)
**Purpose**: Train a predictive model on curated datasets.  
**Consumers**: Data scientists, ML engineers.  
**Inputs**: `trainingSpec{datasetRef, modelType, features, objective, resources}`  
**Outputs/State**: `modelId@version`, metrics  
**Contract (order)**: `kind: Training`, `action: Start`  
**Lifecycle**: Queue → Train → Validate → Complete  
**SLIs/SLOs**: time-to-best, val score, cost/epoch  
**AuthZ**: `models:train`  
**Telemetry/Metering**: gpu/cpu hours, checkpoints  
**Policy checks**: Dataset consent/class, lineage  
**Failure Modes**: Data drift, OOM, early stop  

### O-07 — Model Registry Ops (Promote/Rollback/Retire) (simplified)
**Purpose**: Govern model lifecycle and stages.  
**Consumers**: MLOps, governance.  
**Inputs**: `modelId@version`, `stage{staging|prod}`, `policyCheck`  
**Outputs/State**: `status`, `ModelPromoted|ModelRetired`  
**Contract (order)**: `kind: Model`, `action: Promote|Rollback|Retire`  
**Lifecycle**: Check → Transition → Record  
**SLIs/SLOs**: governance pass rate, rollback time  
**AuthZ**: `models:promote|retire`  
**Telemetry/Metering**: promotions, rollbacks  
**Policy checks**: Model card present, metrics thresholds  
**Failure Modes**: Threshold fail, dependency conflicts  

### O-08 — Model Inference / Query (simplified)
**Purpose**: Serve predictions (batch or realtime).  
**Consumers**: Apps, services, engineers.  
**Inputs**: `modelRef`, `query{inputs, batch|stream}`, `explain?`  
**Outputs/State**: `predictions`, optional `explanations`  
**Contract (order)**: `kind: Inference`, `action: Invoke`  
**Lifecycle**: Request → Response  
**SLIs/SLOs**: P50/P99 latency, throughput, error rate  
**AuthZ**: `inference:invoke`  
**Telemetry/Metering**: qps, latency, tokens/rows processed  
**Policy checks**: Input validation, quota limits  
**Failure Modes**: Timeout, cold start, version mismatch  

### O-09 — Sensor Monitoring / Edge Stream Attach (simplified)
**Purpose**: Attach live streams; detect anomalies/drift.  
**Consumers**: Ops, data scientists.  
**Inputs**: `streamSpec{topic, codec, rate}`, `monitoringRules{threshold, drift, anomaly}`  
**Outputs/State**: `monitorId`, events `AnomalyDetected|DriftAlert`  
**Contract (order)**: `kind: Monitoring`, `action: Attach`  
**Lifecycle**: Attach → Observe → Alert  
**SLIs/SLOs**: e2e lag, detection precision/recall  
**AuthZ**: `monitoring:configure`  
**Telemetry/Metering**: event rate, lag  
**Policy checks**: PII redaction, retention  
**Failure Modes**: Backpressure, decode errors  

### O-10 — Data Curation / Dataset Build (simplified)
**Purpose**: Build governed datasets for modeling/analytics.  
**Consumers**: Data platform, DS/ML.  
**Inputs**: `curationSpec{sources, filters, labeling, augmentation, splits}`  
**Outputs/State**: `datasetRef`, `dataSheet`  
**Contract (order)**: `kind: Dataset`, `action: Build`  
**Lifecycle**: Ingest → Transform → Validate → Publish  
**SLIs/SLOs**: coverage, label quality, build time  
**AuthZ**: `datasets:curate`  
**Telemetry/Metering**: rows, bytes, quality metrics  
**Policy checks**: Data class, consent, lineage  
**Failure Modes**: Schema drift, low quality labels  

### O-11 — Retraining Trigger (Policy/Drift/Calendar) (simplified)
**Purpose**: Automate retraining based on policy or signals.  
**Consumers**: MLOps, governance.  
**Inputs**: `trigger{drift>τ|schedule|manual}`, `trainSpecRef`  
**Outputs/State**: `jobId`, `newModelRef`  
**Contract (order)**: `kind: Retraining`, `action: Trigger`  
**Lifecycle**: Detect → Trigger → Train → Promote  
**SLIs/SLOs**: trigger→start latency, recovery time  
**AuthZ**: `models:train`  
**Telemetry/Metering**: triggers, MTTR  
**Policy checks**: Safe rollout, canary/quarantine  
**Failure Modes**: False positives, compute shortage  

### O-12 — Cost & Usage Report (FinOps) (simplified)
**Purpose**: Attribute costs and usage by scope/window.  
**Consumers**: FinOps, leadership.  
**Inputs**: `scope{tenant|project|app|run}`, `window{from,to}`  
**Outputs/State**: `usage{cpu,gpu,mem,io,bytesOut}`, `costBreakdown`, `unitCosts`  
**Contract (order)**: `kind: Report`, `action: CostUsage`  
**Lifecycle**: Aggregate → Attribute → Render  
**SLIs/SLOs**: attribution accuracy, report latency  
**AuthZ**: `finops:read`  
**Telemetry/Metering**: emitted per report  
**Policy checks**: Anonymization, retention  
**Failure Modes**: Incomplete meters, skewed tags  

### O-13 — SLO Report / Error Forensics (simplified)
**Purpose**: Summarize SLI/SLO attainment; support RCA.  
**Consumers**: SRE, product owners.  
**Inputs**: `scope`, `window`, `dimensions{app,version,node,region}`  
**Outputs/State**: SLO summary, top errors, RCA hints  
**Contract (order)**: `kind: Report`, `action: SLO`  
**Lifecycle**: Aggregate → Analyze → Publish  
**SLIs/SLOs**: MTTR, error-budget burn  
**AuthZ**: `slo:read`  
**Telemetry/Metering**: error histograms, burn rate  
**Policy checks**: PII scrub in logs  
**Failure Modes**: Missing traces, noisy metrics  

### O-14 — Access Grant / Share Result (simplified)
**Purpose**: Grant time-boxed access to results/models/registries.  
**Consumers**: Project owners, data stewards.  
**Inputs**: `resourceRef`, `shareTo{user|group|mesh-service}`, `ttl`, `scopes`  
**Outputs/State**: `grantId`, revocation handle  
**Contract (order)**: `kind: Share`, `action: Grant|Revoke`  
**Lifecycle**: Issue → Monitor → Expire/Revoke  
**SLIs/SLOs**: grant latency, violations=0  
**AuthZ**: `sharing:grant`  
**Telemetry/Metering**: grants active, revocations  
**Policy checks**: Least privilege, residency guardrails  
**Failure Modes**: Over-privilege, stale grants
