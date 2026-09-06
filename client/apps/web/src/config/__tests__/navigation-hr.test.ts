import { appModuleGroups, navigationConfig } from "@/config/navigation.config";
import { isNavGroup, type NavGroup, type NavItem } from "@/config/navigation.types";
import { describe, expect, it } from "vitest";

function flatten(entries: (NavItem | NavGroup)[]): NavItem[] {
  return entries.flatMap((entry) => (isNavGroup(entry) ? entry.items : [entry]));
}

function moduleById(id: string) {
  const found = navigationConfig.modules.find((module) => module.id === id);
  if (!found) throw new Error(`module ${id} is not registered`);
  return found;
}

/** Everything the HR module owns, by nav id. */
const HR_ITEM_IDS = [
  "workers",
  "credential-types",
  "training-courses",
  "checklist-templates",
  "review-templates",
  "pto-policies",
  "holidays",
];

describe("human resource management module", () => {
  it("is registered with its own base path and sits in the operations group", () => {
    const hr = moduleById("hr");
    expect(hr.label).toBe("Human Resource Management");
    expect(hr.basePath).toBe("/hr");

    const operations = appModuleGroups.find((group) => group.id === "operations");
    expect(operations?.moduleIds).toContain("hr");
  });

  it("owns the worker record and every people-related configuration file", () => {
    const ids = flatten(moduleById("hr").navigation).map((item) => item.id);
    for (const id of HR_ITEM_IDS) {
      expect(ids).toContain(id);
    }
  });

  it("puts every one of its pages under /hr", () => {
    for (const item of flatten(moduleById("hr").navigation)) {
      expect(item.path?.startsWith("/hr/")).toBe(true);
    }
  });

  it("leaves dispatch with operations only", () => {
    const dispatch = moduleById("dispatch");
    const ids = flatten(dispatch.navigation).map((item) => item.id);
    for (const id of HR_ITEM_IDS) {
      expect(ids).not.toContain(id);
    }
    expect(ids).toEqual(expect.arrayContaining(["dispatch-console", "locations", "fleet-codes"]));
    for (const item of flatten(dispatch.navigation)) {
      expect(item.path?.startsWith("/dispatch/")).toBe(true);
    }
  });

  it("points the worker quick actions at the HR module", () => {
    const workerActions = (navigationConfig.quickActions ?? []).filter((action) =>
      ["create-worker", "request-worker-pto"].includes(action.id),
    );
    expect(workerActions).toHaveLength(2);
    for (const action of workerActions) {
      expect(action.path).toBe("/hr/workers");
    }
  });
});
