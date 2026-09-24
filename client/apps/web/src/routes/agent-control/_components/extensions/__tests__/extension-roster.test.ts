import type { AgentExtensionCatalogItem } from "@/types/agent-extension";
import { describe, expect, it } from "vitest";
import {
  ALL_CATEGORIES,
  dailyUsageShare,
  extensionState,
  filterExtensions,
  sortExtensions,
} from "../extension-roster";

function extension(overrides: Partial<AgentExtensionCatalogItem>): AgentExtensionCatalogItem {
  return {
    type: "Exa",
    name: "Web search",
    vendor: "Exa",
    summary: "Let agents search the web.",
    description: "",
    category: "WebResearch",
    categoryLabel: "Web research",
    brandDomain: "exa.ai",
    docsUrl: "",
    websiteUrl: "",
    pricingUrl: "",
    capabilities: ["Ranks government sites first"],
    tools: [{ name: "web_search", label: "Search the web", description: "" }],
    dataNotice: "",
    featured: false,
    sortOrder: 10,
    releasedAt: 100,
    enabled: false,
    configured: false,
    availability: "SelectedAgents",
    configSpec: [],
    supportsTestConnect: true,
    dailyRequestLimit: 200,
    usage: { requestsToday: 0, requestsThisMonth: 0, failuresThisMonth: 0, costThisMonthUsd: 0 },
    enabledAt: null,
    updatedAt: 0,
    version: 0,
    ...overrides,
  };
}

describe("extensionState", () => {
  it("is on only when switched on and set up", () => {
    expect(extensionState(extension({ enabled: true, configured: true }))).toBe("on");
    expect(extensionState(extension({ enabled: true, configured: false }))).toBe("needsSetup");
    expect(extensionState(extension({ enabled: false, configured: true }))).toBe("off");
  });
});

describe("filterExtensions", () => {
  const items = [
    extension({ type: "Exa", enabled: true, configured: true }),
    extension({
      type: "Other",
      name: "Rates",
      vendor: "Acme",
      summary: "Market rates.",
      category: "MarketData",
      capabilities: [],
      tools: [],
    }),
  ];

  it("matches every word against the name, vendor, capabilities and tools", () => {
    const found = filterExtensions(items, {
      query: "government exa",
      category: ALL_CATEGORIES,
      status: "all",
    });
    expect(found.map((item) => item.type)).toEqual(["Exa"]);

    const byTool = filterExtensions(items, {
      query: "web_search",
      category: ALL_CATEGORIES,
      status: "all",
    });
    expect(byTool.map((item) => item.type)).toEqual(["Exa"]);
  });

  it("filters by category and by whether the extension is on", () => {
    expect(
      filterExtensions(items, { query: "", category: "MarketData", status: "all" }).map(
        (i) => i.type,
      ),
    ).toEqual(["Other"]);
    expect(
      filterExtensions(items, { query: "", category: ALL_CATEGORIES, status: "on" }).map(
        (i) => i.type,
      ),
    ).toEqual(["Exa"]);
    expect(
      filterExtensions(items, { query: "", category: ALL_CATEGORIES, status: "off" }).map(
        (i) => i.type,
      ),
    ).toEqual(["Other"]);
  });

  it("counts an extension that still needs setup as off", () => {
    const needsSetup = [extension({ enabled: true, configured: false })];
    expect(
      filterExtensions(needsSetup, { query: "", category: ALL_CATEGORIES, status: "off" }),
    ).toHaveLength(1);
  });
});

describe("sortExtensions", () => {
  const items = [
    extension({ type: "b", name: "Beta", sortOrder: 5, releasedAt: 300 }),
    extension({ type: "a", name: "Alpha", sortOrder: 20, releasedAt: 100, featured: true }),
    extension({ type: "c", name: "Gamma", sortOrder: 1, releasedAt: 200 }),
  ];

  it("puts featured first, then the catalog order", () => {
    expect(sortExtensions(items, "featured").map((i) => i.type)).toEqual(["a", "c", "b"]);
  });

  it("sorts by name and by release", () => {
    expect(sortExtensions(items, "name").map((i) => i.type)).toEqual(["a", "b", "c"]);
    expect(sortExtensions(items, "newest").map((i) => i.type)).toEqual(["b", "c", "a"]);
  });

  it("does not reorder the list it was given", () => {
    sortExtensions(items, "name");
    expect(items.map((i) => i.type)).toEqual(["b", "a", "c"]);
  });
});

describe("dailyUsageShare", () => {
  it("is the share of the day's limit used, capped at one", () => {
    const usage = { requestsThisMonth: 0, failuresThisMonth: 0, costThisMonthUsd: 0 };
    expect(dailyUsageShare(extension({ usage: { ...usage, requestsToday: 50 } }))).toBe(0.25);
    expect(dailyUsageShare(extension({ usage: { ...usage, requestsToday: 900 } }))).toBe(1);
    expect(dailyUsageShare(extension({ dailyRequestLimit: 0 }))).toBe(0);
  });
});
