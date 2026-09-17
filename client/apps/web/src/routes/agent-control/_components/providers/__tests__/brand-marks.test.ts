import { describe, expect, it } from "vitest";
import { brandMarkFor } from "@trenova/shared/components/ui/logos/registry";

/**
 * A preset whose mark we ship must resolve without a network call, because
 * VITE_BRANDFETCH_CLIENT_ID is unset in most installs and the CDN returns
 * nothing without it. A preset we have no licensed mark for must resolve to
 * null rather than to an approximation of someone's logo.
 */
describe("brandMarkFor", () => {
  it("ships a mark for the presets people reach for first", () => {
    for (const key of [
      "openai",
      "anthropic",
      "ollama",
      "openrouter",
      "bedrock",
      "vllm",
      "lmstudio",
      "groq",
      "together",
      "fireworks",
    ]) {
      expect(brandMarkFor({ presetKey: key }), `${key} has no bundled mark`).not.toBeNull();
    }
  });

  it("resolves a saved provider by its endpoint domain", () => {
    expect(brandMarkFor({ domain: "anthropic.com" })).not.toBeNull();
    expect(brandMarkFor({ domain: "api.openai.com" })).not.toBeNull();
    expect(brandMarkFor({ domain: "www.ollama.com" })).not.toBeNull();
  });

  it("resolves the three vendors whose marks are not in simple-icons", () => {
    expect(brandMarkFor({ domain: "groq.com" })).not.toBeNull();
    expect(brandMarkFor({ domain: "api.together.xyz" })).not.toBeNull();
    expect(brandMarkFor({ domain: "api.fireworks.ai" })).not.toBeNull();
  });

  it("returns nothing for a vendor we do not ship", () => {
    // sglang, llamacpp and deepinfra have no mark we can draw from a published
    // source, so they fall to the monogram rather than to a guess at a logo.
    expect(brandMarkFor({ presetKey: "sglang" })).toBeNull();
    expect(brandMarkFor({ presetKey: "llamacpp" })).toBeNull();
    expect(brandMarkFor({ domain: "example.invalid" })).toBeNull();
    expect(brandMarkFor({})).toBeNull();
  });
});
