import { afterEach, describe, expect, it, vi } from "vitest";

async function freshConstants() {
  vi.resetModules();
  return import("@trenova/shared/lib/constants");
}

afterEach(() => {
  vi.unstubAllEnvs();
});

describe("vendor addresses", () => {
  it("defaults to no terms, privacy or support address at all", async () => {
    vi.stubEnv("VITE_TERMS_URL", "");
    vi.stubEnv("VITE_PRIVACY_URL", "");
    vi.stubEnv("VITE_SUPPORT_EMAIL", "");

    const constants = await freshConstants();

    expect(constants.TERMS_URL).toBe("");
    expect(constants.PRIVACY_URL).toBe("");
    expect(constants.SUPPORT_EMAIL).toBe("");
  });

  it("takes the build's own addresses", async () => {
    vi.stubEnv("VITE_TERMS_URL", " https://example.com/terms ");
    vi.stubEnv("VITE_PRIVACY_URL", "https://example.com/privacy");
    vi.stubEnv("VITE_SUPPORT_EMAIL", "help@example.com");

    const constants = await freshConstants();

    expect(constants.TERMS_URL).toBe("https://example.com/terms");
    expect(constants.PRIVACY_URL).toBe("https://example.com/privacy");
    expect(constants.SUPPORT_EMAIL).toBe("help@example.com");
  });

  it("refuses a legal address that is not a web page", async () => {
    vi.stubEnv("VITE_TERMS_URL", "javascript:alert(1)");

    const constants = await freshConstants();

    expect(constants.TERMS_URL).toBe("");
  });
});
