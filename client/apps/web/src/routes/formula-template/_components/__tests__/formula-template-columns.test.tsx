import type { FormulaTemplate } from "@trenova/shared/types/formula-template";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { getColumns } from "../formula-template-columns";

function cellFor(accessorKey: string, original: Partial<FormulaTemplate>) {
  const column = getColumns().find((c) => "accessorKey" in c && c.accessorKey === accessorKey);
  if (!column || typeof column.cell !== "function") {
    throw new Error(`no renderable column ${accessorKey}`);
  }
  const row = { original };
  const node = column.cell({ row } as never);
  return render(<>{node}</>).container.textContent ?? "";
}

describe("formula template list columns", () => {
  it("shows how many things rate with a template and how many scenarios guard it", () => {
    expect(cellFor("usageCount", { usageCount: 12 } as Partial<FormulaTemplate>)).toContain("12");
    expect(cellFor("usageCount", { usageCount: 0 } as Partial<FormulaTemplate>)).toContain(
      "Not in use",
    );
    expect(cellFor("scenarioCount", { scenarioCount: 3 } as Partial<FormulaTemplate>)).toContain(
      "3",
    );
    expect(cellFor("scenarioCount", { scenarioCount: 0 } as Partial<FormulaTemplate>)).toContain(
      "None",
    );
  });

  it("shows when a template was last approved, or that it never was", () => {
    expect(cellFor("approvedAt", { approvedAt: null })).toContain("Never");
    expect(cellFor("approvedAt", { approvedAt: 1_700_000_000 })).not.toContain("Never");
  });
});
