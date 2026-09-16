import type { AIProvider } from "@/types/ai-provider";
import { describe, expect, it } from "vitest";
import { aiProvidersCatalogItem } from "../_components/ai-providers/ai-providers-catalog-item";

function provider(overrides: Partial<AIProvider> = {}): AIProvider {
  return {
    id: "aiprv_1",
    businessUnitId: "bu_1",
    organizationId: "org_1",
    name: "Local Ollama",
    description: "",
    kind: "Ollama",
    baseUrl: "",
    model: "qwen2.5:14b",
    allowPrivateNetwork: true,
    structuredOutputMode: "JSONSchema",
    maxTokens: 8192,
    tasks: ["General"],
    priority: 10,
    trusted: false,
    enabled: true,
    version: 0,
    createdAt: 0,
    updatedAt: 0,
    ...overrides,
  };
}

/**
 * AI providers live in their own table rather than the integrations table, but
 * the marketplace is where an administrator goes to connect anything. The card
 * is built client-side from the provider list and has to answer the same
 * questions the server-built cards do, so the status filter treats it alike.
 */
describe("aiProvidersCatalogItem", () => {
  it("sits in the same category as the other AI integrations", () => {
    const item = aiProvidersCatalogItem([]);

    expect(item.type).toBe("AIProviders");
    expect(item.category).toBe("ArtificialIntelligence");
    expect(item.categoryLabel).toBe("AI & Automation");
  });

  it("reads as disconnected and needing setup with no providers", () => {
    const item = aiProvidersCatalogItem([]);

    expect(item.enabled).toBe(false);
    expect(item.configured).toBe(false);
    expect(item.status.connection).toBe("disconnected");
    expect(item.status.configuration).toBe("needs_setup");
  });

  // A provider that is saved but switched off is configured, not connected —
  // the same distinction the server draws for a key that is stored but disabled.
  it("reads as configured but disconnected when every provider is disabled", () => {
    const item = aiProvidersCatalogItem([provider({ enabled: false })]);

    expect(item.configured).toBe(true);
    expect(item.enabled).toBe(false);
    expect(item.status.configuration).toBe("configured");
    expect(item.status.connection).toBe("disconnected");
  });

  it("reads as connected once any provider is enabled", () => {
    const item = aiProvidersCatalogItem([provider({ enabled: false }), provider({ id: "p2" })]);

    expect(item.enabled).toBe(true);
    expect(item.status.connection).toBe("connected");
    expect(item.status.connectionLabel).toBe("Connected");
  });

  it("says how many providers are configured in its action label", () => {
    expect(aiProvidersCatalogItem([]).primaryActionLabel).toBe("Add Provider");
    expect(aiProvidersCatalogItem([provider(), provider({ id: "p2" })]).primaryActionLabel).toBe(
      "Manage 2 Providers",
    );
  });
});
