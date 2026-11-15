import { unavailableFeatures } from "@/test/plan";
import { describe, expect, it } from "vitest";
import { columnsOnPlan } from "../plan-columns";

const columns = [
  { id: "name", meta: { label: "Name" } },
  { id: "compliance", meta: { label: "Compliance", planFeature: "workforce.compliance" } },
  { id: "safety", meta: { label: "Safety", planFeature: "workforce.safety" } },
  { id: "select" },
];

describe("columnsOnPlan", () => {
  it("leaves out a column whose add-on the plan does not include", () => {
    expect(
      columnsOnPlan(columns, unavailableFeatures("workforce.compliance")).map((c) => c.id),
    ).toEqual(["name", "safety", "select"]);
  });

  it("keeps every column, as the same array, when the plan leaves nothing out", () => {
    expect(columnsOnPlan(columns, [])).toBe(columns);
  });
});
