import { translate } from "@trenova/shared/i18n/runtime";
import { getColumns } from "../shipment-columns";
import { describe, expect, it } from "vitest";

describe("shipment coverage column", () => {
  const column = getColumns({ rowActions: [], t: translate, expandedRowId: null, onToggleExpanded: () => {} }).find((entry) => entry.id === "driver");

  it("is registered", () => {
    expect(column).toBeDefined();
  });

  /**
   * The cell under this header renders a driver, an external carrier, or
   * nothing at all, so naming it after one of the three was already wrong for a
   * brokered move and is wrong for every move at an organization without
   * drivers.
   */
  it("names what the column actually holds rather than assuming a driver", () => {
    expect(column?.header).toBe("Coverage");
    expect(column?.meta?.label).toBe("Coverage");
  });
});

describe("shipment board columns", () => {
  const columns = getColumns({ rowActions: [], t: translate, expandedRowId: null, onToggleExpanded: () => {} });

  it("lays the columns out in the board's order, lane first and actions last", () => {
    expect(columns.map((column) => column.id)).toEqual([
      "lane",
      "status",
      "tenderStatus",
      "billing",
      "proBol",
      "order",
      "customer",
      "driver",
      "eta",
      "pickupAppointment",
      "deliveryAppointment",
      "revenue",
      "margin",
      "actions",
    ]);
  });

  it("locks the lane column and exports the lane and coverage as text", () => {
    const lane = columns.find((column) => column.id === "lane");
    expect(lane?.enableHiding).toBe(false);
    const shipment = {
      moves: [
        {
          stops: [
            { type: "Pickup", location: { city: "Des Moines" } },
            { type: "Delivery", location: { city: "Chicago" } },
          ],
          assignment: { primaryWorker: { id: "w", firstName: "Jaime", lastName: "Park" } },
        },
      ],
    };
    expect(lane?.meta?.exportValue?.(shipment)).toBe("Des Moines → Chicago");
    expect(columns.find((column) => column.id === "driver")?.meta?.exportValue?.(shipment)).toBe(
      "Jaime Park",
    );
  });
});
