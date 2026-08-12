import { getColumns } from "@/routes/shipment/_components/shipment-columns";
import { describe, expect, it } from "vitest";

describe("shipment coverage column", () => {
  const column = getColumns([]).find((entry) => entry.id === "driver");

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
