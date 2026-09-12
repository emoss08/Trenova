import type {
  ShipmentBillingAction,
  ShipmentBillingActions,
} from "@/hooks/use-shipment-billing-actions";
import type { Row } from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
import { SendIcon } from "lucide-react";
import { describe, expect, it, vi } from "vitest";
import { buildShipmentRowActions, type ShipmentRowActionHandlers } from "../row-actions";

function billingAction(id: string, isAvailable: (shipment: Shipment) => boolean) {
  return {
    id,
    label: id,
    icon: SendIcon,
    isAvailable,
    run: vi.fn(async (_shipmentId: string) => undefined),
  } satisfies ShipmentBillingAction;
}

function makeBillingActions() {
  return {
    markReadyToBill: billingAction("mark-ready-to-bill", (s) => s.status === "Completed"),
    markReadyAndTransferToBilling: billingAction(
      "mark-ready-and-transfer-to-billing",
      (s) => s.status === "Completed",
    ),
    transferToBilling: billingAction("transfer-to-billing", (s) => s.status === "ReadyToInvoice"),
  } satisfies ShipmentBillingActions;
}

function makeHandlers(overrides?: Partial<ShipmentRowActionHandlers>): ShipmentRowActionHandlers {
  return {
    onEdit: vi.fn(),
    onDuplicate: vi.fn(),
    onCancel: vi.fn(),
    onUncancel: vi.fn(),
    onTransferOwnership: vi.fn(),
    billingActions: makeBillingActions(),
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

describe("billing row actions", () => {
  const ids = [
    "mark-ready-to-bill",
    "mark-ready-and-transfer-to-billing",
    "transfer-to-billing",
  ] as const;

  function actionById(handlers: ShipmentRowActionHandlers, id: string) {
    const action = buildShipmentRowActions(handlers).find((a) => a.id === id);
    if (!action) throw new Error(`${id} action is not registered`);
    return action;
  }

  it("registers every billing action, each hidden by its own availability rule", () => {
    const handlers = makeHandlers();
    const completed = makeRow({ id: "sp_1", status: "Completed" });
    const ready = makeRow({ id: "sp_1", status: "ReadyToInvoice" });

    expect(actionById(handlers, "mark-ready-to-bill").hidden?.(completed)).toBe(false);
    expect(actionById(handlers, "mark-ready-to-bill").hidden?.(ready)).toBe(true);
    expect(actionById(handlers, "mark-ready-and-transfer-to-billing").hidden?.(completed)).toBe(
      false,
    );
    expect(actionById(handlers, "mark-ready-and-transfer-to-billing").hidden?.(ready)).toBe(true);
    expect(actionById(handlers, "transfer-to-billing").hidden?.(completed)).toBe(true);
    expect(actionById(handlers, "transfer-to-billing").hidden?.(ready)).toBe(false);
  });

  it("runs the billing action for the row's shipment and hands back its promise", async () => {
    const handlers = makeHandlers();
    const row = makeRow({ id: "sp_42", status: "Completed" });
    const billing = handlers.billingActions;
    const runs = {
      "mark-ready-to-bill": billing.markReadyToBill.run,
      "mark-ready-and-transfer-to-billing": billing.markReadyAndTransferToBilling.run,
      "transfer-to-billing": billing.transferToBilling.run,
    };

    for (const id of ids) {
      const result = actionById(handlers, id).onClick(row);

      expect(result).toBeInstanceOf(Promise);
      await result;
      expect(runs[id]).toHaveBeenCalledExactlyOnceWith("sp_42");
    }
  });

  it("does nothing for a row without a shipment id", () => {
    const handlers = makeHandlers();
    const row = makeRow({ status: "Completed" });

    expect(actionById(handlers, "mark-ready-to-bill").onClick(row)).toBeUndefined();
    expect(handlers.billingActions.markReadyToBill.run).not.toHaveBeenCalled();
  });
});
