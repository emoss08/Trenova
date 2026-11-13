import {
  cleveland,
  kansasCity,
  move,
  outlook,
  readyQuality,
  slotStarts,
} from "@/lib/graphql/__tests__/freight-outlook-fixtures";
import type { FreightOutlook, FreightPositioning } from "@/lib/graphql/freight-outlook";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FreightOutlook as FreightOutlookView } from "../freight-outlook";
import { slotRange } from "../outlook-parts";

const mocks = vi.hoisted(() => ({
  outlook: vi.fn<() => Promise<FreightOutlook>>(),
  area: vi.fn(),
  positionDriver: vi.fn(),
  callOff: vi.fn(),
  allowed: true,
}));

vi.mock("@/lib/queries/freight-outlook", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/queries/freight-outlook")>();
  return {
    ...actual,
    freightOutlook: {
      _def: ["freightOutlook"],
      outlook: () => ({ queryKey: ["freightOutlook", "outlook"] as const, queryFn: mocks.outlook }),
      area: (cell: number, slotStart: number | null) => ({
        queryKey: ["freightOutlook", "area", cell, slotStart] as const,
        queryFn: () => mocks.area(cell, slotStart),
      }),
    },
  };
});

vi.mock("@/lib/graphql/freight-outlook", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/freight-outlook")>();
  return {
    ...actual,
    positionDriver: mocks.positionDriver,
    callOffPositioning: mocks.callOff,
  };
});

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.allowed, isLoading: false }),
}));

