import { withCsrfHeader } from "@trenova/shared/lib/api";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import type {
  GraphQLExecutableDocument,
  TypedGraphQLDocument,
} from "@trenova/shared/types/graphql";
import {
  type NormalizedApiError,
  type ProblemType,
  type ValidationError,
  apiProblem,
  parseFieldErrors,
  parseProblemType,
} from "@trenova/shared/types/errors";
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

// GraphQLRequestErrorKind separates a failure of the HTTP exchange itself from a
// resolver failure the server reported inside a 200 response body. Callers must be
// able to tell those apart without reading .status, because a resolver error and a
// gateway error can both surface with a status the caller would have to special-case.
export type GraphQLRequestErrorKind = "graphql" | "transport";

export type GraphQLResult<TData> = {
  data: TData;
  errors: NormalizedGraphQLError[];
};

type GraphQLRequestErrorOptions = {
  graphQLErrors?: NormalizedGraphQLError[];
  kind: GraphQLRequestErrorKind;
  message: string;
  status?: number;
};

export class GraphQLRequestError extends Error {
  public readonly code?: string;
  public readonly errors?: unknown;
  public readonly extensions?: GraphQLErrorExtensions;
  public readonly graphQLErrors: NormalizedGraphQLError[];
  public readonly kind: GraphQLRequestErrorKind;
  public readonly params?: unknown;
  public readonly status?: number;
  public readonly traceId?: string;
  public readonly type?: string;

