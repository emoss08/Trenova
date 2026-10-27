import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { CheckCallRow } from "@/lib/graphql/check-calls";
import { setUserDatePreferences } from "@trenova/shared/lib/date";
import type { Shipment } from "@trenova/shared/types/shipment";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import ShipmentCheckCalls from "../shipment-check-calls";

const unix = (iso: string) => Date.parse(iso) / 1000;

// 10:00 on Tuesday 27 October 2026 in Chicago.
const NOW = unix("2026-10-27T15:00:00Z");

const mocks = vi.hoisted(() => ({
  listCheckCalls: vi.fn(),
}));

vi.mock("@/lib/graphql/check-calls", () => ({
  listShipmentCheckCalls: mocks.listCheckCalls,
}));

vi.mock("../log-check-call-dialog", () => ({
  LogCheckCallDialog: ({
    open,
    defaultMoveId,
    moves,
    stopTimesNote,
  }: {
    open: boolean;
    defaultMoveId: string;
    moves: { id: string }[];
    stopTimesNote?: string | null;
  }) =>
    open ? (
      <div
        role="dialog"
        aria-label="Log check call"
        data-default-move={defaultMoveId}
        data-moves={moves.map((move) => move.id).join(",")}
        data-stop-times-note={stopTimesNote ?? ""}
      />
    ) : null,
}));

function call(overrides: Partial<CheckCallRow>): CheckCallRow {
  return {
    id: "chkc_1",
    shipmentId: "shp_1",
    shipmentMoveId: "sm_1",
    stopId: null,
    source: "Dispatcher",
    kind: "Update",
    reportedAt: NOW - 3600,
    receivedAt: NOW - 3600,
    latitude: null,
    longitude: null,
    city: null,
    stateCode: null,
    delayed: false,
    delayReason: null,
    estimatedArrival: null,
    note: null,
    contactName: null,
    reportedById: null,
    trackingEventId: null,
    ...overrides,
  } as CheckCallRow;
}

function shipmentWith(
  moves: Array<{ id: string; status: string }>,
  billing: Partial<Pick<Shipment, "status" | "billingTransferStatus">> = {},
): Shipment {
  return {
    id: "shp_1",
    status: "InTransit",
    billingTransferStatus: null,
    ...billing,
    moves: moves.map((move, index) => ({
      ...move,
      sequence: index,
      stops: [
        {
          id: `stp_${move.id}_pickup`,
          type: "Pickup",
          location: { name: `Origin ${index + 1}` },
        },
        {
          id: `stp_${move.id}_delivery`,
          type: "Delivery",
          location: { name: `Acme DC ${index + 1}` },
        },
      ],
    })),
  } as unknown as Shipment;
}

const oneMove = shipmentWith([{ id: "sm_1", status: "InTransit" }]);

function renderTab(subject: Shipment) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  }
  return render(<ShipmentCheckCalls shipment={subject} />, { wrapper: Wrapper });
}

// Intl separates a time from its meridiem with a narrow no-break space.
const text = (element: Element) => (element.textContent ?? "").replace(/\s+/g, " ").trim();

