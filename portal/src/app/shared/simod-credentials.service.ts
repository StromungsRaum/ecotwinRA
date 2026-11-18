/**
 * SIMOD Credentials - shared Secret upsert
 *
 * Every provider operator that talks to the SIMOD/StrömungsRaum backend
 * (injection-molding, materials) reads the same namespace-local Secret -
 * SIMOD_SECRET_NAME, via each CR's spec.secretRefs[0] (see
 * pkg/simod/auth.go's LoginFromSecret). This service lets the portal set
 * that Secret itself instead of requiring an admin to kubectl-create it:
 * one dialog (SimodCredentialsDialogComponent), reused on every page that
 * can hit a missing-credentials failure, upserts it via the CRD Gateway's
 * generic `applyYaml` mutation (create-or-update semantics, see ApplyYaml
 * in kubernetes-graphql-gateway's resolver package) - so submitting again
 * later just rotates the password instead of failing on "already exists",
 * and the Secret itself persists independently of any CR, surviving a
 * disable/re-enable of any provider.
 */
import { Injectable, inject } from '@angular/core';
import { Observable, catchError, map, of } from 'rxjs';
import { GraphQLClientService } from './graphql-client.service';

export const SIMOD_SECRET_NAME = 'simod-credentials';
export const SIMOD_SECRET_NAMESPACE = 'default';
export const DEFAULT_SIMOD_BASE_URL = 'https://backend.dev.simod.de';

export interface SimodCredentialInfo {
  email: string;
  /** Always set - DEFAULT_SIMOD_BASE_URL if the Secret doesn't override it. */
  baseUrl: string;
}

/**
 * pkg/simod's CredentialsError/AuthRejectedError.Error() always start with
 * one of these exact prefixes - matching on them is how the UI tells "the
 * credentials themselves are the problem" (missing Secret, or a Secret the
 * backend rejected) apart from a generic backend/network failure, without a
 * dedicated status.reason field. Safe to rely on: we own both ends.
 */
const MISSING_CREDENTIALS_PREFIX = 'SIMOD credentials secret';
const AUTH_REJECTED_PREFIX = 'SIMOD login rejected by backend';

export function isMissingCredentialsMessage(message: string | undefined | null): boolean {
  return !!message?.startsWith(MISSING_CREDENTIALS_PREFIX);
}

export function isAuthRejectedMessage(message: string | undefined | null): boolean {
  return !!message?.startsWith(AUTH_REJECTED_PREFIX);
}

/** True for either credentials-class failure - mirrors pkg/simod.NeedsCredentialsFix. */
export function needsCredentialsFixMessage(message: string | undefined | null): boolean {
  return isMissingCredentialsMessage(message) || isAuthRejectedMessage(message);
}

const APPLY_YAML_MUTATION = `
  mutation ApplySimodCredentials($yaml: String!) {
    applyYaml(yaml: $yaml)
  }
`;

/**
 * Core k8s resources sit at the GraphQL schema root under their version
 * (e.g. `v1`), not nested under a group name like every CRD query in this
 * portal is - the gateway's schema generator treats the empty group ("core")
 * as root. No namespace arg on the generated list field (same as every
 * other `{Kind}s { items }` query in this codebase) - filter client-side.
 */
// `data` on a core Secret is the standard k8s API shape: a map of key to its
// base64-encoded value, serialized as a JSON object of strings - decode with
// atob(), the same as `kubectl get secret -o jsonpath` + `base64 -d` does.
const LIST_SECRETS_QUERY = `
  query ListSimodCredentialSecrets {
    v1 {
      Secrets {
        items { metadata { name namespace } data }
      }
    }
  }
`;

@Injectable({ providedIn: 'root' })
export class SimodCredentialsService {
  private graphql = inject(GraphQLClientService);

  private findSecret(): Observable<Record<string, string> | null> {
    return this.graphql.post(LIST_SECRETS_QUERY, {}).pipe(
      map((response: any) => {
        const items = response?.data?.v1?.Secrets?.items || [];
        const secret = items.find(
          (s: any) => s?.metadata?.name === SIMOD_SECRET_NAME && (s?.metadata?.namespace || 'default') === SIMOD_SECRET_NAMESPACE
        );
        return secret?.data ?? null;
      }),
      catchError((error) => {
        console.error('Error reading SIMOD credentials:', error);
        return of(null);
      })
    );
  }

  /** Whether the shared SIMOD credentials Secret currently exists - never reads its data, just presence. */
  hasCredentials(): Observable<boolean> {
    return this.findSecret().pipe(map((data) => data !== null));
  }

  /**
   * The Secret's email and effective backend URL, to prefill the dialog -
   * never the password, which stays write-only. null if no Secret exists
   * yet (a fresh dialog then starts blank, baseUrl defaulting to
   * DEFAULT_SIMOD_BASE_URL).
   */
  getCredentialInfo(): Observable<SimodCredentialInfo | null> {
    return this.findSecret().pipe(
      map((data) => {
        if (!data) {
          return null;
        }
        return {
          email: data['email'] ? atob(data['email']) : '',
          baseUrl: data['baseUrl'] ? atob(data['baseUrl']) : DEFAULT_SIMOD_BASE_URL,
        };
      })
    );
  }

  /** Upserts the shared SIMOD credentials Secret. JSON.stringify produces a quoted YAML scalar, safe against any special characters in email/password/baseUrl. */
  setCredentials(email: string, password: string, baseUrl: string): Observable<boolean> {
    const yaml = [
      'apiVersion: v1',
      'kind: Secret',
      'metadata:',
      `  name: ${SIMOD_SECRET_NAME}`,
      `  namespace: ${SIMOD_SECRET_NAMESPACE}`,
      'type: Opaque',
      'stringData:',
      `  email: ${JSON.stringify(email)}`,
      `  password: ${JSON.stringify(password)}`,
      `  baseUrl: ${JSON.stringify(baseUrl || DEFAULT_SIMOD_BASE_URL)}`,
      '',
    ].join('\n');

    return this.graphql.post(APPLY_YAML_MUTATION, { yaml }).pipe(
      map((response: any) => !response?.errors?.length),
      catchError((error) => {
        console.error('Error saving SIMOD credentials:', error);
        return of(false);
      })
    );
  }
}
