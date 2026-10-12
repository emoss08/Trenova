import type { RecordShipment } from "@/lib/graphql/record-shipments";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CardShipments } from "../card-shipments";
import { shipmentLane, useRecordShipments } from "../record-shipments";

const fetchRecordShipments = vi.hoisted(() => vi.fn());

vi.mock("@/lib/graphql/record-shipments", () => ({ fetchRecordShipments }));

const DAY = 86_400;
const NOW = 1_791_720_000;

function shipment(id: string, overrides: Partial<RecordShipment> = {}): RecordShipment {
  return {
    id,
    proNumber: `PRO-${id}`,
    status: "InTransit",
    totalChargeAmount: "1850",
    moves: [
      {
        sequence: 0,
        stops: [
          {
            sequence: 0,
            type: "Pickup",
            scheduledWindowStart: NOW,
            location: { city: "Memphis", state: { abbreviation: "TN" } },
          },
          {
            sequence: 1,
            type: "Delivery",
            scheduledWindowStart: NOW + DAY,
            location: { city: "Little Rock", state: { abbreviation: "AR" } },
          },
        ],
      },
    ],
    ...overrides,
  } as RecordShipment;
}

function Listed({ onOpen }: { onOpen?: (id: string) => void }) {
  const shipments = useRecordShipments({
    scope: { kind: "tractor", id: "tr_1" },
    pageSize: 2,
    enabled: true,
  });
  return (
    <CardShipments
      title="Recent shipments"
      shipments={shipments}
      onOpen={onOpen}
      emptyText="This tractor has pulled no loads yet."
    />
  );
}

function draw(onOpen?: (id: string) => void) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <Listed onOpen={onOpen} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  fetchRecordShipments.mockReset();
});

describe("CardShipments", () => {
  it("holds rows while the first page is read, then lists each load with its lane", async () => {
    fetchRecordShipments.mockResolvedValue({
      rows: [shipment("shp_1")],
      total: 1,
      endCursor: null,
      hasNextPage: false,
    });
    const { container } = draw();

    expect(container.querySelector(".dk-rec-wait")).not.toBeNull();
    const row = await screen.findByRole("link", { name: /PRO-shp_1/ });
    expect(within(row).getByText(/Memphis, TN/)).toHaveTextContent("Memphis, TN → Little Rock, AR");
    expect(row).toHaveAttribute("href", expect.stringContaining("shp_1"));
    expect(screen.getByText("1 shipment")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Show more" })).toBeNull();
  });

  it("reads the next page on request from where the last one ended, and adds it under", async () => {
    fetchRecordShipments
      .mockResolvedValueOnce({
        rows: [shipment("shp_1"), shipment("shp_2")],
        total: 3,
        endCursor: "c2",
        hasNextPage: true,
      })
      .mockResolvedValueOnce({
        rows: [shipment("shp_3")],
        total: null,
        endCursor: null,
        hasNextPage: false,
      });
    draw();

    expect(await screen.findByText("2 of 3 shown")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Show more" }));

    expect(await screen.findByRole("link", { name: /PRO-shp_3/ })).toBeInTheDocument();
    expect(fetchRecordShipments).toHaveBeenLastCalledWith(
      { kind: "tractor", id: "tr_1" },
      { first: 2, after: "c2" },
      expect.anything(),
    );
    expect(screen.getAllByRole("link").map((link) => link.textContent)).toEqual([
      expect.stringContaining("PRO-shp_1"),
      expect.stringContaining("PRO-shp_2"),
      expect.stringContaining("PRO-shp_3"),
    ]);
    expect(screen.getByText("3 shipments")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Show more" })).toBeNull();
  });

  it("opens a row in the card's place when the shipment has a card here", async () => {
    fetchRecordShipments.mockResolvedValue({
      rows: [shipment("shp_1")],
      total: 1,
      endCursor: null,
      hasNextPage: false,
    });
    const onOpen = vi.fn();
    draw(onOpen);

    fireEvent.click(await screen.findByRole("button", { name: /PRO-shp_1/ }));
    expect(onOpen).toHaveBeenCalledWith("shp_1");
  });

  it("says so when the record has no shipments", async () => {
    fetchRecordShipments.mockResolvedValue({
      rows: [],
      total: 0,
      endCursor: null,
      hasNextPage: false,
    });
    draw();

    expect(await screen.findByText("This tractor has pulled no loads yet.")).toBeInTheDocument();
    expect(screen.queryByText("0 shipments")).toBeNull();
  });

  it("says a first page that failed in its own place", async () => {
    fetchRecordShipments.mockRejectedValue(new Error("Shipments are unavailable"));
    draw();

    expect(await screen.findByText("Shipments are unavailable")).toBeInTheDocument();
  });

  it("keeps the rows shown when a later page fails, and says so under them", async () => {
    fetchRecordShipments
      .mockResolvedValueOnce({
        rows: [shipment("shp_1"), shipment("shp_2")],
        total: 3,
        endCursor: "c2",
        hasNextPage: true,
      })
      .mockRejectedValueOnce(new Error("The server went away"));
    draw();

    fireEvent.click(await screen.findByRole("button", { name: "Show more" }));

    expect(await screen.findByText("The server went away")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /PRO-shp_2/ })).toBeInTheDocument();
  });
});

describe("shipmentLane", () => {
  it("runs from the first pickup to the last delivery across moves", () => {
    const shipment = {
      id: "shp_1",
      proNumber: "S1",
      status: "InTransit",
      totalChargeAmount: "2450",
      moves: [
        {
          sequence: 0,
          stops: [
            {
              sequence: 0,
              type: "Pickup",
              scheduledWindowStart: NOW,
              location: { city: "Atlanta", state: { abbreviation: "GA" } },
            },
            {
              sequence: 1,
              type: "Delivery",
              scheduledWindowStart: NOW + DAY,
              location: { city: "Nashville", state: { abbreviation: "TN" } },
            },
          ],
        },
        {
          sequence: 1,
          stops: [
            {
              sequence: 0,
              type: "Delivery",
              scheduledWindowStart: NOW + 2 * DAY,
              location: { city: "Dallas", state: null },
            },
          ],
        },
      ],
    } as RecordShipment;

    expect(shipmentLane(shipment)).toEqual({
      origin: "Atlanta, GA",
      destination: "Dallas",
      pickupAt: NOW,
    });
  });

  it("has no lane for a shipment with no stops", () => {
    expect(shipmentLane({ moves: [] } as unknown as RecordShipment)).toEqual({
      origin: null,
      destination: null,
      pickupAt: null,
    });
  });
});
