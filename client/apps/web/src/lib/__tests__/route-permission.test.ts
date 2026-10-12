import { createPermissionLoader, ensurePermissionManifest } from "@/lib/route-permission";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { LoaderFunctionArgs } from "react-router";
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

function runPermissionLoader() {
  const loader = createPermissionLoader(Resource.Shipment, Operation.Read);
  return settle(
    Promise.resolve(
      loader({
        request: new Request("http://localhost/shipments/management"),
        params: {},
        context: undefined,
      } as unknown as LoaderFunctionArgs),
    ),
  );
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

    const { value, thrown } = await runPermissionLoader();

    expect(value).toBeUndefined();
    expect(thrown).toBeInstanceOf(ApiRequestError);
    expect((thrown as ApiRequestError).status).toBe(429);
    expect((thrown as ApiRequestError).retryAfter).toBe(7);
  });

  it("surfaces a server error as the route error", async () => {
    answerManifestWith(problemResponse(500, "internal-error", "Internal Error"));

    const { value, thrown } = await runPermissionLoader();

    expect(value).toBeUndefined();
    expect((thrown as ApiRequestError).status).toBe(500);
  });

  it("surfaces a network failure as the route error", async () => {
    const failure = new TypeError("Failed to fetch");
    answerManifestWith(failure);

    const { value, thrown } = await runPermissionLoader();

    expect(value).toBeUndefined();
    expect(thrown).toBe(failure);
  });

  it("still sends an expired session to /login", async () => {
    answerManifestWith(problemResponse(401, "authentication-error", "Authentication Failed"));

    const { value, thrown } = await runPermissionLoader();

    expect(thrown).toBeUndefined();
    expect(value).toBeInstanceOf(Response);
    expect((value as Response).headers.get("Location")).toBe("/login");
  });
});
