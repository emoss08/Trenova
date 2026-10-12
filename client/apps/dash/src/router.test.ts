import { routes } from "@/router";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
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

// The pathless layout every signed-in driver page sits under.
function findSignedInLayout(entries: RouteObject[]): RouteObject | undefined {
  for (const entry of entries) {
    if (entry.children?.some((child) => child.path === "/dash")) {
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
      request: new Request("http://localhost/dash/loads"),
      params: {},
      context: undefined,
    } as unknown as LoaderFunctionArgs),
  ).then(
    (value) => ({ value, thrown: undefined }),
    (thrown: unknown) => ({ value: undefined, thrown }),
  );
}

function signedInUser(isAuthenticated: boolean) {
  useAuthStore.setState({
    user: { id: "usr_01", currentOrganizationId: "org_01" } as unknown as User,
    isAuthenticated,
  });
}

describe("dashProtectedLoader", () => {
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

    const { value, thrown } = await runProtectedLoader();

    expect(value).toBeUndefined();
    expect(thrown).toBeInstanceOf(ApiRequestError);
    expect((thrown as ApiRequestError).status).toBe(429);
  });

  it("lets a network failure through mid-session", async () => {
    signedInUser(true);
    answerSessionCheckWith(new TypeError("Failed to fetch"));

    const { value, thrown } = await runProtectedLoader();

    expect(thrown).toBeUndefined();
    expect(value).toBeNull();
  });

  it("still sends an expired session to /dash/login", async () => {
    signedInUser(true);
    answerSessionCheckWith(problemResponse(401, "authentication-error", "Authentication Failed"));

    const { value, thrown } = await runProtectedLoader();

    expect(thrown).toBeUndefined();
    expect(value).toBeInstanceOf(Response);
    expect((value as Response).headers.get("Location")).toBe("/dash/login");
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
  profilePicUrl: "",
  thumbnailUrl: "",
  timezone: "America/Los_Angeles",
  timeFormat: "12-hour",
  isLocked: false,
  mustChangePassword: false,
  lastLoginAt: 1779811303,
});

// The page modules are not under test; the loaders and revalidation rules are kept exactly
// as the app has them.
function withoutPages(entries: RouteObject[]): RouteObject[] {
  return entries.map((entry) => {
    const { lazy: _lazy, children, ...rest } = entry;
    return (children ? { ...rest, children: withoutPages(children) } : rest) as RouteObject;
  });
}

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
    const router = createMemoryRouter(withoutPages(routes), { initialEntries: [path] });
    await vi.waitFor(() => expect(router.state.initialized).toBe(true));
    return router;
  }

  it("does not re-run the session check when only the search changes", async () => {
    const router = await startAt("/dash/loads?status=active");
    const afterLoad = sessionChecks;
    expect(afterLoad).toBeGreaterThan(0);

    for (const status of ["upcoming", "completed", "active"]) {
      await router.navigate(`/dash/loads?status=${status}`);
    }

    expect(router.state.location.search).toBe("?status=active");
    expect(sessionChecks).toBe(afterLoad);
    router.dispose();
  });

  it("re-runs the session check when the path changes", async () => {
    const router = await startAt("/dash/loads?status=active");
    const afterLoad = sessionChecks;

    await router.navigate("/dash/pay");

    expect(router.state.location.pathname).toBe("/dash/pay");
    expect(sessionChecks).toBeGreaterThan(afterLoad);
    router.dispose();
  });
});
