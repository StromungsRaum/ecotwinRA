# Operations (Runtime)

Simulation runs, DoE/optimization, surrogate/model training, model-registry operations, inference/queries, sensor monitoring, data curation, retraining, artifact egress, and SLO/FinOps reporting.

Operations execute **work inside an existing provisioned context**. This is where value is produced: start/manage simulations and DoE batches, train/promote models, run inference, curate datasets, monitor sensors, export artifacts, and report SLO/FinOps.

- **Audience:** Engineers, data scientists, operators, downstream apps
- **Artifacts:** Runs/batches, datasets, models, results, reports, usage records
- **Outcomes:** Computation performed, artifacts produced and egressed, telemetry emitted
- **Failure domain:** Workload/runtime errors (handled via retries, checkpoints, graceful control)

---

### O-01 — Start Simulation Run (simplified)
**Purpose:** Execute a reproducible high-fidelity simulation.  
**Consumers:** Engineers, analysts, automated pipelines.  
**Inputs:** `appId@version|templateId`, `runSpec{geometryRef, materials, BCs, params, resources}`, optional `SLO`  
**Outputs/State:** `runId`, streams `RunStatus|Logs|Artifacts`  
**Contract (order):** `kind: Run`, `action: Start`, `spec: runSpec`  
**Lifecycle:** Queued → Running → Succeeded|Failed → Archived  
**SLIs/SLOs:** Queue time, TTFR, success rate, cost/run  
**AuthZ:** `runs:start`  
**Telemetry/Metering:** cpu/gpu/mem/io, bytes-out  
**Policy checks:** Data class & residency at egress  
**Failure Modes:** Solver fail, preemption; checkpoint/retry

### O-02 — Start DoE / Optimization Batch (simplified)
**Purpose:** Explore/optimize a design space via many runs.  
**Consumers:** Optimization engineers, data scientists.  
**Inputs:** `designSpace`, `sampler|optimizer`, `budget`, `constraints`  
**Outputs/State:** `batchId`, `datasetRef`, `paretoRef`  
**Contract (order):** `kind: DoEBatch`, `action: Start`  
**Lifecycle:** Plan → Evaluate → Consolidate → Complete  
**SLIs/SLOs:** eval/sec, feasible rate, hypervolume  
**AuthZ:** `runs:start`  
**Telemetry/Metering:** evaluations, cost by eval  
**Policy checks:** Max evaluations, param bounds  
**Failure Modes:** Infeasible space, convergence stall

### O-03 — Manage Run (Pause/Resume/Cancel) (simplified)
**Purpose:** Control lifecycle of running jobs.  
**Consumers:** Operators, support.  
**Inputs:** `runId`, `action{pause|resume|cancel}`  
**Outputs/State:** `status`, events `RunPaused|RunCancelled`  
**Contract (order):** `kind: Run`, `action: Manage`  
**Lifecycle:** Live control → Confirmed state  
**SLIs/SLOs:** Control latency ≤ 30s (P95)  
**AuthZ:** `runs:manage`  
**Telemetry/Metering:** control events  
**Policy checks:** Graceful termination windows  
**Failure Modes:** Non-cooperative job, lost heartbeat

### O-04 — Fetch Status / Logs / Events (simplified)
**Purpose:** Observe progress and diagnostics.  
**Consumers:** Users, SRE.  
**Inputs:** `runId|batchId`, `since`, `follow?`  
**Outputs/State:** `status`, `progress`, `events`, `metrics`  
**Contract (order):** `kind: RunStatus`, `action: Get`  
**Lifecycle:** Read-only  
**SLIs/SLOs:** API P99 ≤ 500ms, freshness  
**AuthZ:** `runs:view`  
**Telemetry/Metering:** request/latency counters  
**Policy checks:** Redaction of sensitive logs  
**Failure Modes:** Over-fetch, throttling

### O-05 — Artifact Export / Data Egress (simplified)
**Purpose:** Publish artifacts into enterprise lake/catalog.  
**Consumers:** Data platform, BI, downstream apps.  
**Inputs:** `artifactQuery{runId,type}`, `egressTarget{lake/cat/topic}`, `mode{copy|link}`  
**Outputs/State:** `artifactUris`, `catalogEntry`, `ArtifactReady`  
**Contract (order):** `kind: Egress`, `action: Export`  
**Lifecycle:** Validate → Transfer → Register  
**SLIs/SLOs:** Egress time, violations=0  
**AuthZ:** `artifacts:egress`  
**Telemetry/Metering:** bytes-out, transfer duration  
**Policy checks:** Residency, DLP, retention  
**Failure Modes:** Expired creds, size limits, timeout

### O-06 — Train Surrogate / Foundation Model (simplified)
**Purpose:** Train a predictive model on curated datasets.  
**Consumers:** Data scientists, ML engineers.  
**Inputs:** `trainingSpec{datasetRef, modelType, features, objective, resources}`  
**Outputs/State:** `modelId@version`, metrics  
**Contract (order):** `kind: Training`, `action: Start`  
**Lifecycle:** Queue → Train → Validate → Complete  
**SLIs/SLOs:** time-to-best, val score, cost/epoch  
**AuthZ:** `models:train`  
**Telemetry/Metering:** gpu/cpu hours, checkpoints  
**Policy checks:** Dataset consent/class, lineage  
**Failure Modes:** Data drift, OOM, early stop

