import { withCsrfHeader } from "@/lib/api";
import { API_BASE_URL } from "@/lib/constants";
import type { GraphQLExecutableDocument, TypedGraphQLDocument } from "@/types/graphql";
import {
  type NormalizedApiError,
  type ProblemType,
  type ValidationError,
  apiProblem,
  parseFieldErrors,
  parseProblemType,
} from "@/types/errors";
import type { ResultOf, VariablesOf } from "@graphql-typed-document-node/core";

type GraphQLErrorResponse = {
  message?: unknown;
  extensions?: unknown;
  locations?: unknown;
  path?: unknown;
};

type GraphQLResponse<TData> = {
  data?: TData;
  errors?: GraphQLErrorResponse[];
};

type GraphQLRequestBody<TVariables> = {
  extensions?: {
    persistedQuery: {
      sha256Hash: string;
      version: 1;
    };
  };
  operationName?: string;
  query: string;
  variables?: TVariables;
};

type GraphQLRequestParams<TVariables = Record<string, unknown>> = {
  document: GraphQLExecutableDocument;
  operationName?: string;
  variables?: TVariables;
};

type TypedGraphQLRequestParams<TDocument extends TypedGraphQLDocument<unknown, never>> = {
  document: TDocument;
  operationName?: string;
  variables?: VariablesOf<TDocument>;
};

export type GraphQLErrorExtensions = Record<string, unknown> & {
  code?: string;
  errors?: unknown;
  params?: unknown;
  traceId?: string;
  type?: string;
};

export type NormalizedGraphQLError = {
  code?: string;
  errors?: unknown;
  extensions: GraphQLErrorExtensions;
  locations?: unknown;
  message: string;
  params?: unknown;
  path?: unknown;
  traceId?: string;
  type?: string;
};

type GraphQLRequestErrorOptions = {
  graphQLErrors?: NormalizedGraphQLError[];
  message: string;
  status?: number;
};

export class GraphQLRequestError extends Error {
  public readonly code?: string;
  public readonly errors?: unknown;
  public readonly extensions?: GraphQLErrorExtensions;
  public readonly graphQLErrors: NormalizedGraphQLError[];
  public readonly params?: unknown;
  public readonly status?: number;
  public readonly traceId?: string;
  public readonly type?: string;

  public constructor({ graphQLErrors = [], message, status }: GraphQLRequestErrorOptions) {
    super(message);
    this.name = "GraphQLRequestError";
    this.status = status;
    this.graphQLErrors = graphQLErrors;

    const firstError = graphQLErrors[0];
    if (firstError) {
      this.extensions = firstError.extensions;
      this.code = firstError.code;
      this.type = firstError.type;
      this.traceId = firstError.traceId;
      this.params = firstError.params;
      this.errors = firstError.errors;
    }
  }

  public normalize(): NormalizedApiError {
    const fieldErrors = this.getFieldErrors();
    const detail = fieldErrors[0]?.message ?? this.message;
    return {
      problemType: this.getProblemType(),
      status: this.status ?? 0,
      fieldErrors,
      message: detail,
      detail,
      traceId: this.traceId,
    };
  }

  public getProblemType(): ProblemType | null {
    return parseProblemType(this.type);
  }

  public getFieldErrors(): ValidationError[] {
    return parseFieldErrors(this.errors);
  }

  public getFieldError(field: string): ValidationError | undefined {
    return this.getFieldErrors().find((e) => e.field === field);
  }

  public isValidationError(): boolean {
    return apiProblem.isValidationError(this.normalize());
  }

  public isBusinessError(): boolean {
    return apiProblem.isBusinessError(this.normalize());
  }

  public isAuthenticationError(): boolean {
    return apiProblem.isAuthenticationError(this.normalize());
  }

  public isAuthorizationError(): boolean {
    return apiProblem.isAuthorizationError(this.normalize());
  }

  public isNotFoundError(): boolean {
    return apiProblem.isNotFoundError(this.normalize());
  }

  public isRateLimitError(): boolean {
    return apiProblem.isRateLimitError(this.normalize());
  }

