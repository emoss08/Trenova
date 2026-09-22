import { agentRunPath, customerPath, shipmentPath } from "@/lib/record-paths";
import { routes } from "@/router";
import type { RouteObject } from "react-router";
import { describe, expect, it } from "vitest";

function joinPath(parent: string, child: string): string {
  return child.startsWith("/") ? child : `${parent.replace(/\/$/, "")}/${child}`;
}

function registered(entries: RouteObject[], parent = "", found = new Set<string>()): Set<string> {
  for (const entry of entries) {
    const path = entry.path === undefined ? parent : joinPath(parent, entry.path);
    if (entry.path !== undefined) found.add(path);
    if (entry.children) registered(entry.children, path, found);
  }
  return found;
}

/**
 * A link to a record is only as good as the route it names. The inbox's first
 * version linked to a customers page that did not exist and opened shipments
 * with parameters the shipments page does not read, so both links landed on
 * something other than the record they promised.
 */
describe("record paths", () => {
  const paths = registered(routes);

  it("names a registered page and the panel parameters that page reads", () => {
    for (const build of [shipmentPath, customerPath]) {
      const url = new URL(build("rec_1"), "http://localhost");
      expect(paths.has(url.pathname)).toBe(true);
      expect(url.searchParams.get("panelType")).toBe("edit");
      expect(url.searchParams.get("panelEntityId")).toBe("rec_1");
    }
  });

  it("opens a run in AI Control's activity list filtered to that run", () => {
    const url = new URL(agentRunPath("run_1"), "http://localhost");
    expect(paths.has(url.pathname)).toBe(true);
    expect(url.searchParams.get("tab")).toBe("activity");
    expect(url.searchParams.get("activity")).toBe("runs");
    expect(JSON.parse(url.searchParams.get("fieldFilters") ?? "[]")).toEqual([
      { field: "id", operator: "eq", value: "run_1" },
    ]);
  });

  it("encodes the id rather than trusting it", () => {
    expect(shipmentPath("a&b=c")).toContain("panelEntityId=a%26b%3Dc");
  });
});
