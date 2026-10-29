/**
 * Injection Molding Service - GraphQL API Client
 *
 * HttpBin-style: zero resources by default, the portal's "+" button creates
 * one InjectionMoldingService per card. Each card monitors at most one SIMOD
 * Simulation (spec.simulationId), picked from a Refresh-populated dropdown
 * (status.availableSimulations), polled server-side by the operator every 5
 * seconds (operator/injectionmolding/injectionmoldingservice_controller.go)
 * until it completes.
 *
 * NOTE: createInjectionMoldingService/updateInjectionMoldingService follow
 * the same create{Kind}/delete{Kind} naming convention models.service.ts's
 * (live, documented) createModel/deleteModel use - but an update mutation is
 * not documented anywhere in this repo or independently confirmed against a
 * live gateway schema. Verify it in a real Luigi session.
 *
 * SECRET_NAME: every new card references a Secret named "simod-credentials"
 * in the same namespace (email/password keys) - this must already exist in
 * the account (same SecretRef pattern as the SIMOD CRDs), it is not created
 * by this UI.
 */
import { Injectable, inject } from '@angular/core';
import { Observable, of, catchError, map } from 'rxjs';
import { GraphQLClientService } from '../shared/graphql-client.service';

const SECRET_NAME = 'simod-credentials';

export interface SimulationSummary {
  id: string;
  name: string;
  status: string;
}

export interface MonitoredSimulationStatus {
  id: string;
  name: string;
  status: string;
  complete: boolean;
  resultsUrl?: string;
  lastPolledAt?: string;
}

export interface InjectionMoldingService {
  metadata: {
    name: string;
    namespace?: string;
  };
  spec?: {
    refreshRequestedAt?: string;
    simulationId?: string;
  };
  status?: {
    phase?: string;
    url?: string;
    availableSimulations?: SimulationSummary[];
    lastRefreshedAt?: string;
    monitoredSimulation?: MonitoredSimulationStatus;
    message?: string;
  };
}

interface InjectionMoldingServiceListResponse {
  injectionmolding_platform_mesh_io: {
    v1alpha1: {
      InjectionMoldingServices: {
        items: InjectionMoldingService[];
      };
    };
  };
}

const LIST_QUERY = `
  query ListInjectionMoldingServices {
    injectionmolding_platform_mesh_io {
      v1alpha1 {
        InjectionMoldingServices {
          items {
            metadata { name namespace }
            spec { refreshRequestedAt simulationId }
            status {
              phase
              url
              availableSimulations { id name status }
              lastRefreshedAt
              monitoredSimulation { id name status complete resultsUrl lastPolledAt }
              message
            }
          }
        }
      }
    }
  }
`;

const CREATE_MUTATION = `
  mutation CreateInjectionMoldingService($name: String!, $namespace: String!, $secretName: String!) {
    injectionmolding_platform_mesh_io {
      v1alpha1 {
        createInjectionMoldingService(
          namespace: $namespace
          object: { metadata: { name: $name }, spec: { secretRefs: [{ name: $secretName }] } }
        ) {
          metadata { name namespace }
        }
      }
    }
  }
`;

const UPDATE_REFRESH_MUTATION = `
  mutation RequestRefresh($name: String!, $namespace: String!, $refreshRequestedAt: String!) {
    injectionmolding_platform_mesh_io {
      v1alpha1 {
        updateInjectionMoldingService(
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

const DELETE_MUTATION = `
  mutation DeleteInjectionMoldingService($name: String!, $namespace: String!) {
    injectionmolding_platform_mesh_io {
      v1alpha1 {
        deleteInjectionMoldingService(name: $name, namespace: $namespace)
      }
    }
  }
`;

const UPDATE_SIMULATION_MUTATION = `
  mutation SelectSimulation($name: String!, $namespace: String!, $simulationId: String!) {
    injectionmolding_platform_mesh_io {
      v1alpha1 {
        updateInjectionMoldingService(
          name: $name
          namespace: $namespace
          object: { spec: { simulationId: $simulationId } }
        ) {
          metadata { name namespace }
        }
      }
    }
  }
`;

@Injectable({ providedIn: 'root' })
export class InjectionMoldingServiceClient {
  private graphql = inject(GraphQLClientService);
  private post = (query: string, variables: Record<string, unknown>) => this.graphql.post(query, variables);

  list(): Observable<InjectionMoldingService[]> {
    return this.post(LIST_QUERY, {}).pipe(
      map((response: { data: InjectionMoldingServiceListResponse }) => {
        return (
          response.data?.injectionmolding_platform_mesh_io?.v1alpha1?.InjectionMoldingServices
            ?.items || []
        );
      }),
      catchError((error) => {
        console.error('Error fetching injection molding services:', error);
        return of([]);
      })
    );
  }

  /** Creates a new card, referencing the account's "simod-credentials" Secret. */
  create(name: string, namespace: string): Observable<boolean> {
    return this.post(CREATE_MUTATION, { name, namespace, secretName: SECRET_NAME }).pipe(
      map((response: any) => !!response.data?.injectionmolding_platform_mesh_io?.v1alpha1?.createInjectionMoldingService),
      catchError((error) => {
        console.error('Error creating injection molding service:', error);
        return of(false);
      })
    );
  }

  /** Bumps spec.refreshRequestedAt to re-fetch status.availableSimulations. */
  requestRefresh(name: string, namespace: string): Observable<boolean> {
    return this.post(UPDATE_REFRESH_MUTATION, {
      name,
      namespace,
      refreshRequestedAt: new Date().toISOString(),
    }).pipe(
      map((response: any) => !!response.data?.injectionmolding_platform_mesh_io?.v1alpha1?.updateInjectionMoldingService),
      catchError((error) => {
        console.error('Error requesting injection molding service refresh:', error);
        return of(false);
      })
    );
  }

  /** Deletes a card. */
  delete(name: string, namespace: string): Observable<boolean> {
    return this.post(DELETE_MUTATION, { name, namespace }).pipe(
      map((response: any) => !!response.data?.injectionmolding_platform_mesh_io?.v1alpha1?.deleteInjectionMoldingService),
      catchError((error) => {
        console.error('Error deleting injection molding service:', error);
        return of(false);
      })
    );
  }

  /** Sets spec.simulationId, starting the operator's 5-second status poll. */
  selectSimulation(name: string, namespace: string, simulationId: string): Observable<boolean> {
    return this.post(UPDATE_SIMULATION_MUTATION, { name, namespace, simulationId }).pipe(
      map((response: any) => !!response.data?.injectionmolding_platform_mesh_io?.v1alpha1?.updateInjectionMoldingService),
      catchError((error) => {
        console.error('Error selecting simulation to monitor:', error);
        return of(false);
      })
    );
  }
}
