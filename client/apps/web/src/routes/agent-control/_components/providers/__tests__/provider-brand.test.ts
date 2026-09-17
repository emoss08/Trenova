import { describe, expect, it } from "vitest";
import { providerBrandDomain, type BrandPreset } from "../provider-brand";

const presets: BrandPreset[] = [
  { kind: "OpenAIChat", baseUrl: "https://openrouter.ai/api/v1", domain: "openrouter.ai" },
  { kind: "OpenAIChat", baseUrl: "https://api.groq.com/openai/v1", domain: "groq.com" },
  { kind: "OpenAIChat", baseUrl: "http://localhost:8000/v1", domain: "vllm.ai" },
  { kind: "AnthropicMessages", baseUrl: "https://api.anthropic.com", domain: "anthropic.com" },
];

/**
 * A saved provider knows its protocol and endpoint, not the vendor behind it,
 * so the brand has to be inferred. The endpoint is the strongest signal; the
 * protocol is the fallback for vendors whose default endpoint is implied.
 */
describe("providerBrandDomain", () => {
  it("matches a preset by endpoint host, ignoring path and case", () => {
    expect(
      providerBrandDomain(
        { kind: "OpenAIChat", baseUrl: "https://API.GROQ.com/openai/v1/" },
        presets,
      ),
    ).toBe("groq.com");
  });

  it("never claims a self-hosted preset's brand for a private endpoint", () => {
    // Several presets share localhost; vLLM at :8000 is the preset, but a
    // llama.cpp server on the same port is just as likely. A private host gets
    // no brand and the monogram renders.
    expect(
      providerBrandDomain({ kind: "OpenAIChat", baseUrl: "http://localhost:8000/v1" }, presets),
    ).toBeNull();
    expect(
      providerBrandDomain({ kind: "OpenAIChat", baseUrl: "http://10.0.0.5:8000/v1" }, presets),
    ).toBeNull();
    expect(
      providerBrandDomain(
        { kind: "OpenAIChat", baseUrl: "http://gpu-box.internal:1234/v1" },
        presets,
      ),
    ).toBeNull();
  });

  it("falls back to the protocol's vendor when no endpoint is set", () => {
    expect(providerBrandDomain({ kind: "AnthropicMessages", baseUrl: "" }, presets)).toBe(
      "anthropic.com",
    );
    expect(providerBrandDomain({ kind: "OpenAIResponses", baseUrl: "" }, presets)).toBe(
      "openai.com",
    );
    expect(providerBrandDomain({ kind: "Ollama", baseUrl: "" }, presets)).toBe("ollama.com");
    expect(providerBrandDomain({ kind: "OpenAIChat", baseUrl: "" }, presets)).toBeNull();
  });

  it("derives a public vendor domain from an unknown hosted endpoint", () => {
    expect(
      providerBrandDomain({ kind: "OpenAIChat", baseUrl: "https://api.together.xyz/v1" }, presets),
    ).toBe("together.xyz");
    expect(
      providerBrandDomain(
        { kind: "OpenAIChat", baseUrl: "https://bedrock-mantle.us-east-1.amazonaws.com/openai/v1" },
        presets,
      ),
    ).toBe("amazonaws.com");
  });

  it("returns null for an endpoint that is not a URL", () => {
    expect(providerBrandDomain({ kind: "OpenAIChat", baseUrl: "not a url" }, presets)).toBeNull();
  });
});
