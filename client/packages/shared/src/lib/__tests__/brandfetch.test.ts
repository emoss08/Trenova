import { describe, expect, it } from "vitest";
import { brandfetchLogoUrl } from "../brandfetch";

/**
 * The Logo CDN contract (https://docs.brandfetch.com/logo-api): a public
 * client id is the only credential the URL carries, so the id is the one thing
 * that decides whether a logo can be requested at all.
 */
describe("brandfetchLogoUrl", () => {
  it("builds a CDN URL from the domain, size and client id", () => {
    expect(brandfetchLogoUrl("openai.com", { clientId: "abc123", width: 64, height: 64 })).toBe(
      "https://cdn.brandfetch.io/openai.com/w/64/h/64?c=abc123",
    );
  });

  it("returns null without a client id so callers fall back instead of hitting the CDN", () => {
    expect(brandfetchLogoUrl("openai.com", { clientId: "", width: 64 })).toBeNull();
    expect(brandfetchLogoUrl("openai.com", { clientId: undefined, width: 64 })).toBeNull();
  });

  it("returns null for a blank domain", () => {
    expect(brandfetchLogoUrl("   ", { clientId: "abc123", width: 64 })).toBeNull();
  });

  it("defaults the height to the width for a square tile", () => {
    expect(brandfetchLogoUrl("groq.com", { clientId: "abc123", width: 48 })).toBe(
      "https://cdn.brandfetch.io/groq.com/w/48/h/48?c=abc123",
    );
  });

  it("normalizes a domain pasted with a scheme, path or upper case", () => {
    expect(brandfetchLogoUrl("https://Anthropic.com/pricing", { clientId: "abc", width: 32 })).toBe(
      "https://cdn.brandfetch.io/anthropic.com/w/32/h/32?c=abc",
    );
  });

  it("never lets the domain carry a path or query into the CDN URL", () => {
    expect(brandfetchLogoUrl("open ai.com/../x", { clientId: "abc", width: 32 })).toBeNull();
    expect(brandfetchLogoUrl("evil.com?c=other", { clientId: "abc", width: 32 })).toBe(
      "https://cdn.brandfetch.io/evil.com/w/32/h/32?c=abc",
    );
  });
});
