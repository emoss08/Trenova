import type { AgentScope } from "@/lib/agent-scope";
import type { AgentExtensionCatalogItem } from "@/types/agent-extension";
import { describe, expect, it } from "vitest";
import {
  ALL_CATEGORIES,
  capabilityHolder,
  capabilityServer,
  offersWebWatches,
  webWatchServer,
  dailyUsageShare,
  extensionState,
  filterExtensions,
  sortExtensions,
  holdsExtension,
  withExtensionTools,
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
    capability: "WebResearch",
    capabilityLabel: "Web research",
    capabilityHeldBy: "",
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

describe("capabilityHolder", () => {
  it("names the vendor that already serves the capability while this one is off", () => {
    expect(capabilityHolder(extension({ enabled: false, capabilityHeldBy: "Exa" }))).toBe("Exa");
  });

  it("is empty when nothing else serves the capability", () => {
    expect(capabilityHolder(extension({ enabled: false, capabilityHeldBy: "" }))).toBe("");
  });

  it("does not stand in the way of an extension that is already on", () => {
    expect(capabilityHolder(extension({ enabled: true, capabilityHeldBy: "Exa" }))).toBe("");
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

/**
 * An agent holds an extension when it holds every tool the extension adds; giving it the
 * extension adds the tools it lacks, and taking it away removes only the extension's.
 */
describe("holdsExtension and withExtensionTools", () => {
  const tools = [
    { name: "web_search", label: "", description: "" },
    { name: "web_fetch", label: "", description: "" },
  ];
  const named = (capabilities: string[], over: Partial<AgentScope> = {}): AgentScope => ({
    areas: [],
    capabilities,
    denyAreas: [],
    denyCapabilities: [],
    ...over,
  });

  it("holds only when the scope allows every one of the extension's tools", () => {
    expect(holdsExtension(named(["web_search", "web_fetch", "shipment_list"]), tools)).toBe(true);
    expect(holdsExtension(named(["web_search", "shipment_list"]), tools)).toBe(false);
    expect(holdsExtension(named([], { areas: ["web"] }), tools)).toBe(true);
    expect(holdsExtension(named([]), tools)).toBe(true);
    expect(holdsExtension(named([], { denyCapabilities: ["web_fetch"] }), tools)).toBe(false);
    expect(holdsExtension(named(["shipment_list"]), [])).toBe(false);
  });

  it("names the missing tools once and keeps the agent's own order", () => {
    expect(withExtensionTools(named(["shipment_list", "web_search"]), tools, true)).toEqual(
      named(["shipment_list", "web_search", "web_fetch"]),
    );
  });

  it("lifts a switch-off rather than naming a tool everything already allows", () => {
    expect(withExtensionTools(named([], { denyCapabilities: ["web_fetch"] }), tools, true)).toEqual(
      named([]),
    );
  });

  it("takes only the extension's tools, switching off what an area still gives", () => {
    expect(
      withExtensionTools(named(["web_search", "shipment_list", "web_fetch"]), tools, false),
    ).toEqual(named(["shipment_list"]));
    expect(withExtensionTools(named([], { areas: ["web"] }), tools, false)).toEqual(
      named([], { areas: ["web"], denyCapabilities: ["web_search", "web_fetch"] }),
    );
  });
});

describe("capabilityServer", () => {
  const exa = extension({ type: "Exa", capability: "WebResearch" });
  const parallel = extension({ type: "Parallel", capability: "WebResearch", enabled: true });
  const browserUse = extension({ type: "BrowserUse", capability: "PortalAutomation" });

  it("is the extension turned on for the capability", () => {
    expect(capabilityServer([exa, parallel, browserUse], "WebResearch")).toBe(parallel);
  });

  it("is the first that provides it when none is on", () => {
    expect(capabilityServer([exa, parallel, browserUse], "PortalAutomation")).toBe(browserUse);
  });

  it("is nothing when no extension provides it", () => {
    expect(capabilityServer([exa, parallel], "PortalAutomation")).toBeNull();
  });
});

describe("webWatchServer", () => {
  const watching = [
    { name: "web_search", label: "Search the web", description: "" },
    { name: "web_watch", label: "Watch the web", description: "" },
  ];
  const exa = extension({ type: "Exa", capability: "WebResearch", tools: watching });
  const parallel = extension({ type: "Parallel", capability: "WebResearch", enabled: true });

  it("is the vendor that can watch, even while another serves web research", () => {
    expect(offersWebWatches(exa)).toBe(true);
    expect(offersWebWatches(parallel)).toBe(false);
    expect(webWatchServer([parallel, exa])).toBe(exa);
  });

  it("prefers the one turned on", () => {
    const exaOn = extension({
      type: "Exa",
      capability: "WebResearch",
      tools: watching,
      enabled: true,
    });
    expect(webWatchServer([exa, exaOn])).toBe(exaOn);
  });

  it("is nothing when no vendor can watch", () => {
    expect(webWatchServer([parallel])).toBeNull();
  });
});
