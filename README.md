# ECOTWIN Reference Architecture

The **E**dge-**C**loud **O**perative Digital **TWIN** project attempts to development of a powerful and sustainable multi-provider edge-cloud continuum.
This will enable next-generation AI and industrial process simulation services.
The aim of the ECOTWIN reference architecture is to prescribe how all the technologies fit together to provide the edge-cloud continuum.

## Concepts

Many components are networked together to provide an edge-cloud continuum.
The entities that make the network can be categorized into:

- Software resources
- Hardware Resources
- Network Resources
- People
- Business Resources

They have been outlined in the [concepts](./docs/concepts.md) document.

## Digital Twin Controller **twinctl**

In order to manage the edge-cloud continuum a controller is proposed.
This controller will tie into all the software stacks that are used.
Each individual twin will be configured using a set of YAML files.
The controller will be use the twin information to make sure that dependent tools have consistent information.

Twin controller is further documented in [twinctl](./docs/twinctl.md).
