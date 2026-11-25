import { queryClient } from "@/lib/query-client";
import { routeTree } from "@/route-tree.gen";
import { gateOutcomeById } from "@/test/route-tree";
import { publicConfigQueryOptions } from "@trenova/shared/hooks/use-public-config";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { SELF_HOSTED_PUBLIC_CONFIG } from "@trenova/shared/types/platform";
import { Resource, type PermissionManifest } from "@trenova/shared/types/permission";
import { userSchema, type User } from "@trenova/shared/types/user";
import { createTestRouter } from "@trenova/shared/testing/test-router";
import { isRedirect, type AnyRouter } from "@tanstack/react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// The pages these tests open are not under test; their route files, gates and loaders are, so
// each page module stands in as an empty page with nothing to prefetch.
vi.mock("@/routes/invoice/page", () => ({ InvoicesPage: () => null }));
vi.mock("@/routes/invoice-approval/page", () => ({ InvoiceApprovalPage: () => null }));
vi.mock("@/routes/desk/page", () => ({ DeskPage: () => null, prefetch: () => [] }));
vi.mock("@/routes/desk/triage-page", () => ({ DeskTriagePage: () => null }));
vi.mock("@/routes/inbox/page", () => ({ prefetch: () => [] }));

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

// The gate of the pathless layout every signed-in page sits under: the one that guards them all.
async function runProtectedGate(): Promise<{ thrown: unknown }> {
  return { thrown: await gateOutcomeById("/_app", { pathname: "/shipments/management" }) };
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

describe("the signed-in session gate", () => {
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

    const { thrown } = await runProtectedGate();

    expect(thrown).toBeInstanceOf(ApiRequestError);
    expect((thrown as ApiRequestError).status).toBe(429);
    expect((thrown as ApiRequestError).retryAfter).toBe(7);
  });

  it("surfaces a cold-load network failure instead of redirecting", async () => {
    coldLoad();
    const failure = new TypeError("Failed to fetch");
    answerSessionCheckWith(failure);

    const { thrown } = await runProtectedGate();

    expect(thrown).toBe(failure);
  });

  it("lets a rate-limited check through mid-session", async () => {
    midSession();
    answerSessionCheckWith(problemResponse(429, "rate-limit-exceeded", "Rate Limit Exceeded"));

    const { thrown } = await runProtectedGate();

    expect(thrown).toBeUndefined();
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

    const { thrown } = await runProtectedGate();

    expect(thrown).toBeUndefined();
  });

  it("still sends an expired session to /login, carrying the page it was on", async () => {
    midSession();
    answerSessionCheckWith(problemResponse(401, "authentication-error", "Authentication Failed"));

    const { thrown } = await runProtectedGate();

    expect(isRedirect(thrown)).toBe(true);
    expect((thrown as { options: { href?: string } }).options.href).toBe(
      "/login?redirect=%2Fshipments%2Fmanagement",
    );
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
  photo: null,
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
    unavailableFeatures: [],
    checksum: "abc123",
    expiresAt: 1_782_403_304,
  } as unknown as PermissionManifest;
}

function failedMatches(router: AnyRouter) {
  return router.state.matches.filter((match) => match.status === "error");
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
        const url = new URL(
          input instanceof Request ? input.url : String(input),
          "http://localhost",
        );
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
    const router = createTestRouter(routeTree, [path], { queryClient });
    await router.load();
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
      await router.navigate({ href: `/billing/invoices?status=${status}` });
    }

    expect(router.state.location.searchStr).toBe("?status=draft");
    expect(failedMatches(router)).toEqual([]);
    expect(sessionChecks).toBe(afterLoad);
  });

  it("re-runs the session check when the path changes", async () => {
    const router = await startAt("/billing/invoices?status=open");
    const afterLoad = sessionChecks;

    await router.navigate({ href: "/billing/pending-approvals" });

    expect(router.state.location.pathname).toBe("/billing/pending-approvals");
    expect(sessionChecks).toBeGreaterThan(afterLoad);
  });

  // Keys pressed faster than a navigation commits read the message that is still open, so
  // each one navigates to the URL already shown. Re-running the session check on each kept a
  // navigation pending for good.
  it("does not re-run the session check when quick steps land on the URL already shown", async () => {
    const router = await startAt("/billing/invoices?status=open");
    const afterLoad = sessionChecks;

    for (let step = 0; step < 5; step += 1) {
      await router.navigate({ href: "/billing/invoices?status=open" });
    }

    expect(failedMatches(router)).toEqual([]);
    expect(sessionChecks).toBe(afterLoad);
  });

  it("guards Triage the same way", async () => {
    const router = await startAt("/desk/triage?message=msg_1");
    const afterLoad = sessionChecks;
    expect(afterLoad).toBeGreaterThan(0);

    await router.navigate({ href: "/desk/triage?message=msg_2" });

    expect(router.state.location.searchStr).toBe("?message=msg_2");
    expect(failedMatches(router)).toEqual([]);
    expect(sessionChecks).toBe(afterLoad);
  });
});
