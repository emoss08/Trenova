import { cleanup, render, screen } from "@testing-library/react";
import { translate } from "@trenova/shared/i18n/runtime";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import type { FuelCardRow } from "@/lib/graphql/fuel-card";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { getColumns } from "../fuel-card-columns";

afterEach(cleanup);

type Meta = { apiField?: string; exportValue?: (row: FuelCardRow) => string };

const row: FuelCardRow = {
  id: "fcrd_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  provider: "Comdata",
  lastFour: "4821",
  label: "Unit 118 card",
  externalCardId: null,
  assignedWorkerId: null,
  assignedTractorId: null,
  status: "Active",
  expiresAt: null,
  cancelledAt: null,
  cancelReason: null,
  notes: null,
  discoveredAt: null,
  version: 1,
  createdAt: 1,
  updatedAt: 2,
  assignedWorker: null,
  assignedTractor: null,
};

function cardColumn(): ColumnDef<FuelCardRow> {
  const column = getColumns(translate).find(
    (candidate) => (candidate.meta as Meta | undefined)?.apiField === "lastFour",
  );
  if (!column) {
    throw new Error("no card column");
  }
  return column;
}

describe("fuel card table card column", () => {
  it("shows the card's network beside its masked number", () => {
    const cell = cardColumn().cell as (context: { row: { original: FuelCardRow } }) => ReactNode;
    render(<>{cell({ row: { original: row } })}</>);

    expect(screen.getByRole("img", { name: "Comdata" })).toBeTruthy();
    expect(screen.getByText("•••• 4821")).toBeTruthy();
  });

  it("exports the masked number alone", () => {
    expect((cardColumn().meta as Meta).exportValue?.(row)).toBe("•••• 4821");
  });
});
