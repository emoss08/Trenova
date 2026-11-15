import { queryClient } from "@/lib/query-client";
import { anyOfGates, ensurePermissionManifest, requirePermission } from "@/lib/route-permission";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { planRestrictedRefusal } from "@trenova/shared/lib/plan-limit";
import { RouteRefusal } from "@trenova/shared/lib/router/refusal";
import { enterContext } from "@trenova/shared/testing/gate-context";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import {
  Operation,
  Resource,
  type PermissionManifest,
  type UnavailableFeature,
} from "@trenova/shared/types/permission";
import { isRedirect, redirect } from "@tanstack/react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// The manifest is GET /me/permissions over REST. Every failure the server sends is an
// RFC 9457 problem document; the rate limiter's carries Retry-After.
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

function answerManifestWith(outcome: Response | Error) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = new URL(input instanceof Request ? input.url : String(input), "http://localhost");
      if (!url.pathname.endsWith("/me/permissions")) {
        throw new Error(`unexpected request to ${url.pathname}`);
      }
      if (outcome instanceof Error) {
        throw outcome;
      }
      return outcome;
    }),
  );
}

async function settle(promise: Promise<unknown>): Promise<{ value: unknown; thrown: unknown }> {
  return promise.then(
    (value) => ({ value, thrown: undefined }),
    (thrown: unknown) => ({ value: undefined, thrown }),
  );
}

function runPermissionGate() {
  const gate = requirePermission(Resource.Shipment, Operation.Read);
  return settle(Promise.resolve(gate(enterContext("/shipments/management"))));
}

describe("ensurePermissionManifest without a manifest loaded", () => {
  beforeEach(() => {
    usePermissionStore.getState().clearPermissions();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    usePermissionStore.getState().clearPermissions();
  });

  it("reports a 401 as no session", async () => {
    answerManifestWith(problemResponse(401, "authentication-error", "Authentication Failed"));

    await expect(ensurePermissionManifest()).resolves.toBe(false);
  });

  it("throws a rate-limited fetch rather than reporting no session", async () => {
    answerManifestWith(problemResponse(429, "rate-limit-exceeded", "Rate Limit Exceeded"));

    const { value, thrown } = await settle(ensurePermissionManifest());

    expect(value).toBeUndefined();
    expect(thrown).toBeInstanceOf(ApiRequestError);
    expect((thrown as ApiRequestError).status).toBe(429);
  });
});

describe("createPermissionLoader without a manifest loaded", () => {
  beforeEach(() => {
    usePermissionStore.getState().clearPermissions();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    usePermissionStore.getState().clearPermissions();
  });

  it("does not send a rate-limited manifest fetch to /login", async () => {
    answerManifestWith(problemResponse(429, "rate-limit-exceeded", "Rate Limit Exceeded"));

    const { value, thrown } = await runPermissionGate();

    expect(value).toBeUndefined();
    expect(thrown).toBeInstanceOf(ApiRequestError);
    expect((thrown as ApiRequestError).status).toBe(429);
    expect((thrown as ApiRequestError).retryAfter).toBe(7);
  });

  it("surfaces a server error as the route error", async () => {
    answerManifestWith(problemResponse(500, "internal-error", "Internal Error"));

    const { value, thrown } = await runPermissionGate();

    expect(value).toBeUndefined();
    expect((thrown as ApiRequestError).status).toBe(500);
  });

  it("surfaces a network failure as the route error", async () => {
    const failure = new TypeError("Failed to fetch");
    answerManifestWith(failure);

    const { value, thrown } = await runPermissionGate();

    expect(value).toBeUndefined();
    expect(thrown).toBe(failure);
  });

  it("still sends an expired session to /login", async () => {
    answerManifestWith(problemResponse(401, "authentication-error", "Authentication Failed"));

    const { thrown } = await runPermissionGate();

    expect(isRedirect(thrown)).toBe(true);
    expect((thrown as { options: { href?: string } }).options.href).toBe("/login");
  });
});

