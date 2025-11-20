# IANUS Simulation Integration to ApeiroRA

## Use Case:

_Third party_(TP) wants to simulate product iterations as part of their product development.
The _TP_ has their own k8s cluster to manage product development through continuous integration(CI).
_IANUS Simulation_(IANUS) publishes their simulation services, _simod_(Simulation On Demand), to Apeiro _Platform Mesh_.
The _TP_ obtains _simod_ service definitions, k8s manifests, from _Platform Mesh_.
These manifests are used to create the _simod service_ inside the _TP_ k8s cluster.
With each new version of the _TP_ product a simulation is ordered through _simod service_.

[diagram](./integration-diagram.md)
