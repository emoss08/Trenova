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

  it("shows the current version, and when it was approved or that it never was", () => {
    expect(cellFor("currentVersionNumber", { currentVersionNumber: 4 })).toBe("v4");
    expect(cellFor("approvedAt", { approvedAt: null })).toBe("Never");
    expect(cellFor("approvedAt", { approvedAt: 1_700_000_000 })).not.toContain("Never");
  });

  it("counts the tests its source holds", () => {
    expect(cellFor("testCount", { testCount: 3 })).toBe("3");
    expect(cellFor("testCount", { testCount: 0 })).toBe("None");
  });

  it("lists the libraries it imports, or none", () => {
    const column = getColumns(translate).find((c) => c.id === "imports");
    if (!column || typeof column.cell !== "function") throw new Error("no imports column");
    const cell = column.cell;
    const text = (imports: FormulaLibraryRow["imports"]) =>
      render(<>{cell({ row: { original: { imports } } } as never)}</>).container.textContent;
    expect(text([{ path: "org.zones", libraryId: "fl_2", versionNumber: 2 }])).toBe("org.zones");
    expect(text([])).toBe("None");
  });
});
