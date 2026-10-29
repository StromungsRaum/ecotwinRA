/**
 * SKZ Materials - GraphQL API Client
 *
 * SKZ is a dummy/demo provider: no backend, no operator. HttpBin-style - the
 * portal's "+" button creates one Material card per material. The user only
 * types a name (e.g. "Luran 368R"); this client fills in canned placeholder
 * parameter values at creation time (DUMMY_PARAMETERS below) - there is no
 * real material characterization behind these numbers.
 *
 * NOTE: the CRD Kind is "Material", not "MaterialParameters" - a Kind name
 * ending in "s" broke the Platform Mesh CRD Gateway's pluralization
 * (confirmed live: it generated no list/collection field at all, only a
 * singular `MaterialParameters(name, namespace)` getter). Renamed before
 * wiring this up for real. The `spec.parameters` map[string]string field is
 * represented as the gateway's `StringMap_Input` scalar, not a generic JSON
 * type (also confirmed live via introspection - this gateway has no JSON
 * scalar at all).
 */
import { Injectable, inject } from '@angular/core';
import { Observable, of, catchError, map } from 'rxjs';
import { GraphQLClientService } from '../shared/graphql-client.service';

/**
 * Canned placeholder material parameters, loosely modeled on ianus-simod's
 * real Platform API `POST /api/platform/materials` concept (name + flat
 * parameter values) - not measured data, just illustrative numbers.
 */
const DUMMY_PARAMETERS: Record<string, string> = {
  density: '965.9',
  heatconductivity: '0.17',
  heatcapacity: '2334.8',
  meltTemperature: '250',
  ejectTemperature: '98',
};

export interface Material {
  metadata: {
    name: string;
    namespace?: string;
  };
  spec?: {
    name?: string;
    parameters?: Record<string, string>;
  };
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
            spec { name parameters }
          }
        }
      }
    }
  }
`;

const CREATE_MUTATION = `
  mutation CreateMaterial($name: String!, $namespace: String!, $materialName: String!, $parameters: StringMap_Input) {
    skz_platform_mesh_io {
      v1alpha1 {
        createMaterial(
          namespace: $namespace
          object: { metadata: { name: $name }, spec: { name: $materialName, parameters: $parameters } }
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

  /** Creates a new material card. Only the name comes from the user; the
   * parameter values are canned placeholders (see DUMMY_PARAMETERS). */
  create(crName: string, namespace: string, materialName: string): Observable<boolean> {
    return this.graphql
      .post(CREATE_MUTATION, { name: crName, namespace, materialName, parameters: DUMMY_PARAMETERS })
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
