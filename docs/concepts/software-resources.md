# Software Resources

## Workload Manager / Batch Manager

Schedules and dispatches HPC/Cloud jobs with queuing, priorities, resource quotas, and policy enforcement.

## Containerized Application (Apptainer)

Portable, reproducible runtime image for Sim/AI workloads optimized for HPC environments.

## Application

User-facing UI/API component that orchestrates processes and accesses models through explicit model-gates.

## Optimization Agent

Rule/ML-driven service that tunes control variables to optimize targets (e.g., kWh/m³, throughput) under constraints.

## AI Agent

Goal-driven orchestrator that plans/runs sims, updates surrogates, and proposes/executes control actions autonomously.

## Service

- **Heavy-weight** – Resource-intensive, usually batch/cluster bound (HiFi simulation, large training).
- **Light-weight** – Low-latency edge/API service (surrogate inference, stream analytics).
- **asynch** – Non-blocking execution using queues/callbacks/events.
- **synch** – Request/response with bounded latency on the critical path.

## Sim-as-a-Service (Sim-aaS)

On-demand HiFi simulation on HPC/Cloud with automated meshing, scheduling, and artifact capture.

## Opt-as-a-Service (Opt-aaS)

Autonomous optimization loop (e.g., Bayesian/EAs) orchestrating simulations and surrogates against defined KPIs.

## Model

- **Simulation Model (High-Fidelity)** – Physics-accurate, compute-heavy ground-truth model (e.g., CFD/FEM).
- **Surrogate Model** – Fast approximation of HiFi behavior (ML/ROM) suitable for real-time use.
- **Reduced Order Model** – Physics-preserving model reduction (e.g., MOR/POD) with low dimensional state.

## Model Gate

Contracted access point from apps/processes to models (I/O schema, latency/SLA, validation status).

## Data Bridge

Streaming/batch interoperability layer for model↔model and model↔edge data (schema mapping, QoS, lineage).

## Sensor Bridge

Edge adapter for timestamped P/Q/T/vib/strain signals incl. filtering, sync, unit and non-dimensional transforms.

## Thread / Process (technical)

OS-level execution units: a **process** has its own address space; **threads** share memory within a process—mapped to containers/Pods/VMs with explicit CPU/memory/NUMA/SIMD/GPU binding.

## Job

Concrete execution instance of a workload with inputs, resources, lifecycle, and artifacts.

## Batch

Grouped jobs executed under a shared policy/time window and consistent data snapshot.

## Artifact Registry

Versioned store for meshes, configs, checkpoints, and surrogate snapshots with provenance/SBOM.

## HPC Queue / QoS Class

Workload class with preemption/priority, reservation, and accelerator constraints under scheduler control.

## Layer

- **Process Layer** – Domain-specific step chain with **process-gates** (temporal/spatial causality, handover/rollback criteria).
- **Application Layer** – Verticals and apps initiating and coordinating processes.
- **Model Layer** – HiFi/Surrogate/ROM assets exposed via model-gates.
- **Domain Layer** – Multiphysics domains (e.g., flow, heat, deformation) and their couplings.
- **Data Layer** – Persistence, data-bridge, feature pipelines, metadata/catalog.
- **Execution Layer** – Runtime targets (edge/cluster/cloud) and control loops.

## Process (industrial)

Business/engineering workflow step (domain-specific) with defined inputs/outputs, QoS, KPIs and **process-gates** across the value chain.

## Virtual Representation (VR)

Abstract data-driven view of an asset/system without full simulation synchronization.

## Virtual Prototype (VP)

Composable, simulatable bundle of domains and models representing a system design.

## Digital Twin (DT)

Bi-directionally synchronized VP/VR with live data, state estimation, and actuation path.
