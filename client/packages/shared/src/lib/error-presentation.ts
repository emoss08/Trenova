import { ApiRequestError } from "@trenova/shared/lib/api";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { isRecord } from "@trenova/shared/lib/utils";
import type { NormalizedApiError, ProblemType } from "@trenova/shared/types/errors";
import { RETRYABLE_PROBLEM_TYPES, apiProblem } from "@trenova/shared/types/errors";
import { type ErrorResponse, isRouteErrorResponse } from "react-router";

// ErrorKind is what the person looking at the screen can do about a failure, not where it
// came from. Two failures with different sources and the same remedy share a kind, so the
// screen that explains them cannot drift apart.
export type ErrorKind =
  | "offline"
  | "network"
  | "stale-build"
  | "session-expired"
  | "forbidden"
  | "not-found"
  | "rate-limited"
  | "timeout"
  | "server"
  | "request"
  | "unexpected";

export type ErrorTone = "neutral" | "info" | "warning" | "danger";

export type ErrorDescription = {
  kind: ErrorKind;
  tone: ErrorTone;
  // retryable says whether running the same request again can succeed without the person
  // changing anything first. It decides whether "Try again" is the primary action.
  retryable: boolean;
  name: string;
  message: string;
  // detail is the server's own explanation when it wrote one for people (a validation or
  // business-rule message, a missing permission). Transport and crash text never lands here.
  detail?: string;
  status?: number;
  problemType?: ProblemType;
  code?: string;
  traceId?: string;
  stack?: string;
};

export type StackFrame = {
  fn: string;
  file: string;
  line: string;
  column: string;
};

const NETWORK_FAILURE_PATTERN =
  /failed to fetch|networkerror|network request failed|load failed|network connection was lost|err_internet_disconnected|err_network_changed/i;

const STALE_BUILD_PATTERN =
  /failed to fetch dynamically imported module|error loading dynamically imported module|importing a module script failed|unable to preload css|chunkloaderror|loading (?:css )?chunk [\w-]+ failed/i;

const KIND_TONE: Record<ErrorKind, ErrorTone> = {
  offline: "warning",
  network: "warning",
  "stale-build": "info",
  "session-expired": "info",
  forbidden: "neutral",
  "not-found": "neutral",
  "rate-limited": "warning",
  timeout: "warning",
  server: "danger",
  request: "warning",
  unexpected: "danger",
};

const RETRYABLE_KINDS: ReadonlySet<ErrorKind> = new Set<ErrorKind>([
  "offline",
  "network",
  "rate-limited",
  "timeout",
  "server",
  "unexpected",
]);

function isBrowserOffline(): boolean {
  return typeof navigator !== "undefined" && navigator.onLine === false;
}

export function isNetworkFailure(error: unknown): boolean {
  return error instanceof TypeError && NETWORK_FAILURE_PATTERN.test(error.message);
}

export function isStaleBuildFailure(error: unknown): boolean {
  if (!(error instanceof Error)) {
    return false;
  }
  return error.name === "ChunkLoadError" || STALE_BUILD_PATTERN.test(error.message);
}

function kindFromProblem(problem: NormalizedApiError): ErrorKind {
  const { status, problemType } = problem;

  if (apiProblem.isAuthenticationError(problem) || status === 401) {
    return "session-expired";
  }
  if (apiProblem.isAuthorizationError(problem) || status === 403) {
    return "forbidden";
  }
  if (apiProblem.isNotFoundError(problem) || status === 404) {
    return "not-found";
  }
  if (apiProblem.isRateLimitError(problem)) {
    return "rate-limited";
  }
  if (apiProblem.isTimeoutError(problem) || status === 408) {
    return "timeout";
  }
  if ((problemType && RETRYABLE_PROBLEM_TYPES.includes(problemType)) || status >= 500) {
    return "server";
  }
  if (problemType || (status >= 400 && status < 500)) {
    return "request";
  }
  return "unexpected";
}

function kindFromStatus(status: number): ErrorKind {
  return kindFromProblem({ status, problemType: null, fieldErrors: [], message: "" });
}

// A server message is shown to people only when the server wrote it for them. Anything the
// server classifies as its own fault carries internals (a SQL state, a panic) and stays in
// the technical details instead.
function exposesServerDetail(kind: ErrorKind): boolean {
  return kind === "request" || kind === "forbidden" || kind === "not-found";
}

function routeErrorDetail(error: ErrorResponse): string | undefined {
  if (typeof error.data === "string" && error.data.trim() !== "") {
    return error.data.trim();
  }
  if (isRecord(error.data) && typeof error.data.message === "string") {
    return error.data.message;
  }
  return error.statusText || undefined;
}

function describe(
  kind: ErrorKind,
  fields: Omit<ErrorDescription, "kind" | "tone" | "retryable">,
): ErrorDescription {
  return {
    kind,
    tone: KIND_TONE[kind],
    retryable: RETRYABLE_KINDS.has(kind),
    ...fields,
  };
}

