import type { RecordShipment } from "@/lib/graphql/record-shipments";
import type { DeskTrailer as DeskTrailerRecord } from "@/lib/graphql/trailer-card";
import {
  FakeIntersectionObserver,
  installIntersectionObserver,
} from "@/test/intersection-observer";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskTrailer, DeskTrailerArtifact } from "../desk-trailer";

const fetchers = vi.hoisted(() => ({
  trailer: vi.fn(),
  shipments: vi.fn(),
}));

vi.mock("@/lib/graphql/trailer-card", () => ({ fetchDeskTrailer: fetchers.trailer }));
vi.mock("@/lib/graphql/record-shipments", () => ({ fetchRecordShipments: fetchers.shipments }));
// The shipment card is its own chunk; here it stands in as what it was given,
// with its way back.
vi.mock("../../shipment/desk-shipment", () => ({
  DeskShipment: ({
    recordId,
    onBack,
    backLabel,
  }: {
    recordId: string;
    onBack: () => void;
    backLabel?: string;
  }) => (
    <div>
      <p>Shipment card {recordId}</p>
      <button type="button" onClick={onBack}>
        {backLabel}
      </button>
    </div>
  ),
  DeskShipmentArtifact: () => null,
}));

const DAY = 86_400;
const NOW = Math.floor(Date.now() / 1000);

/** A trailer as the DeskTrailerCard query returns it. */
function trailer(overrides: Partial<DeskTrailerRecord> = {}): DeskTrailerRecord {
  return {
    id: "trl_1",
    code: "TL-501",
    status: "Available",
    make: "Wabash",
    model: "DuraPlate",
    year: 2021,
    vin: "1JJV532D8ML123456",
    licensePlateNumber: "TX-T5501",
    registrationNumber: "REG-T5501",
    registrationExpiry: NOW + 200 * DAY,
    lastInspectionDate: NOW - 100 * DAY,
    maxLoadWeight: 45_000,
    externalId: "",
    ownershipType: "CompanyOwned",
    lessorName: "",
    leaseReference: "",
    leaseEndDate: null,
    createdAt: 1_700_000_000,
    registrationState: { abbreviation: "TX" },
    equipmentType: { id: "et_dv", code: "DV53", description: "53' dry van" },
    equipmentManufacturer: { id: "em_wb", name: "Wabash" },
    fleetCode: { id: "fc_dal", code: "DAL" },
    ownerWorker: null,
    lastKnownLocation: {
      id: "loc_dal",
      name: "Dallas yard",
      latitude: 32.7767,
      longitude: -96.797,
    },
    ...overrides,
  } as DeskTrailerRecord;
}

const shipment = {
  id: "shp_1",
  proNumber: "S26100090038417",
  status: "Completed",
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
          location: { city: "Dallas", state: { abbreviation: "TX" } },
        },
      ],
    },
  ],
} as RecordShipment;

function page(rows: RecordShipment[], total: number) {
  return { rows, total, endCursor: null, hasNextPage: false };
}

function draw(props: Partial<Parameters<typeof DeskTrailer>[0]> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handlers = { onSelect: vi.fn(), onBack: vi.fn() };
  const tree = (more: Partial<typeof props> = {}) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskTrailer
          recordId="trl_1"
          order={["trl_0", "trl_1"]}
          {...handlers}
          {...props}
          {...more}
        />
      </MemoryRouter>
    </QueryClientProvider>
  );
  const view = render(tree());
  return {
    ...view,
    ...handlers,
    step: (recordId: string) => view.rerender(tree({ recordId })),
  };
}

function section(title: string): HTMLElement {
  const heading = screen.getByRole("heading", { name: title, level: 3 });
  const found = heading.closest("section");
  if (!found) throw new Error(`No section titled ${title}`);
  return found;
}

beforeEach(() => {
  seedPermissions({ [Resource.Trailer]: Operation.Read, [Resource.Shipment]: Operation.Read });
  fetchers.trailer.mockReset().mockResolvedValue(trailer());
  fetchers.shipments.mockReset().mockResolvedValue(page([shipment], 9));
});

afterEach(() => {
  clearPermissions();
  vi.unstubAllGlobals();
});

