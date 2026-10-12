import type { ActivityEntry } from "@/lib/graphql/record-activity";
import type { RecordFailure } from "@/lib/graphql/record-failures";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskServiceFailure, DeskServiceFailureArtifact } from "../desk-service-failure";
import { NOW, serviceFailure } from "./fixtures";

const fetchers = vi.hoisted(() => ({
  failure: vi.fn(),
  failures: vi.fn(),
  activity: vi.fn(),
}));

vi.mock("@/lib/graphql/service-failure-card", () => ({
  fetchDeskServiceFailure: fetchers.failure,
}));
vi.mock("@/lib/graphql/record-failures", () => ({ fetchRecordFailures: fetchers.failures }));
vi.mock("@/lib/graphql/record-activity", () => ({ fetchRecordActivity: fetchers.activity }));
// The shipment and customer cards are their own chunks; here each stands in as
// what it was given, with its way back.
function standIn(kind: string) {
  return ({
    recordId,
    onBack,
    backLabel,
  }: {
    recordId: string;
    onBack: () => void;
    backLabel?: string;
  }) => (
    <div>
      <p>
        {kind} card {recordId}
      </p>
      <button type="button" onClick={onBack}>
        {backLabel}
      </button>
    </div>
  );
}
vi.mock("../../shipment/desk-shipment", () => ({
  DeskShipment: standIn("Shipment"),
  DeskShipmentArtifact: () => null,
}));
vi.mock("../../customer/desk-customer", () => ({
  DeskCustomer: standIn("Customer"),
  DeskCustomerArtifact: () => null,
}));

/** The failure the card is about and one more on the same load, as the load's list reads them. */
const thisOne = {
  id: "sf_1",
  number: "SF-2610-0042",
  type: "LateDelivery",
  status: "Open",
  stopType: "Delivery",
  lateMinutes: 105,
  actualArrival: NOW,
  shipment: { id: "shp_1", proNumber: "S2610-1366" },
  stop: null,
} as RecordFailure;
const pickup = {
  ...thisOne,
  id: "sf_0",
  number: "SF-2610-0039",
  type: "LatePickup",
  status: "Resolved",
  stopType: "Pickup",
  lateMinutes: 40,
  stop: {
    id: "stp_1",
    location: { id: "loc_9", name: "Acme Plant 2", city: "Rogers", state: { abbreviation: "AR" } },
  },
} as RecordFailure;

const entry = {
  id: "ae_1",
  operation: "update",
  timestamp: NOW - 600,
  changes: { notes: { from: "", to: "Called the DC" } },
  comment: null,
  principalType: "session_user",
  user: { id: "usr_1", name: "Mia Chen" },
} as ActivityEntry;

function draw(props: Partial<Parameters<typeof DeskServiceFailure>[0]> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handlers = { onSelect: vi.fn(), onBack: vi.fn() };
  const view = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskServiceFailure recordId="sf_1" order={["sf_0", "sf_1"]} {...handlers} {...props} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { ...view, ...handlers };
}

function section(title: string): HTMLElement {
  const heading = screen.getByRole("heading", { name: title, level: 3 });
  const found = heading.closest("section");
  if (!found) throw new Error(`No section titled ${title}`);
  return found;
}

beforeEach(() => {
  seedPermissions({
    [Resource.ServiceFailure]: Operation.Read,
    [Resource.AuditLog]: Operation.Read,
    [Resource.Shipment]: Operation.Read,
    [Resource.Customer]: Operation.Read,
  });
  fetchers.failure
    .mockReset()
    .mockImplementation(async (id: string) =>
      id === "sf_0"
        ? serviceFailure({ id: "sf_0", number: "SF-2610-0039", type: "LatePickup" })
        : serviceFailure(),
    );
  fetchers.failures.mockReset().mockResolvedValue({
    rows: [thisOne, pickup],
    total: 2,
    endCursor: null,
    hasNextPage: false,
  });
  fetchers.activity.mockReset().mockResolvedValue({
    rows: [entry],
    total: 3,
    endCursor: "c1",
    hasNextPage: true,
  });
});

afterEach(() => {
  clearPermissions();
});

