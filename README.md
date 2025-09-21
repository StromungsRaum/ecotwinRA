# ECOTWIN Reference Architecture

## Concepts

### Software Resources

#### Workload Manager / Batch Manager
Schedules and dispatches HPC/Cloud jobs with queuing, priorities, resource quotas, and policy enforcement.

#### Containerized Application (Apptainer)
Portable, reproducible runtime image for Sim/AI workloads optimized for HPC environments.

#### Application
User-facing UI/API component that orchestrates processes and accesses models through explicit model-gates.

#### Optimization Agent
Rule/ML-driven service that tunes control variables to optimize targets (e.g., kWh/m³, throughput) under constraints.

#### AI Agent
Goal-driven orchestrator that plans/runs sims, updates surrogates, and proposes/executes control actions autonomously.

#### Service
- **Heavy-weight** – Resource-intensive, usually batch/cluster bound (HiFi simulation, large training).
- **Light-weight** – Low-latency edge/API service (surrogate inference, stream analytics).
- **asynch** – Non-blocking execution using queues/callbacks/events.
- **synch** – Request/response with bounded latency on the critical path.

#### Sim-as-a-Service (Sim-aaS)
On-demand HiFi simulation on HPC/Cloud with automated meshing, scheduling, and artifact capture.

#### Opt-as-a-Service (Opt-aaS)
Autonomous optimization loop (e.g., Bayesian/EAs) orchestrating simulations and surrogates against defined KPIs.

#### Model
- **Simulation Model (High-Fidelity)** – Physics-accurate, compute-heavy ground-truth model (e.g., CFD/FEM).
- **Surrogate Model** – Fast approximation of HiFi behavior (ML/ROM) suitable for real-time use.
- **Reduced Order Model** – Physics-preserving model reduction (e.g., MOR/POD) with low dimensional state.

#### Model Gate
Contracted access point from apps/processes to models (I/O schema, latency/SLA, validation status).

#### Data Bridge
Streaming/batch interoperability layer for model↔model and model↔edge data (schema mapping, QoS, lineage).

#### Sensor Bridge
Edge adapter for timestamped P/Q/T/vib/strain signals incl. filtering, sync, unit and non-dimensional transforms.

#### Thread / Process (technical)
OS-level execution units: a **process** has its own address space; **threads** share memory within a process—mapped to containers/Pods/VMs with explicit CPU/memory/NUMA/SIMD/GPU binding.

#### Job
Concrete execution instance of a workload with inputs, resources, lifecycle, and artifacts.

#### Batch
Grouped jobs executed under a shared policy/time window and consistent data snapshot.

#### Artifact Registry
Versioned store for meshes, configs, checkpoints, and surrogate snapshots with provenance/SBOM.

#### HPC Queue / QoS Class
Workload class with preemption/priority, reservation, and accelerator constraints under scheduler control.

#### Layer
- **Process Layer** – Domain-specific step chain with **process-gates** (temporal/spatial causality, handover/rollback criteria).
- **Application Layer** – Verticals and apps initiating and coordinating processes.
- **Model Layer** – HiFi/Surrogate/ROM assets exposed via model-gates.
- **Domain Layer** – Multiphysics domains (e.g., flow, heat, deformation) and their couplings.
- **Data Layer** – Persistence, data-bridge, feature pipelines, metadata/catalog.
- **Execution Layer** – Runtime targets (edge/cluster/cloud) and control loops.

#### Process (industrial)
Business/engineering workflow step (domain-specific) with defined inputs/outputs, QoS, KPIs and **process-gates** across the value chain.

#### Virtual Representation
Abstract data-driven view of an asset/system without full simulation synchronization.

#### Virtual Prototype
Composable, simulatable bundle of domains and models representing a system design.

#### Digital Twin
Bi-directionally synchronized VP/VR with live data, state estimation, and actuation path.

### Hardware Resources

#### On Device
Runs directly on controller/PLC/SoC of the target equipment.

#### On Premise
Executed within the operator’s facility or datacenter for sovereignty and low latency.

#### Far Edge
Site-adjacent compute focused on aggregation and pre-processing outside the plant core.

#### Near Edge
Plant-near compute for hard-latency, closed-loop control.

#### Cloud
Elastic, scalable infrastructure/platform beyond the local perimeter.

#### Edge–Cloud Continuum
Placement strategy from near/far edge to general cloud per latency, cost, data sovereignty, and compliance.

#### Sensors
Measurement sources (pressure, flow, temperature, vibration, strain) streaming to models/agents.

#### Storage
Persistence layer (object/block/time-series) with versioning, retention, and access control.

#### Cluster
Pool of nodes (CPU/GPU) with a scheduler for batch and service workloads.

#### Server
Single node with defined CPU/GPU/memory/IO configuration.

#### Device
Edge-capable unit (gateway/IPC) for local inference and bridging.

#### Host
Physical/virtual execution environment for VMs/containers.

#### Compute Grid
Federated resources across clusters/sites with unified orchestration and policy.

### Network Resources

#### Interconnect
Low-latency/high-throughput fabric (e.g., InfiniBand/RoCE/TSN) for MPI, storage, and edge backhaul.

### People

#### Creator
Builds models/services (numerics, ML, software) and delivers validated artifacts.

#### Attractor
Drives adoption/community and curates high-value use cases.

#### Extractor
Operationalizes value from data/models (Ops, BI, operations excellence).

#### Vendor
Provides components and support (hardware, software, managed services).

#### Maintainer
Owns operations, updates, security/compliance, and SLO management.

### Business Resources

#### Product / Vertical
Marketed domain solution (e.g., Pump Suite) built on the ECOTWIN platform.

#### Tenant
Isolated customer space with dedicated data, policies, SLAs, and billing.
