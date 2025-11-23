# IANUS Integration to ApeiroRA Diagram

```mermaid
flowchart TD
    subgraph Apeiro
        direction LR
        IP@{ shape: brace-r, label: "IANUS publishes it's service to Apeiro" }
        ARA1(Portal)
        IP-->ARA1
    end

    subgraph ss [IANUS k8s]
        id23[**simod** backend]
        id24[**simod** frontend]
        id24-->|Existing|id23
    end

    subgraph tpc [Third Party k8s cluster]
        subgraph sp [simod service proxy]
            id21(REST API)
            id22(**twinctl**)
            id21-->|Forwards API requests to **twinctl**|id22
        end
        id32(Third Party)
        id32-->|Third party consumes IANUS service in there own k8s cluster|id21
    end


%%    Apeiro----|simod publishes manifests to ApeiroRA|simod
    Apeiro--->|Third party gets IANUS manifests from ApeiroRA|sp
%%    id22-->|Fulfills request by contacting IANUS simod|id23
    id22-->|**twinctl** fulfills request by communicating with **simod** server|id23

```

```mermaid
flowchart TB
classDef cluster fill:white,color:black,stroke:black;
classDef reconciledBy fill:#dedede,stroke:black,stroke-dasharray: 5,color:black;
classDef k8sObject fill:#b3b3b3,color:black,stroke:black;
classDef information fill:#b3b3b3,color:black,stroke:none;
classDef templateOf fill:#b3b3b3,color:black,stroke:black,stroke-dasharray: 2;
classDef ocm fill:white,stroke:black,color:black;
classDef legendStyle fill:white,stroke:black,color:black,stroke-dasharray: 2;
classDef legendStartEnd height:0px;
classDef legendItems fill:#b3b3b3,stroke:none,color:black;
subgraph legend[Legend]
start1[ ] ---references[referenced by] --> end1[ ]
start2[ ] -.-creates -.-> end2[ ]
start3[ ] ---instanceOf[instance of] --> end3[ ]
start4[ ] ~~~reconciledBy[reconciled by] ~~~ end4[ ]
start5[ ] ~~~k8sObject[k8s object] ~~~ end5[ ]
start6[ ] ~~~templateOf[template of] ~~~ end6[ ]
end
subgraph background[ ]
direction TB
subgraph ocmRepo[OCM Repository]
subgraph ocmCV[OCM Component Version]
direction RL
subgraph ocmResourceHelm[OCM Resource: HelmChart]
end
subgraph ocmResourceImage[OCM Resource: Image]
end
subgraph ocmResourceRGD[OCM Resource: RGD]
end
end
end
subgraph k8sCluster[Kubernetes Cluster]
subgraph bootstrap[OCM Controllers]
k8sRepo[OCMRepository]
k8sComponent[Component]
k8sResourceRGD[Resource: RGD]
k8sDeployer[Deployer]
end
subgraph kro[kro]
subgraph rgd[RGD: Bootstrap]
rgdResourceHelm[Resource: HelmChart]
rgdResourceImage[Resource: Image]
rgdSource[FluxCD: OCI Repository]
rgdHelmRelease[FluxCD: HelmRelease]
end
crdBootstrap[CRD: Bootstrap]
subgraph instanceBootstrap[Instance: Bootstrap]
subgraph ocmControllers[OCM Controllers]
k8sResourceHelm[Resource: HelmChart]
k8sResourceImage[Resource: Image]
end
subgraph fluxCD[FluxCD]
source[OCI Repository]
helmRelease[HelmRelease]
end
k8sResourceImage ---info[localization reference] --> helmRelease
end
end
helmRelease --> deployment[Deployment: Helm chart]
end
ocmRepo --> k8sRepo --> k8sComponent --> k8sResourceRGD --> k8sDeployer --> rgd --> crdBootstrap --> instanceBootstrap
k8sComponent --> k8sResourceHelm & k8sResourceImage
k8sResourceHelm --> source --> helmRelease
end
linkStyle default fill:none,stroke:black,color:black;
linkStyle 2,3,20 stroke:black,fill:none,color:black,stroke-dasharray: 10;
linkStyle 4,5,21 stroke:black,fill:none,color:black,stroke-dasharray: 4;
class start1,end1,start2,end2,start3,end3,start4,end4,start5,end5,start6,end6 legendStartEnd;
class references,creates,instanceOf legendItems;
class templateOf,rgdResourceHelm,rgdResourceImage,rgdSource,rgdHelmRelease templateOf;
class info information;
class reconciledBy,ocmK8sToolkit,bootstrap,fluxCD,kro reconciledBy;
class k8sObject,rgd,k8sRepo,k8sComponent,k8sResourceRGD,k8sDeployer,k8sResourceHelm,k8sResourceImage,source,helmRelease,deployment,crdBootstrap,instanceBootstrap k8sObject;
class ocmRepo,ocmCV,ocmResourceHelm,ocmResourceRGD,ocmResourceImage ocm;
class k8sCluster cluster;
class legend legendStyle;
```
