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

  it("groups the AI administration pages together in the settings sidebar", () => {
    const byHref = new Map(adminLinks.map((link) => [link.href, link]));

    const providers = byHref.get("/admin/ai-providers");
    expect(providers?.group).toBe("AI & Automation");
    expect(providers?.resource).toBe(Resource.AIProvider);
    expect(providers?.requiredOperation).toBe(Operation.Read);

    const agentControl = byHref.get("/admin/agent-control");
    expect(agentControl?.group).toBe("AI & Automation");
    expect(agentControl?.resource).toBe(Resource.AgentControl);
  });

  // Agents are configured from Agent Control rather than from a page of their
  // own, so nothing may point people at the retired route.
  it("does not list a separate agents page", () => {
    expect(adminLinks.some((link) => link.href.startsWith("/admin/agents"))).toBe(false);
  });

  it("offers the assistant and the insights page from the command palette", () => {
    const actions = navigationConfig.quickActions ?? [];

    const assistant = actions.find((action) => action.id === "open-assistant");
    expect(assistant?.path).toBe("/assistant");
    expect(assistant?.resource).toBe(Resource.Assistant);
    expect(assistant?.requiredOperation).toBe(Operation.Read);

    const insights = actions.find((action) => action.id === "open-insights");
    expect(insights?.path).toBe("/insights");
  });
});
