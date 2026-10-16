import { afterEach, describe, expect, it, vi } from "vitest";
import { platformService } from "@trenova/shared/services/platform";
import { SELF_HOSTED_PUBLIC_CONFIG, isSignupAvailable } from "@trenova/shared/types/platform";

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

// The documented shape of GET /api/v1/system/public-config.
const cloudConfig = {
  platformMode: "cloud",
  signupEnabled: true,
  turnstileSiteKey: "1x00000000000000000000AA",
  termsUrl: "https://trenova.app/legal/terms/",
  privacyUrl: "https://trenova.app/legal/privacy/",
  freePlan: { limits: { "shipments.total": 12, "documents.storage_bytes": 104857600 } },
  session: { idleTimeoutSeconds: 1800, idleWarningSeconds: 120, keepAliveSeconds: 60 },
};

const IDLE_OFF = { idleTimeoutSeconds: 0, idleWarningSeconds: 0, keepAliveSeconds: 0 };

describe("platformService.getPublicConfig", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("returns the server's config when it matches the contract", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(cloudConfig)),
    );

    const config = await platformService.getPublicConfig();

    expect(config).toEqual(cloudConfig);
    expect(isSignupAvailable(config)).toBe(true);
  });

  it("falls back to self-hosted with signup off when the request fails", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        throw new TypeError("Failed to fetch");
      }),
    );

    const config = await platformService.getPublicConfig();

    expect(config).toEqual(SELF_HOSTED_PUBLIC_CONFIG);
    expect(isSignupAvailable(config)).toBe(false);
  });

  it("falls back when an older server has no such route", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({ type: "resource-not-found", title: "Not Found", status: 404 }, 404),
      ),
    );

    expect(await platformService.getPublicConfig()).toEqual(SELF_HOSTED_PUBLIC_CONFIG);
  });

  it("falls back when the body is not an object", async () => {
    vi.spyOn(console, "warn").mockImplementation(() => {});
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(["not", "a", "config"])),
    );

    expect(await platformService.getPublicConfig()).toEqual(SELF_HOSTED_PUBLIC_CONFIG);
  });

  it("never enables signup on an unknown platform mode", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ ...cloudConfig, platformMode: "saas-v2" })),
    );

    const config = await platformService.getPublicConfig();

    expect(config.platformMode).toBe("self_hosted");
    expect(isSignupAvailable(config)).toBe(false);
  });

  it("keeps signup off when the flag is missing or not a boolean", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ platformMode: "cloud", signupEnabled: "yes" })),
    );

    const config = await platformService.getPublicConfig();

    expect(config.signupEnabled).toBe(false);
    expect(config.turnstileSiteKey).toBe("");
    expect(config.freePlan).toEqual({ limits: {} });
  });

  it("drops legal links that are not http(s) URLs", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({ ...cloudConfig, termsUrl: "javascript:alert(1)", privacyUrl: null }),
      ),
    );

    const config = await platformService.getPublicConfig();

    expect(config.termsUrl).toBe("");
    expect(config.privacyUrl).toBe("");
  });

  it("turns the idle sign-out off when an older server sends no session policy", async () => {
    const { session: _session, ...olderServer } = cloudConfig;
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(olderServer)),
    );

    const config = await platformService.getPublicConfig();

    expect(config.session).toEqual(IDLE_OFF);
  });

  it("keeps a disabled idle policy disabled", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ ...cloudConfig, session: IDLE_OFF })),
    );

    expect((await platformService.getPublicConfig()).session).toEqual(IDLE_OFF);
  });

  it.each([
    [
      "a warning as long as the timeout",
      { idleTimeoutSeconds: 120, idleWarningSeconds: 120, keepAliveSeconds: 60 },
    ],
    ["no warning", { idleTimeoutSeconds: 1800, idleWarningSeconds: 0, keepAliveSeconds: 60 }],
    ["no keepalive", { idleTimeoutSeconds: 1800, idleWarningSeconds: 120, keepAliveSeconds: 0 }],
    [
      "a negative timeout",
      { idleTimeoutSeconds: -5, idleWarningSeconds: 120, keepAliveSeconds: 60 },
    ],
    [
      "a fractional timeout",
      { idleTimeoutSeconds: 1800.5, idleWarningSeconds: 120, keepAliveSeconds: 60 },
    ],
    [
      "a string timeout",
      { idleTimeoutSeconds: "1800", idleWarningSeconds: 120, keepAliveSeconds: 60 },
    ],
    ["a policy that is not an object", "30m"],
  ])("turns the idle sign-out off for %s", async (_name, session) => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ ...cloudConfig, session })),
    );

    const config = await platformService.getPublicConfig();

    expect(config.session).toEqual(IDLE_OFF);
    expect(config.platformMode).toBe("cloud");
  });

  it("propagates an abort instead of caching the fallback", async () => {
    const controller = new AbortController();
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        controller.abort();
        throw new DOMException("Aborted", "AbortError");
      }),
    );

    await expect(platformService.getPublicConfig(controller.signal)).rejects.toThrow("Aborted");
  });
});