describe("DeskTrailer", () => {
  it("draws the next trailer's outline, not the one it left, while the next one is read", async () => {
    const { container, step } = draw();
    await screen.findByRole("heading", { name: "2021 Wabash DuraPlate", level: 2 });

    fetchers.trailer.mockReturnValue(new Promise(() => {}));
    step("trl_0");

    expect(screen.queryByRole("heading", { name: "2021 Wabash DuraPlate", level: 2 })).toBeNull();
    expect(container.querySelector(".dk-rec-skel")).not.toBeNull();
  });

  it("holds the card's outline while the trailer is read, then draws what it is and where", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the trailer…");
    expect(container.querySelector(".dk-rec-skel .dk-rec-where")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "2021 Wabash DuraPlate", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(screen.getByText("TL-501")).toBeInTheDocument();
    expect(document.querySelector(".dk-rec-amt b")).toHaveTextContent("45,000 lb");
    expect(screen.getByText("maximum load")).toBeInTheDocument();
    expect(screen.getByText("Available")).toBeInTheDocument();

    const where = section("Where it is");
    expect(within(where).getByText("Where its last move left it")).toBeInTheDocument();
    expect(within(where).getByText("Dallas yard")).toBeInTheDocument();
    expect(within(where).getByRole("link", { name: /Open location/ })).toHaveAttribute(
      "href",
      expect.stringContaining("loc_dal"),
    );
    const calendar = section("Service calendar");
    expect(within(calendar).getByText("Nothing due")).toBeInTheDocument();
    expect(within(calendar).getByRole("img", { name: "Service calendar" })).toBeInTheDocument();
    expect(within(calendar).getByText("Today")).toBeInTheDocument();
    expect(
      Array.from(calendar.querySelectorAll(".dk-tl-mark > b")).map((b) => b.textContent),
    ).toEqual(expect.arrayContaining(["Last inspected", "Inspection due", "Registration ends"]));
    expect(within(section("Data plate")).getByText("Max load")).toBeInTheDocument();
    expect(await screen.findByText("9 shipments")).toBeInTheDocument();
    expect(fetchers.shipments).toHaveBeenCalledWith(
      { kind: "trailer", id: "trl_1" },
      { first: 5, after: null },
      expect.anything(),
    );
  });

  it("draws the last location on the map once its square comes into view", async () => {
    installIntersectionObserver();
    const { container } = draw();

    await screen.findByRole("heading", { name: "2021 Wabash DuraPlate", level: 2 });
    expect(container.querySelector(".dk-rec-map")).toBeNull();
    const square = container.querySelector(".dk-rec-scope");
    if (!square) throw new Error("No map square");
    act(() => FakeIntersectionObserver.show(square));

    const map = await screen.findByRole("img", { name: "Map of Dallas yard" }, { timeout: 3000 });
    expect(map).not.toHaveClass("dk-live");
  });

  it("checks what falls due, worst first", async () => {
    fetchers.trailer.mockResolvedValue(
      trailer({
        status: "OutOfService",
        lastInspectionDate: NOW - 400 * DAY,
        registrationExpiry: NOW - DAY,
      }),
    );
    draw();

    await screen.findByRole("heading", { name: "Service calendar", level: 3 });
    const due = section("Service calendar");
    expect(within(due).getByText("3 need a look")).toBeInTheDocument();
    const titles = Array.from(due.querySelectorAll(".dk-tl-legend b")).map((b) => b.textContent);
    expect(titles).toEqual(["Out of service", "Annual inspection overdue", "Registration expired"]);
    expect(due.querySelector(".dk-tl-mark.dk-bad > b")).toHaveTextContent("Inspection due");
  });

  it("names an owner-operator and opens the shipments it carried in the card's place", async () => {
    fetchers.trailer.mockResolvedValue(
      trailer({
        ownershipType: "OwnerOperator",
        ownerWorker: { id: "wrk_9", firstName: "Ana", lastName: "Ruiz" },
      }),
    );
    draw();

    const owner = await screen.findByRole("heading", { name: "Owner", level: 3 });
    expect(
      within(owner.closest("section") as HTMLElement).getByRole("link", { name: /Ana Ruiz/ }),
    ).toHaveAttribute("href", expect.stringContaining("wrk_9"));

    fireEvent.click(await screen.findByRole("button", { name: /S26100090038417/ }));
    expect(await screen.findByText("Shipment card shp_1")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to TL-501" }));
    expect(
      await screen.findByRole("heading", { name: "2021 Wabash DuraPlate", level: 2 }),
    ).toBeInTheDocument();
  });

  it("stands with nothing around it: no location, no inspection, no loads", async () => {
    fetchers.trailer.mockResolvedValue(
      trailer({
        make: "",
        model: "",
        year: null,
        vin: "",
        maxLoadWeight: null,
        lastInspectionDate: null,
        registrationExpiry: null,
        lastKnownLocation: null,
      }),
    );
    fetchers.shipments.mockResolvedValue(page([], 0));
    draw();

    expect(await screen.findByRole("heading", { name: "TL-501", level: 2 })).toBeInTheDocument();
    expect(screen.getAllByText("No known location yet").length).toBeGreaterThan(0);
    expect(screen.getByRole("img", { name: "No known location" })).toBeInTheDocument();
    expect(screen.getByText("No maximum load on file")).toBeInTheDocument();
    expect(screen.getByText("No inspection on file")).toBeInTheDocument();
    expect(screen.getByText("No expiry on file")).toBeInTheDocument();
    expect(await screen.findByText("This trailer has not carried a load yet.")).toBeInTheDocument();
  });

  it("says a location that was never geocoded, and leaves the dot off", async () => {
    fetchers.trailer.mockResolvedValue(
      trailer({
        lastKnownLocation: { id: "loc_x", name: "Gate 4", latitude: null, longitude: null },
      }),
    );
    draw();

    expect(await screen.findByText("Not geocoded")).toBeInTheDocument();
    expect(
      screen.getByRole("img", { name: "This location has not been geocoded" }),
    ).toBeInTheDocument();
  });

  it("says the trailer could not be read, with the way back", async () => {
    fetchers.trailer.mockRejectedValue(new Error("Trailer not found"));
    const { onBack } = draw();

    expect(
      await screen.findByRole("heading", { name: "This trailer could not be read", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.getByText("Trailer not found")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalled();
  });

  it("opens with only what the person may read, asking for nothing else", async () => {
    seedPermissions({ [Resource.Trailer]: Operation.Read });
    draw();

    expect(
      await screen.findByRole("heading", { name: "2021 Wabash DuraPlate", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Recent shipments" })).toBeNull();
    expect(fetchers.shipments).not.toHaveBeenCalled();
  });

  it("leaves the shipments' outline out of the skeleton when the person may not read them", () => {
    seedPermissions({ [Resource.Trailer]: Operation.Read });
    fetchers.trailer.mockReturnValue(new Promise(() => {}));
    const { container } = draw();

    expect(container.querySelectorAll(".dk-rec-skel .dk-rec-sec")).toHaveLength(3);
  });

  it("draws the same card as an artifact, with nothing to step through", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DeskTrailerArtifact artifact={{} as never} recordId="trl_1" fallback={null} />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(
      await screen.findByRole("heading", { name: "2021 Wabash DuraPlate", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Next trailer" })).toBeNull();
  });
});
