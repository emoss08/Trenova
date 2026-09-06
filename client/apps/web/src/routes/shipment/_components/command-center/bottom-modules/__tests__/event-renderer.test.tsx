import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import type {
  ShipmentAssignmentEvent,
  ShipmentCarrierEvent,
  ShipmentCommentEvent,
  ShipmentHoldEvent,
  ShipmentLifecycleEvent,
  ShipmentMoveEvent,
  ShipmentOwnershipEvent,
  ShipmentTenderEvent,
} from "@/types/shipment-event";
import { renderEvent } from "../event-renderer";

const ENVELOPE = {
  id: "se_1",
  organizationId: "org_1",
  businessUnitId: "bu_1",
  shipmentId: "shp_1",
  severity: "muted",
  actorType: "user",
  actorLabel: "",
  summary: "Shipment created",
  metadata: {},
  occurredAt: 1_700_000_000,
  actor: { name: "System Administrator", username: "sysadmin" },
  shipment: { id: "shp_1", proNumber: "PRO-2026-1042" },
} satisfies Omit<ShipmentLifecycleEvent, "__typename" | "type">;

type WithType<E extends { type: string }> = Partial<E> & Pick<E, "type">;

function lifecycleEvent(overrides: Partial<ShipmentLifecycleEvent> = {}): ShipmentLifecycleEvent {
  return {
    ...ENVELOPE,
    __typename: "ShipmentLifecycleEvent",
    type: "ShipmentCreated",
    ...overrides,
  };
}

function ownershipEvent(overrides: Partial<ShipmentOwnershipEvent> = {}): ShipmentOwnershipEvent {
  return {
    ...ENVELOPE,
    __typename: "ShipmentOwnershipEvent",
    type: "OwnershipTransferred",
    ...overrides,
  };
}

function moveEvent(overrides: WithType<ShipmentMoveEvent>): ShipmentMoveEvent {
  return { ...ENVELOPE, __typename: "ShipmentMoveEvent", ...overrides };
}

function assignmentEvent(overrides: WithType<ShipmentAssignmentEvent>): ShipmentAssignmentEvent {
  return { ...ENVELOPE, __typename: "ShipmentAssignmentEvent", ...overrides };
}

function carrierEvent(overrides: WithType<ShipmentCarrierEvent>): ShipmentCarrierEvent {
  return { ...ENVELOPE, __typename: "ShipmentCarrierEvent", ...overrides };
}

function tenderEvent(overrides: WithType<ShipmentTenderEvent>): ShipmentTenderEvent {
  return {
    ...ENVELOPE,
    __typename: "ShipmentTenderEvent",
    reasons: [],
    warnings: [],
    ...overrides,
  };
}

function holdEvent(overrides: WithType<ShipmentHoldEvent>): ShipmentHoldEvent {
  return { ...ENVELOPE, __typename: "ShipmentHoldEvent", ...overrides };
}

