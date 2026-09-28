import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import { clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { vi } from "vitest";

// Fixtures follow the wire contract: the operations in
// packages/graphql/src/operations/capture/capture.graphql, and errors shaped the
// way services/tms/internal/api/graphql/errors.go presents them — a problem type
// URI under the configured base, the errortypes code, and a trace id.

export type GraphQLBody = { operationName: string; variables: Record<string, unknown> };

type Handler = (body: GraphQLBody) => Response | Promise<Response>;

const PROBLEM_BASE = "https://api.trenova.app/problems/";

export function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

export function resolverError(
  field: string,
  problemType: string,
  code: string,
  message: string,
  fieldErrors?: { field: string; code: string; message: string }[],
): Response {
  return json({
    data: null,
    errors: [
      {
        message,
        path: [field],
        extensions: {
          code,
          type: `${PROBLEM_BASE}${problemType}`,
          traceId: "req_test",
          ...(fieldErrors ? { errors: fieldErrors } : {}),
        },
      },
    ],
  });
}

/**
 * Answers each GraphQL operation by name. A handler may be a list, used in
 * order and then repeating its last entry, so a test can fail a request once
 * and succeed on the retry.
 */
export function stubGraphQL(handlers: Partial<Record<string, Handler | Handler[]>>) {
  setCsrfToken("token");
  const bodies: GraphQLBody[] = [];
  const calls = new Map<string, number>();
  const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
    const body = JSON.parse(init?.body as string) as GraphQLBody;
    bodies.push(body);
    const entry = handlers[body.operationName];
    if (entry === undefined) {
      throw new Error(`no handler for ${body.operationName}`);
    }
    const count = calls.get(body.operationName) ?? 0;
    calls.set(body.operationName, count + 1);
    const handler = Array.isArray(entry) ? entry[Math.min(count, entry.length - 1)] : entry;
    return handler(body);
  });
  vi.stubGlobal("fetch", fetchMock);
  return bodies;
}

export function resetGraphQL() {
  clearCsrfToken();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
}

/** What fetch itself rejects with when the connection never produces a response. */
export function networkFailure(): Promise<Response> {
  return Promise.reject(new TypeError("Failed to fetch"));
}

export function captureDevice(overrides: Record<string, unknown> = {}) {
  return {
    id: "cdev_office",
    userId: "usr_1",
    name: "Office",
    machineName: "OFFICE-PC",
    windowsUser: "jdoe",
    agentVersion: "1.4.0",
    architecture: "X64",
    osVersion: "Windows 10.0.22631",
    status: "Active",
    lastSeenAt: null,
    isOnline: false,
    lastIp: null,
    revokedAt: null,
    revokedReason: "",
    version: 1,
    createdAt: 1_700_000_000,
    sources: [],
    ...overrides,
  };
}

export function renderWithClient(ui: ReactElement, { route = "/" }: { route?: string } = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[route]}>{ui}</MemoryRouter>
    </QueryClientProvider>,
  );
}