describe("DeskServiceFailure", () => {
  it("holds the card's outline while the failure is read, then draws where and how late it was", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the service failure…");
    expect(container.querySelector(".dk-rec-skel .dk-rec-path")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-rec-where")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "Tyson Foods DC", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(document.querySelector(".dk-rec-kick")).toHaveTextContent(
      /SF-2610-0042.*Late delivery.*Detected/u,
    );
    expect(
      screen.getByText("Springdale, AR 72762", { selector: ".dk-rec-who span" }),
    ).toBeInTheDocument();
    expect(document.querySelector(".dk-rec-amt b")).toHaveTextContent("1h 45m");
    expect(document.querySelector(".dk-rec-st")).toHaveTextContent("Open");
    expect(screen.getByText("Shipment S2610-1366")).toBeInTheDocument();

    const happened = section("What happened");
    expect(within(happened).getByText("Late by").nextElementSibling).toHaveTextContent("1h 45m");
    expect(within(happened).getByText("Grace").nextElementSibling).toHaveTextContent("30m");

    expect(within(section("Where")).getByText("2200 W Don Tyson Pkwy")).toBeInTheDocument();
    expect(within(section("Load")).getByText("Tyson Foods")).toBeInTheDocument();
    expect(within(section("Cause")).getByText("No reason has been given yet.")).toHaveClass(
      "dk-svf-missing",
    );

    const review = within(section("Review")).getByRole("list", { name: "Review" });
    expect(review.querySelector('[class~="dk-now"]')).toHaveTextContent("Waiting on a reviewer");

    const activity = section("Activity");
    expect(await within(activity).findByText("Changed notes")).toBeInTheDocument();
    expect(fetchers.activity).toHaveBeenCalledWith(
      "sf_1",
      { first: 5, after: null },
      expect.anything(),
    );
  });

  it("lists the load's other failures without this one, each at its own stop", async () => {
    draw();

    const others = await screen.findByRole("heading", { name: "Other failures on this load" });
    const list = others.closest("section") as HTMLElement;
    expect(await within(list).findByText("SF-2610-0039")).toBeInTheDocument();
    expect(within(list).queryByText("SF-2610-0042")).toBeNull();
    expect(within(list).getByText("1 other")).toBeInTheDocument();
    expect(
      within(list).getByText(/^Late pickup · Acme Plant 2 · 40m late · /u),
    ).toBeInTheDocument();
    expect(fetchers.failures).toHaveBeenCalledWith(
      { kind: "shipment", id: "shp_1" },
      { first: 5, after: null },
      expect.anything(),
    );
  });

  it("opens another failure on the load in the card's place, and comes back", async () => {
    draw();

    fireEvent.click(await screen.findByRole("button", { name: /SF-2610-0039/ }));
    expect(await screen.findByRole("button", { name: "Back to SF-2610-0042" })).toBeInTheDocument();
    expect(fetchers.failure).toHaveBeenCalledWith("sf_0", expect.anything());

    fireEvent.click(screen.getByRole("button", { name: "Back to SF-2610-0042" }));
    expect(
      await screen.findByRole("heading", { name: "Tyson Foods DC", level: 2 }),
    ).toBeInTheDocument();
  });

  it("opens the load and its customer in the card's place", async () => {
    draw();

    fireEvent.click(await screen.findByRole("button", { name: /S2610-1366/ }));
    expect(await screen.findByText("Shipment card shp_1")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to SF-2610-0042" }));

    fireEvent.click(await screen.findByRole("button", { name: /Tyson Foods/ }));
    expect(await screen.findByText("Customer card cus_1")).toBeInTheDocument();
  });

  it("states why it happened, with its notes, the internal one marked as kept off the customer's record", async () => {
    fetchers.failure.mockResolvedValue(
      serviceFailure({
        status: "Reviewed",
        reviewedAt: NOW,
        reviewedBy: { id: "usr_2", name: "Omar Haddad" },
        notes: "Dock 4 was closed for repairs.",
        internalNotes: "Third miss here this month.",
        reasonCode: {
          id: "rc_1",
          code: "CNS-DLY",
          label: "Consignee delayed unloading",
          description: "",
          category: "Consignee",
        },
      }),
    );
    draw();

    const cause = await screen.findByRole("heading", { name: "Cause" });
    const body = cause.closest("section") as HTMLElement;
    expect(within(body).getByText("Consignee delayed unloading")).toBeInTheDocument();
    expect(within(body).getByText("CNS-DLY")).toBeInTheDocument();
    expect(within(body).getByText("Dock 4 was closed for repairs.")).toBeInTheDocument();
    expect(
      within(body).getByText("Internal note, kept off the customer's record").closest("figure"),
    ).toHaveClass("dk-internal");
    const review = within(section("Review")).getByRole("list", { name: "Review" });
    expect(review.querySelector('[class~="dk-now"]')).toHaveTextContent("Waiting to be resolved");
    expect(review).toHaveTextContent("Omar Haddad");
  });

  it("closes a voided failure on its rail with the reason it was voided", async () => {
    fetchers.failure.mockResolvedValue(
      serviceFailure({
        status: "Voided",
        voidedAt: NOW,
        voidedBy: { id: "usr_2", name: "Omar Haddad" },
        voidReason: "Appointment was moved by the consignee",
      }),
    );
    draw();

    expect(await screen.findByText("Appointment was moved by the consignee")).toBeInTheDocument();
    expect(screen.getByText("No reason was given.")).not.toHaveClass("dk-svf-missing");
    expect(document.querySelector(".dk-rec-path li.dk-closed")).toHaveTextContent("Voided");
    expect(document.querySelector(".dk-rec-st")).toHaveTextContent("Voided");
  });

  it("stands with nothing around it: no place on file, no other failures, no activity", async () => {
    fetchers.failure.mockResolvedValue(
      serviceFailure({ stop: null, shipment: null, stopType: "Pickup", type: "MissedPickup" }),
    );
    fetchers.failures.mockResolvedValue({
      rows: [thisOne],
      total: 1,
      endCursor: null,
      hasNextPage: false,
    });
    fetchers.activity.mockResolvedValue({
      rows: [],
      total: 0,
      endCursor: null,
      hasNextPage: false,
    });
    draw();

    expect(
      await screen.findByRole("heading", { name: "Unknown stop", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.getByText("The load this failure was recorded on is gone.")).toBeInTheDocument();
    expect(
      await screen.findByText("Nothing else has gone wrong on this load."),
    ).toBeInTheDocument();
    expect(
      await screen.findByText("Nothing has been done to this failure yet."),
    ).toBeInTheDocument();
    expect(screen.getByText("No street address on file")).toBeInTheDocument();
  });

  it("says a section that could not be read, and draws the rest", async () => {
    fetchers.failures.mockRejectedValue(new Error("Service failures are unavailable"));
    draw();

    expect(await screen.findByText("Service failures are unavailable")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Review", level: 3 })).toBeInTheDocument();
  });

  it("says the failure could not be read, with the way back", async () => {
    fetchers.failure.mockRejectedValue(new Error("Service failure not found"));
    const { onBack } = draw();

    expect(
      await screen.findByRole("heading", {
        name: "This service failure could not be read",
        level: 2,
      }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalled();
    expect(fetchers.activity).not.toHaveBeenCalled();
    expect(fetchers.failures).not.toHaveBeenCalled();
  });

  it("opens with only what the person may read, asking for nothing else", async () => {
    seedPermissions({ [Resource.ServiceFailure]: Operation.Read });
    draw();

    expect(
      await screen.findByRole("heading", { name: "Tyson Foods DC", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Activity" })).toBeNull();
    expect(fetchers.activity).not.toHaveBeenCalled();
    expect(within(section("Load")).getByRole("link", { name: /S2610-1366/ })).toHaveAttribute(
      "href",
      expect.stringContaining("shp_1"),
    );
  });

  it("leaves the outline of activity the person may not read out of the skeleton", () => {
    seedPermissions({ [Resource.ServiceFailure]: Operation.Read });
    fetchers.failure.mockReturnValue(new Promise(() => {}));
    const { container } = draw();

    expect(container.querySelector(".dk-rec-skel .dk-rec-sk-log")).toBeNull();
    expect(container.querySelectorAll(".dk-rec-skel .dk-rec-sec")).toHaveLength(7);
  });

  it("draws the same card as an artifact, with nothing to step through", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DeskServiceFailureArtifact artifact={{} as never} recordId="sf_1" fallback={null} />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(
      await screen.findByRole("heading", { name: "Tyson Foods DC", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Next failure" })).toBeNull();
  });
});