  public constructor({ graphQLErrors = [], kind, message, status }: GraphQLRequestErrorOptions) {
    super(message);
    this.name = "GraphQLRequestError";
    this.kind = kind;
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

  public isTransportError(): boolean {
    return this.kind === "transport";
  }

  public isResolverError(): boolean {
    return this.kind === "graphql";
  }

  public normalize(): NormalizedApiError {
    const fieldErrors = this.getFieldErrors();
    // With a single field error its message IS the story. With several, picking the
    // first would headline one arbitrary field while the form shows all of them, so the
    // server's own summary is the honest choice.
    const detail = fieldErrors.length === 1 ? fieldErrors[0].message : this.message;
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

  // getProblemTypes covers every error in the response, not just the first. A
  // mutation that trips two resolvers can report a validation failure alongside a
  // database failure, and collapsing that to graphQLErrors[0] loses whichever one
  // happened to be serialised second.
  public getProblemTypes(): ProblemType[] {
    return dedupe(
      this.graphQLErrors
        .map((error) => parseProblemType(error.type))
        .filter((type): type is ProblemType => type !== null),
    );
  }

  public hasProblemType(problemType: ProblemType): boolean {
    return this.getProblemTypes().includes(problemType);
  }

  public getCodes(): string[] {
    return dedupe(this.graphQLErrors.map((error) => error.code).filter(isNonEmptyString));
  }

  public getTraceIds(): string[] {
    return dedupe(this.graphQLErrors.map((error) => error.traceId).filter(isNonEmptyString));
  }

  // getFieldErrors returns every field error in document order. Passing a field narrows
  // to that field's messages — two validators can reject the same input, and a caller
  // that wants to show both needs them all rather than whichever came first.
  public getFieldErrors(field?: string): ValidationError[] {
    const fieldErrors = collectFieldErrors(this.graphQLErrors);
    return field === undefined ? fieldErrors : fieldErrors.filter((e) => e.field === field);
  }

  public getFieldError(field: string): ValidationError | undefined {
    return this.getFieldErrors(field)[0];
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

// documentContainsMutation decides whether a response document is allowed to be a partial
// success.
//
// A query that resolves some fields and fails others is still useful — that is what
// GraphQL null-propagation is for. A mutation is not: a write that partially failed must
// not reach onSuccess, because TanStack would then run success handlers and
// useOptimisticMutation would commit the optimistic value instead of rolling it back. On
// billing, settlement and journal paths that turns a half-applied write into a UI that
// claims it saved.
//
// Scans at brace depth zero so a field or fragment that happens to be named "mutation"
// cannot be mistaken for an operation definition. Comments and block strings are removed
// first because either can contain the keyword.
export function documentContainsMutation(query: string): boolean {
  const source = query.replace(/"""[\s\S]*?"""/g, '""').replace(/#[^\n]*/g, "");

  let depth = 0;
  let index = 0;

  while (index < source.length) {
    const char = source[index];

    if (char === "{") {
      depth += 1;
      index += 1;
      continue;
    }
    if (char === "}") {
      depth -= 1;
      index += 1;
      continue;
    }
    if (depth > 0 || !/[A-Za-z_]/.test(char)) {
      index += 1;
      continue;
    }

    const word = /^[A-Za-z_]\w*/.exec(source.slice(index))?.[0] ?? "";
    if (word === "mutation") {
      return true;
    }
    index += word.length;
  }

  return false;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function stringExtension(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}

function isNonEmptyString(value: string | undefined): value is string {
  return typeof value === "string" && value.length > 0;
}

function dedupe<T>(values: T[]): T[] {
  return values.length > 1 ? [...new Set(values)] : values;
}

// collectFieldErrors flattens the per-error validation payloads in document order and
// drops exact duplicates, so a form binding to a field gets one message even when two
// resolvers rejected the same input.
function collectFieldErrors(graphQLErrors: NormalizedGraphQLError[]): ValidationError[] {
  if (graphQLErrors.length === 0) {
    return [];
  }

  const collected: ValidationError[] = [];
  const seen = new Set<string>();

  for (const graphQLError of graphQLErrors) {
    for (const fieldError of parseFieldErrors(graphQLError.errors)) {
      const key = `${fieldError.field} ${fieldError.message}`;
      if (seen.has(key)) {
        continue;
      }
      seen.add(key);
      collected.push(fieldError);
    }
  }

  return collected;
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

type SessionExpiryHandler = () => void;
type PartialErrorReporter = (
  errors: NormalizedGraphQLError[],
  context: { operationName?: string },
) => void;

let sessionExpiryHandler: SessionExpiryHandler | null = null;
let partialErrorReporter: PartialErrorReporter | null = null;

// setSessionExpiryHandler lets the app own what an expired session does — the transport
// sits below the router and the auth store, so it can only signal. Registered once at
// bootstrap; without it an authentication-error still throws, it just does not redirect.
export function setSessionExpiryHandler(handler: SessionExpiryHandler | null): void {
  sessionExpiryHandler = handler;
}

// setPartialErrorReporter receives the errors from responses that still carried usable
// data. Those no longer throw, so without somewhere to send them the field-level
// failures would vanish instead of merely being non-fatal.
export function setPartialErrorReporter(reporter: PartialErrorReporter | null): void {
  partialErrorReporter = reporter;
}

function isAuthenticationExpiry(graphQLErrors: NormalizedGraphQLError[], status: number): boolean {
  if (status === 401) {
    return true;
  }

  return graphQLErrors.some((error) => parseProblemType(error.type) === "authentication-error");
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

// requestGraphQLResult exposes the full response: the data and any errors that came
// back beside it. Prefer requestGraphQL unless the caller actually renders the
// partial-failure state, since this shape forces every caller to reach through .data.
export async function requestGraphQLResult<TDocument extends TypedGraphQLDocument<unknown, never>>(
  params: TypedGraphQLRequestParams<TDocument>,
): Promise<GraphQLResult<ResultOf<TDocument>>>;
export async function requestGraphQLResult<TData, TVariables = Record<string, unknown>>(
  params: GraphQLRequestParams<TVariables>,
): Promise<GraphQLResult<TData>>;
export async function requestGraphQLResult<TData, TVariables = Record<string, unknown>>({
  document,
  operationName,
  variables,
}: GraphQLRequestParams<TVariables>): Promise<GraphQLResult<TData>> {
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

  // An expired session has to reach the auth store from here. The router only
  // re-checks on navigation, so a long-lived screen would otherwise sit on a dead
  // session and surface nothing but a failed panel.
  if (isAuthenticationExpiry(graphQLErrors, response.status)) {
    sessionExpiryHandler?.();
  }

  // The HTTP exchange is checked first so a failed exchange is never mistaken for a
  // resolver failure. Any errors the body did carry stay attached — a 422 from query
  // validation still has its field detail, it is just classified as transport.
  if (!response.ok) {
    throw new GraphQLRequestError({
      graphQLErrors,
      kind: "transport",
      message: graphQLErrors[0]?.message ?? `GraphQL request failed with HTTP ${response.status}`,
      status: response.status,
    });
  }

  // Null-propagation means a resolver failure can still leave a usable document. For a
  // query, data present alongside errors is a partial success, so it is returned rather
  // than thrown and the errors ride along for the caller to render.
  //
  // A mutation gets no such latitude: a partially-failed write must surface as a failure
  // so success handlers do not run and optimistic updates roll back. The operation type is
  // read off the document rather than passed in, so the safe path cannot be forgotten at a
  // call site.
  if (payload.data != null) {
    if (graphQLErrors.length === 0) {
      return { data: payload.data, errors: graphQLErrors };
    }

    if (!documentContainsMutation(query)) {
      partialErrorReporter?.(graphQLErrors, { operationName: operationLabel });
      return { data: payload.data, errors: graphQLErrors };
    }

    throw new GraphQLRequestError({
      graphQLErrors,
      kind: "graphql",
      message: graphQLErrors[0].message,
      status: response.status,
    });
  }

  const firstError = graphQLErrors[0];
  if (firstError) {
    throw new GraphQLRequestError({
      graphQLErrors,
      kind: "graphql",
      message: firstError.message,
      status: response.status,
    });
  }

  throw new Error("GraphQL response did not include data");
}

export async function requestGraphQL<TDocument extends TypedGraphQLDocument<unknown, never>>(
  params: TypedGraphQLRequestParams<TDocument>,
): Promise<ResultOf<TDocument>>;
export async function requestGraphQL<TData, TVariables = Record<string, unknown>>(
  params: GraphQLRequestParams<TVariables>,
): Promise<TData>;
export async function requestGraphQL<TData, TVariables = Record<string, unknown>>(
  params: GraphQLRequestParams<TVariables>,
): Promise<TData> {
  const { data } = await requestGraphQLResult<TData, TVariables>(params);
  return data;
}
