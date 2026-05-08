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