function describeApiError(
  error: ApiRequestError | GraphQLRequestError,
  problem: NormalizedApiError,
): ErrorDescription {
  const kind = kindFromProblem(problem);
  const code = error instanceof GraphQLRequestError ? error.getCodes()[0] : undefined;

  return describe(kind, {
    name: error.name,
    message: error.message,
    detail: exposesServerDetail(kind) ? problem.message : undefined,
    status: problem.status || undefined,
    problemType: problem.problemType ?? undefined,
    code,
    traceId: problem.traceId,
    stack: error.stack,
  });
}

// describeError turns anything thrown — a fetch TypeError, an API problem, a GraphQL
// response, a route Response, a render crash, a non-Error value — into one description a
// screen can present. It never throws.
export function describeError(error: unknown): ErrorDescription {
  if (isStaleBuildFailure(error)) {
    const err = error as Error;
    return describe("stale-build", { name: err.name, message: err.message, stack: err.stack });
  }

  if (isNetworkFailure(error)) {
    const err = error as Error;
    return describe(isBrowserOffline() ? "offline" : "network", {
      name: err.name,
      message: err.message,
      stack: err.stack,
    });
  }

  if (error instanceof ApiRequestError || error instanceof GraphQLRequestError) {
    return describeApiError(error, error.normalize());
  }

  if (isRouteErrorResponse(error)) {
    const kind = kindFromStatus(error.status);
    const detail = routeErrorDetail(error);
    return describe(kind, {
      name: "RouteError",
      message: detail ?? `${error.status}`,
      detail: exposesServerDetail(kind) ? detail : undefined,
      status: error.status,
    });
  }

  if (error instanceof Error) {
    return describe(isBrowserOffline() ? "offline" : "unexpected", {
      name: error.name || "Error",
      message: error.message,
      stack: error.stack,
    });
  }

  return describe("unexpected", {
    name: "Error",
    message: typeof error === "string" ? error : String(error),
  });
}

const CHROMIUM_FRAME = /^\s*at\s+(?:(.+?)\s+\()?(.+?):(\d+):(\d+)\)?\s*$/;
const GECKO_FRAME = /^\s*(.*?)@(.+?):(\d+):(\d+)\s*$/;
const DEV_SERVER_PREFIX = /^https?:\/\/[^/]+\/(?:@fs\/)?/;
const CACHE_BUST_SUFFIX = /\?(?:t|v)=[\w.]+$/;

// shortenSourcePath drops the dev server origin and Vite's cache-busting query, so a frame
// reads as the file a developer would open rather than a URL.
export function shortenSourcePath(file: string): string {
  const withoutOrigin = file.replace(DEV_SERVER_PREFIX, "").replace(CACHE_BUST_SUFFIX, "");
  const clientIndex = withoutOrigin.search(/(?:^|\/)(?:client|src)\//);
  return clientIndex > 0 ? withoutOrigin.slice(clientIndex + 1) : withoutOrigin;
}

export function parseStackFrames(stack: string | undefined): StackFrame[] {
  if (!stack) {
    return [];
  }

  const frames: StackFrame[] = [];
  for (const line of stack.split("\n")) {
    const match = CHROMIUM_FRAME.exec(line) ?? GECKO_FRAME.exec(line);
    if (!match) {
      continue;
    }
    frames.push({
      fn: match[1]?.trim() || "anonymous",
      file: shortenSourcePath(match[2]),
      line: match[3],
      column: match[4],
    });
  }
  return frames;
}

export type ErrorReportContext = {
  title: string;
  occurredAt: Date;
  location?: string;
  componentStack?: string | null;
};

// formatErrorReport is the text a person pastes into a support ticket. It leads with what
// support searches by (the reference and the time) and ends with what engineering reads.
export function formatErrorReport(
  description: ErrorDescription,
  context: ErrorReportContext,
): string {
  const lines = [
    context.title,
    "",
    `Kind: ${description.kind}`,
    `Occurred: ${context.occurredAt.toISOString()}`,
  ];

  if (description.traceId) {
    lines.push(`Reference: ${description.traceId}`);
  }
  if (description.status) {
    lines.push(`Status: ${description.status}`);
  }
  if (description.problemType) {
    lines.push(`Problem: ${description.problemType}`);
  }
  if (description.code) {
    lines.push(`Code: ${description.code}`);
  }
  if (context.location) {
    lines.push(`Page: ${context.location}`);
  }
  if (typeof navigator !== "undefined") {
    lines.push(`Browser: ${navigator.userAgent}`);
  }

  lines.push("", `${description.name}: ${description.message}`);

  if (description.stack) {
    lines.push("", "Stack trace:", description.stack);
  }
  if (context.componentStack) {
    lines.push("", "Component stack:", context.componentStack.trim());
  }

  return lines.join("\n");
}
