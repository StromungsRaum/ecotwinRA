# Provisioning (Control Plane)

Tenant/project creation, quotas/policies, application (process-chain) CRUD, secret/connection bindings, observability, and FinOps plans.

Provisioning defines **who can do what, where, and under which guardrails**. It creates and configures the *context* in which workloads will later run: tenants, projects, quotas, policies, process-chains (applications), secrets/bindings, and telemetry/FinOps plans.

- **Audience:** Platform/Mesh admins, security, FinOps, app publishers
- **Artifacts:** Tenants, projects, app definitions, policy attachments, bindings
- **Outcomes:** Governed, auditable environments ready for use; no payload data processed yet
- **Failure domain:** Misconfiguration & policy drift (fixed via validation, idempotent updates)

---

### P-01 — Create Tenant (simplified)
**Purpose:** Provision a tenant with quotas, policies, SLO tiers.  
**Consumers:** Platform/Mesh admins.  
**Inputs:** `tenantSpec{name, tier, quota, residency, policies}`  
**Outputs/State:** `tenantId`, `status{Pending,Ready,Error}`  
**Contract (order):** `kind: Tenant`, `action: Create`, `spec: tenantSpec`  
**Lifecycle:** Validate → Create → Attach policies/secrets → Ready  
**SLIs/SLOs:** Provision time ≤ 5m (P95)  
**AuthZ:** `tenant.admin:create`  
**Telemetry/Metering:** `tenant.provision.duration`, `policy.attach.count`  
**Policy checks:** Residency, data class, quota bounds  
**Failure Modes:** Policy conflict, quota overflow, credentials missing

### P-02 — Update Tenant / Quotas / Policies (simplified)
**Purpose:** Adjust quotas, residency, policies for compliance.  
**Consumers:** Platform admins, compliance.  
**Inputs:** `tenantId`, `patch{quota,policyRefs,residency}`  
**Outputs/State:** `tenantId`, updated spec  
**Contract (order):** `kind: Tenant`, `action: Update`, `spec: patch`  
**Lifecycle:** Validate patch → Apply → Conformity check  
**SLIs/SLOs:** Time-to-effect ≤ 5m (P95), violations=0  
**AuthZ:** `tenant.admin:update`  
**Telemetry/Metering:** `tenant.update.duration`, `policy.violations`  
**Policy checks:** Residency EU-only, data-class allowlist, quota ceilings  
**Failure Modes:** Policy violation, quota conflict, key-rotation error

### P-03 — Delete/Suspend Tenant (simplified)
**Purpose:** Decommission or freeze a tenant.  
**Consumers:** Central ops, legal/compliance.  
**Inputs:** `tenantId`, `retentionAction{archive|purge}`, `effectiveAt`  
**Outputs/State:** Finalizers done, `TenantDeleted|TenantSuspended`  
**Contract (order):** `kind: Tenant`, `action: Delete|Suspend`  
**Lifecycle:** Quiesce → Egress/Archive → Disable → Delete  
**SLIs/SLOs:** Deletion window, zero orphaned resources  
**AuthZ:** `tenant.admin:delete`  
**Telemetry/Metering:** `tenant.delete.duration`  
**Policy checks:** Legal hold, retention, export controls  
**Failure Modes:** Active runs, legal hold blocks, dangling bindings

### P-04 — Create Project (Workspace) (simplified)
**Purpose:** Create an isolated workspace under a tenant.  
**Consumers:** Project admins, operators.  
**Inputs:** `projectSpec{name, labels, dataDomains, artifactBuckets}`  
**Outputs/State:** `projectId`, `status{Pending,Ready,Error}`  
**Contract (order):** `kind: Project`, `action: Create`, `spec: projectSpec`  
**Lifecycle:** Create → Bind roles/buckets → Ready  
**SLIs/SLOs:** Provision ≤ 2m (P95)  
**AuthZ:** `project.admin:create`  
**Telemetry/Metering:** `project.provision.duration`  
**Policy checks:** Namespace isolation, bucket residency  
**Failure Modes:** Bucket binding failure, RBAC misconfig

