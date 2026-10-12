import { queryClient } from "@/lib/query-client";
import { routes } from "@/router";
import { publicConfigQueryOptions } from "@trenova/shared/hooks/use-public-config";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { SELF_HOSTED_PUBLIC_CONFIG } from "@trenova/shared/types/platform";
import { Resource, type PermissionManifest } from "@trenova/shared/types/permission";
import { userSchema, type User } from "@trenova/shared/types/user";
import { createMemoryRouter, type LoaderFunctionArgs, type RouteObject } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// The session check is GET /users/me/. Every failure the server sends is an RFC 9457
// problem document; the rate limiter's carries Retry-After.
function problemResponse(status: number, problem: string, title: string): Response {
  return new Response(
    JSON.stringify({
      type: `https://api.trenova.app/problems/${problem}`,
      title,
      status,
      detail: title,
    }),
    {
      status,
      headers: {
        "Content-Type": "application/problem+json",
        ...(status === 429 ? { "Retry-After": "7" } : {}),
      },
    },
  );
}

function answerSessionCheckWith(outcome: Response | Error) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = new URL(input instanceof Request ? input.url : String(input), "http://localhost");
      if (!url.pathname.endsWith("/users/me/")) {
        throw new Error(`unexpected request to ${url.pathname}`);
      }
      if (outcome instanceof Error) {
        throw outcome;
      }
      return outcome;
    }),
  );
}

// The pathless layout every signed-in page sits under: the one that guards them all.
function findSignedInLayout(entries: RouteObject[]): RouteObject | undefined {
  for (const entry of entries) {
    if (entry.children?.some((child) => child.path === "/")) {
      return entry;
    }
    if (entry.children) {
      const found = findSignedInLayout(entry.children);
      if (found) {
        return found;
      }
    }
  }
  return undefined;
}

async function runProtectedLoader(): Promise<{ value: unknown; thrown: unknown }> {
  const layout = findSignedInLayout(routes);
  if (typeof layout?.loader !== "function") {
    throw new Error("the signed-in layout has no loader");
  }
  return Promise.resolve(
    layout.loader({
      request: new Request("http://localhost/shipments/management"),
      params: {},
      context: undefined,
    } as unknown as LoaderFunctionArgs),
  ).then(
    (value) => ({ value, thrown: undefined }),
    (thrown: unknown) => ({ value: undefined, thrown }),
  );
}

function midSession() {
  useAuthStore.setState({
    user: { id: "usr_01", currentOrganizationId: "org_01", memberships: [] } as unknown as User,
    isAuthenticated: true,
  });
}

// The store persists only the user, so a full page load starts with isAuthenticated false
// even for somebody who is signed in.
function coldLoad() {
  useAuthStore.setState({
    user: { id: "usr_01", currentOrganizationId: "org_01", memberships: [] } as unknown as User,
    isAuthenticated: false,
  });
}

