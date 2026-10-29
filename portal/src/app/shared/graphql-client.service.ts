/**
 * Shared GraphQL Client Base - Luigi context -> CRD Gateway -> fetch
 *
 * Every provider page in this portal talks to the Platform Mesh CRD Gateway
 * the same way: pull the bearer token and GraphQL endpoint out of the Luigi
 * context, then POST a query/mutation with an Authorization header. This was
 * duplicated once already (injection-molding.service.ts); a second provider
 * (skz) duplicating it a third time is the point where sharing it stops being
 * premature.
 */
import { Injectable, inject } from '@angular/core';
import { LuigiContextService } from '@luigi-project/client-support-angular';
import { from, map, Observable, switchMap, filter } from 'rxjs';

interface GraphQLConfig {
  endpoint: string;
  token: string | null;
}

@Injectable({ providedIn: 'root' })
export class GraphQLClientService {
  private luigiContextService = inject(LuigiContextService);

  private getGraphQLConfig(): Observable<GraphQLConfig> {
    return this.luigiContextService.contextObservable().pipe(
      filter((ctx) => !!ctx?.context),
      map((ctx) => {
        const context = ctx.context as any;
        const token = context.token || null;
        let endpoint = context.portalContext?.crdGatewayApiUrl;
        if (!endpoint) {
          console.warn('crdGatewayApiUrl not found in context, falling back to default');
          endpoint = context.portalBaseUrl + '/graphql';
        }
        return { endpoint, token };
      })
    );
  }

  private buildHeaders(token: string | null): Record<string, string> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
    return headers;
  }

  /** Sends a GraphQL query/mutation and returns the raw `{data, errors?}` response. */
  post(query: string, variables: Record<string, unknown>): Observable<any> {
    return this.getGraphQLConfig().pipe(
      switchMap(({ endpoint, token }) =>
        from(
          fetch(endpoint, {
            method: 'POST',
            headers: this.buildHeaders(token),
            body: JSON.stringify({ query, variables }),
          }).then((res) => res.json())
        )
      )
    );
  }
}
