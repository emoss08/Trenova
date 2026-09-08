import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import RandomTestingConsole from "../random-testing-console";

const mocks = vi.hoisted(() => ({
  fetchDotRandomPools: vi.fn(),
  fetchDotRandomDraws: vi.fn(),
  fetchDotRandomDraw: vi.fn(),
  runDotRandomDraw: vi.fn(),
  finalizeDotRandomDraw: vi.fn(),
  cancelDotRandomDraw: vi.fn(),
  updateDotRandomDrawEntry: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/lib/graphql/worker-drug-alcohol", () => ({
  fetchDotRandomPools: mocks.fetchDotRandomPools,
  fetchDotRandomDraws: mocks.fetchDotRandomDraws,
  fetchDotRandomDraw: mocks.fetchDotRandomDraw,
  runDotRandomDraw: mocks.runDotRandomDraw,
  finalizeDotRandomDraw: mocks.finalizeDotRandomDraw,
  cancelDotRandomDraw: mocks.cancelDotRandomDraw,
  updateDotRandomDrawEntry: mocks.updateDotRandomDrawEntry,
  DOT_RANDOM_POOLS_KEY: "dot-random-pools",
  DOT_RANDOM_DRAWS_KEY: "dot-random-draws",
  DOT_RANDOM_DRAW_KEY: "dot-random-draw",
}));

vi.mock("../random-pool-dialog", () => ({
  RandomPoolDialog: ({ open }: { open: boolean }) =>
    open ? <div role="dialog" aria-label="Pool dialog" /> : null,
}));