function commentEvent(overrides: Partial<ShipmentCommentEvent> = {}): ShipmentCommentEvent {
  return {
    ...ENVELOPE,
    __typename: "ShipmentCommentEvent",
    type: "CommentPosted",
    mentionedUserIds: [],
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
      commentEvent({ severity: "info", commentBody: "hello @ops-night", commentType: "Dispatch" }),
    );

    render(harness(result));

    const headline = screen.getByTestId("headline").textContent;
    expect(headline).toBe("System Administrator added a comment to #PRO-2026-1042");
    expect(screen.getByTestId("detail").textContent).toBe("hello @ops-night");
    expect(screen.getByTestId("handle").textContent).toBe("@sysadmin");
  });

  it("omits the comment detail when the body is empty", () => {
    const result = renderEvent(commentEvent({ commentBody: "" }));
    render(harness(result));
    expect(screen.queryByTestId("detail")).toBeNull();
  });

  it("clamps a long comment body on the detail line", () => {
    const body = "x".repeat(300);
    const result = renderEvent(commentEvent({ commentBody: body }));
    render(harness(result));
    expect(screen.getByTestId("detail").textContent).toBe("x".repeat(240) + "…");
  });

  it("renders status changes with new status appended", () => {
    const result = renderEvent(
      lifecycleEvent({ type: "StatusChanged", previousStatus: "New", newStatus: "InTransit" }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator marked #PRO-2026-1042 as InTransit",
    );
  });

  it("renders a status change without a recorded status as a plain update", () => {
    const result = renderEvent(
      lifecycleEvent({ type: "StatusChanged", actorType: "edi", actor: undefined }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe("EDI updated #PRO-2026-1042");
  });

  it("renders driver assignment with the typed driver name", () => {
    const result = renderEvent(
      assignmentEvent({ type: "DriverAssigned", driverName: "S. Ndiaye", moveId: "smv_1" }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator assigned S. Ndiaye to #PRO-2026-1042",
    );
  });

  it("renders driver reassignment and falls back to a generic driver", () => {
    const result = renderEvent(assignmentEvent({ type: "DriverReassigned" }));
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator reassigned a driver on #PRO-2026-1042",
    );
  });

  it("renders driver unassignment", () => {
    const result = renderEvent(assignmentEvent({ type: "DriverUnassigned" }));
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator unassigned a driver from #PRO-2026-1042",
    );
  });

  it("renders cancellation reason on the detail line", () => {
    const result = renderEvent(
      lifecycleEvent({ type: "ShipmentCanceled", severity: "danger", reason: "Customer request" }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator canceled #PRO-2026-1042",
    );
    expect(screen.getByTestId("detail").textContent).toBe("Reason: Customer request");
  });

  it("renders uncancel, update and ownership transfer headlines", () => {
    const reopened = renderEvent(lifecycleEvent({ type: "ShipmentUncanceled" }));
    const updated = renderEvent(lifecycleEvent({ type: "ShipmentUpdated" }));
    const transferred = renderEvent(
      ownershipEvent({ previousOwnerId: "usr_1", newOwnerId: "usr_2" }),
    );

    render(
      <div>
        <div data-testid="reopened">{reopened.headline}</div>
        <div data-testid="updated">{updated.headline}</div>
        <div data-testid="transferred">{transferred.headline}</div>
      </div>,
    );

    expect(screen.getByTestId("reopened").textContent).toBe(
      "System Administrator reopened #PRO-2026-1042",
    );
    expect(screen.getByTestId("updated").textContent).toBe(
      "System Administrator updated #PRO-2026-1042",
    );
    expect(screen.getByTestId("transferred").textContent).toBe(
      "System Administrator transferred ownership of #PRO-2026-1042",
    );
  });

  it("renders hold placement with the hold type woven in", () => {
    const result = renderEvent(
      holdEvent({
        type: "HoldPlaced",
        severity: "danger",
        holdType: "OperationalHold",
        holdSeverity: "Blocking",
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator placed a OperationalHold hold on #PRO-2026-1042",
    );
  });

  it("renders hold update and release without a hold type", () => {
    const updated = renderEvent(holdEvent({ type: "HoldUpdated" }));
    const released = renderEvent(holdEvent({ type: "HoldReleased", holdType: "FinanceHold" }));

    render(
      <div>
        <div data-testid="updated">{updated.headline}</div>
        <div data-testid="released">{released.headline}</div>
      </div>,
    );

    expect(screen.getByTestId("updated").textContent).toBe(
      "System Administrator updated a hold on #PRO-2026-1042",
    );
    expect(screen.getByTestId("released").textContent).toBe(
      "System Administrator released a FinanceHold hold on #PRO-2026-1042",
    );
  });

  it("falls back to system actor label when no user is attached", () => {
    const result = renderEvent(
      moveEvent({ type: "MoveDeparted", actorType: "system", actor: undefined }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System dispatched a move on #PRO-2026-1042",
    );
    expect(screen.getByTestId("handle").textContent).toBe("system");
  });

  it("renders a move status change with the status trail", () => {
    const result = renderEvent(
      moveEvent({ type: "MoveStatusChanged", previousStatus: "New", newStatus: "Assigned" }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator updated a move on #PRO-2026-1042 (New → Assigned)",
    );
  });

  it("renders a move status change without a trail when one side is missing", () => {
    const result = renderEvent(moveEvent({ type: "MoveStatusChanged", newStatus: "Canceled" }));
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator updated a move on #PRO-2026-1042",
    );
  });

  it("renders move arrival and stop completion", () => {
    const arrived = renderEvent(moveEvent({ type: "MoveArrived" }));
    const stop = renderEvent(moveEvent({ type: "StopCompleted", stopId: "stp_1" }));

    render(
      <div>
        <div data-testid="arrived">{arrived.headline}</div>
        <div data-testid="stop">{stop.headline}</div>
      </div>,
    );

    expect(screen.getByTestId("arrived").textContent).toBe(
      "System Administrator completed a move on #PRO-2026-1042",
    );
    expect(screen.getByTestId("stop").textContent).toBe(
      "System Administrator completed a stop on #PRO-2026-1042",
    );
  });

  it("falls back to a shipment without pro number", () => {
    const result = renderEvent(lifecycleEvent({ shipment: undefined }));
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator created a shipment",
    );
  });

  it("uses the actor label and then the actor type when no user is attached", () => {
    const labelled = renderEvent(
      lifecycleEvent({ actor: undefined, actorType: "apikey", actorLabel: "Dispatch bot" }),
    );
    const unlabelled = renderEvent(lifecycleEvent({ actor: undefined, actorType: "apikey" }));

    render(
      <div>
        <div data-testid="labelled">{labelled.headline}</div>
        <div data-testid="labelled-handle">{labelled.actorHandle}</div>
        <div data-testid="unlabelled">{unlabelled.headline}</div>
        <div data-testid="unlabelled-handle">{unlabelled.actorHandle}</div>
      </div>,
    );

    expect(screen.getByTestId("labelled").textContent).toBe("Dispatch bot created #PRO-2026-1042");
    expect(screen.getByTestId("labelled-handle").textContent).toBe("Dispatch bot");
    expect(screen.getByTestId("unlabelled").textContent).toBe("API key created #PRO-2026-1042");
    expect(screen.getByTestId("unlabelled-handle").textContent).toBe("apikey");
  });

  it("renders carrier assignment with the typed carrier name", () => {
    const result = renderEvent(
      carrierEvent({
        type: "CarrierAssigned",
        carrierName: "Blue Ridge Freight",
        carrierId: "car_01",
        totalCost: "2450.00",
        proNumber: "BRF-88213",
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator assigned Blue Ridge Freight to #PRO-2026-1042",
    );
  });

  it("renders carrier unassignment with the reason on the detail line", () => {
    const result = renderEvent(
      carrierEvent({
        type: "CarrierUnassigned",
        carrierName: "Blue Ridge Freight",
        reason: "Move was covered outside the tender",
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
      tenderEvent({
        type: "TenderOffered",
        severity: "info",
        tenderId: "tdr_01",
        moveId: "smv_01",
        offerId: "tof_01",
        carrierName: "Blue Ridge Freight",
        rank: 1,
        channel: "Email",
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator offered #PRO-2026-1042 to Blue Ridge Freight via Email",
    );
  });

  it("renders a tender offer without a channel when the payload omits it", () => {
    const result = renderEvent(
      tenderEvent({
        type: "TenderOffered",
        severity: "info",
        tenderId: "tdr_01",
        carrierName: "Blue Ridge Freight",
        rank: 2,
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator offered #PRO-2026-1042 to Blue Ridge Freight",
    );
  });

  it("renders a tender acceptance with the carrier as the subject", () => {
    const result = renderEvent(
      tenderEvent({
        type: "TenderAccepted",
        severity: "success",
        actorType: "system",
        actor: undefined,
        actorLabel: "System",
        tenderId: "tdr_01",
        offerId: "tof_01",
        carrierName: "Blue Ridge Freight",
        source: "Email",
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
      tenderEvent({
        type: "TenderDeclined",
        severity: "info",
        tenderId: "tdr_01",
        carrierName: "Blue Ridge Freight",
        reason: "No available power",
        source: "EDI",
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
      tenderEvent({
        type: "TenderDeclined",
        severity: "info",
        tenderId: "tdr_01",
        carrierName: "Blue Ridge Freight",
        reason: "",
        source: "Manual",
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
      tenderEvent({
        type: "TenderExpired",
        severity: "muted",
        tenderId: "tdr_01",
        carrierName: "Blue Ridge Freight",
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "Offer to Blue Ridge Freight expired on #PRO-2026-1042",
    );
  });

  it("renders a tender withdrawal with the reason as detail", () => {
    const result = renderEvent(
      tenderEvent({
        type: "TenderWithdrawn",
        severity: "muted",
        tenderId: "tdr_01",
        reason: "Move was covered outside the tender",
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
      tenderEvent({
        type: "TenderNeedsReview",
        severity: "danger",
        tenderId: "tdr_01",
        carrierName: "Blue Ridge Freight",
        reason: "Carrier insurance expired before pickup",
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
      tenderEvent({
        type: "RoutingGuideExhausted",
        severity: "danger",
        tenderId: "tdr_01",
        mode: "Waterfall",
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator exhausted the routing guide on #PRO-2026-1042",
    );
  });

  it("names the spot tender when a non-waterfall mode is exhausted", () => {
    const result = renderEvent(
      tenderEvent({
        type: "RoutingGuideExhausted",
        severity: "danger",
        tenderId: "tdr_01",
        mode: "SpotBroadcast",
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator exhausted the spot tender on #PRO-2026-1042",
    );
  });

  it("renders a late response with its action and carrier", () => {
    const result = renderEvent(
      tenderEvent({
        type: "TenderLateResponse",
        severity: "muted",
        tenderId: "tdr_01",
        carrierName: "Blue Ridge Freight",
        action: "Decline",
        source: "Email",
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator recorded a late Decline from Blue Ridge Freight on #PRO-2026-1042",
    );
  });

  it("renders a late response without an action as a generic response", () => {
    const result = renderEvent(
      tenderEvent({ type: "TenderLateResponse", tenderId: "tdr_01", carrierName: "Blue Ridge" }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator recorded a late response from Blue Ridge on #PRO-2026-1042",
    );
  });

  it("renders a delivery failure with the transport error as detail", () => {
    const result = renderEvent(
      tenderEvent({
        type: "TenderDeliveryFailed",
        severity: "danger",
        tenderId: "tdr_01",
        carrierName: "Blue Ridge Freight",
        channel: "Email",
        error: "smtp: 550 recipient rejected",
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
      tenderEvent({
        type: "TenderEntrySkipped",
        severity: "danger",
        tenderId: "tdr_01",
        carrierName: "Sunset Logistics",
        rank: 3,
        reasons: ["Insurance policy expired", "Compliance status is Unqualified"],
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
      tenderEvent({
        type: "TenderEntryWarned",
        severity: "info",
        tenderId: "tdr_01",
        carrierName: "Sunset Logistics",
        rank: 2,
        warnings: ["Auto liability policy expires in 12 days"],
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
      tenderEvent({
        type: "TenderEntrySkipped",
        severity: "danger",
        tenderId: "tdr_01",
        carrierName: "Sunset Logistics",
        reasons: [],
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "System Administrator skipped Sunset Logistics on #PRO-2026-1042",
    );
    expect(screen.queryByTestId("detail")).toBeNull();
  });

  it("drops blank entries from reasons and warnings", () => {
    const result = renderEvent(
      tenderEvent({
        type: "TenderEntryWarned",
        tenderId: "tdr_01",
        carrierName: "Sunset Logistics",
        warnings: ["", "Policy expires soon", ""],
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("detail").textContent).toBe("Policy expires soon");
  });

  it("falls back to a generic carrier when tender payload omits the carrier name", () => {
    const result = renderEvent(
      tenderEvent({ type: "TenderAccepted", severity: "success", tenderId: "tdr_01" }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe(
      "a carrier accepted the tender for #PRO-2026-1042",
    );
  });

  it("falls back to summary for an event type its category does not know", () => {
    const result = renderEvent(
      lifecycleEvent({
        // @ts-expect-error intentional unknown type
        type: "FutureUnknownType",
        summary: "Something happened",
      }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe("Something happened");
  });

  it("falls back to summary when a known type arrives on the wrong category", () => {
    const result = renderEvent(
      holdEvent({ type: "CommentPosted", summary: "Comment posted on a hold" }),
    );
    render(harness(result));
    expect(screen.getByTestId("headline").textContent).toBe("Comment posted on a hold");
  });
});
