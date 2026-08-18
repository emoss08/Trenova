import type { Row } from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
import { describe, expect, it, vi } from "vitest";
import { buildShipmentRowActions, type ShipmentRowActionHandlers } from "../row-actions";

function makeHandlers(overrides?: Partial<ShipmentRowActionHandlers>): ShipmentRowActionHandlers {
  return {
    onEdit: vi.fn(),
    onDuplicate: vi.fn(),
    onCancel: vi.fn(),
    onUncancel: vi.fn(),
    onTransferOwnership: vi.fn(),
    onTransferToBilling: vi.fn(),
    onSendEDI: vi.fn(),
    canSendEDI: true,
    ...overrides,
  };
}

function makeRow(shipment: Partial<Shipment>): Row<Shipment> {
  return { original: shipment as Shipment } as Row<Shipment>;
}

function sendEDIAction(handlers: ShipmentRowActionHandlers) {
  const action = buildShipmentRowActions(handlers).find((a) => a.id === "send-edi-load-tender");
  if (!action) throw new Error("send-edi-load-tender action is not registered");
  return action;
}

const ediPartner = { id: "edip_1", name: "Acme Logistics", code: "ACME" };

describe("send-edi-load-tender row action", () => {
  it("is hidden when the shipment customer has no linked EDI partner", () => {
    const action = sendEDIAction(makeHandlers());
    const row = makeRow({
      id: "sp_1",
      status: "New",
      tenderStatus: undefined,
      customer: { id: "cus_1", name: "No EDI Inc", ediPartner: null } as Shipment["customer"],
    });

    expect(action.hidden?.(row)).toBe(true);
  });

  it("is hidden when the shipment has no customer attached", () => {
    const action = sendEDIAction(makeHandlers());
    const row = makeRow({ id: "sp_1", status: "New", tenderStatus: undefined, customer: null });

    expect(action.hidden?.(row)).toBe(true);
  });

  it("is visible when the customer is linked to an EDI partner", () => {
    const action = sendEDIAction(makeHandlers());
    const row = makeRow({
      id: "sp_1",
      status: "New",
      tenderStatus: undefined,
      customer: { id: "cus_1", name: "EDI Inc", ediPartner } as Shipment["customer"],
    });

    expect(action.hidden?.(row)).toBe(false);
  });

  it("stays hidden without the EDI create permission", () => {
    const action = sendEDIAction(makeHandlers({ canSendEDI: false }));
    const row = makeRow({
      id: "sp_1",
      status: "New",
      tenderStatus: undefined,
      customer: { id: "cus_1", name: "EDI Inc", ediPartner } as Shipment["customer"],
    });

    expect(action.hidden?.(row)).toBe(true);
  });

  it("stays hidden when the shipment is not in New status", () => {
    const action = sendEDIAction(makeHandlers());
    const row = makeRow({
      id: "sp_1",
      status: "InTransit",
      tenderStatus: undefined,
      customer: { id: "cus_1", name: "EDI Inc", ediPartner } as Shipment["customer"],
    });

    expect(action.hidden?.(row)).toBe(true);
  });

  it("stays hidden when the shipment is already tendered", () => {
    const action = sendEDIAction(makeHandlers());
    const row = makeRow({
      id: "sp_1",
      status: "New",
      tenderStatus: "Tendered",
      customer: { id: "cus_1", name: "EDI Inc", ediPartner } as Shipment["customer"],
    });

    expect(action.hidden?.(row)).toBe(true);
  });
});
