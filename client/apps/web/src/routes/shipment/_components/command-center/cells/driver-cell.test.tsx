import { cleanup, render, screen } from "@testing-library/react";
import type { Shipment, ShipmentMove } from "@trenova/shared/types/shipment";
import { afterEach, describe, expect, it } from "vitest";
import { DriverCell } from "./driver-cell";

function makeShipment(move: Partial<ShipmentMove>): Shipment {
  return {
    id: "shp_1",
    status: "New",
    moves: [
      {
        id: "mov_1",
        status: "New",
        sequence: 0,
        loaded: true,
        stops: [],
        ...move,
      },
    ],
  } as unknown as Shipment;
}

const carrierAssignment = {
  id: "casn_1",
  shipmentMoveId: "mov_1",
  carrierId: "car_1",
  status: "Confirmed",
  rateMethod: "Flat",
  carrier: { id: "car_1", name: "Acme Freight", scac: "ACME" },
} as unknown as ShipmentMove["carrierAssignment"];

describe("DriverCell", () => {
  afterEach(cleanup);

  it("never asks a brokerage for a driver it does not employ", () => {
    render(<DriverCell shipment={makeShipment({ coverageType: "unassigned" })} />);

    expect(screen.queryByText("Needs driver")).toBeNull();
  });

  it("renders an uncovered move as needing coverage rather than a driver", () => {
    render(<DriverCell shipment={makeShipment({ coverageType: "unassigned" })} />);

    expect(screen.getByText("Needs coverage")).toBeTruthy();
  });

  it("keeps an uncovered move uncovered even with a lingering carrier record", () => {
    render(
      <DriverCell shipment={makeShipment({ coverageType: "unassigned", carrierAssignment })} />,
    );

    expect(screen.getByText("Needs coverage")).toBeTruthy();
    expect(screen.queryByText("Acme Freight")).toBeNull();
  });

  it("renders the carrier only when the move is carrier covered", () => {
    render(<DriverCell shipment={makeShipment({ coverageType: "carrier", carrierAssignment })} />);

    expect(screen.getByText("Acme Freight")).toBeTruthy();
    expect(screen.queryByText("Needs coverage")).toBeNull();
  });

  it("renders the driver when the move is driver covered", () => {
    render(
      <DriverCell
        shipment={makeShipment({
          coverageType: "driver",
          assignment: {
            id: "asn_1",
            status: "New",
            primaryWorkerId: "wrk_1",
            primaryWorker: { id: "wrk_1", firstName: "Dana", lastName: "Reyes" },
            tractor: { id: "trc_1", code: "T-100" },
          },
        } as Partial<ShipmentMove>)}
      />,
    );

    expect(screen.getByText("D. Reyes")).toBeTruthy();
    expect(screen.queryByText("Needs coverage")).toBeNull();
  });
});
