import { describe, expect, it } from "vitest";
import type { AIProviderPlatform } from "@/types/ai-provider";
import { novaSetup } from "../nova-setup";

const NOVA_PLATFORMS: readonly AIProviderPlatform[] = [
  "Direct",
  "ClaudePlatformAWS",
  "ClaudePlatformGoogleCloud",
  "VertexAI",
];

type Provider = Parameters<typeof novaSetup>[0][number];

function provider(overrides: Partial<Provider> & Pick<Provider, "id">): Provider {
  return {
    name: overrides.id,
    kind: "AnthropicMessages",
    platform: "Direct",
    enabled: true,
    nova: false,
    priority: 0,
    version: 1,
    ...overrides,
  } as Provider;
}

describe("novaSetup", () => {
  it("is ready when the provider Nova runs on is on", () => {
    const nova = provider({ id: "aiprv_a", nova: true });

    expect(novaSetup([nova, provider({ id: "aiprv_b" })], NOVA_PLATFORMS)).toEqual({
      state: "ready",
      provider: nova,
    });
  });

  it("is paused when the provider Nova runs on is off", () => {
    const nova = provider({ id: "aiprv_a", nova: true, enabled: false });

    expect(novaSetup([nova], NOVA_PLATFORMS)).toEqual({ state: "paused", provider: nova });
  });

  it("offers the first provider on, in routing order, that can run Nova", () => {
    const off = provider({ id: "aiprv_off", priority: 0, enabled: false });
    const later = provider({ id: "aiprv_later", priority: 2 });
    const first = provider({ id: "aiprv_first", priority: 1 });
    const bedrock = provider({ id: "aiprv_bedrock", priority: 0, platform: "Bedrock" });
    const openai = provider({ id: "aiprv_openai", priority: 0, kind: "OpenAIChat" });

    expect(novaSetup([off, later, first, bedrock, openai], NOVA_PLATFORMS)).toEqual({
      state: "choose",
      provider: first,
    });
  });

  it("offers a provider that is off when no provider that can run Nova is on", () => {
    const off = provider({ id: "aiprv_off", enabled: false });

    expect(novaSetup([off], NOVA_PLATFORMS)).toEqual({ state: "choose", provider: off });
  });

  it("asks for a provider when none can run Nova", () => {
    expect(
      novaSetup(
        [
          provider({ id: "aiprv_openai", kind: "OpenAIChat" }),
          provider({ id: "aiprv_bedrock", platform: "Bedrock" }),
        ],
        NOVA_PLATFORMS,
      ),
    ).toEqual({ state: "connect" });
    expect(novaSetup([], NOVA_PLATFORMS)).toEqual({ state: "connect" });
  });

  it("leaves the providers it was given in their order", () => {
    const providers = [provider({ id: "aiprv_b", priority: 2 }), provider({ id: "aiprv_a" })];
    novaSetup(providers, NOVA_PLATFORMS);

    expect(providers.map((p) => p.id)).toEqual(["aiprv_b", "aiprv_a"]);
  });
});