describe("ShipmentCheckCalls", () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(NOW * 1000);
    setUserDatePreferences({ timezone: "America/Chicago", timeFormat: "12-hour" });
    mocks.listCheckCalls.mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
    setUserDatePreferences({});
  });

  it("leads with where the load was last reported and whether it is late", async () => {
    mocks.listCheckCalls.mockResolvedValue([
      call({ id: "chkc_old", reportedAt: NOW - 7200, city: "Joplin", stateCode: "MO" }),
      call({
        id: "chkc_latest",
        reportedAt: NOW - 1800,
        city: "Tulsa",
        stateCode: "OK",
        source: "Carrier",
        delayed: true,
        delayReason: "Weather on I-44",
        estimatedArrival: unix("2026-10-27T18:30:00Z"),
      }),
    ]);

    renderTab(oneMove);

    const latest = await screen.findByRole("region", { name: "Last reported" });
    expect(within(latest).getByRole("heading", { level: 3 })).toHaveTextContent("Tulsa, OK");
    expect(text(within(latest).getByText(/Position update/))).toBe(
      "Position update · 30m ago · Carrier link",
    );
    const alert = within(latest).getByRole("alert");
    expect(within(alert).getByText("Running late")).toBeInTheDocument();
    expect(within(alert).getByText("Weather on I-44")).toBeInTheDocument();
    expect(text(within(latest).getByText("Expected at next stop").nextElementSibling!)).toBe(
      "Oct 27, 1:30 PM",
    );
    expect(within(latest).getByText("2 check calls")).toBeInTheDocument();
  });

  it("names the stop when an arrival gave no city", async () => {
    mocks.listCheckCalls.mockResolvedValue([
      call({ id: "chkc_arrived", kind: "Arrival", stopId: "stp_sm_1_delivery" }),
    ]);

    renderTab(oneMove);

    const latest = await screen.findByRole("region", { name: "Last reported" });
    expect(within(latest).getByRole("heading", { level: 3 })).toHaveTextContent(
      "2. Delivery · Acme DC 1",
    );
    expect(within(latest).queryByRole("alert")).not.toBeInTheDocument();
    expect(text(within(latest).getByText("Expected at next stop").nextElementSibling!)).toBe(
      "Not given",
    );
  });

  it("lays the history out by day in Chicago time, newest first", async () => {
    mocks.listCheckCalls.mockResolvedValue([
      call({ id: "chkc_older", reportedAt: unix("2026-10-24T16:00:00Z"), city: "Dallas" }),
      // 03:30 UTC on the 27th is 22:30 on the 26th in Chicago.
      call({ id: "chkc_night", reportedAt: unix("2026-10-27T03:30:00Z"), city: "Ardmore" }),
      call({
        id: "chkc_morning",
        reportedAt: unix("2026-10-27T14:00:00Z"),
        kind: "Departure",
        stopId: "stp_sm_1_pickup",
        contactName: "Jim",
        note: "Loaded and rolling",
      }),
    ]);

    renderTab(oneMove);

    const history = await screen.findByRole("list", { name: "Check call history" });
    const days = within(history).getAllByRole("heading", { level: 4 });
    expect(days.map(text)).toEqual(["Today", "Yesterday", "Oct 24, 2026"]);

    const entries = within(history)
      .getAllByRole("listitem")
      .filter((item) => item.querySelector(":scope > time"));
    expect(entries.map((entry) => text(entry.querySelector(":scope > time")!))).toEqual([
      "9:00 AM",
      "10:30 PM",
      "11:00 AM",
    ]);
    expect(within(entries[0]).getByText("Departed")).toBeInTheDocument();
    expect(
      within(entries[0]).getByText("1. Pickup · Origin 1 · Dispatcher · Spoke to Jim"),
    ).toBeInTheDocument();
    expect(within(entries[0]).getByText("Loaded and rolling")).toBeInTheDocument();
    expect(text(entries[1].querySelector("p")!)).toBe("Position update · Ardmore");
  });

  it("narrows the summary and history to one move and logs against it", async () => {
    const user = userEvent.setup();
    mocks.listCheckCalls.mockResolvedValue([
      call({ id: "chkc_m1", shipmentMoveId: "sm_1", reportedAt: NOW - 600, city: "Tulsa" }),
      call({ id: "chkc_m2", shipmentMoveId: "sm_2", reportedAt: NOW - 3000, city: "Wichita" }),
    ]);

    renderTab(
      shipmentWith([
        { id: "sm_1", status: "InTransit" },
        { id: "sm_2", status: "Assigned" },
      ]),
    );

    const latest = await screen.findByRole("region", { name: "Last reported" });
    expect(within(latest).getByRole("heading", { level: 3 })).toHaveTextContent("Tulsa");
    expect(screen.getByText("Move 1 · Dispatcher")).toBeInTheDocument();

    await user.click(screen.getByRole("radio", { name: "Move 2" }));

    const narrowed = screen.getByRole("region", { name: "Last reported" });
    expect(within(narrowed).getByRole("heading", { level: 3 })).toHaveTextContent("Wichita");
    expect(within(narrowed).getByText("1 check call")).toBeInTheDocument();
    expect(screen.queryByText(/Tulsa/)).not.toBeInTheDocument();

    await user.click(within(narrowed).getByRole("button", { name: "Log check call" }));
    expect(screen.getByRole("dialog", { name: "Log check call" })).toHaveAttribute(
      "data-default-move",
      "sm_2",
    );
  });

  it("offers the first call from an empty history, on the move in transit", async () => {
    const user = userEvent.setup();
    mocks.listCheckCalls.mockResolvedValue([]);

    renderTab(
      shipmentWith([
        { id: "sm_1", status: "Completed" },
        { id: "sm_2", status: "InTransit" },
      ]),
    );

    expect(await screen.findByText("No check calls yet")).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Last reported" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Log check call" }));
    const dialog = screen.getByRole("dialog", { name: "Log check call" });
    expect(dialog).toHaveAttribute("data-default-move", "sm_2");
    expect(dialog).toHaveAttribute("data-moves", "sm_1,sm_2");
  });

  it("keeps the history but offers no logging once every move is canceled", async () => {
    mocks.listCheckCalls.mockResolvedValue([call({ id: "chkc_1", city: "Tulsa" })]);

    renderTab(shipmentWith([{ id: "sm_1", status: "Canceled" }]));

    const latest = await screen.findByRole("region", { name: "Last reported" });
    expect(within(latest).getByRole("heading", { level: 3 })).toHaveTextContent("Tulsa");
    expect(screen.queryByRole("button", { name: "Log check call" })).not.toBeInTheDocument();
  });

  it("explains why nothing can be logged when there is no history either", async () => {
    mocks.listCheckCalls.mockResolvedValue([]);

    renderTab(shipmentWith([{ id: "sm_1", status: "Canceled" }]));

    expect(
      await screen.findByText("This shipment has no active move to log a check call against."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Log check call" })).not.toBeInTheDocument();
  });

  it("says why the history could not be loaded", async () => {
    mocks.listCheckCalls.mockRejectedValue(new Error("Check calls are unavailable"));

    renderTab(oneMove);

    expect(await screen.findByText("Check calls are unavailable")).toBeInTheDocument();
  });

  it("warns up front that arrivals will not move stop times while billing holds the load", async () => {
    const user = userEvent.setup();
    mocks.listCheckCalls.mockResolvedValue([call({ id: "chkc_1", city: "Tulsa" })]);
    const note =
      "Arrivals and departures logged now won't change stop times — this load is in the billing queue.";

    renderTab(
      shipmentWith([{ id: "sm_1", status: "Completed" }], {
        status: "ReadyToInvoice",
        billingTransferStatus: "InReview",
      }),
    );

    expect(await screen.findByText(note)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Log check call" }));
    expect(screen.getByRole("dialog", { name: "Log check call" })).toHaveAttribute(
      "data-stop-times-note",
      note,
    );
  });

  it("names the invoice as the reason once the load is invoiced", async () => {
    mocks.listCheckCalls.mockResolvedValue([]);

    renderTab(
      shipmentWith([{ id: "sm_1", status: "Completed" }], {
        status: "Invoiced",
        billingTransferStatus: "Posted",
      }),
    );

    expect(
      await screen.findByText(
        "Arrivals and departures logged now won't change stop times — this load has been invoiced.",
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Log where the load is each time the driver or carrier reports in."),
    ).toBeInTheDocument();
  });

  it("says nothing about stop times when billing has handed the load back", async () => {
    const user = userEvent.setup();
    mocks.listCheckCalls.mockResolvedValue([call({ id: "chkc_1", city: "Tulsa" })]);

    renderTab(
      shipmentWith([{ id: "sm_1", status: "InTransit" }], {
        status: "ReadyToInvoice",
        billingTransferStatus: "SentBackToOps",
      }),
    );

    await screen.findByRole("region", { name: "Last reported" });
    expect(screen.queryByText(/won't change stop times/)).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Log check call" }));
    expect(screen.getByRole("dialog", { name: "Log check call" })).toHaveAttribute(
      "data-stop-times-note",
      "",
    );
  });

  it("leaves out the note when nothing can be logged anyway", async () => {
    mocks.listCheckCalls.mockResolvedValue([call({ id: "chkc_1", city: "Tulsa" })]);

    renderTab(
      shipmentWith([{ id: "sm_1", status: "Canceled" }], {
        status: "Invoiced",
        billingTransferStatus: "Posted",
      }),
    );

    await screen.findByRole("region", { name: "Last reported" });
    expect(screen.queryByText(/won't change stop times/)).not.toBeInTheDocument();
  });
});
