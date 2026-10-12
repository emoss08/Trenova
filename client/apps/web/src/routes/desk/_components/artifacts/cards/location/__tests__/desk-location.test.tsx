import type {
  DeskLocation as DeskLocationRecord,
  LocationDwell,
} from "@/lib/graphql/location-card";
import type { RecordFailure } from "@/lib/graphql/record-failures";
import type { RecordShipment } from "@/lib/graphql/record-shipments";
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
import { DeskLocation, DeskLocationArtifact } from "../desk-location";

const fetchers = vi.hoisted(() => ({
  location: vi.fn(),
  dwell: vi.fn(),
  policies: vi.fn(),
  failures: vi.fn(),
  shipments: vi.fn(),
}));

vi.mock("@/lib/graphql/location-card", () => ({
  fetchDeskLocation: fetchers.location,
  fetchLocationDwell: fetchers.dwell,
  fetchLocationPolicies: fetchers.policies,
}));
vi.mock("@/lib/graphql/record-failures", () => ({ fetchRecordFailures: fetchers.failures }));
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
vi.mock("../../service-failure/desk-service-failure", () => ({
  DeskServiceFailure: ({
    recordId,
    onBack,
    backLabel,
  }: {
    recordId: string;
    onBack: () => void;
    backLabel?: string;
  }) => (
    <div>
      <p>Service failure card {recordId}</p>
      <button type="button" onClick={onBack}>
        {backLabel}
      </button>
    </div>
  ),
  DeskServiceFailureArtifact: () => null,
}));

const DAY = 86_400;
const NOW = Math.floor(Date.now() / 1000);

/** A location as the DeskLocationCard query returns it. */
function location(overrides: Partial<DeskLocationRecord> = {}): DeskLocationRecord {
  return {
    id: "loc_1",
    status: "Active",
    code: "DAL-YD",
    name: "Dallas yard",
    description: "",
    addressLine1: "1801 Commerce St",
    addressLine2: "",
    city: "Dallas",
    postalCode: "75201",
    timezone: "America/Chicago",
    isGeocoded: true,
    latitude: 32.7801,
    longitude: -96.8005,
    geofenceType: "circle",
    geofenceRadiusMeters: 250,
    geofenceVertices: [],
    createdAt: NOW - 400 * DAY,
    state: { abbreviation: "TX" },
    locationCategory: {
      id: "lc_1",
      name: "Company yard",
      type: "Terminal",
      facilityType: null,
      hasSecureParking: true,
      requiresAppointment: true,
      allowsOvernight: false,
      hasRestroom: false,
    },
    ...overrides,
  } as DeskLocationRecord;
}

const dwell = {
  locationId: "loc_1",
  stopCount: 40,
  breachCount: 6,
  avgDwellMinutes: 96,
  medianDwellMinutes: 80,
  p90DwellMinutes: 150,
  billedAmount: "1250",
  waivedAmount: "0",
  disputeCount: 0,
} as LocationDwell;

const failure = {
  id: "sf_1",
  number: "SF-2610-0042",
  type: "LateDelivery",
  status: "Open",
  stopType: "Delivery",
  lateMinutes: 45,
  actualArrival: NOW - DAY,
  shipment: { id: "shp_9", proNumber: "S2610-1366" },
  stop: null,
} as RecordFailure;

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

function draw(props: Partial<Parameters<typeof DeskLocation>[0]> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handlers = { onSelect: vi.fn(), onBack: vi.fn() };
  const view = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskLocation recordId="loc_1" order={["loc_0", "loc_1"]} {...handlers} {...props} />
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
    [Resource.Location]: Operation.Read,
    [Resource.DetentionPolicy]: Operation.Read,
    [Resource.Shipment]: Operation.Read,
    [Resource.ServiceFailure]: Operation.Read,
  });
  fetchers.location.mockReset().mockResolvedValue(location());
  fetchers.dwell.mockReset().mockResolvedValue(dwell);
  fetchers.policies.mockReset().mockResolvedValue({ rows: [], total: 0 });
  fetchers.failures.mockReset().mockResolvedValue({
    rows: [failure],
    total: 3,
    endCursor: "c1",
    hasNextPage: true,
  });
  fetchers.shipments.mockReset().mockResolvedValue({
    rows: [shipment],
    total: 11,
    endCursor: null,
    hasNextPage: false,
  });
});

afterEach(() => {
  clearPermissions();
  vi.unstubAllGlobals();
});

