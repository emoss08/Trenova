import type { AIProviderRow } from "@/lib/graphql/ai-provider";
import { describe, expect, it } from "vitest";
import { filterProviders, sortProvidersByRouting } from "../provider-roster";

function provider(overrides: Partial<AIProviderRow>): AIProviderRow {
  return {
    id: "aip_1",
    name: "OpenAI",
    model: "gpt-5",
    baseUrl: "",
    priority: 10,
    enabled: true,
    ...overrides,
  } as AIProviderRow;
}

describe("sortProvidersByRouting", () => {
  // The list is the routing order: the lowest number is offered work first.
  it("orders by priority, then live before parked, then name", () => {
    const sorted = sortProvidersByRouting([
      provider({ id: "a", name: "Local", priority: 20 }),
      provider({ id: "b", name: "Backup", priority: 10, enabled: false }),
      provider({ id: "c", name: "Anthropic", priority: 10 }),
      provider({ id: "d", name: "Zeta", priority: 10 }),
    ]);

    expect(sorted.map((p) => p.id)).toEqual(["c", "d", "b", "a"]);
  });
});

describe("filterProviders", () => {
  it("matches name, model or endpoint, ignoring case", () => {
    const providers = [
      provider({ id: "a", name: "Anthropic", model: "claude" }),
      provider({ id: "b", name: "Local", model: "llama", baseUrl: "http://10.0.0.5:11434" }),
    ];

    expect(filterProviders(providers, "CLAUDE").map((p) => p.id)).toEqual(["a"]);
    expect(filterProviders(providers, "11434").map((p) => p.id)).toEqual(["b"]);
    expect(filterProviders(providers, "")).toHaveLength(2);
  });
});
