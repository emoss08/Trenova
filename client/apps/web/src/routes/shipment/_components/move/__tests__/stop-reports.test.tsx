import type { TrackingEventRow } from "@/lib/tracking-events";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { StopReports } from "../stop-reports";

const { listShipmentTrackingEventsGraphQL } = vi.hoisted(() => ({
  listShipmentTrackingEventsGraphQL: vi.fn(),
}));

vi.mock("@/lib/graphql/shipment", () => ({
  listShipmentTrackingEventsGraphQL,
  getShipmentProfitabilityGraphQL: vi.fn(),
}));

function report(overrides: Partial<TrackingEventRow>): TrackingEventRow {
  return {
    id: "tkev_1",
    shipmentMoveId: "sm_1",
    stopId: "stp_pickup",
    source: "Telematics",
    kind: "Arrival",
    matchMethod: "Location",
    eventAt: 1767787200,
    receivedAt: 1767787260,
    sourceStatus: null,
    outcome: "Applied",
    outcomeReason: null,
    ...overrides,
  };
}

function Harness({ shipmentId }: { shipmentId?: string }) {
  const form = useForm({
    defaultValues: {
      id: shipmentId,
      moves: [
        {
          id: "sm_1",
          sequence: 0,
          stops: [
            { id: "stp_pickup", type: "Pickup", sequence: 0, location: { name: "Acme DC" } },
            { id: "stp_drop", type: "Delivery", sequence: 1, addressLine: "12 Main St" },
          ],
        },
      ],
    },
  });
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } }),
  );

  return (
    <QueryClientProvider client={client}>
      <FormProvider {...form}>
        <StopReports />
      </FormProvider>
    </QueryClientProvider>
  );
}

describe("StopReports", () => {
  beforeEach(() => {
    listShipmentTrackingEventsGraphQL.mockReset();
  });

  it("renders nothing for a shipment that has not been saved", () => {
    const { container } = render(<Harness />);

    expect(container).toBeEmptyDOMElement();
    expect(listShipmentTrackingEventsGraphQL).not.toHaveBeenCalled();
  });

  it("says so when nothing has been reported", async () => {
    listShipmentTrackingEventsGraphQL.mockResolvedValue([]);

    render(<Harness shipmentId="shp_1" />);

    expect(
      await screen.findByText("No arrivals or departures have been reported yet."),
    ).toBeInTheDocument();
    expect(listShipmentTrackingEventsGraphQL).toHaveBeenCalledWith("shp_1", {
      signal: expect.any(AbortSignal),
    });
  });

  it("lists each stop's reports in route order with what became of them", async () => {
    listShipmentTrackingEventsGraphQL.mockResolvedValue([
      report({
        id: "tkev_3",
        stopId: "stp_drop",
        source: "EDI",
        sourceStatus: "X1",
        outcome: "Pending",
        outcomeReason: "Waiting for the earlier stop to be completed",
        eventAt: 1767800000,
      }),
      report({
        id: "tkev_2",
        source: "Driver",
        outcome: "Superseded",
        outcomeReason: "Another report was used",
        eventAt: 1767787500,
      }),
      report({ id: "tkev_1" }),
      report({ id: "tkev_9", stopId: "stp_removed", outcome: "Refused", outcomeReason: "Gone" }),
      report({ id: "tkev_4", outcome: "Duplicate", eventAt: 1767787200 }),
    ]);

    render(<Harness shipmentId="shp_1" />);

    const headings = await screen.findAllByRole("heading", { level: 4 });
    expect(headings.map((heading) => heading.textContent)).toEqual([
      "1. Pickup · Acme DC",
      "2. Delivery · 12 Main St",
      "A stop no longer on this shipment",
    ]);

    const pickup = within(headings[0]!.closest("section")!);
    const pickupRows = pickup.getAllByRole("listitem");
    expect(pickupRows).toHaveLength(3);
    expect(within(pickupRows[0]!).getByText("Telematics")).toBeInTheDocument();
    expect(within(pickupRows[0]!).getByText("Used")).toBeInTheDocument();
    expect(within(pickupRows[1]!).getByText("Already reported")).toBeInTheDocument();
    expect(within(pickupRows[2]!).getByText("Driver")).toBeInTheDocument();
    expect(within(pickupRows[2]!).getByText("Not used")).toBeInTheDocument();
    expect(within(pickupRows[2]!).getByText("Another report was used")).toBeInTheDocument();

    const drop = within(headings[1]!.closest("section")!);
    expect(drop.getByText("EDI (X1)")).toBeInTheDocument();
    expect(drop.getByText("Waiting")).toBeInTheDocument();
    expect(drop.getByText("Waiting for the earlier stop to be completed")).toBeInTheDocument();

    const removed = within(headings[2]!.closest("section")!);
    expect(removed.getByText("Rejected")).toBeInTheDocument();
  });

  it("shows the error when the reports cannot be loaded", async () => {
    listShipmentTrackingEventsGraphQL.mockRejectedValue(new Error("You cannot view shipments"));

    render(<Harness shipmentId="shp_1" />);

    expect(await screen.findByText("You cannot view shipments")).toBeInTheDocument();
  });
});