const onTheWay: FreightPositioning = {
  id: "dpo_1",
  workerId: "wrk_ana",
  workerName: "Ana Ruiz",
  fromLabel: "Kansas City, MO",
  toCell: cleveland.cell,
  toLabel: "Cleveland, OH",
  slotStart: slotStarts[1],
  freeAt: slotStarts[0],
  arriveAt: slotStarts[1],
  miles: 52.4,
  valueCents: 104_000,
  status: "Active",
  createdAt: slotStarts[0],
};

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <FreightOutlookView horizon="24" />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("FreightOutlook", () => {
  beforeEach(() => {
    window.localStorage.clear();
    mocks.allowed = true;
    mocks.area.mockImplementation(() => new Promise(() => {}));
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("reads the outlook in one paragraph written from its own figures", async () => {
    mocks.outlook.mockResolvedValue(outlook());

    renderPage();

    const summary = await screen.findByRole("region", { name: "Summary" });
    expect(summary).toHaveTextContent(/Freight outruns your drivers in Cleveland, OH through /);
    expect(summary).toHaveTextContent(", 3 loads with no truck nearby.");
    expect(summary).toHaveTextContent(
      "3 drivers will sit with little to haul, most in Kansas City, MO, and moving one of them first is worth $845.",
    );
    const figures = screen.getByRole("group", { name: "Outlook figures" });
    expect(within(figures).getByText("Loads with no truck nearby")).toBeInTheDocument();
    expect(within(figures).getByText("Enough drivers")).toBeInTheDocument();
    expect(within(figures).getByText("$845")).toBeInTheDocument();
  });

  it("counts the days the range held, out of the days on record", async () => {
    mocks.outlook.mockResolvedValue(
      outlook({
        record: [
          { dayStart: slotStarts[0] - 86_400, low: 10, high: 20, mean: 15, actual: 15 },
          { dayStart: slotStarts[0] - 172_800, low: 10, high: 20, mean: 15, actual: 25 },
        ],
      }),
    );

    renderPage();

    const record = await screen.findByRole("region", { name: "Track record" });
    expect(within(record).getByText("1 of 2 days")).toBeInTheDocument();
    expect(record).toHaveTextContent(
      "Each day forecast only from the weeks before it. 1 day was busier than the range; a typical day lands ±5 loads from the middle.",
    );
  });

  it("sends a driver and keeps them in the brief, with a way to undo it", async () => {
    const user = userEvent.setup();
    mocks.outlook
      .mockResolvedValueOnce(outlook())
      .mockResolvedValue(outlook({ moves: [], positionings: [onTheWay] }));
    mocks.positionDriver.mockResolvedValue(onTheWay);

    renderPage();

    const row = await screen.findByRole("listitem", { name: "Send Ana Ruiz to Cleveland, OH" });
    await user.click(within(row).getByRole("button", { name: "Send" }));

    expect(mocks.positionDriver).toHaveBeenCalledWith({
      workerId: "wrk_ana",
      toCell: cleveland.cell,
      slotStart: slotStarts[1],
    });
    const sent = await screen.findByText("Ana Ruiz");
    expect(sent.closest("li")).toHaveTextContent("Sent Ana Ruiz to Cleveland, OH");
    mocks.callOff.mockResolvedValue({ ...onTheWay, status: "Canceled" });
    await user.click(screen.getByRole("button", { name: "On the way · Undo" }));
    expect(mocks.callOff).toHaveBeenCalledWith("dpo_1");
  });

  it("offers no send to someone who may not position drivers", async () => {
    mocks.allowed = false;
    mocks.outlook.mockResolvedValue(outlook());

    renderPage();

    const row = await screen.findByRole("listitem", { name: "Send Ana Ruiz to Cleveland, OH" });
    expect(within(row).queryByRole("button", { name: "Send" })).toBeNull();
  });

  it("opens a keep in the console with the other driver picked", async () => {
    mocks.outlook.mockResolvedValue(
      outlook({
        moves: [
          move({
            kind: "Keep",
            workerName: "Dana Ortiz",
            moveId: "sm_out",
            proNumber: "PRO-OUT",
            partnerId: "wrk_joe",
            partnerName: "Joe Kowalski",
            partnerCell: kansasCity.cell,
            partnerLabel: "Indianapolis, IN",
            partnerFreeAt: slotStarts[0],
          }),
        ],
      }),
    );

    renderPage();

    const review = await screen.findByText("Review");
    expect(review.closest("a")).toHaveAttribute(
      "href",
      "/dispatch/console?move=sm_out&worker=wrk_joe",
    );
    expect(screen.getByText("Joe Kowalski takes their next load, PRO-OUT")).toBeInTheDocument();
  });

  it("hides a move for now and brings it back", async () => {
    const user = userEvent.setup();
    mocks.outlook.mockResolvedValue(outlook());

    renderPage();

    const row = await screen.findByRole("listitem", { name: "Send Ana Ruiz to Cleveland, OH" });
    await user.click(within(row).getByRole("button", { name: "Not now" }));

    expect(screen.queryByRole("listitem", { name: "Send Ana Ruiz to Cleveland, OH" })).toBeNull();
    await user.click(screen.getByRole("button", { name: "Show 1 hidden move" }));
    expect(
      screen.getByRole("listitem", { name: "Send Ana Ruiz to Cleveland, OH" }),
    ).toBeInTheDocument();
  });

  it("keeps a hidden move hidden after an area is opened and closed", async () => {
    const user = userEvent.setup();
    mocks.outlook.mockResolvedValue(outlook());

    renderPage();

    const row = await screen.findByRole("listitem", { name: "Send Ana Ruiz to Cleveland, OH" });
    await user.click(within(row).getByRole("button", { name: "Not now" }));
    const board = screen.getByRole("region", { name: "Loads against drivers" });
    await user.click(within(board).getByRole("button", { name: /^Cleveland, OH/ }));
    await user.keyboard("{Escape}");

    expect(screen.queryByRole("listitem", { name: "Send Ana Ruiz to Cleveland, OH" })).toBeNull();
    expect(screen.getByRole("button", { name: "Show 1 hidden move" })).toBeInTheDocument();
  });

  it("re-reads the page at one window when it is picked on the board", async () => {
    const user = userEvent.setup();
    mocks.outlook.mockResolvedValue(outlook());

    renderPage();
    await screen.findByRole("region", { name: "Summary" });

    const first = screen
      .getAllByRole("button", { pressed: false })
      .find((button) => button.getAttribute("title") === slotRange(slotStarts[0]));
    expect(first).toBeDefined();
    await user.click(first as HTMLElement);

    expect(screen.getByRole("button", { name: "Show all" })).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Summary" }).textContent).toContain(
      `Your drivers cover the freight on ${slotRange(slotStarts[0])}.`,
    );
  });

  it("opens an area in the inspector at its worst window, and Escape brings the brief back", async () => {
    const user = userEvent.setup();
    mocks.outlook.mockResolvedValue(outlook());

    renderPage();

    const board = await screen.findByRole("region", { name: "Loads against drivers" });
    await user.click(within(board).getByRole("button", { name: /^Cleveland, OH/ }));

    await waitFor(() => expect(mocks.area).toHaveBeenCalledWith(cleveland.cell, slotStarts[1]));
    expect(await screen.findByLabelText("Loading the area")).toBeInTheDocument();
    expect(screen.getByRole("complementary", { name: "Area" })).toBeInTheDocument();

    await user.keyboard("{Escape}");
    expect(screen.getByRole("complementary", { name: "Brief" })).toBeInTheDocument();
  });

  it("folds the brief to a rail that still counts the moves, and unfolds it again", async () => {
    const user = userEvent.setup();
    mocks.outlook.mockResolvedValue(outlook());

    renderPage();

    await user.click(await screen.findByRole("button", { name: "Hide the brief" }));

    expect(screen.queryByRole("listitem", { name: "Send Ana Ruiz to Cleveland, OH" })).toBeNull();
    const rail = screen.getByRole("button", { name: "Show the brief" }).parentElement;
    expect(rail).toHaveTextContent("Brief1");
    expect(window.localStorage.getItem("trenova-freight-outlook-brief-open")).toBe("false");

    await user.click(screen.getByRole("button", { name: "Show the brief" }));
    expect(
      screen.getByRole("listitem", { name: "Send Ana Ruiz to Cleveland, OH" }),
    ).toBeInTheDocument();
  });

  it("unfolds a folded brief when an area is opened", async () => {
    const user = userEvent.setup();
    window.localStorage.setItem("trenova-freight-outlook-brief-open", "false");
    mocks.outlook.mockResolvedValue(outlook());

    renderPage();

    const board = await screen.findByRole("region", { name: "Loads against drivers" });
    expect(screen.getByRole("button", { name: "Show the brief" })).toBeInTheDocument();
    await user.click(within(board).getByRole("button", { name: /^Cleveland, OH/ }));

    expect(screen.queryByRole("button", { name: "Show the brief" })).toBeNull();
    expect(await screen.findByLabelText("Loading the area")).toBeInTheDocument();
  });

  it("says why one more driver is worth nothing instead of leaving it blank", async () => {
    mocks.outlook.mockResolvedValue(
      outlook({
        areas: [
          { ...cleveland, valueCents: 0 },
          { ...kansasCity, valueCents: 0 },
        ],
      }),
    );

    renderPage();

    const board = await screen.findByRole("region", { name: "Loads against drivers" });
    expect(within(board).getByRole("button", { name: /^Kansas City, MO/ })).toHaveTextContent(
      "Not needed",
    );
    expect(within(board).getByRole("button", { name: /^Cleveland, OH/ })).toHaveTextContent(
      "No price",
    );
  });

  it("says why there is no outlook while the forecast is not ready", async () => {
    mocks.outlook.mockResolvedValue(
      outlook({ quality: { ...readyQuality, status: "NoSkill" }, areas: [], moves: [] }),
    );

    renderPage();

    expect(
      await screen.findByRole("heading", { name: "Not better than last week yet" }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("group", { name: "History counted so far" })).toBeNull();
    expect(screen.getByRole("status")).toHaveTextContent(
      `Last checked ${formatUnixDateTimeMedium(readyQuality.fittedAt)}. Checked again every night, and whenever this is opened more than an hour later.`,
    );
    expect(screen.queryByRole("region", { name: "Summary" })).toBeNull();
  });

  it("shows how much history the forecast has and that it is being counted again", async () => {
    mocks.outlook.mockResolvedValue(
      outlook({
        quality: {
          ...readyQuality,
          status: "InsufficientHistory",
          historyWeeks: 11,
          historyLoads: 35,
          refitting: true,
        },
        areas: [],
        moves: [],
      }),
    );

    renderPage();

    expect(
      await screen.findByRole("heading", { name: "Not enough history yet" }),
    ).toBeInTheDocument();
    const history = screen.getByRole("group", { name: "History counted so far" });
    expect(within(history).getByText("11 of 8")).toBeInTheDocument();
    expect(within(history).getByText("35 of 200")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent(
      `Last checked ${formatUnixDateTimeMedium(readyQuality.fittedAt)}. Checking your shipment history again now; this updates when it finishes.`,
    );
  });
});
