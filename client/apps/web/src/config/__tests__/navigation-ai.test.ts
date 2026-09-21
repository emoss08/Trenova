import { adminLinks, navigationConfig } from "@/config/navigation.config";
import { isNavGroup, type NavGroup, type NavItem } from "@/config/navigation.types";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { describe, expect, it } from "vitest";

function flatten(entries: (NavItem | NavGroup)[]): NavItem[] {
  return entries.flatMap((entry) => (isNavGroup(entry) ? entry.items : [entry]));
}

function moduleById(id: string) {
  const found = navigationConfig.modules.find((module) => module.id === id);
  if (!found) throw new Error(`module ${id} is not registered`);
  return found;
}

/**
 * Every AI surface has to be reachable from the navigation, not only from a
 * URL somebody remembers. These pin where each one lives so a page cannot be
 * shipped as a route with no way to get to it.
 */
describe("AI surfaces in navigation", () => {
  it("files the insights page under Reports, gated on the insight permission", () => {
    const item = flatten(moduleById("reports").navigation).find(
      (entry) => entry.path === "/insights",
    );

    expect(item).toBeDefined();
    expect(item?.resource).toBe(Resource.Insight);
  });

  // Providers, agents and activity are tabs of one hub. A second settings link
  // for providers would send people to two places for the same configuration.
  it("lists AI Control once in the settings sidebar and no separate providers page", () => {
    const byHref = new Map(adminLinks.map((link) => [link.href, link]));

    const hub = byHref.get("/admin/agent-control");
    expect(hub?.title).toBe("AI control");
    expect(hub?.group).toBe("AI & Automation");
    expect(hub?.resource).toBe(Resource.AgentControl);
    expect(hub?.requiredOperation).toBe(Operation.Read);

    expect(adminLinks.some((link) => link.href.startsWith("/admin/ai-providers"))).toBe(false);
    expect(adminLinks.some((link) => link.href.startsWith("/admin/agents"))).toBe(false);
  });

  it("offers the assistant and the insights page from the command palette", () => {
    const actions = navigationConfig.quickActions ?? [];

    const assistant = actions.find((action) => action.id === "open-assistant");
    expect(assistant?.resource).toBe(Resource.Assistant);
    expect(assistant?.requiredOperation).toBe(Operation.Read);
    // The assistant is a floating panel, not a page: the palette opens it in
    // place rather than leaving whatever the person was looking at.
    expect(assistant?.action).toBe("open-assistant");

    const insights = actions.find((action) => action.id === "open-insights");
    expect(insights?.path).toBe("/insights");
  });
});
