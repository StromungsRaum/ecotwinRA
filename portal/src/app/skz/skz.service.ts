/**
 * SKZ Materials - GraphQL API Client
 *
 * SKZ is a dummy/demo provider: no backend, no operator. HttpBin-style - the
 * portal's "+" button creates one Material card per material. Unlike the
 * provider's original name-only form with canned placeholder values, the CRD
 * now mirrors a real material characterization file 1:1
 * (docs/assets/EastmanPlastics1000.json, apis/skz/v1alpha1/material_types.go)
 * so the user pastes a JSON blob in that shape and it's sent through
 * unmodified (see skz.component.ts).
 *
 * NOTE: the CRD Kind is "Material", not "MaterialParameters" - a Kind name
 * ending in "s" broke the Platform Mesh CRD Gateway's pluralization
 * (confirmed live: it generated no list/collection field at all, only a
 * singular `MaterialParameters(name, namespace)` getter). Renamed before
 * wiring this up for real.
 *
 * NOTE: nested structs do NOT get a flat `<GoTypeName>_Input` type - the
 * gateway names each nested input type after its full field PATH from the
 * Kind, e.g. `SkzPlatformMeshIoV1alpha1MaterialSpecData_Input` for
 * `spec.data`, not `MaterialData_Input` (confirmed via live `__schema`
 * introspection, 2026-11-04, after CreateMaterial failed with "Unknown
 * type" errors). See CREATE_MUTATION below for the real names.
 */
import { Injectable, inject } from '@angular/core';
import { Observable, of, catchError, map } from 'rxjs';
import { GraphQLClientService } from '../shared/graphql-client.service';

export interface MaterialData {
  type: string;
  norm: string;
  material_number: string;
  density: number;
  heatconductivity: number;
  heatcapacity: number;
}

export interface ViscoModel {
  model: string;
  exponent: number;
  zero_viscosity: number;
  critical_stress_level: number;
}

export interface ThermalDataPoint {
  temperature: number;
  heatcapacity: number;
  heatconductivity: number;
  density: number;
}

export interface HeatModel {
  model: string;
  thermaldata: ThermalDataPoint[];
}

export interface TempModel {
  model: string;
  c1: number;
  c2: number;
  D3: number;
  referenceTemperature: number;
}

export interface PVTModel {
  model: string;
  b1s: number;
  b2s: number;
  b3s: number;
  b4s: number;
  b1m: number;
  b2m: number;
  b3m: number;
  b4m: number;
  b5: number;
  b6: number;
  b7: number;
  b8: number;
  b9: number;
}

export interface MechProperties {
  E1: number;
  E2: number;
  E_avg: number;
  v12: number;
  v23: number;
  alpha_1: number;
  alpha_2: number;
  G12: number;
}

export interface ShrinkageProperties {
  parallel: number;
  perpendicular: number;
  avg_nominal: number;
}

/** The full material characterization shape - matches EastmanPlastics1000.json. */
export interface MaterialSpec {
  name: string;
  data: MaterialData;
  viscoModel: ViscoModel;
  heatModel: HeatModel;
  tempModel: TempModel;
  pvtmodel: PVTModel;
  structure: string;
  min_melt_temp: number;
  max_melt_temp: number;
  abs_max_melt_temp: number;
  eject_temp: number;
  freeze_temp: number;
  melt_density: number;
  solid_density: number;
  mech_properties: MechProperties;
  shrinkage_properties: ShrinkageProperties;
  max_shear_stress: number;
  max_shear_rate: number;
}

export interface Material {
  metadata: {
    name: string;
    namespace?: string;
  };
  spec?: MaterialSpec;
}

interface MaterialListResponse {
  skz_platform_mesh_io: {
    v1alpha1: {
      Materials: {
        items: Material[];
      };
    };
  };
}

const LIST_QUERY = `
  query ListMaterials {
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

const CREATE_MUTATION = `
  mutation CreateMaterial(
    $name: String!
    $materialName: String!
    $data: SkzPlatformMeshIoV1alpha1MaterialSpecData_Input!
    $viscoModel: SkzPlatformMeshIoV1alpha1MaterialSpecViscoModel_Input!
    $heatModel: SkzPlatformMeshIoV1alpha1MaterialSpecHeatModel_Input!
    $tempModel: SkzPlatformMeshIoV1alpha1MaterialSpecTempModel_Input!
    $pvtmodel: SkzPlatformMeshIoV1alpha1MaterialSpecPvtmodel_Input!
    $structure: String!
    $minMeltTemp: Float!
    $maxMeltTemp: Float!
    $absMaxMeltTemp: Float!
    $ejectTemp: Float!
    $freezeTemp: Float!
    $meltDensity: Float!
    $solidDensity: Float!
    $mechProperties: SkzPlatformMeshIoV1alpha1MaterialSpecMech_properties_Input!
    $shrinkageProperties: SkzPlatformMeshIoV1alpha1MaterialSpecShrinkage_properties_Input!
    $maxShearStress: Float!
    $maxShearRate: Float!
  ) {
    skz_platform_mesh_io {
      v1alpha1 {
        createMaterial(
          namespace: "default"
          object: {
            metadata: { name: $name }
            spec: {
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
        ) {
          metadata { name namespace }
        }
      }
    }
  }
`;

const DELETE_MUTATION = `
  mutation DeleteMaterial($name: String!, $namespace: String!) {
    skz_platform_mesh_io {
      v1alpha1 {
        deleteMaterial(name: $name, namespace: $namespace)
      }
    }
  }
`;

@Injectable({ providedIn: 'root' })
export class SkzService {
  private graphql = inject(GraphQLClientService);

  list(): Observable<Material[]> {
    return this.graphql.post(LIST_QUERY, {}).pipe(
      map((response: { data: MaterialListResponse }) => {
        return response.data?.skz_platform_mesh_io?.v1alpha1?.Materials?.items || [];
      }),
      catchError((error) => {
        console.error('Error fetching materials:', error);
        return of([]);
      })
    );
  }

  /** Creates a new material card from a fully parsed MaterialSpec (pasted JSON). */
  create(crName: string, spec: MaterialSpec): Observable<boolean> {
    return this.graphql
      .post(CREATE_MUTATION, {
        name: crName,
        materialName: spec.name,
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
        map((response: any) => !!response.data?.skz_platform_mesh_io?.v1alpha1?.createMaterial),
        catchError((error) => {
          console.error('Error creating material:', error);
          return of(false);
        })
      );
  }

  delete(name: string, namespace: string): Observable<boolean> {
    return this.graphql.post(DELETE_MUTATION, { name, namespace }).pipe(
      map((response: any) => !!response.data?.skz_platform_mesh_io?.v1alpha1?.deleteMaterial),
      catchError((error) => {
        console.error('Error deleting material:', error);
        return of(false);
      })
    );
  }
}
