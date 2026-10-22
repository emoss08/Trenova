import type { FormulaLibraryRow } from "@/lib/graphql/formula-library";
import { render } from "@testing-library/react";
import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { getColumns } from "../formula-library-columns";

function cellFor(accessorKey: string, original: Partial<FormulaLibraryRow>) {
  const column = getColumns(translate).find(
    (c) => "accessorKey" in c && c.accessorKey === accessorKey,
  );
  if (!column || typeof column.cell !== "function") {
    throw new Error(`no renderable column ${accessorKey}`);
  }
  const node = column.cell({ row: { original } } as never);
  return render(<>{node}</>).container.textContent ?? "";
}

describe("formula library list columns", () => {
  it("names a library by the path templates import it with, and its description", () => {
    expect(cellFor("name", { name: "accessorials", description: "Fees we share" })).toBe(
      "org.accessorialsFees we share",
    );
    expect(cellFor("name", { name: "accessorials", description: "" })).toBe("org.accessorials");
  });

  it("shows the approved version, or that none was ever approved", () => {
    expect(cellFor("approvedVersionNumber", { approvedVersionNumber: 4 })).toBe("v4");
    expect(cellFor("approvedVersionNumber", { approvedVersionNumber: null })).toBe("Never");
  });

  it("counts tests and the template versions that import it", () => {
    expect(cellFor("testCount", { testCount: 3 })).toBe("3");
    expect(cellFor("testCount", { testCount: 0 })).toBe("None");
    expect(cellFor("usageCount", { usageCount: 1 })).toBe("1 template version");
    expect(cellFor("usageCount", { usageCount: 5 })).toBe("5 template versions");
    expect(cellFor("usageCount", { usageCount: 0 })).toBe("Not imported");
  });
});