  public isVersionMismatchError(): boolean {
    return apiProblem.isVersionMismatchError(this.normalize());
  }

  public isConflictError(): boolean {
    return apiProblem.isConflictError(this.normalize());
  }

  public isTimeoutError(): boolean {
    return apiProblem.isTimeoutError(this.normalize());
  }
}

export function resolveGraphQLURL(apiBaseURL = API_BASE_URL, operationName?: string): string {
  const suffix = operationName ? `?op=${encodeURIComponent(operationName)}` : "";

  if (!apiBaseURL.startsWith("http")) {
    return `/graphql${suffix}`;
  }

  const url = new URL(apiBaseURL);
  url.pathname = "/graphql";
  url.search = "";
  url.hash = "";

  return `${url.toString()}${suffix}`;
}

const operationNamePattern = /\b(?:query|mutation|subscription)\s+(\w+)/;

// extractOperationName reads the operation name from a GraphQL document string so the
// request URL can be labelled per-operation (e.g. /graphql?op=ShipmentDetail) instead of the
// opaque /graphql that persisted-query POSTs would otherwise show in the network tab.
export function extractOperationName(query: string): string | undefined {
  return operationNamePattern.exec(query)?.[1];
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function stringExtension(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}

function normalizeGraphQLError(error: GraphQLErrorResponse): NormalizedGraphQLError {
  const extensions: GraphQLErrorExtensions = isRecord(error.extensions)
    ? { ...error.extensions }
    : {};

  return {
    code: stringExtension(extensions.code),
    errors: extensions.errors,
    extensions,
    locations: error.locations,
    message: typeof error.message === "string" ? error.message : "GraphQL request failed",
    params: extensions.params,
    path: error.path,
    traceId: stringExtension(extensions.traceId),
    type: stringExtension(extensions.type),
  };
}

// graphQLErrorMessage extracts the server's field-level or top-level error detail for
// inline mutation toasts, falling back to a caller-provided message.
export function graphQLErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof GraphQLRequestError) {
    const detail = error.normalize().detail;
    if (detail && detail !== "GraphQL request failed") {
      return detail;
    }
    return fallback;
  }
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return fallback;
}

export async function requestGraphQL<
  TDocument extends TypedGraphQLDocument<unknown, never>,
>(
  params: TypedGraphQLRequestParams<TDocument>,
): Promise<ResultOf<TDocument>>;
export async function requestGraphQL<
  TData,
  TVariables = Record<string, unknown>,
>(params: GraphQLRequestParams<TVariables>): Promise<TData>;
export async function requestGraphQL<
  TData,
  TVariables = Record<string, unknown>,
>({
  document,
  operationName,
  variables,
}: GraphQLRequestParams<TVariables>): Promise<TData> {
  const query = document.toString();
  const body: GraphQLRequestBody<TVariables> = {
    query,
    operationName,
    variables,
  };
  const persistedHash = typeof document === "string" ? undefined : document.__meta__?.hash;
  if (persistedHash) {
    body.extensions = {
      persistedQuery: {
        version: 1,
        sha256Hash: persistedHash,
      },
    };
  }

  const operationLabel = operationName ?? extractOperationName(query);
  const response = await fetch(resolveGraphQLURL(API_BASE_URL, operationLabel), {
    body: JSON.stringify(body),
    credentials: "include",
    headers: await withCsrfHeader("POST", { "Content-Type": "application/json" }, "/graphql"),
    method: "POST",
  });

  const payload = (await response.json().catch(() => ({}))) as GraphQLResponse<TData>;
  const graphQLErrors = payload.errors?.map(normalizeGraphQLError) ?? [];
  const firstError = graphQLErrors[0];
  if (firstError) {
    throw new GraphQLRequestError({
      graphQLErrors,
      message: firstError.message,
      status: response.status,
    });
  }

  if (!response.ok) {
    throw new GraphQLRequestError({
      message: `GraphQL request failed with HTTP ${response.status}`,
      status: response.status,
    });
  }

  if (!payload.data) {
    throw new Error("GraphQL response did not include data");
  }

  return payload.data;
}
