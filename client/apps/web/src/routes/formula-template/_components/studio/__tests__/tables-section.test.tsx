import type { FormulaTemplateTable } from "@/lib/graphql/formula-template-tables";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { studioTableRows, TablePinState } from "../sections/tables-section";

function pin(overrides: Partial<FormulaTemplateTable> = {}): FormulaTemplateTable {
  return {
    code: "Lanes",
    rateMatrixId: "rmx_1",
    versionNumber: 2,
    latestVersionNumber: 2,
    archived: false,
    ...overrides,
  };
}

describe("studioTableRows", () => {
  it("joins what the check reads with what the template pins, by code", () => {
    const rows = studioTableRows(
      {
        tables: [
          { code: "Zones", version: 5 },
          { code: "Lanes", version: 3 },
        ],
      },
      {
        tables: [
          {
            code: "Lanes",
            name: "Lane tariff",
            version: 3,
            axes: [],
            valueType: "Money",
            cells: undefined,
            loaded: undefined,
          },
          {
            code: "Zones",
            name: "Zones",
            version: 5,
            axes: [],
            valueType: "Money",
            cells: undefined,
            loaded: undefined,
          },
        ],
      },
      [
        pin({ versionNumber: 2, latestVersionNumber: 3 }),
        pin({ code: "Old", rateMatrixId: "rmx_9" }),
      ],
    );

    expect(rows).toEqual([
      {
        code: "Lanes",
        name: "Lane tariff",
        reads: 3,
        pin: pin({ versionNumber: 2, latestVersionNumber: 3 }),
      },
      { code: "Old", name: "", reads: null, pin: pin({ code: "Old", rateMatrixId: "rmx_9" }) },
      { code: "Zones", name: "Zones", reads: 5, pin: null },
    ]);
  });

  it("is empty for a tariff that reads no tables and pins none", () => {
    expect(studioTableRows({ tables: [] }, null, [])).toEqual([]);
  });
});

describe("TablePinState", () => {
  const row = { code: "Lanes", name: "Lane tariff", reads: 3 };

  it("says a newer published version is out and what contracts keep pricing with", () => {
    render(
      <TablePinState row={{ ...row, pin: pin({ versionNumber: 2, latestVersionNumber: 3 }) }} />,
    );

    expect(screen.getByText("Newer version")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Pinned v2; v3 is published. Contracts keep pricing with v2 until a new template version is approved.",
      ),
    ).toBeInTheDocument();
  });

  it("says the pin is the newest when nothing newer is published", () => {
    render(
      <TablePinState row={{ ...row, pin: pin({ versionNumber: 3, latestVersionNumber: 3 }) }} />,
    );

    expect(screen.getByText("Pinned v3, the newest published version.")).toBeInTheDocument();
    expect(screen.queryByText("Newer version")).toBeNull();
  });

  it("warns that an archived table can be read by no new template version", () => {
    render(
      <TablePinState
        row={{ ...row, pin: pin({ versionNumber: 2, latestVersionNumber: 4, archived: true }) }}
      />,
    );

    expect(screen.getByText("Archived")).toBeInTheDocument();
    expect(screen.queryByText("Newer version")).toBeNull();
  });

  it("says saving pins the version when the template has not pinned the table yet", () => {
    render(<TablePinState row={{ ...row, pin: null }} />);

    expect(screen.getByText("Saving pins the version the check reads.")).toBeInTheDocument();
  });
});