describe("anyOfGates, for a page whose parts answer to different permissions", () => {
  const args = enterContext("/desk/needs");
  const refuse = (what: string) => () => {
    throw new RouteRefusal({
      status: 403,
      statusText: "Forbidden",
      data: `Missing permission: ${what}`,
    });
  };

  it("lets the person in when a later gate does, though an earlier one refused", async () => {
    const gate = anyOfGates(refuse("watchtower:Read"), () => undefined);

    expect(await settle(Promise.resolve(gate(args)))).toEqual({
      value: undefined,
      thrown: undefined,
    });
  });

  it("refuses with the last gate's answer when every gate refuses", async () => {
    const gate = anyOfGates(refuse("watchtower:Read"), refuse("agent_proposal:Read"));

    const { thrown } = await settle(Promise.resolve(gate(args)));

    expect(thrown).toBeInstanceOf(RouteRefusal);
    expect((thrown as RouteRefusal).status).toBe(403);
    expect((thrown as RouteRefusal).data).toBe("Missing permission: agent_proposal:Read");
  });

  const offPlan = (addOn: string) => () => {
    throw planRestrictedRefusal({ feature: "tms.factoring", addOn });
  };

  it("says the page is off the plan only when the plan refused every gate", async () => {
    const gate = anyOfGates(offPlan("Accounting & Cash"), offPlan("Trenova AI"));

    const { thrown } = await settle(Promise.resolve(gate(args)));

    expect(thrown).toBeInstanceOf(RouteRefusal);
    expect((thrown as RouteRefusal).status).toBe(403);
    expect((thrown as RouteRefusal).data).toMatchObject({
      code: "PLAN_RESTRICTED",
      params: { addOn: "Accounting & Cash" },
    });
  });

  it("reports the permission refusal when any gate was refused by the roles", async () => {
    for (const gate of [
      anyOfGates(refuse("watchtower:Read"), offPlan("Accounting & Cash")),
      anyOfGates(offPlan("Accounting & Cash"), refuse("watchtower:Read")),
    ]) {
      const { thrown } = await settle(Promise.resolve(gate(args)));

      expect(thrown).toBeInstanceOf(RouteRefusal);
      expect((thrown as RouteRefusal).status).toBe(403);
      expect((thrown as RouteRefusal).data).toBe("Missing permission: watchtower:Read");
    }
  });

  it("obeys a gate that sends the person to sign in, without asking the next", async () => {
    const next = vi.fn(() => undefined);
    const toLogin = redirect({ href: "/login" });
    const gate = anyOfGates(() => {
      throw toLogin;
    }, next);

    expect((await settle(Promise.resolve(gate(args)))).thrown).toBe(toLogin);
    expect(next).not.toHaveBeenCalled();
  });

  it("does not hide a failure that is not a refusal", async () => {
    const failure = new TypeError("Failed to fetch");
    const gate = anyOfGates(
      () => {
        throw failure;
      },
      () => undefined,
    );

    expect((await settle(Promise.resolve(gate(args)))).thrown).toBe(failure);
  });
});

describe("requirePermission for a resource missing from the manifest", () => {
  const factoring: UnavailableFeature = {
    feature: "tms.factoring",
    featureName: "Factoring",
    addOn: "ACC",
    addOnName: "Accounting & Cash",
  };

  // GET /me/permissions: the plan leaves Factoring out, so factoring companies are not in
  // permissions; customers are left out by the person's roles.
  function manifest(unavailableFeatures: UnavailableFeature[]): PermissionManifest {
    return {
      version: "1.0",
      userId: "usr_1",
      organizationId: "org_1",
      activeRoleIds: [],
      authorizedRoleIds: [],
      activeRoles: [],
      authorizedRoles: [],
      requiresRoleActivation: false,
      maxSensitivity: "internal",
      permissions: { [Resource.Shipment]: Operation.Read },
      routeAccess: {},
      availableOrgs: [],
      unavailableFeatures,
      checksum: "abc",
      expiresAt: 1_900_000_000,
    };
  }

  // GET /permissions/resources: the catalogue declares each resource's feature.
  const catalogue = [
    {
      category: "Billing",
      resources: [
        {
          resource: Resource.FactoringCompany,
          displayName: "Factoring Company",
          description: "",
          category: "Billing",
          feature: "tms.factoring",
          operations: [],
          defaultSensitivity: "internal",
        },
        {
          resource: Resource.Customer,
          displayName: "Customer",
          description: "",
          category: "Billing",
          feature: "tms.billing",
          operations: [],
          defaultSensitivity: "internal",
        },
      ],
    },
  ];

  let catalogueReads = 0;

  function serve(current: PermissionManifest) {
    catalogueReads = 0;
    usePermissionStore.setState({ manifest: current, lastFetched: Date.now() });
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = new URL(
          input instanceof Request ? input.url : String(input),
          "http://localhost",
        );
        const json = (body: unknown) =>
          new Response(JSON.stringify(body), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          });
        if (url.pathname.endsWith("/me/permissions")) return json(current);
        if (url.pathname.endsWith("/permissions/resources")) {
          catalogueReads += 1;
          return json(catalogue);
        }
        throw new Error(`unexpected request to ${url.pathname}`);
      }),
    );
  }

  function run(resource: string) {
    const gate = requirePermission(resource, Operation.Read);
    return settle(Promise.resolve(gate(enterContext(`/${resource}`))));
  }

  afterEach(() => {
    vi.unstubAllGlobals();
    queryClient.clear();
    usePermissionStore.getState().clearPermissions();
  });

  it("throws the plan refusal, naming the add-on, for a resource the plan leaves out", async () => {
    serve(manifest([factoring]));

    const { thrown } = await run(Resource.FactoringCompany);

    expect(thrown).toBeInstanceOf(RouteRefusal);
    const refusal = thrown as RouteRefusal;
    expect(refusal.status).toBe(403);
    expect(refusal.data).toEqual({
      type: "plan-restricted",
      title: "Plan Restricted",
      status: 403,
      code: "PLAN_RESTRICTED",
      params: { feature: "tms.factoring", addOn: "Accounting & Cash" },
    });
  });

  it("throws the permission refusal for a resource the person's roles leave out", async () => {
    serve(manifest([factoring]));

    const { thrown } = await run(Resource.Customer);

    expect(thrown).toBeInstanceOf(RouteRefusal);
    expect((thrown as RouteRefusal).data).toBe("Missing permission: customer:Read");
  });

  it("does not read the catalogue when the plan leaves nothing out", async () => {
    serve(manifest([]));

    const { thrown } = await run(Resource.FactoringCompany);

    expect((thrown as RouteRefusal).data).toBe("Missing permission: factoring_company:Read");
    expect(catalogueReads).toBe(0);
  });
});