### O-07 — Model Registry Ops (Promote/Rollback/Retire) (simplified)
**Purpose:** Govern model lifecycle and stages.  
**Consumers:** MLOps, governance.  
**Inputs:** `modelId@version`, `stage{staging|prod}`, `policyCheck`  
**Outputs/State:** `status`, `ModelPromoted|ModelRetired`  
**Contract (order):** `kind: Model`, `action: Promote|Rollback|Retire`  
**Lifecycle:** Check → Transition → Record  
**SLIs/SLOs:** Governance pass rate, rollback time  
**AuthZ:** `models:promote|retire`  
**Telemetry/Metering:** promotions, rollbacks  
**Policy checks:** Model card present, metrics thresholds  
**Failure Modes:** Threshold fail, dependency conflicts

### O-08 — Model Inference / Query (simplified)
**Purpose:** Serve predictions (batch or realtime).  
**Consumers:** Apps, services, engineers.  
**Inputs:** `modelRef`, `query{inputs, batch|stream}`, `explain?`  
**Outputs/State:** `predictions`, optional `explanations`  
**Contract (order):** `kind: Inference`, `action: Invoke`  
**Lifecycle:** Request → Response  
**SLIs/SLOs:** P50/P99 latency, throughput, error rate  
**AuthZ:** `inference:invoke`  
**Telemetry/Metering:** qps, latency, tokens/rows processed  
**Policy checks:** Input validation, quota limits  
**Failure Modes:** Timeout, cold start, version mismatch

### O-09 — Sensor Monitoring / Edge Stream Attach (simplified)
**Purpose:** Attach live streams; detect anomalies/drift.  
**Consumers:** Ops, data scientists.  
**Inputs:** `streamSpec{topic, codec, rate}`, `monitoringRules{threshold, drift, anomaly}`  
**Outputs/State:** `monitorId`, events `AnomalyDetected|DriftAlert`  
**Contract (order):** `kind: Monitoring`, `action: Attach`  
**Lifecycle:** Attach → Observe → Alert  
**SLIs/SLOs:** e2e lag, detection precision/recall  
**AuthZ:** `monitoring:configure`  
**Telemetry/Metering:** event rate, lag  
**Policy checks:** PII redaction, retention  
**Failure Modes:** Backpressure, decode errors

### O-10 — Data Curation / Dataset Build (simplified)
**Purpose:** Build governed datasets for modeling/analytics.  
**Consumers:** Data platform, DS/ML.  
**Inputs:** `curationSpec{sources, filters, labeling, augmentation, splits}`  
**Outputs/State:** `datasetRef`, `dataSheet`  
**Contract (order):** `kind: Dataset`, `action: Build`  
**Lifecycle:** Ingest → Transform → Validate → Publish  
**SLIs/SLOs:** Coverage, label quality, build time  
**AuthZ:** `datasets:curate`  
**Telemetry/Metering:** rows, bytes, quality metrics  
**Policy checks:** Data class, consent, lineage  
**Failure Modes:** Schema drift, low quality labels

### O-11 — Retraining Trigger (Policy/Drift/Calendar) (simplified)
**Purpose:** Automate retraining based on policy or signals.  
**Consumers:** MLOps, governance.  
**Inputs:** `trigger{drift>τ|schedule|manual}`, `trainSpecRef`  
**Outputs/State:** `jobId`, `newModelRef`  
**Contract (order):** `kind: Retraining`, `action: Trigger`  
**Lifecycle:** Detect → Trigger → Train → Promote  
**SLIs/SLOs:** trigger→start latency, recovery time  
**AuthZ:** `models:train`  
**Telemetry/Metering:** triggers, MTTR  
**Policy checks:** Safe rollout, canary/quarantine  
**Failure Modes:** False positives, compute shortage

### O-12 — Cost & Usage Report (FinOps) (simplified)
**Purpose:** Attribute costs and usage by scope/window.  
**Consumers:** FinOps, leadership.  
**Inputs:** `scope{tenant|project|app|run}`, `window{from,to}`  
**Outputs/State:** `usage{cpu,gpu,mem,io,bytesOut}`, `costBreakdown`, `unitCosts`  
**Contract (order):** `kind: Report`, `action: CostUsage`  
**Lifecycle:** Aggregate → Attribute → Render  
**SLIs/SLOs:** Attribution accuracy, report latency  
**AuthZ:** `finops:read`  
**Telemetry/Metering:** Emitted per report  
**Policy checks:** Anonymization, retention  
**Failure Modes:** Incomplete meters, skewed tags

### O-13 — SLO Report / Error Forensics (simplified)
**Purpose:** Summarize SLI/SLO attainment; support RCA.  
**Consumers:** SRE, product owners.  
**Inputs:** `scope`, `window`, `dimensions{app,version,node,region}`  
**Outputs/State:** SLO summary, top errors, RCA hints  
**Contract (order):** `kind: Report`, `action: SLO`  
**Lifecycle:** Aggregate → Analyze → Publish  
**SLIs/SLOs:** MTTR, error-budget burn  
**AuthZ:** `slo:read`  
**Telemetry/Metering:** Error histograms, burn rate  
**Policy checks:** PII scrub in logs  
**Failure Modes:** Missing traces, noisy metrics

### O-14 — Access Grant / Share Result (simplified)
**Purpose:** Grant time-boxed access to results/models/registries.  
**Consumers:** Project owners, data stewards.  
**Inputs:** `resourceRef`, `shareTo{user|group|mesh-service}`, `ttl`, `scopes`  
**Outputs/State:** `grantId`, revocation handle  
**Contract (order):** `kind: Share`, `action: Grant|Revoke`  
**Lifecycle:** Issue → Monitor → Expire/Revoke  
**SLIs/SLOs:** Grant latency, violations=0  
**AuthZ:** `sharing:grant`  
**Telemetry/Metering:** Grants active, revocations  
**Policy checks:** Least privilege, residency guardrails  
**Failure Modes:** Over-privilege, stale grants
