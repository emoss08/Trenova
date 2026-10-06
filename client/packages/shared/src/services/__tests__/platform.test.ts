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
};

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