describe("protectedLoader", () => {
  beforeEach(() => {
    queryClient.setQueryData(publicConfigQueryOptions.queryKey, SELF_HOSTED_PUBLIC_CONFIG);
    vi.spyOn(usePermissionStore.getState(), "clearPermissions").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    queryClient.clear();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  // Reproduced live: fast keyboard navigation exhausted the per-user rate limit and the
  // next full page load's session check got a 429, which sent a signed-in person to /login.
  it("does not send a rate-limited cold load to the login form", async () => {
    coldLoad();
    answerSessionCheckWith(problemResponse(429, "rate-limit-exceeded", "Rate Limit Exceeded"));

    const { value, thrown } = await runProtectedLoader();

    expect(value).toBeUndefined();
    expect(thrown).toBeInstanceOf(ApiRequestError);
    expect((thrown as ApiRequestError).status).toBe(429);
    expect((thrown as ApiRequestError).retryAfter).toBe(7);
  });

  it("surfaces a cold-load network failure instead of redirecting", async () => {
    coldLoad();
    const failure = new TypeError("Failed to fetch");
    answerSessionCheckWith(failure);

    const { value, thrown } = await runProtectedLoader();

    expect(value).toBeUndefined();
    expect(thrown).toBe(failure);
  });

  it("lets a rate-limited check through mid-session", async () => {
    midSession();
    answerSessionCheckWith(problemResponse(429, "rate-limit-exceeded", "Rate Limit Exceeded"));

    const { value, thrown } = await runProtectedLoader();

    expect(thrown).toBeUndefined();
    expect(value).toBeNull();
    expect(useAuthStore.getState().isAuthenticated).toBe(true);
  });

  // A 503 during a deploy comes from the proxy in front of the API, not the API: plain
  // text, no problem document.
  it("lets a proxy's 503 through mid-session", async () => {
    midSession();
    answerSessionCheckWith(
      new Response("Service Unavailable", {
        status: 503,
        headers: { "Content-Type": "text/plain" },
      }),
    );

    const { value, thrown } = await runProtectedLoader();

    expect(thrown).toBeUndefined();
    expect(value).toBeNull();
  });

  it("still sends an expired session to /login", async () => {
    midSession();
    answerSessionCheckWith(problemResponse(401, "authentication-error", "Authentication Failed"));

    const { value, thrown } = await runProtectedLoader();

    expect(thrown).toBeUndefined();
    expect(value).toBeInstanceOf(Response);
    expect((value as Response).headers.get("Location")).toBe("/login");
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });
});

// Parsed through userSchema so the session check accepts it exactly as it would the API's.
const currentUser = userSchema.parse({
  id: "usr_01KSJG9AG6XFQ8HDX5309293A9",
  version: 0,
  createdAt: 1779811265,
  updatedAt: 1779811264,
  businessUnitId: "bu_01KSJG9ADGK726S3YXAWMWQ2G3",
  currentOrganizationId: "org_01KSJG9ADP4CFEFS4P1CZ699NB",
  status: "Active",
  name: "System Administrator",
  username: "admin",
  emailAddress: "admin@trenova.app",
  profilePicUrl: "",
  thumbnailUrl: "",
  timezone: "America/Los_Angeles",
  timeFormat: "12-hour",
  isLocked: false,
  mustChangePassword: false,
  lastLoginAt: 1779811303,
});

function manifestReadingEverything(): PermissionManifest {
  return {
    version: "1.0",
    userId: currentUser.id,
    organizationId: currentUser.currentOrganizationId,
    activeRoleIds: [],
    authorizedRoleIds: [],
    activeRoles: [],
    authorizedRoles: [],
    requiresRoleActivation: false,
    maxSensitivity: "internal",
    permissions: Object.fromEntries(Object.values(Resource).map((resource) => [resource, 0xffff])),
    routeAccess: {},
    availableOrgs: [],
    checksum: "abc123",
    expiresAt: 1_782_403_304,
  } as unknown as PermissionManifest;
}

// The page modules are not under test, and loading them would pull every page's code
// into this run; the loaders and revalidation rules are kept exactly as the app has them.
function withoutPages(entries: RouteObject[]): RouteObject[] {
  return entries.map((entry) => {
    const { lazy: _lazy, children, ...rest } = entry;
    return (children ? { ...rest, children: withoutPages(children) } : rest) as RouteObject;
  });
}

describe("signed-in route revalidation", () => {
  let sessionChecks = 0;

  beforeEach(() => {
    sessionChecks = 0;
    queryClient.setQueryData(publicConfigQueryOptions.queryKey, SELF_HOSTED_PUBLIC_CONFIG);
    vi.spyOn(usePermissionStore.getState(), "checkForUpdates").mockResolvedValue(false);
    usePermissionStore.setState({
      manifest: manifestReadingEverything(),
      lastFetched: Date.now(),
      isLoading: false,
    });
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = new URL(input instanceof Request ? input.url : String(input), "http://localhost");
        if (!url.pathname.endsWith("/users/me/")) {
          throw new Error(`unexpected request to ${url.pathname}`);
        }
        sessionChecks += 1;
        return new Response(JSON.stringify(currentUser), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }),
    );
    useAuthStore.setState({ user: currentUser, isAuthenticated: true });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    queryClient.clear();
    usePermissionStore.getState().clearPermissions();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  async function startAt(path: string) {
    const router = createMemoryRouter(withoutPages(routes), { initialEntries: [path] });
    await vi.waitFor(() => expect(router.state.initialized).toBe(true));
    return router;
  }

  // Triage keeps the open message in ?message=, so stepping through messages with J/K is a
  // run of search-only navigations. Each one re-ran the session check and twenty of them
  // exhausted the per-user rate limit.
  it("does not re-run the session check when only the search changes", async () => {
    const router = await startAt("/billing/invoices?status=open");
    const afterLoad = sessionChecks;
    expect(afterLoad).toBeGreaterThan(0);

    for (const status of ["paid", "void", "draft"]) {
      await router.navigate(`/billing/invoices?status=${status}`);
    }

    expect(router.state.location.search).toBe("?status=draft");
    expect(router.state.errors).toBeNull();
    expect(sessionChecks).toBe(afterLoad);
    router.dispose();
  });

  it("re-runs the session check when the path changes", async () => {
    const router = await startAt("/billing/invoices?status=open");
    const afterLoad = sessionChecks;

    await router.navigate("/billing/pending-approvals");

    expect(router.state.location.pathname).toBe("/billing/pending-approvals");
    expect(sessionChecks).toBeGreaterThan(afterLoad);
    router.dispose();
  });

  it("re-runs the session check on an explicit revalidate", async () => {
    const router = await startAt("/billing/invoices?status=open");
    const afterLoad = sessionChecks;

    await router.revalidate();

    expect(sessionChecks).toBeGreaterThan(afterLoad);
    router.dispose();
  });

  it("guards Triage the same way", async () => {
    const triage = [...withoutPages(routes)]
      .flatMap(function flatten(entry): RouteObject[] {
        return [entry, ...(entry.children ?? []).flatMap(flatten)];
      })
      .find((entry) => entry.path === "/triage");

    expect(
      triage?.shouldRevalidate?.({
        currentUrl: new URL("http://localhost/triage?message=msg_1"),
        nextUrl: new URL("http://localhost/triage?message=msg_2"),
        currentParams: {},
        nextParams: {},
        defaultShouldRevalidate: true,
      }),
    ).toBe(false);
  });
});
