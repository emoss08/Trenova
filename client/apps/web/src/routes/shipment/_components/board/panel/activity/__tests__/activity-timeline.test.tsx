import type { ShipmentLifecycleEvent } from "@/types/shipment-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ActivityTimeline } from "../activity-timeline";

const listShipmentEventsMock = vi.hoisted(() => vi.fn());

vi.mock("@/services/api", () => ({
  apiService: {
    shipmentEventService: {
      list: listShipmentEventsMock,
    },
  },
}));

const NEWEST = 1_800_000_000;
const ROW_HEIGHT = 31;

function createdEvent(index: number): ShipmentLifecycleEvent {
  const number = String(index + 1).padStart(4, "0");
  return {
    __typename: "ShipmentLifecycleEvent",
    type: "ShipmentCreated",
    id: `se_${number}`,
    organizationId: "org_1",
    businessUnitId: "bu_1",
    shipmentId: `shp_${number}`,
    severity: "muted",
    actorType: "user",
    actorLabel: "",
    summary: "Shipment created",
    metadata: {},
    occurredAt: NEWEST - index * 60,
    actor: { name: "System Administrator", username: "sysadmin" },
    shipment: { id: `shp_${number}`, proNumber: `PRO-${number}` },
  };
}

function eventsPage(from: number, length: number) {
  return Array.from({ length }, (_, offset) => createdEvent(from + offset));
}

function renderTimeline() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
      </MemoryRouter>
    );
  }
  return render(<ActivityTimeline scrollElement={null} />, { wrapper: Wrapper });
}

describe("ActivityTimeline", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockImplementation(
      function (this: HTMLElement) {
        return this.tagName === "LI" ? ROW_HEIGHT : 0;
      },
    );
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("mounts only the newest events near the viewport of a long feed", async () => {
    listShipmentEventsMock.mockResolvedValueOnce(eventsPage(0, 200));

    renderTimeline();

    const list = await screen.findByRole("list", { name: "Shipment activity" });
    const rows = within(list).getAllByRole("listitem");
    expect(rows.length).toBeGreaterThan(0);
    expect(rows.length).toBeLessThan(200);
    expect(rows[0]).toHaveTextContent("PRO-0001");
    expect(within(list).getByText("#PRO-0001")).toBeInTheDocument();
    expect(within(list).queryByText("#PRO-0200")).not.toBeInTheDocument();
  });

  it("does not ask for older events while the window is far from the end", async () => {
    listShipmentEventsMock.mockResolvedValueOnce(eventsPage(0, 200));

    renderTimeline();

    await screen.findByRole("list", { name: "Shipment activity" });
    expect(listShipmentEventsMock).toHaveBeenCalledTimes(1);
  });

  it("asks for the events before the oldest one read once the window reaches the end", async () => {
    const firstPage = eventsPage(0, 20);
    listShipmentEventsMock
      .mockResolvedValueOnce(firstPage)
      .mockResolvedValueOnce(eventsPage(20, 5));

    renderTimeline();

    await waitFor(() => {
      expect(listShipmentEventsMock).toHaveBeenCalledTimes(2);
    });
    expect(listShipmentEventsMock).toHaveBeenNthCalledWith(2, {
      shipmentId: undefined,
      types: undefined,
      limit: 20,
      before: firstPage[19].occurredAt,
    });
    const list = await screen.findByRole("list", { name: "Shipment activity" });
    expect(await within(list).findByText("#PRO-0025")).toBeInTheDocument();
  });

  it("ends the list in rows waiting to load while older events remain", async () => {
    listShipmentEventsMock
      .mockResolvedValueOnce(eventsPage(0, 20))
      .mockReturnValueOnce(new Promise(() => {}));

    renderTimeline();

    const list = await screen.findByRole("list", { name: "Shipment activity" });
    const rows = within(list).getAllByRole("listitem");
    expect(rows).toHaveLength(21);
    expect(within(rows[20]).getByLabelText("Loading older activity")).toBeInTheDocument();
    expect(rows[19]).toHaveTextContent("#PRO-0020");
  });

  it("stops asking once a short page says there is nothing older", async () => {
    listShipmentEventsMock.mockResolvedValueOnce(eventsPage(0, 7));

    renderTimeline();

    const list = await screen.findByRole("list", { name: "Shipment activity" });
    expect(within(list).getAllByRole("listitem")).toHaveLength(7);
    expect(screen.queryByLabelText("Loading older activity")).not.toBeInTheDocument();
    expect(listShipmentEventsMock).toHaveBeenCalledTimes(1);
  });

  it("keeps the events read and offers a retry when older events fail to load", async () => {
    listShipmentEventsMock
      .mockResolvedValueOnce(eventsPage(0, 20))
      .mockRejectedValueOnce(new Error("network"))
      .mockResolvedValueOnce(eventsPage(20, 3));

    renderTimeline();

    const retry = await screen.findByRole("button", { name: "Try again" });
    const list = screen.getByRole("list", { name: "Shipment activity" });
    expect(within(list).getByText("Older activity could not be loaded.")).toBeInTheDocument();
    expect(within(list).getByText("#PRO-0001")).toBeInTheDocument();
    expect(listShipmentEventsMock).toHaveBeenCalledTimes(2);

    fireEvent.click(retry);

    expect(await within(list).findByText("#PRO-0023")).toBeInTheDocument();
    expect(listShipmentEventsMock).toHaveBeenCalledTimes(3);
    expect(listShipmentEventsMock).toHaveBeenLastCalledWith({
      shipmentId: undefined,
      types: undefined,
      limit: 20,
      before: NEWEST - 19 * 60,
    });
    expect(screen.queryByRole("button", { name: "Try again" })).not.toBeInTheDocument();
  });
});
