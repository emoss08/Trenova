import type { RecordShipment } from "@/lib/graphql/record-shipments";
import type {
  DeskTractor as DeskTractorRecord,
  DeskTractorInspection,
  DeskTractorPosition,
} from "@/lib/graphql/tractor-card";
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
import { DeskTractor, DeskTractorArtifact } from "../desk-tractor";

const fetchers = vi.hoisted(() => ({
  tractor: vi.fn(),
  position: vi.fn(),
  inspections: vi.fn(),
  shipments: vi.fn(),
}));

vi.mock("@/lib/graphql/tractor-card", () => ({
  fetchDeskTractor: fetchers.tractor,
  fetchTractorPosition: fetchers.position,
  fetchTractorInspections: fetchers.inspections,
}));
vi.mock("@/lib/graphql/record-shipments", () => ({
  fetchRecordShipments: fetchers.shipments,
}));
// The shipment card is its own chunk; here it stands in as what it was given,
// with its way back.
vi.mock("../../shipment/desk-shipment", () => ({
  DeskShipment: ({
    recordId,
    order,
    onBack,
    backLabel,
  }: {
    recordId: string;
    order: readonly string[];
    onBack: () => void;
    backLabel?: string;
  }) => (
    <div>
      <p>Shipment card {recordId}</p>
      <p>Stepping through {order.join(", ")}</p>
      <button type="button" onClick={onBack}>
        {backLabel}
      </button>
    </div>
  ),
  DeskShipmentArtifact: () => null,
}));

const DAY = 86_400;
const NOW = Math.floor(Date.now() / 1000);

/** A tractor as the DeskTractorCard query returns it. */
function tractor(overrides: Partial<DeskTractorRecord> = {}): DeskTractorRecord {
  return {
    id: "tr_1",
    code: "T-101",
    status: "Available",
    make: "Freightliner",
    model: "Cascadia",
    year: 2022,
    vin: "3AKJHHDR5NSNA1234",
    licensePlateNumber: "TX-48812",
    registrationNumber: "REG-99812",
    registrationExpiry: NOW + 200 * DAY,
    externalId: "SAM-4471",
    fuelType: "Diesel",
    iftaQualified: true,
    ownershipType: "CompanyOwned",
    lessorName: "",
    leaseReference: "",
    leaseEndDate: null,
    createdAt: 1_700_000_000,
    state: { abbreviation: "TX" },
    equipmentType: { id: "et_1", code: "SLP", description: "Sleeper tractor" },
    equipmentManufacturer: { id: "em_1", name: "Freightliner" },
    fleetCode: { id: "fc_1", code: "DAL" },
    primaryWorker: { id: "wrk_1", firstName: "Rosa", lastName: "Diaz" },
    secondaryWorker: { id: "wrk_2", firstName: "Sam", lastName: "Lee" },
    ownerWorker: null,
    lastKnownLocation: { id: "loc_1", name: "Dallas yard" },
    ...overrides,
  } as DeskTractorRecord;
}

function position(overrides: Partial<DeskTractorPosition> = {}): DeskTractorPosition {
  return {
    latitude: 32.7767,
    longitude: -96.797,
    headingDegrees: 40,
    speedMph: 58,
    engineState: "On",
    fuelPercent: 62,
    odometerMeters: 293_100_000,
    formattedLocation: "I-35E near Waxahachie, TX",
    recordedAt: NOW - 120,
    ...overrides,
  };
}

function inspection(id: string, overrides: Partial<DeskTractorInspection> = {}) {
  return {
    id,
    inspectionType: "preTrip",
    safetyStatus: "safe",
    workerName: "Rosa Diaz",
    startedAt: NOW - DAY,
    location: "Dallas, TX",
    odometerMeters: 293_000_000,
    defectCount: 0,
    unresolvedDefectCount: 0,
    ...overrides,
  } satisfies DeskTractorInspection;
}

const shipment = {
  id: "shp_1",
  proNumber: "S26100090038417",
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
          location: { city: "Dallas", state: { abbreviation: "TX" } },
        },
      ],
    },
  ],
} as RecordShipment;

const EVERYTHING = {
  [Resource.Tractor]: Operation.Read,
  [Resource.Shipment]: Operation.Read,
};

