/**
 * IANUS Simulation - Materials - GraphQL API Client
 *
 * Two CRDs, both queried/mutated through the same CRD Gateway as every other
 * provider page in this portal:
 *  - MaterialCatalog: a singleton per account (this client creates the first
 *    one it doesn't find, named "catalog"), purely for browsing
 *    StromungsRaum's existing materials via Refresh -> status.availableMaterials.
 * NOTE: nested CREATE_SUBMISSION_MUTATION input types are named after their
 * full field PATH from the Kind (e.g.
 * `MaterialsPlatformMeshIoV1alpha1MaterialSubmissionSpecMaterialData_Input`
 * for `spec.material.data`), not a flat `<GoTypeName>_Input` - confirmed via
 * live `__schema` introspection, 2026-11-04, same bug/fix as
 * skz.service.ts's CreateMaterial mutation.
 *
 *  - MaterialSubmission: HttpBin-style, one per "+" click. The user picks a
 *    source from the account's SKZ Materials (a *different* provider's CRD -
 *    see skz.service.ts's shape, duplicated here since this is a separate
 *    Angular project/page, not a shared library); this client copies that
 *    Material's full spec into the new MaterialSubmission's spec.material at
 *    creation time (the cross-provider composition happens here in the
 *    portal, not in either operator - see
 *    apis/materials/v1alpha1/materialsubmission_types.go's package doc).
 *
 * SECRET_NAME: every new MaterialCatalog/MaterialSubmission references a
 * Secret named "simod-credentials" in the same namespace (email/password
 * keys) - same convention as injection-molding.service.ts.
 */
import { Injectable, inject } from '@angular/core';
import { Observable, of, catchError, map } from 'rxjs';
import { GraphQLClientService } from '../shared/graphql-client.service';

const SECRET_NAME = 'simod-credentials';
const CATALOG_NAME = 'catalog';
const NAMESPACE = 'default';

// --- SKZ Material (selection source) - duplicated shape from skz.service.ts ---

export interface SkzMaterialSpec {
  name: string;
  data: Record<string, unknown>;
  viscoModel: Record<string, unknown>;
  heatModel: Record<string, unknown>;
  tempModel: Record<string, unknown>;
  pvtmodel: Record<string, unknown>;
  structure: string;
  min_melt_temp: number;
  max_melt_temp: number;
  abs_max_melt_temp: number;
  eject_temp: number;
  freeze_temp: number;
  melt_density: number;
  solid_density: number;
  mech_properties: Record<string, unknown>;
  shrinkage_properties: Record<string, unknown>;
  max_shear_stress: number;
  max_shear_rate: number;
}

export interface SkzMaterial {
  metadata: { name: string; namespace?: string };
  spec?: SkzMaterialSpec;
}

// --- MaterialCatalog ---

export interface MaterialSummary {
  id: string;
  name: string;
  createdAt: string;
}

export interface MaterialCatalog {
  metadata: { name: string; namespace?: string };
  spec?: {
    refreshRequestedAt?: string;
  };
  status?: {
    phase?: string;
    availableMaterials?: MaterialSummary[];
    lastRefreshedAt?: string;
    lastProcessedRefreshRequestedAt?: string;
    message?: string;
  };
}

// --- MaterialSubmission ---

export interface MaterialSubmission {
  metadata: { name: string; namespace?: string };
  spec?: {
    sourceMaterialRef?: { name: string; namespace: string };
    material?: SkzMaterialSpec;
  };
  status?: {
    phase?: string;
    materialId?: string;
    submittedAt?: string;
    message?: string;
  };
}

const LIST_SKZ_MATERIALS_QUERY = `
  query ListSkzMaterials {
    skz_platform_mesh_io {
      v1alpha1 {
        Materials {
          items {
            metadata { name namespace }
            spec {
              name
              data { type norm material_number density heatconductivity heatcapacity }
              viscoModel { model exponent zero_viscosity critical_stress_level }
              heatModel { model thermaldata { temperature heatcapacity heatconductivity density } }
              tempModel { model c1 c2 D3 referenceTemperature }
              pvtmodel { model b1s b2s b3s b4s b1m b2m b3m b4m b5 b6 b7 b8 b9 }
              structure
              min_melt_temp
              max_melt_temp
              abs_max_melt_temp
              eject_temp
              freeze_temp
              melt_density
              solid_density
              mech_properties { E1 E2 E_avg v12 v23 alpha_1 alpha_2 G12 }
              shrinkage_properties { parallel perpendicular avg_nominal }
              max_shear_stress
              max_shear_rate
            }
          }
        }
      }
    }
  }
`;