### P-05 — Application / Process-Chain CRUD (simplified)
**Purpose:** Publish and manage executable chains (sim/doe/train/etl).  
**Consumers:** App publishers, platform engineers.  
**Inputs:** `appSpec{type, version, graph, resources, inputsSchema, outputsSchema, SLO}`  
**Outputs/State:** `appId@version`, validation report  
**Contract (order):** `kind: Application`, `action: Publish|Update|Delete`  
**Lifecycle:** Draft → Validate → Publish → Deprecate  
**SLIs/SLOs:** Validation pass rate, time-to-publish  
**AuthZ:** `app.publisher:*`  
**Telemetry/Metering:** `app.publish.duration`, `schema.validation.errors`  
**Policy checks:** Input/output schema, resource ceilings  
**Failure Modes:** Schema invalid, incompatible graph, SLO unsatisfied

### P-06 — Capability Template Registry (simplified)
**Purpose:** Provide reusable order templates (e.g., Hi-Fi Run, DoE Batch).  
**Consumers:** Catalog curators, end users.  
**Inputs:** `templateSpec{params, defaults, SLOs, costModel}`  
**Outputs/State:** `templateId`, catalog entry  
**Contract (order):** `kind: Template`, `action: Publish|Update|Delete`  
**Lifecycle:** Author → Validate → Publish  
**SLIs/SLOs:** Catalog availability, lint pass  
**AuthZ:** `catalog.curator:publish`  
**Telemetry/Metering:** `template.publish.duration`  
**Policy checks:** Param bounds, SLO sanity  
**Failure Modes:** Invalid defaults, cost model missing

### P-07 — Secret & Connection Binding (simplified)
**Purpose:** Bind secure connections to sources/sinks (S3/DB/MQTT/OPC UA).  
**Consumers:** Project admins, data engineers.  
**Inputs:** `binding{type, endpoint, role, secretRef, test?}`  
**Outputs/State:** `bindingId`, connection test result  
**Contract (order):** `kind: Binding`, `action: Create|Update|Delete`  
**Lifecycle:** Create → Test → Activate  
**SLIs/SLOs:** Bind success rate, rotation latency  
**AuthZ:** `binding.admin:*`  
**Telemetry/Metering:** `binding.test.result`, `secret.rotate.count`  
**Policy checks:** Least privilege, TTL, residency  
**Failure Modes:** Credential invalid, CA mismatch, scope too broad

### P-08 — Policy-as-Code Attach (simplified)
**Purpose:** Attach DLP/geo/residency/retention/export controls.  
**Consumers:** SecOps, compliance.  
**Inputs:** `policyRefs`, `rego/k8s-policy`, `mode{enforce|warn}`  
**Outputs/State:** `policyAttachmentId`, conformance report  
**Contract (order):** `kind: PolicyAttachment`, `action: Attach|Detach`  
**Lifecycle:** Validate → Attach → Monitor  
**SLIs/SLOs:** Enforcement rate, violations=0  
**AuthZ:** `policy.admin:attach`  
**Telemetry/Metering:** `policy.violations`, `enforcement.latency`  
**Policy checks:** Data class, residency, export, retention  
**Failure Modes:** Conflicting policies, false positives

### P-09 — Observability & FinOps Plan Attach (simplified)
**Purpose:** Declare telemetry and usage metering for scope.  
**Consumers:** SRE, FinOps.  
**Inputs:** `telemetryPlan{metrics,traces,logs}`, `usagePlan{meteringKeys}`  
**Outputs/State:** `planId`, activation status  
**Contract (order):** `kind: TelemetryPlan`, `action: Attach|Update|Detach`  
**Lifecycle:** Validate → Attach → Emit  
**SLIs/SLOs:** Meter coverage, attribution accuracy  
**AuthZ:** `telemetry.admin:configure`  
**Telemetry/Metering:** Emitted per plan; coverage %  
**Policy checks:** PII redaction, log retention  
**Failure Modes:** Missing meters, incorrect labels
