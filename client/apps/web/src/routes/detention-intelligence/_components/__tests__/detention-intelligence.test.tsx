import type { DetentionIntelligence as Intelligence } from "@/lib/graphql/detention-intelligence";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DetentionIntelligence } from "../detention-intelligence";

const mocks = vi.hoisted(() => ({ fetchDetentionIntelligence: vi.fn() }));

vi.mock("@/lib/graphql/detention-intelligence", () => ({
  fetchDetentionIntelligence: mocks.fetchDetentionIntelligence,
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const split = { stopCount: 0, detainedCount: 0, pastFreeMinutes: 0 };

function totals(overrides: Partial<Intelligence["totals"]> = {}): Intelligence["totals"] {
  return {
    stopCount: 120,
    detainedCount: 60,
    pastFreeMinutes: 6_000,
    dockMinutes: 4_200,
    openMinutes: 600,
    earlyMinutes: 600,
    lateMinutes: 600,
    billedAmount: "9000",
    driverPayAmount: "4000",
    netMargin: "5000",
    waivedAmount: "500",
    disputeCount: 1,
    suppressedCount: 0,
    noticeDueCount: 10,
    noticeOnTimeCount: 9,
    ...overrides,
  };
}

function intelligence(overrides: Partial<Intelligence> = {}): Intelligence {
  return {
    days: 90,
    through: "2030-05-29",
    from: 1_900_000_000,
    to: 1_907_776_000,
    previousFrom: 1_892_224_000,
    previousTo: 1_899_999_999,
    timezone: "UTC",
    truncated: false,
    totals: totals(),
    previous: totals({ pastFreeMinutes: 5_000 }),
    facilities: [
      {
        locationId: "loc_ridgeway",
        locationName: "Ridgeway DC",
        locationCity: "Joliet",
        locationState: "IL",
        stopCount: 80,
        detainedCount: 50,
        pastFreeMinutes: 5_000,
        previousPastFreeMinutes: 4_000,
        dockMinutes: 4_000,
        openMinutes: 400,
        earlyMinutes: 300,
        lateMinutes: 300,
        avgDwellMinutes: 190,
        medianDwellMinutes: 188,
        p90DwellMinutes: 412,
        medianFreeMinutes: 120,
        billedAmount: "7000",
        driverPayAmount: "3000",
        netMargin: "4000",
        waivedAmount: "400",
        disputeCount: 1,
        suppressedCount: 0,
        noticeDueCount: 8,
        noticeOnTimeCount: 7,
      },
    ],
    customers: [],
    waivers: [],
    recovery: {
      accrued: "10000",
      billed: "8000",
      invoiced: "7000",
      collected: "6000",
      steps: [{ bucket: "Waived", occurrenceCount: 4, amount: "500" }],
    },
    arrivals: [],
    splits: {
      pickup: split,
      delivery: split,
      appointment: split,
      open: split,
      live: split,
      drop: split,
    },
    weeks: [],
    policyChanges: [],
    summary: [
      {
        segments: [
          { text: "Trucks waited ", strong: false, tone: null, target: null, targetId: null },
          { text: "100 h", strong: true, tone: null, target: null, targetId: null },
          {
            text: " past free time, most of it at ",
            strong: false,
            tone: null,
            target: null,
            targetId: null,
          },
          {
            text: "Ridgeway DC",
            strong: false,
            tone: null,
            target: "Facility",
            targetId: "loc_ridgeway",
          },
          { text: ".", strong: false, tone: null, target: null, targetId: null },
        ],
      },
    ],
    nextSteps: [
      {
        kind: "ShipperReview",
        title: [
          {
            text: "Take the scorecard to ",
            strong: false,
            tone: null,
            target: null,
            targetId: null,
          },
          {
            text: "Ridgeway DC",
            strong: false,
            tone: null,
            target: "Facility",
            targetId: "loc_ridgeway",
          },
        ],
        detail: "67 h of waiting with the truck on time for its appointment.",
        impactMinutes: 4_000,
        impactAmount: null,
        impactCaption: "on-time waiting",
        locationId: "loc_ridgeway",
        customerId: null,
      },
    ],
    ...overrides,
  };
}

/** Where the page has navigated, so a test can assert the route an action opens. */
function Location() {
  const location = useLocation();
  return <output aria-label="Location">{`${location.pathname}${location.search}`}</output>;
}

function renderPage() {
  render(
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
    >
      <MemoryRouter>
        <DetentionIntelligence days={90} onWiden={() => {}} />
        <Location />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("detention intelligence", () => {
  it("leads with the server's summary and the next steps it ranked", async () => {
    mocks.fetchDetentionIntelligence.mockResolvedValue(intelligence());
    renderPage();

    const summary = await screen.findByRole("region", { name: "Summary" });
    expect(summary).toHaveTextContent(
      "Trucks waited 100 h past free time, most of it at Ridgeway DC.",
    );
    const steps = screen.getByRole("region", { name: "What to do next" });
    expect(within(steps).getByText("Shipper")).toBeInTheDocument();
    expect(within(steps).getByText("67 h")).toBeInTheDocument();
    expect(within(steps).getByText("on-time waiting")).toBeInTheDocument();
  });

  it("asks for the page's number of days ending today, leaving the days to the organization's zone", async () => {
    mocks.fetchDetentionIntelligence.mockResolvedValue(intelligence());
    renderPage();
    await screen.findByRole("region", { name: "Summary" });

    expect(mocks.fetchDetentionIntelligence).toHaveBeenCalledTimes(1);
    const [input] = mocks.fetchDetentionIntelligence.mock.calls[0] ?? [];
    expect(input).toEqual({ days: 90 });
  });

  it("opens the facility's scorecard from the step that names it", async () => {
    mocks.fetchDetentionIntelligence.mockResolvedValue(intelligence());
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("button", { name: /Open scorecard/ }));

    const sheet = await screen.findByRole("dialog");
    expect(within(sheet).getByText("Facility scorecard · last 90 days")).toBeInTheDocument();
    expect(within(sheet).getByRole("heading", { name: "Ridgeway DC" })).toBeInTheDocument();
    expect(within(sheet).getByRole("button", { name: /Export scorecard/ })).toBeInTheDocument();
    expect(
      within(sheet).getByText(
        "The typical stop here takes 1h 8m longer than the free time granted, so detention is built into the schedule.",
      ),
    ).toBeInTheDocument();
  });

  it("opens the detention desk searched to the scorecard's facility from See occurrences", async () => {
    mocks.fetchDetentionIntelligence.mockResolvedValue(intelligence());
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByRole("button", { name: /Open scorecard/ }));
    const sheet = await screen.findByRole("dialog");

    await user.click(within(sheet).getByRole("button", { name: "See occurrences" }));

    expect(screen.getByRole("status", { name: "Location" })).toHaveTextContent(
      "/detention/desk?q=Ridgeway+DC",
    );
  });

  it("reads the scorecard over the window the page read, narrowed to the facility", async () => {
    mocks.fetchDetentionIntelligence.mockResolvedValue(intelligence());
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("button", { name: /Open scorecard/ }));
    await screen.findByRole("dialog");

    expect(mocks.fetchDetentionIntelligence).toHaveBeenCalledTimes(2);
    const [scoped] = mocks.fetchDetentionIntelligence.mock.calls[1] ?? [];
    expect(scoped).toEqual({
      days: 90,
      through: "2030-05-29",
      locationId: "loc_ridgeway",
    });
  });

  it("opens the scorecard from a facility the summary names", async () => {
    mocks.fetchDetentionIntelligence.mockResolvedValue(intelligence());
    const user = userEvent.setup();
    renderPage();

    const summary = await screen.findByRole("region", { name: "Summary" });
    await user.click(within(summary).getByRole("link", { name: "Ridgeway DC" }));

    expect(await screen.findByRole("dialog")).toHaveTextContent("Ridgeway DC");
  });

  it("draws what was kept as hidden to a reader without the margins group", async () => {
    mocks.fetchDetentionIntelligence.mockResolvedValue(
      intelligence({ totals: totals({ driverPayAmount: null, netMargin: null }) }),
    );
    renderPage();

    await screen.findByRole("region", { name: "Summary" });
    expect(screen.getByText("Needs access to margins")).toBeInTheDocument();
    expect(screen.getAllByText("Hidden").length).toBeGreaterThan(0);
    expect(screen.queryByText("$5,000")).toBeNull();
  });

  it("shows the empty sheet when no stop settled detention", async () => {
    mocks.fetchDetentionIntelligence.mockResolvedValue(
      intelligence({
        totals: totals({ stopCount: 0 }),
        facilities: [],
        summary: [],
        nextSteps: [],
      }),
    );
    renderPage();

    expect(await screen.findByText("No detention in this window")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Look back 180 days" })).toBeInTheDocument();
  });
});