const LIST_CATALOGS_QUERY = `
  query ListMaterialCatalogs {
    materials_platform_mesh_io {
      v1alpha1 {
        MaterialCatalogs {
          items {
            metadata { name namespace }
            spec { refreshRequestedAt }
            status {
              phase
              availableMaterials { id name createdAt }
              lastRefreshedAt
              lastProcessedRefreshRequestedAt
              message
            }
          }
        }
      }
    }
  }
`;

const CREATE_CATALOG_MUTATION = `
  mutation CreateMaterialCatalog($name: String!, $namespace: String!, $secretName: String!) {
    materials_platform_mesh_io {
      v1alpha1 {
        createMaterialCatalog(
          namespace: $namespace
          object: { metadata: { name: $name }, spec: { secretRefs: [{ name: $secretName }] } }
        ) {
          metadata { name namespace }
        }
      }
    }
  }
`;

const REQUEST_CATALOG_REFRESH_MUTATION = `
  mutation RequestCatalogRefresh($name: String!, $namespace: String!, $refreshRequestedAt: String!) {
    materials_platform_mesh_io {
      v1alpha1 {
        updateMaterialCatalog(
          name: $name
          namespace: $namespace
          object: { spec: { refreshRequestedAt: $refreshRequestedAt } }
        ) {
          metadata { name namespace }
        }
      }
    }
  }
`;

const LIST_SUBMISSIONS_QUERY = `
  query ListMaterialSubmissions {
    materials_platform_mesh_io {
      v1alpha1 {
        MaterialSubmissions {
          items {
            metadata { name namespace }
            spec {
              sourceMaterialRef { name namespace }
              material {
                name
                structure
                data { type density }
                min_melt_temp
                max_melt_temp
              }
            }
            status { phase materialId submittedAt message }
          }
        }
      }
    }
  }
`;

const CREATE_SUBMISSION_MUTATION = `
  mutation CreateMaterialSubmission(
    $name: String!
    $namespace: String!
    $secretName: String!
    $sourceName: String!
    $sourceNamespace: String!
    $materialName: String!
    $data: MaterialsPlatformMeshIoV1alpha1MaterialSubmissionSpecMaterialData_Input!
    $viscoModel: MaterialsPlatformMeshIoV1alpha1MaterialSubmissionSpecMaterialViscoModel_Input!
    $heatModel: MaterialsPlatformMeshIoV1alpha1MaterialSubmissionSpecMaterialHeatModel_Input!
    $tempModel: MaterialsPlatformMeshIoV1alpha1MaterialSubmissionSpecMaterialTempModel_Input!
    $pvtmodel: MaterialsPlatformMeshIoV1alpha1MaterialSubmissionSpecMaterialPvtmodel_Input!
    $structure: String!
    $minMeltTemp: Float!
    $maxMeltTemp: Float!
    $absMaxMeltTemp: Float!
    $ejectTemp: Float!
    $freezeTemp: Float!
    $meltDensity: Float!
    $solidDensity: Float!
    $mechProperties: MaterialsPlatformMeshIoV1alpha1MaterialSubmissionSpecMaterialMech_properties_Input!
    $shrinkageProperties: MaterialsPlatformMeshIoV1alpha1MaterialSubmissionSpecMaterialShrinkage_properties_Input!
    $maxShearStress: Float!
    $maxShearRate: Float!
  ) {
    materials_platform_mesh_io {
      v1alpha1 {
        createMaterialSubmission(
          namespace: $namespace
          object: {
            metadata: { name: $name }
            spec: {
              secretRefs: [{ name: $secretName }]
              sourceMaterialRef: { name: $sourceName, namespace: $sourceNamespace }
              material: {
                name: $materialName
                data: $data
                viscoModel: $viscoModel
                heatModel: $heatModel
                tempModel: $tempModel
                pvtmodel: $pvtmodel
                structure: $structure
                min_melt_temp: $minMeltTemp
                max_melt_temp: $maxMeltTemp
                abs_max_melt_temp: $absMaxMeltTemp
                eject_temp: $ejectTemp
                freeze_temp: $freezeTemp
                melt_density: $meltDensity
                solid_density: $solidDensity
                mech_properties: $mechProperties
                shrinkage_properties: $shrinkageProperties
                max_shear_stress: $maxShearStress
                max_shear_rate: $maxShearRate
              }
            }
          }
        ) {
          metadata { name namespace }
        }
      }
    }
  }
`;

const DELETE_SUBMISSION_MUTATION = `
  mutation DeleteMaterialSubmission($name: String!, $namespace: String!) {
    materials_platform_mesh_io {
      v1alpha1 {
        deleteMaterialSubmission(name: $name, namespace: $namespace)
      }
    }
  }
`;

