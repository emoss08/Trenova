import { routeTree } from "@/router";
import type { RouteGateContext } from "@trenova/shared/lib/router/gates";
import { enterContext } from "@trenova/shared/testing/gate-context";
import { createTestRouter } from "@trenova/shared/testing/test-router";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { userSchema, type User } from "@trenova/shared/types/user";
import { isRedirect, type AnyRoute } from "@tanstack/react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// The page modules are not under test, and loading them would pull every page's code into
// this run; the gates and the router's rules for when they run are kept exactly as they are.
vi.mock("@tanstack/react-router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@tanstack/react-router")>()),
  lazyRouteComponent: () => () => null,
}));

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

// The gate of the pathless layout every signed-in driver page sits under.
async function runProtectedGate(): Promise<{ thrown: unknown }> {
  const layout = (createTestRouter(routeTree).routesById as Record<string, AnyRoute>)["/signed-in"];
  const gate = layout?.options.beforeLoad as ((context: RouteGateContext) => unknown) | undefined;
  if (!gate) {
    throw new Error("the signed-in layout has no gate");
  }
  try {
    await gate(enterContext("/dash/loads"));
    return { thrown: undefined };
  } catch (thrown) {
    return { thrown };
  }
}

function signedInUser(isAuthenticated: boolean) {
  useAuthStore.setState({
    user: { id: "usr_01", currentOrganizationId: "org_01" } as unknown as User,
    isAuthenticated,
  });
}

describe("the dash session gate", () => {
  beforeEach(() => {
    vi.spyOn(usePermissionStore.getState(), "clearPermissions").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  // A full page load: the store persists only the user, so nothing proves the session.
  it("does not send a rate-limited cold load to the login form", async () => {
    signedInUser(false);
    answerSessionCheckWith(problemResponse(429, "rate-limit-exceeded", "Rate Limit Exceeded"));

    const { thrown } = await runProtectedGate();

    expect(thrown).toBeInstanceOf(ApiRequestError);
    expect((thrown as ApiRequestError).status).toBe(429);
  });

  it("lets a network failure through mid-session", async () => {
    signedInUser(true);
    answerSessionCheckWith(new TypeError("Failed to fetch"));

    const { thrown } = await runProtectedGate();

    expect(thrown).toBeUndefined();
  });

  it("still sends an expired session to /dash/login, carrying the page it was on", async () => {
    signedInUser(true);
    answerSessionCheckWith(problemResponse(401, "authentication-error", "Authentication Failed"));

    const { thrown } = await runProtectedGate();

    expect(isRedirect(thrown)).toBe(true);
    expect((thrown as { options: { href?: string } }).options.href).toBe(
      "/dash/login?redirect=%2Fdash%2Floads",
    );
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
  name: "Driver One",
  username: "driver",
  emailAddress: "driver@trenova.app",
  photo: null,
  timezone: "America/Los_Angeles",
  timeFormat: "12-hour",
  isLocked: false,
  mustChangePassword: false,
  lastLoginAt: 1779811303,
});

describe("dash signed-in route revalidation", () => {
  let sessionChecks = 0;

  beforeEach(() => {
    sessionChecks = 0;
    vi.spyOn(usePermissionStore.getState(), "checkForUpdates").mockResolvedValue(false);
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
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  async function startAt(path: string) {
    const router = createTestRouter(routeTree, [path]);
    await router.load();
    return router;
  }

  it("does not re-run the session check when only the search changes", async () => {
    const router = await startAt("/dash/loads?status=active");
    const afterLoad = sessionChecks;
    expect(afterLoad).toBeGreaterThan(0);

    for (const status of ["upcoming", "completed", "active"]) {
      await router.navigate({ href: `/dash/loads?status=${status}` });
    }

    expect(router.state.location.searchStr).toBe("?status=active");
    expect(sessionChecks).toBe(afterLoad);
  });

  it("re-runs the session check when the path changes", async () => {
    const router = await startAt("/dash/loads?status=active");
    const afterLoad = sessionChecks;

    await router.navigate({ href: "/dash/pay" });

    expect(router.state.location.pathname).toBe("/dash/pay");
    expect(sessionChecks).toBeGreaterThan(afterLoad);
  });
});
