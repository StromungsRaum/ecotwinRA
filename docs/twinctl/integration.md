# IANUS Simulation Integration to ApeiroRA

The integration will roughly consist of 2 phases.
During the first phase a basic service from StrömungsRaum will be exposed to Platform Mesh.
This service will just list available applications in StrömungsRaum.
In the second phase a full application will be provided.

## Terminology

KCP
: Kubernates like control plane [kcp.io](https://kcp.io)

MSP
: Managed Service Provider

## Plan

### IANUS Team Tasks

1. IANUS needs to you host their own operator in their own cluster
2. IANUS installs sync agent there
3. IANUS needs to host a UI that can be used in Platform Mesh (the UI that allows us to have CRUD operations against the resource that the IANUS operator will expose, e.g. a model).

### Showroom Team Tasks

1. They will create a provider workspace e.g. provider-ianus in platform mesh.
2. Provide you with a kubeconfig that connects to that workspace, and this is the kubeconfig you use with the sync agent.
3. They provide you with access to a demo account that will represent a consumer account that has Ianus service enabled to order the CR exposed via the sync agent. e.g. the model.
4. Users that access that account will see the Ianus service with its resources below in the left hand panel and be able to order resource via CRUD operations in the UI.... OR using the terminal and kubectl. FYI this ui that shows the CRUD operations will be your UI that you host in your cluster. The showroom can help with configuring and enabling that ui in platform mesh.

The first thing I would think about is creating the operator and what the resource should be that shall be presented in the Platform Mesh and what its fields should be.

![MSP Cluster](../assets/platform_mesh.png)

![Mesh Platform to IANUS Simulation Flow](../assets/platform_mesh_flow.png)

## Use Case:

_Third party_(TP) wants to simulate product iterations as part of their product development.
The _TP_ has their own k8s cluster to manage product development through continuous integration(CI).
_IANUS Simulation_(IANUS) publishes their simulation services, _simod_(Simulation On Demand), to Apeiro _Platform Mesh_.
The _TP_ obtains _simod_ service definitions, k8s manifests, from _Platform Mesh_.
These manifests are used to create the _simod service_ inside the _TP_ k8s cluster.
With each new version of the _TP_ product a simulation is ordered through _simod service_.

[diagram](./integration-diagram.md)