function draw(props: Partial<Parameters<typeof DeskTractor>[0]> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handlers = { onSelect: vi.fn(), onBack: vi.fn() };
  const view = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskTractor recordId="tr_1" order={["tr_0", "tr_1", "tr_2"]} {...handlers} {...props} />
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
  seedPermissions(EVERYTHING);
  for (const fn of Object.values(fetchers)) fn.mockReset();
  fetchers.tractor.mockResolvedValue(tractor());
  fetchers.position.mockResolvedValue(position());
  fetchers.inspections.mockResolvedValue({ rows: [inspection("vinsp_1")], next: null });
  fetchers.shipments.mockResolvedValue({
    rows: [shipment],
    total: 14,
    endCursor: null,
    hasNextPage: false,
  });
});

afterEach(() => {
  clearPermissions();
});

describe("DeskTractor", () => {
  it("holds the card's outline while the tractor is read, then draws what it is and where", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the tractor…");
    expect(container.querySelector(".dk-rec-skel .dk-rec-where")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-tr-spec")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-tr-dash")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "2022 Freightliner Cascadia", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(screen.getByText("T-101")).toBeInTheDocument();
    expect(screen.getByText("Fleet DAL")).toBeInTheDocument();
    expect(screen.getByText("VIN 3AKJHHDR5NSNA1234")).toBeInTheDocument();
    expect(screen.getByText("Available")).toBeInTheDocument();
    expect(screen.getByText("Driven by Rosa Diaz")).toBeInTheDocument();

    const where = section("Where it is");
    expect(await within(where).findByText("I-35E near Waxahachie, TX")).toBeInTheDocument();
    expect(within(where).getByText("Moving 58 mph NE")).toBeInTheDocument();

    expect(within(where).getByRole("link", { name: /Dallas yard/ })).toHaveAttribute(
      "href",
      expect.stringContaining("loc_1"),
    );
    expect(await screen.findByText("182,124 mi")).toBeInTheDocument();

    const dash = where.querySelector(".dk-tr-dash") as HTMLElement;
    expect(within(dash).getByText("58")).toBeInTheDocument();
    expect(within(dash).getByText("Heading NE")).toBeInTheDocument();
    expect(within(dash).getByText("Engine on")).toBeInTheDocument();
    expect(within(dash).getByRole("meter", { name: "Fuel" })).toBeInTheDocument();
    expect(within(dash).getByText("182,124")).toBeInTheDocument();

    const drivers = section("Drivers");
    expect(drivers.querySelector(".dk-tr-seat .dk-rec-av")).toHaveTextContent("RD");
    expect(within(section("Specification")).getByText("3AKJHHDR5NSNA1234")).toBeInTheDocument();
  });

  it("reads the map only once its square comes within reach of the screen", async () => {
    installIntersectionObserver();
    try {
      const { container } = draw();

      await screen.findByText("I-35E near Waxahachie, TX");
      expect(container.querySelector(".dk-rec-map")).toBeNull();
      expect(container.querySelector('img[src*="openstreetmap"]')).toBeNull();

      const square = container.querySelector(".dk-rec-scope");
      if (!square) throw new Error("No map square");
      act(() => FakeIntersectionObserver.show(square));

      const map = await screen.findByRole(
        "img",
        { name: "Map of I-35E near Waxahachie, TX" },
        { timeout: 3000 },
      );
      const tiles = map.querySelectorAll("img");
      expect(tiles.length).toBeGreaterThan(0);
      for (const tile of tiles) {
        expect(tile.getAttribute("src")).toMatch(/^https:\/\/tile\.openstreetmap\.org\/12\//u);
        expect(tile).toHaveAttribute("loading", "lazy");
        expect(tile).toHaveAttribute("draggable", "false");
      }
      expect(within(map).getByRole("link", { name: "© OpenStreetMap" })).toHaveAttribute(
        "href",
        "https://www.openstreetmap.org/copyright",
      );
      expect(map).toHaveClass("dk-live");
    } finally {
      vi.unstubAllGlobals();
    }
  });

  it("checks what falls due, worst first, once the last inspection is read", async () => {
    fetchers.tractor.mockResolvedValue(tractor({ registrationExpiry: NOW + 10 * DAY }));
    fetchers.inspections.mockResolvedValue({
      rows: [inspection("vinsp_1", { unresolvedDefectCount: 2, defectCount: 2 })],
      next: null,
    });
    draw();

    const due = await screen.findByRole("heading", { name: "Due", level: 3 });
    const checks = due.closest("section") as HTMLElement;
    expect(await within(checks).findByText("2 open defects")).toBeInTheDocument();
    expect(within(checks).getByText(/Registration expires in/)).toBeInTheDocument();
    expect(within(checks).getByText("2 need a look")).toBeInTheDocument();
    const titles = Array.from(checks.querySelectorAll(".dk-rec-cb > b")).map((b) => b.textContent);
    expect(titles[0]).toBe("2 open defects");
  });

  it("lists the drivers, the loads it pulled and its inspections", async () => {
    draw();

    const drivers = await screen.findByRole("heading", { name: "Drivers", level: 3 });
    const people = drivers.closest("section") as HTMLElement;
    expect(within(people).getByRole("link", { name: /Rosa Diaz/ })).toHaveAttribute(
      "href",
      expect.stringContaining("wrk_1"),
    );
    expect(within(people).getByText("Co-driver")).toBeInTheDocument();

    const row = await screen.findByRole("button", { name: /S26100090038417/ });
    expect(within(row).getByText(/Atlanta, GA/)).toHaveTextContent("Atlanta, GA → Dallas, TX");
    expect(within(section("Recent shipments")).getByText("14 shipments")).toBeInTheDocument();
    expect(fetchers.shipments).toHaveBeenCalledWith(
      { kind: "tractor", id: "tr_1" },
      { first: 5, after: null },
      expect.anything(),
    );

    const inspections = section("Inspections");
    expect(await within(inspections).findByText("No defects")).toBeInTheDocument();
    expect(within(inspections).getByText(/Rosa Diaz · Dallas, TX/)).toBeInTheDocument();
  });

  it("opens a listed shipment in the card's place, and goes back to the tractor", async () => {
    fetchers.shipments.mockResolvedValue({
      rows: [shipment, { ...shipment, id: "shp_2", proNumber: "S2" }],
      total: 2,
      endCursor: null,
      hasNextPage: false,
    });
    draw();

    fireEvent.click(await screen.findByRole("button", { name: /S26100090038417/ }));

    expect(await screen.findByText("Shipment card shp_1")).toBeInTheDocument();
    expect(screen.getByText("Stepping through shp_1, shp_2")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to T-101" }));
    expect(
      await screen.findByRole("heading", { name: "2022 Freightliner Cascadia", level: 2 }),
    ).toBeInTheDocument();
  });

  it("reads older inspections on request, from before the last one shown", async () => {
    fetchers.inspections
      .mockResolvedValueOnce({
        rows: [inspection("vinsp_3"), inspection("vinsp_2")],
        next: { before: NOW - DAY, beforeId: "vinsp_2" },
      })
      .mockResolvedValueOnce({
        rows: [inspection("vinsp_1", { inspectionType: "postTrip", defectCount: 1 })],
        next: null,
      });
    draw();

    const inspections = await screen.findByRole("heading", { name: "Inspections", level: 3 });
    const list = inspections.closest("section") as HTMLElement;
    fireEvent.click(await within(list).findByRole("button", { name: "Show older" }));

    expect(await within(list).findByText("1 defect fixed")).toBeInTheDocument();
    expect(fetchers.inspections).toHaveBeenLastCalledWith(
      "tr_1",
      { size: 5, cursor: { before: NOW - DAY, beforeId: "vinsp_2" } },
      expect.anything(),
    );
    expect(within(list).getAllByRole("listitem")).toHaveLength(3);
    expect(within(list).queryByRole("button", { name: "Show older" })).toBeNull();
  });

  it("stands with nothing around it: no GPS, no inspections, no loads", async () => {
    fetchers.tractor.mockResolvedValue(
      tractor({ lastKnownLocation: null, secondaryWorker: null, vin: "" }),
    );
    fetchers.position.mockResolvedValue(null);
    fetchers.inspections.mockResolvedValue({ rows: [], next: null });
    fetchers.shipments.mockResolvedValue({
      rows: [],
      total: 0,
      endCursor: null,
      hasNextPage: false,
    });
    draw();

    expect(await screen.findByText("No GPS report from this tractor")).toBeInTheDocument();
    expect(screen.getByText("No known location yet")).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "No position reported" })).toBeInTheDocument();
    expect(document.querySelector(".dk-rec-map")).toBeNull();
    expect(screen.getByText("No odometer reading")).toBeInTheDocument();
    expect(screen.getByText("No VIN on file")).toBeInTheDocument();
    expect(
      await screen.findByText("No driver has inspected this tractor through telematics."),
    ).toBeInTheDocument();
    expect(await screen.findByText("No driver inspection on record")).toBeInTheDocument();
    expect(await screen.findByText("This tractor has not pulled a load yet.")).toBeInTheDocument();
  });

  it("places a tractor with no GPS by the last move that left it somewhere", async () => {
    fetchers.position.mockResolvedValue(null);
    draw();

    expect(await screen.findByText("Placed by its last move, not by GPS")).toBeInTheDocument();
    expect(within(section("Where it is")).getByText("Dallas yard")).toBeInTheDocument();
  });

  it("says a stale report is where the tractor was, not where it is", async () => {
    fetchers.position.mockResolvedValue(
      position({ recordedAt: NOW - 5 * 3600, speedMph: 0, engineState: "Off" }),
    );
    const { container } = draw();

    expect(await screen.findByText("Was stopped")).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-where.dk-live")).toBeNull();
  });

  it("says the tractor could not be read, with the way back", async () => {
    fetchers.tractor.mockRejectedValue(new Error("Tractor not found"));
    const { onBack } = draw();

    expect(
      await screen.findByRole("heading", { name: "This tractor could not be read", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.getByText("Tractor not found")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalled();
  });

  it("says a section could not be read in its own place, and the rest of the card stands", async () => {
    fetchers.position.mockRejectedValue(new Error("Telematics is down"));
    fetchers.inspections.mockRejectedValue(new Error("Inspections are unavailable"));
    draw();

    expect(await screen.findByText("Telematics is down")).toBeInTheDocument();
    expect(section("Where it is")).toContainElement(screen.getByText("Telematics is down"));
    expect(await screen.findByText("Inspections are unavailable")).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: /S26100090038417/ })).toBeInTheDocument();
  });

  // A card opens for anyone who may read its record. What it draws from other
  // records is left out for someone who may not read them, and never asked for.
  it("opens with only what the person may read, asking for nothing else", async () => {
    seedPermissions({ [Resource.Tractor]: Operation.Read });
    draw();

    expect(
      await screen.findByRole("heading", { name: "2022 Freightliner Cascadia", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Recent shipments" })).toBeNull();
    expect(fetchers.shipments).not.toHaveBeenCalled();
    expect(screen.getByRole("link", { name: /Open tractor/ })).toBeInTheDocument();
  });

  it("leaves a section's outline out of the skeleton when the person may not read it", () => {
    seedPermissions({ [Resource.Tractor]: Operation.Read });
    fetchers.tractor.mockReturnValue(new Promise(() => {}));
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the tractor…");
    const titles = container.querySelectorAll(".dk-rec-skel .dk-rec-sec");
    expect(titles).toHaveLength(5);
  });

  it("keeps long names and places inside the card", async () => {
    const longPlace =
      "Pilot Travel Center 4471, Interstate 35 East Frontage Road, northbound exit 401B, " +
      "Waxahachie, Ellis County, Texas 75165, United States";
    fetchers.tractor.mockResolvedValue(
      tractor({ make: "Freightliner Trucks of North America", model: "Cascadia 126 DayCab" }),
    );
    fetchers.position.mockResolvedValue(position({ formattedLocation: longPlace }));
    draw();

    expect(
      await screen.findByRole("heading", {
        name: "2022 Freightliner Trucks of North America Cascadia 126 DayCab",
        level: 2,
      }),
    ).toBeInTheDocument();
    const place = await screen.findByText(longPlace);
    expect(place.tagName).toBe("B");
    expect(place.closest(".dk-rec-at")).not.toBeNull();
  });

  it("steps through the table's rows from its head", async () => {
    const { onSelect } = draw();

    await screen.findByRole("heading", { name: "2022 Freightliner Cascadia", level: 2 });
    expect(screen.getByText("2 of 3")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Next tractor" }));
    expect(onSelect).toHaveBeenCalledWith("tr_2");
  });

  it("draws the same card as an artifact, with nothing to step through", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DeskTractorArtifact artifact={{} as never} recordId="tr_1" fallback={null} />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(
      await screen.findByRole("heading", { name: "2022 Freightliner Cascadia", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Next tractor" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Back to the table" })).toBeNull();
  });
});
