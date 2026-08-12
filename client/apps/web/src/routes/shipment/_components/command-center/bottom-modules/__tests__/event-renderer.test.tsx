import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import type { ShipmentEvent } from "@/types/shipment-event";
import { renderEvent } from "../event-renderer";

function baseEvent(overrides: Partial<ShipmentEvent> = {}): ShipmentEvent {
  return {
    id: "se_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    shipmentId: "shp_1",
    type: "ShipmentCreated",
    severity: "muted",
    actorType: "user",
    actorLabel: "",
    summary: "Shipment created",
    metadata: {},
    occurredAt: 1_700_000_000,
    actor: { name: "System Administrator", username: "sysadmin" },
    shipment: { id: "shp_1", proNumber: "PRO-2026-1042" },
    ...overrides,
  };
}

function harness(rendered: ReturnType<typeof renderEvent>) {
  return (
    <div>
      <div data-testid="headline">{rendered.headline}</div>
      {rendered.detail !== undefined && <div data-testid="detail">{rendered.detail}</div>}
      <div data-testid="handle">{rendered.actorHandle}</div>
    </div>
  );
}

describe("renderEvent", () => {
  afterEach(() => cleanup());

  it("renders comments with actor + target headline and the body as detail", () => {
    const result = renderEvent(
      baseEvent({
        type: "CommentPosted",
        severity: "info",
        metadata: { commentBody: "hello @ops-night" },
      }),
    );

    render(harness(result));

    const headline = screen.getByTestId("headline").textContent;
    expect(headline).toBe("System Administrator added a comment to #PRO-2026-1042");
    expect(screen.getByTestId("detail").textContent).toBe("hello @ops-night");
    expect(screen.getByTestId("handle").textContent).toBe("@sysadmin");
  });

  it("renders status changes with new status appended", () => {
    const result = renderEvent(
      baseEvent({
        type: "StatusChanged",
        metadata: { previousStatus: "New", newStatus: "InTransit" },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator marked #PRO-2026-1042 as InTransit",
    );
  });

  it("renders driver assignment with driver name from metadata", () => {
    const result = renderEvent(
      baseEvent({
        type: "DriverAssigned",
        metadata: { driverName: "S. Ndiaye" },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator assigned S. Ndiaye to #PRO-2026-1042",
    );
  });

  it("renders cancellation reason on the detail line", () => {
    const result = renderEvent(
      baseEvent({
        type: "ShipmentCanceled",
        severity: "danger",
        metadata: { reason: "Customer request" },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator canceled #PRO-2026-1042",
    );
    expect(screen.getByTestId("detail").textContent).toBe("Reason: Customer request");
  });

  it("renders hold placement with the hold type woven in", () => {
    const result = renderEvent(
      baseEvent({
        type: "HoldPlaced",
        severity: "danger",
        metadata: { holdType: "Operational" },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator placed a Operational hold on #PRO-2026-1042",
    );
  });

  it("falls back to system actor label when no user is attached", () => {
    const result = renderEvent(
      baseEvent({
        type: "MoveDeparted",
        actorType: "system",
        actor: undefined,
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System dispatched a move on #PRO-2026-1042",
    );
    expect(screen.getByTestId("handle").textContent).toBe("system");
  });

  it("falls back to a shipment without pro number", () => {
    const result = renderEvent(
      baseEvent({
        type: "ShipmentCreated",
        shipment: undefined,
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator created a shipment",
    );
  });

  it("renders carrier assignment with the carrier name from metadata", () => {
    const result = renderEvent(
      baseEvent({
        type: "CarrierAssigned",
        metadata: {
          carrierName: "Blue Ridge Freight",
          carrierId: "car_01",
          totalCost: "2450.00",
          proNumber: "BRF-88213",
        },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator assigned Blue Ridge Freight to #PRO-2026-1042",
    );
  });

  it("renders carrier unassignment with the reason on the detail line", () => {
    const result = renderEvent(
      baseEvent({
        type: "CarrierUnassigned",
        metadata: {
          carrierName: "Blue Ridge Freight",
          reason: "Move was covered outside the tender",
        },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator unassigned Blue Ridge Freight from #PRO-2026-1042",
    );
    expect(screen.getByTestId("detail").textContent).toBe(
      "Reason: Move was covered outside the tender",
    );
  });

  it("renders a tender offer with the carrier and channel trail", () => {
    const result = renderEvent(
      baseEvent({
        type: "TenderOffered",
        severity: "info",
        metadata: {
          tenderId: "tdr_01",
          moveId: "smv_01",
          offerId: "tof_01",
          carrierName: "Blue Ridge Freight",
          rank: 1,
          channel: "Email",
        },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator offered #PRO-2026-1042 to Blue Ridge Freight via Email",
    );
  });

  it("renders a tender offer without a channel when the metadata omits it", () => {
    const result = renderEvent(
      baseEvent({
        type: "TenderOffered",
        severity: "info",
        metadata: { tenderId: "tdr_01", carrierName: "Blue Ridge Freight", rank: 2 },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator offered #PRO-2026-1042 to Blue Ridge Freight",
    );
  });

  it("renders a tender acceptance with the carrier as the subject", () => {
    const result = renderEvent(
      baseEvent({
        type: "TenderAccepted",
        severity: "success",
        actorType: "system",
        actor: undefined,
        actorLabel: "System",
        metadata: {
          tenderId: "tdr_01",
          offerId: "tof_01",
          carrierName: "Blue Ridge Freight",
          source: "Email",
        },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "Blue Ridge Freight accepted the tender for #PRO-2026-1042",
    );
    expect(screen.getByTestId("handle").textContent).toBe("System");
  });

  it("renders a tender decline with the quoted reason as detail", () => {
    const result = renderEvent(
      baseEvent({
        type: "TenderDeclined",
        severity: "info",
        metadata: {
          tenderId: "tdr_01",
          carrierName: "Blue Ridge Freight",
          reason: "No available power",
          source: "EDI",
        },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "Blue Ridge Freight declined the tender for #PRO-2026-1042",
    );
    expect(screen.getByTestId("detail").textContent).toBe('"No available power"');
  });

  it("renders a tender decline with an empty reason and no detail line", () => {
    const result = renderEvent(
      baseEvent({
        type: "TenderDeclined",
        severity: "info",
        metadata: {
          tenderId: "tdr_01",
          carrierName: "Blue Ridge Freight",
          reason: "",
          source: "Manual",
        },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "Blue Ridge Freight declined the tender for #PRO-2026-1042",
    );
    expect(screen.queryByTestId("detail")).toBeNull();
  });

  it("renders an expired offer keyed on the carrier", () => {
    const result = renderEvent(
      baseEvent({
        type: "TenderExpired",
        severity: "muted",
        metadata: { tenderId: "tdr_01", carrierName: "Blue Ridge Freight" },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "Offer to Blue Ridge Freight expired on #PRO-2026-1042",
    );
  });

  it("renders a tender withdrawal with the reason as detail", () => {
    const result = renderEvent(
      baseEvent({
        type: "TenderWithdrawn",
        severity: "muted",
        metadata: { tenderId: "tdr_01", reason: "Move was covered outside the tender" },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator withdrew the tender on #PRO-2026-1042",
    );
    expect(screen.getByTestId("detail").textContent).toBe(
      "Reason: Move was covered outside the tender",
    );
  });

  it("renders a needs-review flag with the carrier and reason", () => {
    const result = renderEvent(
      baseEvent({
        type: "TenderNeedsReview",
        severity: "danger",
        metadata: {
          tenderId: "tdr_01",
          carrierName: "Blue Ridge Freight",
          reason: "Carrier insurance expired before pickup",
        },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator flagged Blue Ridge Freight's acceptance on #PRO-2026-1042 for review",
    );
    expect(screen.getByTestId("detail").textContent).toBe(
      "Reason: Carrier insurance expired before pickup",
    );
  });

  it("names the routing guide when the waterfall mode is exhausted", () => {
    const result = renderEvent(
      baseEvent({
        type: "RoutingGuideExhausted",
        severity: "danger",
        metadata: { tenderId: "tdr_01", mode: "Waterfall" },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator exhausted the routing guide on #PRO-2026-1042",
    );
  });

  it("names the spot tender when a non-waterfall mode is exhausted", () => {
    const result = renderEvent(
      baseEvent({
        type: "RoutingGuideExhausted",
        severity: "danger",
        metadata: { tenderId: "tdr_01", mode: "SpotBroadcast" },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator exhausted the spot tender on #PRO-2026-1042",
    );
  });

  it("renders a late response with its action and carrier", () => {
    const result = renderEvent(
      baseEvent({
        type: "TenderLateResponse",
        severity: "muted",
        metadata: {
          tenderId: "tdr_01",
          carrierName: "Blue Ridge Freight",
          action: "Decline",
          source: "Email",
        },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator recorded a late Decline from Blue Ridge Freight on #PRO-2026-1042",
    );
  });

  it("renders a delivery failure with the transport error as detail", () => {
    const result = renderEvent(
      baseEvent({
        type: "TenderDeliveryFailed",
        severity: "danger",
        metadata: {
          tenderId: "tdr_01",
          carrierName: "Blue Ridge Freight",
          channel: "Email",
          error: "smtp: 550 recipient rejected",
        },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator could not deliver the offer to Blue Ridge Freight for #PRO-2026-1042",
    );
    expect(screen.getByTestId("detail").textContent).toBe("smtp: 550 recipient rejected");
  });

  it("renders a skipped guide entry with its rank and joined reasons", () => {
    const result = renderEvent(
      baseEvent({
        type: "TenderEntrySkipped",
        severity: "danger",
        metadata: {
          tenderId: "tdr_01",
          carrierName: "Sunset Logistics",
          rank: 3,
          reasons: ["Insurance policy expired", "Compliance status is Unqualified"],
        },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator skipped Sunset Logistics (rank 3) on #PRO-2026-1042",
    );
    expect(screen.getByTestId("detail").textContent).toBe(
      "Insurance policy expired; Compliance status is Unqualified",
    );
  });

  it("renders a warned guide entry with its rank and joined warnings", () => {
    const result = renderEvent(
      baseEvent({
        type: "TenderEntryWarned",
        severity: "info",
        metadata: {
          tenderId: "tdr_01",
          carrierName: "Sunset Logistics",
          rank: 2,
          warnings: ["Auto liability policy expires in 12 days"],
        },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator tendered Sunset Logistics (rank 2) with warnings on #PRO-2026-1042",
    );
    expect(screen.getByTestId("detail").textContent).toBe(
      "Auto liability policy expires in 12 days",
    );
  });

  it("omits the rank suffix when a guide-entry event carries no rank", () => {
    const result = renderEvent(
      baseEvent({
        type: "TenderEntrySkipped",
        severity: "danger",
        metadata: { tenderId: "tdr_01", carrierName: "Sunset Logistics", reasons: [] },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator skipped Sunset Logistics on #PRO-2026-1042",
    );
    expect(screen.queryByTestId("detail")).toBeNull();
  });

  it("falls back to a generic carrier when tender metadata omits the carrier name", () => {
    const result = renderEvent(
      baseEvent({
        type: "TenderAccepted",
        severity: "success",
        metadata: { tenderId: "tdr_01" },
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "a carrier accepted the tender for #PRO-2026-1042",
    );
  });

  it("falls back to summary for unknown event types", () => {
    const result = renderEvent(
      baseEvent({
        // @ts-expect-error intentional unknown type
        type: "FutureUnknownType",
        summary: "Something happened",
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe("Something happened");
  });
});