describe("DeskLocation", () => {
  it("holds the card's outline while the location is read, then draws where it is and how it treats trucks", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the location…");
    expect(container.querySelector(".dk-rec-skel .dk-rec-where")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-lo-dwell")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-lo-file")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "Dallas yard", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(screen.getByText("1801 Commerce St, Dallas, TX 75201")).toBeInTheDocument();
    expect(screen.getByText("Terminal")).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();

    expect(await screen.findByText("average dwell")).toBeInTheDocument();
    expect(document.querySelector(".dk-rec-amt b")).toHaveTextContent("1h 36m");

    const where = section("Where it is");
    expect(within(where).getByText("Circle fence, 250 m")).toBeInTheDocument();
    expect(within(where).getByText("Appointment required").closest("li")).toHaveClass("dk-off");
    expect(within(where).getByText("No overnight").closest("li")).toHaveClass("dk-off");

    const dwellSection = section("Dwell, last 90 days");
    expect(
      within(dwellSection).getByRole("img", { name: "Dwell distribution" }),
    ).toBeInTheDocument();
    expect(dwellSection.querySelector(".dk-lo-pin.dk-avg")).toHaveTextContent("1h 36mAverage");
    expect(dwellSection.querySelector(".dk-lo-breach")).toHaveTextContent(
      "15%6 of 40 stops ran past free time",
    );
    expect(
      within(dwellSection).getByText("$1,250.00 detention billed on 40 stops"),
    ).toBeInTheDocument();
    expect(fetchers.dwell).toHaveBeenCalledWith(
      "loc_1",
      expect.objectContaining({ from: expect.any(Number), to: expect.any(Number) }),
      expect.anything(),
    );

    expect(
      await within(section("Detention terms")).findByText(
        "No policy is written for this location; the customer's or the organization's applies.",
      ),
    ).toBeInTheDocument();
    expect(await screen.findByText("11 shipments")).toBeInTheDocument();
    expect(fetchers.shipments).toHaveBeenCalledWith(
      { kind: "location", id: "loc_1" },
      { first: 5, after: null },
      expect.anything(),
    );

    const failures = section("Service failures");
    expect(await within(failures).findByText("SF-2610-0042")).toBeInTheDocument();
    expect(within(failures).getByText("3 failures")).toBeInTheDocument();
    fireEvent.click(within(failures).getByRole("button", { name: /SF-2610-0042/ }));
    expect(await screen.findByText("Service failure card sf_1")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to Dallas yard" }));
    expect(
      await screen.findByRole("heading", { name: "Dallas yard", level: 2 }),
    ).toBeInTheDocument();
  });

  it("draws the fence round the dot once the map comes into view", async () => {
    installIntersectionObserver();
    const { container } = draw();

    await screen.findByRole("heading", { name: "Dallas yard", level: 2 });
    const square = container.querySelector(".dk-rec-scope");
    if (!square) throw new Error("No map square");
    act(() => FakeIntersectionObserver.show(square));

    const map = await screen.findByRole("img", { name: "Map of Dallas yard" }, { timeout: 3000 });
    expect(map.querySelector(".dk-rec-map-fence circle")).not.toBeNull();
  });

  it("lists the policies written for it, and how many more there are", async () => {
    fetchers.policies.mockResolvedValue({
      rows: [
        {
          id: "dp_1",
          name: "Dallas yard terms",
          code: "DAL",
          status: "Active",
          priority: 10,
          customerId: null,
          stopTypes: [],
          appointmentStopsOnly: false,
          billingFreeMinutes: 120,
          pickupFreeMinutes: 60,
          deliveryFreeMinutes: null,
          effectiveStartDate: null,
          effectiveEndDate: null,
        },
      ],
      total: 4,
    });
    draw();

    const terms = await screen.findByRole("heading", { name: "Detention terms", level: 3 });
    const sectionEl = terms.closest("section") as HTMLElement;
    expect(await within(sectionEl).findByText("Dallas yard terms")).toBeInTheDocument();
    expect(within(sectionEl).getByText("1h at pickup · 2h at delivery")).toBeInTheDocument();
    expect(within(sectionEl).getByText("1 of 4 policies")).toBeInTheDocument();
  });

  it("reads the next page of service failures on request", async () => {
    draw();
    expect(await screen.findByText("SF-2610-0042")).toBeInTheDocument();

    fetchers.failures.mockResolvedValue({
      rows: [{ ...failure, id: "sf_2", number: "SF-2610-0043" }],
      total: null,
      endCursor: null,
      hasNextPage: false,
    });
    fireEvent.click(await screen.findByRole("button", { name: "Show more" }));

    expect(await screen.findByText("SF-2610-0043")).toBeInTheDocument();
    expect(fetchers.failures).toHaveBeenLastCalledWith(
      { kind: "location", id: "loc_1" },
      { first: 5, after: "c1" },
      expect.anything(),
    );
  });

  it("opens a load that stopped here in the card's place", async () => {
    draw();

    fireEvent.click(await screen.findByRole("button", { name: /S26100090038417/ }));
    expect(await screen.findByText("Shipment card shp_1")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to Dallas yard" }));
    expect(
      await screen.findByRole("heading", { name: "Dallas yard", level: 2 }),
    ).toBeInTheDocument();
  });

  it("stands with nothing around it: no address, no coordinates, no stops, no failures", async () => {
    fetchers.location.mockResolvedValue(
      location({
        addressLine1: "",
        city: "",
        postalCode: "",
        state: null,
        latitude: null,
        longitude: null,
        isGeocoded: false,
        geofenceType: "auto",
        geofenceRadiusMeters: null,
        locationCategory: null,
      }),
    );
    fetchers.dwell.mockResolvedValue(null);
    fetchers.failures.mockResolvedValue({
      rows: [],
      total: 0,
      endCursor: null,
      hasNextPage: false,
    });
    fetchers.shipments.mockResolvedValue({
      rows: [],
      total: 0,
      endCursor: null,
      hasNextPage: false,
    });
    draw();

    expect(
      await screen.findByRole("heading", { name: "Dallas yard", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.getByText("No address on file")).toBeInTheDocument();
    expect(
      screen.getByRole("img", { name: "This location has not been geocoded" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Automatic fence")).toBeInTheDocument();
    expect(await screen.findByText("No stops scored yet")).toBeInTheDocument();
    expect(
      screen.getByText("No stop here has been scored for detention in the last 90 days."),
    ).toBeInTheDocument();
    expect(await screen.findByText("No load has stopped here yet.")).toBeInTheDocument();
    expect(
      await screen.findByText("No service failure has been recorded here."),
    ).toBeInTheDocument();
  });

  it("says a section that could not be read, and draws the rest", async () => {
    fetchers.dwell.mockRejectedValue(new Error("Detention analytics are not available"));
    fetchers.failures.mockRejectedValue(new Error("boom"));
    draw();

    expect(await screen.findByText("Detention analytics are not available")).toBeInTheDocument();
    expect(await screen.findByText("boom")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "On file", level: 3 })).toBeInTheDocument();
  });

  it("says the location could not be read, with the way back", async () => {
    fetchers.location.mockRejectedValue(new Error("Location not found"));
    const { onBack } = draw();

    expect(
      await screen.findByRole("heading", { name: "This location could not be read", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.getByText("Location not found")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalled();
    expect(fetchers.dwell).not.toHaveBeenCalled();
  });

  it("opens with only what the person may read, asking for nothing else", async () => {
    seedPermissions({ [Resource.Location]: Operation.Read });
    draw();

    expect(
      await screen.findByRole("heading", { name: "Dallas yard", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByText("average dwell")).toBeNull();
    expect(screen.queryByRole("heading", { name: "Dwell, last 90 days" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Recent shipments" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Service failures" })).toBeNull();
    expect(fetchers.dwell).not.toHaveBeenCalled();
    expect(fetchers.policies).not.toHaveBeenCalled();
    expect(fetchers.shipments).not.toHaveBeenCalled();
    expect(fetchers.failures).not.toHaveBeenCalled();
  });

  it("leaves the outlines of sections the person may not read out of the skeleton", () => {
    seedPermissions({ [Resource.Location]: Operation.Read });
    fetchers.location.mockReturnValue(new Promise(() => {}));
    const { container } = draw();

    expect(container.querySelectorAll(".dk-rec-skel .dk-rec-sec")).toHaveLength(2);
    expect(container.querySelector(".dk-rec-skel .dk-rec-amt")).toBeNull();
  });

  it("draws the same card as an artifact, with nothing to step through", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DeskLocationArtifact artifact={{} as never} recordId="loc_1" fallback={null} />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(
      await screen.findByRole("heading", { name: "Dallas yard", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Next location" })).toBeNull();
  });
});
