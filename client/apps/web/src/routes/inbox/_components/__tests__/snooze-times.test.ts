import { messageRow, SHIPMENT } from "@/lib/triage/__tests__/message-fixture";
import { describe, expect, it } from "vitest";
import { eventSnoozeOptions } from "../snooze-times";

describe("event snoozes", () => {
  it("waits on events only on a linked shipment", () => {
    expect(eventSnoozeOptions(messageRow({ matchedShipment: null }))).toEqual([]);
  });

  it("offers delivery before it has happened and payment after", () => {
    expect(
      eventSnoozeOptions(messageRow({ classification: "Tender", matchedShipment: SHIPMENT })).map(
        (o) => o.event,
      ),
    ).toEqual(["ShipmentDelivered", "DocumentAttached"]);
    expect(
      eventSnoozeOptions(
        messageRow({ classification: "DetentionDispute", matchedShipment: SHIPMENT }),
      ).map((o) => o.event),
    ).toEqual(["DocumentAttached", "InvoicePaid"]);
  });
});