vi.mock("sonner", () => ({ toast: mocks.toast }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("@number-flow/react", () => ({
  default: ({ value, className, ...rest }: { value: number; className?: string }) => (
    <span className={className} {...rest}>
      {value}
    </span>
  ),
}));

// 2026-08-15T00:00:00Z: Q3 of a quarterly year.
const NOW = Date.UTC(2026, 7, 15) / 1000;

const dotPool = {
  id: "pool_1",
  code: "DOT",
  name: "DOT drivers",
  description: "Every CDL holder on the road",
  status: "Active",
  period: "Quarterly",
  drugRatePercent: 50,
  alcoholRatePercent: 10,
  includedDriverTypes: [],
  isDefault: true,
  meetsDotMinimums: true,
  version: 1,
  createdAt: NOW,
  updatedAt: NOW,
};

const ownerPool = {
  ...dotPool,
  id: "pool_2",
  code: "OO",
  name: "Owner operators",
  description: null,
  includedDriverTypes: ["OwnerOperator"],
  isDefault: false,
  drugRatePercent: 25,
  meetsDotMinimums: false,
};

const retiredPool = { ...dotPool, id: "pool_3", code: "OLD", name: "Retired", status: "Inactive" };

function draw(over: Record<string, unknown> = {}) {
  return {
    id: "draw_1",
    poolId: "pool_1",
    periodKey: "2026-Q1",
    periodStart: Date.UTC(2026, 0, 1) / 1000,
    periodEnd: Date.UTC(2026, 3, 1) / 1000,
    status: "Final",
    poolSize: 40,
    drugTarget: 5,
    alcoholTarget: 1,
    drugSelected: 5,
    alcoholSelected: 1,
    seed: "seed",
    method: "hmac-sha256",
    notes: null,
    drawnAt: Date.UTC(2026, 0, 4) / 1000,
    finalizedAt: Date.UTC(2026, 0, 5) / 1000,
    version: 1,
    pool: { id: "pool_1", code: "DOT", name: "DOT drivers", period: "Quarterly" },
    ...over,
  };
}

const draws = [
  draw(),
  draw({
    id: "draw_2",
    periodKey: "2026-Q2",
    periodStart: Date.UTC(2026, 3, 1) / 1000,
    periodEnd: Date.UTC(2026, 6, 1) / 1000,
    status: "Cancelled",
    finalizedAt: null,
    drawnAt: Date.UTC(2026, 3, 2) / 1000,
  }),
  draw({
    id: "draw_3",
    periodKey: "2026-Q3",
    periodStart: Date.UTC(2026, 6, 1) / 1000,
    periodEnd: Date.UTC(2026, 9, 1) / 1000,
    status: "Draft",
    drugSelected: 3,
    finalizedAt: null,
    drawnAt: Date.UTC(2026, 6, 3) / 1000,
  }),
];

function renderConsole(pools: unknown[] = [dotPool, ownerPool, retiredPool], rounds = draws) {
  mocks.fetchDotRandomPools.mockResolvedValue(pools);
  mocks.fetchDotRandomDraws.mockResolvedValue(rounds);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <RandomTestingConsole />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.spyOn(Date, "now").mockReturnValue(NOW * 1000);
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.clearAllMocks();
});

describe("RandomTestingConsole", () => {
  // The loading state is the loaded page drawn in grey: the KPI strip, the
  // pools with their year of slots and the rounds table, at the sizes they
  // take once the programme lands, so the page does not reflow when it does.
  it("draws the page's own shape while the programme is still being read", () => {
    mocks.fetchDotRandomPools.mockReturnValue(new Promise(() => {}));
    mocks.fetchDotRandomDraws.mockReturnValue(new Promise(() => {}));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <MemoryRouter>
        <QueryClientProvider client={client}>
          <RandomTestingConsole />
        </QueryClientProvider>
      </MemoryRouter>,
    );

    const loading = screen.getByLabelText("Loading random testing");
    expect(loading).toHaveAttribute("aria-busy", "true");
    const hidden = { hidden: true } as const;
    const pools = within(loading).getByRole("list", { name: "Pools", ...hidden });
    expect(within(pools).getAllByRole("listitem", hidden).length).toBeGreaterThan(0);
    const rounds = within(loading).getByRole("table", { name: "Rounds", ...hidden });
    expect(within(rounds).getAllByRole("row", hidden).length).toBeGreaterThan(1);
  });

  it("heads the page with the pools, the hat, the year's rounds and the collections", async () => {
    renderConsole();

    expect(await screen.findByLabelText("Owed now", { selector: "span" })).toHaveTextContent("1");
    expect(screen.getByText("2 rounds missed this year")).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("2 rounds this year were never drawn");
    expect(screen.getByLabelText("Rounds this year", { selector: "span" })).toHaveTextContent("2");
    expect(screen.getByText("1 final · 1 draft")).toBeInTheDocument();
    expect(screen.getByLabelText("In the hat", { selector: "span" })).toHaveTextContent("40");
    expect(screen.getByText("2 active pools · 1 below the DOT minimum")).toBeInTheDocument();
    expect(screen.getByLabelText("Selected this year", { selector: "span" })).toHaveTextContent(
      "10",
    );
    expect(screen.getByRole("img", { name: "Drug: 8 of 10" })).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "Alcohol: 2 of 2" })).toBeInTheDocument();
  });

  // The year's rounds are the audit: a slot that was never drawn is the
  // finding, and it has to read as such rather than as a shorter list.
  it("shows each pool's year as slots, with the owed round on the draw button", async () => {
    renderConsole();

    const dot = await screen.findByRole("listitem", { name: "DOT drivers" });
    const slots = within(dot).getAllByRole("img");
    expect(slots.map((slot) => slot.getAttribute("aria-label"))).toEqual([
      "Q1: Final",
      "Q2: Never drawn",
      "Q3: Drawn, not final",
      "Q4: Not yet",
    ]);
    expect(
      within(dot).getByText("Drug 8 of 10 · Alcohol 2 of 2 · a round fell short"),
    ).toBeInTheDocument();
    expect(within(dot).getByRole("button", { name: "Run draw" })).toBeInTheDocument();

    const owner = screen.getByRole("listitem", { name: "Owner operators" });
    expect(within(owner).getByText("Below the DOT minimum")).toBeInTheDocument();
    expect(
      within(owner).getByText("Quarterly · 25% drug · 10% alcohol · OwnerOperator"),
    ).toBeInTheDocument();
    expect(within(owner).getByRole("button", { name: "Draw Q3" })).toBeInTheDocument();

    const retired = screen.getByRole("listitem", { name: "Retired" });
    expect(within(retired).getByText("Inactive")).toBeInTheDocument();
    expect(within(retired).queryByRole("button", { name: /draw/i })).not.toBeInTheDocument();
  });

  it("lists the rounds newest first and narrows them by status", async () => {
    const user = userEvent.setup();
    renderConsole();

    const table = await screen.findByRole("table", { name: "Rounds" });
    const periods = () =>
      within(table)
        .getAllByRole("row")
        .slice(1)
        .map((row) => within(row).getAllByRole("cell")[0].textContent?.slice(0, 7));
    expect(periods()).toEqual(["2026-Q3", "2026-Q2", "2026-Q1"]);
    expect(within(table).getByText("Voided")).toBeInTheDocument();
    expect(within(table).getByText("Short")).toBeInTheDocument();
    expect(within(table).getByRole("img", { name: "Drug: 3 of 5" })).toBeInTheDocument();

    await user.click(
      within(screen.getByRole("radiogroup", { name: "Round status" })).getByRole("radio", {
        name: "Final",
      }),
    );
    expect(periods()).toEqual(["2026-Q1"]);
    expect(screen.getByText("1 of 3")).toBeInTheDocument();
  });

  it("narrows the rounds to one pool from its card", async () => {
    const user = userEvent.setup();
    renderConsole(
      [dotPool, ownerPool],
      [
        ...draws,
        draw({
          id: "draw_9",
          poolId: "pool_2",
          periodKey: "2026-Q3",
          status: "Draft",
          pool: { id: "pool_2", code: "OO", name: "Owner operators", period: "Quarterly" },
          drawnAt: Date.UTC(2026, 7, 1) / 1000,
        }),
      ],
    );

    const owner = await screen.findByRole("listitem", { name: "Owner operators" });
    await user.click(within(owner).getByRole("button", { name: "Show rounds for OO" }));

    const table = screen.getByRole("table", { name: "Rounds" });
    expect(within(table).getAllByRole("row")).toHaveLength(2);
    expect(within(table).getByText("OO")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Clear pool OO" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Clear pool OO" }));
    expect(within(screen.getByRole("table", { name: "Rounds" })).getAllByRole("row")).toHaveLength(
      5,
    );
  });

  // Finalising is one way; a table row must not do it on a single click.
  it("asks before finalising a draft round, then finalises it", async () => {
    const user = userEvent.setup();
    mocks.finalizeDotRandomDraw.mockResolvedValue(draw({ id: "draw_3", status: "Final" }));
    renderConsole();

    await user.click(await screen.findByRole("button", { name: "Finalise 2026-Q3" }));
    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent("Finalise round 2026-Q3?");
    expect(dialog).toHaveTextContent("3 drivers for drug testing and 1 for alcohol");
    expect(mocks.finalizeDotRandomDraw).not.toHaveBeenCalled();

    await user.click(within(dialog).getByRole("button", { name: "Finalise" }));
    await waitFor(() => expect(mocks.finalizeDotRandomDraw).toHaveBeenCalledWith("draw_3"));
    expect(mocks.toast.success).toHaveBeenCalledWith("Round finalised", expect.anything());
  });

  it("asks before voiding a draft round, then voids it", async () => {
    const user = userEvent.setup();
    mocks.cancelDotRandomDraw.mockResolvedValue(draw({ id: "draw_3", status: "Cancelled" }));
    renderConsole();

    await user.click(await screen.findByRole("button", { name: "Void 2026-Q3" }));
    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent("Void round 2026-Q3?");

    await user.click(within(dialog).getByRole("button", { name: "Void the round" }));
    await waitFor(() =>
      expect(mocks.cancelDotRandomDraw).toHaveBeenCalledWith("draw_3", expect.any(String)),
    );
  });

  it("runs a pool's draw and opens the round it produced", async () => {
    const user = userEvent.setup();
    const produced = draw({
      id: "draw_new",
      poolId: "pool_2",
      periodKey: "2026-Q3",
      status: "Draft",
      drugSelected: 2,
      drugTarget: 2,
      alcoholSelected: 1,
      alcoholTarget: 1,
      poolSize: 12,
    });
    mocks.runDotRandomDraw.mockResolvedValue(produced);
    mocks.fetchDotRandomDraw.mockResolvedValue({
      ...produced,
      pool: { ...produced.pool, drugRatePercent: 25, alcoholRatePercent: 10 },
      entries: [
        {
          id: "e1",
          drawId: "draw_new",
          workerId: "w1",
          substance: "Drug",
          rank: 1,
          status: "Selected",
          notifiedAt: null,
          completedAt: null,
          testId: null,
          excuseReason: null,
          version: 1,
          worker: { id: "w1", firstName: "Ada", lastName: "Byrne" },
        },
        {
          id: "e2",
          drawId: "draw_new",
          workerId: "w2",
          substance: "Drug",
          rank: 2,
          status: "Completed",
          notifiedAt: null,
          completedAt: NOW,
          testId: "t1",
          excuseReason: null,
          version: 1,
          worker: { id: "w2", firstName: "Cal", lastName: "Diaz" },
        },
        {
          id: "e3",
          drawId: "draw_new",
          workerId: "w3",
          substance: "Alcohol",
          rank: 1,
          status: "Notified",
          notifiedAt: NOW,
          completedAt: null,
          testId: null,
          excuseReason: null,
          version: 1,
          worker: { id: "w3", firstName: "Eve", lastName: "Fox" },
        },
      ],
    });
    renderConsole();

    const owner = await screen.findByRole("listitem", { name: "Owner operators" });
    await user.click(within(owner).getByRole("button", { name: "Draw Q3" }));

    await waitFor(() => expect(mocks.runDotRandomDraw).toHaveBeenCalledWith({ poolId: "pool_2" }));
    const sheet = await screen.findByRole("dialog", { name: "Round 2026-Q3" });
    const collections = within(sheet).getByRole("region", { name: "Collections" });
    expect(collections).toHaveTextContent("1/2 collected");
    expect(collections).toHaveTextContent("1 to collect");
    expect(collections).toHaveTextContent("0/1 collected");
    expect(collections).toHaveTextContent("1 to collect · 1 notified");
  });

  it("offers to make the first pool when there is none", async () => {
    const user = userEvent.setup();
    renderConsole([], []);

    expect(await screen.findByText("No pool is configured")).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
    await user.click(screen.getAllByRole("button", { name: "New pool" })[0]);
    expect(screen.getByRole("dialog", { name: "Pool dialog" })).toBeInTheDocument();
  });
});
