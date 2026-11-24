/**
 * Material Provider A - Materials - GraphQL API Client
 *
 * (Internally still "skz" - only the API group, URL path and labels were
 * renamed.) A dummy/demo provider: no backend, no operator. Its Materials are a
 * static, provider-owned catalog - the same for every consumer. They are
 * created once in the provider workspace by `make init-skz` (from
 * config/catalog-skz/materials/*.json) and projected read-only into every
 * consumer workspace through kcp cached resources (ClusterCachedResource +
 * an APIExport with virtual storage, see pkg/bootstrap/bootstrap_skz.go).
 *
 * So this client only lists: consumers cannot create, change or delete a
 * Material (the gateway may still advertise mutations for the Kind, but kcp
 * rejects them). Material is cluster-scoped - there is no namespace.
 *
 * The CRD mirrors a real material characterization file 1:1
 * (docs/assets/EastmanPlastics1000.json, apis/skz/v1alpha1/material_types.go).
 *
 * NOTE: the CRD Kind is "Material", not "MaterialParameters" - a Kind name
 * ending in "s" broke the Platform Mesh CRD Gateway's pluralization
 * (confirmed live: it generated no list/collection field at all, only a
 * singular `MaterialParameters(name, namespace)` getter).
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
  rawJson?: string;
}

export interface Material {
  metadata: {
    name: string;
  };
  spec?: MaterialSpec;
}

interface MaterialListResponse {
  genericmaterials_platform_mesh_io: {
    v1alpha1: {
      Materials: {
        items: Material[];
      };
    };
  };
}

const LIST_QUERY = `
  query ListMaterials {
    genericmaterials_platform_mesh_io {
      v1alpha1 {
        Materials {
          items {
            metadata { name }
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

@Injectable({ providedIn: 'root' })
export class SkzService {
  private graphql = inject(GraphQLClientService);

  list(): Observable<Material[]> {
    return this.graphql.post(LIST_QUERY, {}).pipe(
      map((response: { data: MaterialListResponse }) => {
        return response.data?.genericmaterials_platform_mesh_io?.v1alpha1?.Materials?.items || [];
      }),
      catchError((error) => {
        console.error('Error fetching materials:', error);
        return of([]);
      })
    );
  }
}