@Injectable({ providedIn: 'root' })
export class MaterialsService {
  private graphql = inject(GraphQLClientService);

  // --- SKZ Materials (selection source) ---

  listSkzMaterials(): Observable<SkzMaterial[]> {
    return this.graphql.post(LIST_SKZ_MATERIALS_QUERY, {}).pipe(
      map((response: any) => response.data?.skz_platform_mesh_io?.v1alpha1?.Materials?.items || []),
      catchError((error) => {
        console.error('Error fetching SKZ materials:', error);
        return of([]);
      })
    );
  }

  // --- MaterialCatalog (singleton) ---

  listCatalogs(): Observable<MaterialCatalog[]> {
    return this.graphql.post(LIST_CATALOGS_QUERY, {}).pipe(
      map((response: any) => response.data?.materials_platform_mesh_io?.v1alpha1?.MaterialCatalogs?.items || []),
      catchError((error) => {
        console.error('Error fetching material catalogs:', error);
        return of([]);
      })
    );
  }

  /** Creates the singleton MaterialCatalog if it doesn't already exist. */
  ensureCatalog(): Observable<boolean> {
    return this.graphql
      .post(CREATE_CATALOG_MUTATION, { name: CATALOG_NAME, namespace: NAMESPACE, secretName: SECRET_NAME })
      .pipe(
        map((response: any) => !!response.data?.materials_platform_mesh_io?.v1alpha1?.createMaterialCatalog),
        catchError((error) => {
          // Most likely "already exists" - that's fine, not a real failure.
          console.warn('createMaterialCatalog did not succeed (may already exist):', error);
          return of(false);
        })
      );
  }

  requestCatalogRefresh(name: string, namespace: string, refreshRequestedAt: string): Observable<boolean> {
    return this.graphql
      .post(REQUEST_CATALOG_REFRESH_MUTATION, { name, namespace, refreshRequestedAt })
      .pipe(
        map((response: any) => !!response.data?.materials_platform_mesh_io?.v1alpha1?.updateMaterialCatalog),
        catchError((error) => {
          console.error('Error requesting material catalog refresh:', error);
          return of(false);
        })
      );
  }

  // --- MaterialSubmission ---

  listSubmissions(): Observable<MaterialSubmission[]> {
    return this.graphql.post(LIST_SUBMISSIONS_QUERY, {}).pipe(
      map((response: any) => response.data?.materials_platform_mesh_io?.v1alpha1?.MaterialSubmissions?.items || []),
      catchError((error) => {
        console.error('Error fetching material submissions:', error);
        return of([]);
      })
    );
  }

  /** Creates a new submission card, embedding a chosen SKZ Material's full spec. */
  createSubmission(crName: string, source: SkzMaterial, materialName?: string): Observable<boolean> {
    const spec = source.spec;
    if (!spec) {
      return of(false);
    }
    return this.graphql
      .post(CREATE_SUBMISSION_MUTATION, {
        name: crName,
        namespace: NAMESPACE,
        secretName: SECRET_NAME,
        sourceName: source.metadata.name,
        sourceNamespace: source.metadata.namespace || NAMESPACE,
        materialName: materialName ?? spec.name,
        data: spec.data,
        viscoModel: spec.viscoModel,
        heatModel: spec.heatModel,
        tempModel: spec.tempModel,
        pvtmodel: spec.pvtmodel,
        structure: spec.structure,
        minMeltTemp: spec.min_melt_temp,
        maxMeltTemp: spec.max_melt_temp,
        absMaxMeltTemp: spec.abs_max_melt_temp,
        ejectTemp: spec.eject_temp,
        freezeTemp: spec.freeze_temp,
        meltDensity: spec.melt_density,
        solidDensity: spec.solid_density,
        mechProperties: spec.mech_properties,
        shrinkageProperties: spec.shrinkage_properties,
        maxShearStress: spec.max_shear_stress,
        maxShearRate: spec.max_shear_rate,
      })
      .pipe(
        map((response: any) => !!response.data?.materials_platform_mesh_io?.v1alpha1?.createMaterialSubmission),
        catchError((error) => {
          console.error('Error creating material submission:', error);
          return of(false);
        })
      );
  }

  deleteSubmission(name: string, namespace: string): Observable<boolean> {
    return this.graphql.post(DELETE_SUBMISSION_MUTATION, { name, namespace }).pipe(
      map((response: any) => !!response.data?.materials_platform_mesh_io?.v1alpha1?.deleteMaterialSubmission),
      catchError((error) => {
        console.error('Error deleting material submission:', error);
        return of(false);
      })
    );
  }
}
