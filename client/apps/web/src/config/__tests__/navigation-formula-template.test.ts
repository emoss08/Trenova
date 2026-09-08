import { navigationConfig } from "@/config/navigation.config";
import { formulaTemplateRoutes } from "@/lib/formula-template-routes";
import { describe, expect, it } from "vitest";

describe("create formula template quick action", () => {
  it("opens the Formula Studio create route rather than the retired list panel", () => {
    const action = (navigationConfig.quickActions ?? []).find(
      (entry) => entry.id === "create-formula-template",
    );
    if (!action) throw new Error("create-formula-template quick action is not registered");

    expect(action.path).toBe(formulaTemplateRoutes.new);
    expect(action.query).toBeUndefined();
  });
});
