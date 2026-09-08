import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import BenefitsConsole from "../benefits-console";

const mocks = vi.hoisted(() => ({
  fetchBenefitPlans: vi.fn(),
  fetchBenefitCosts: vi.fn(),
  fetchBenefitEnrollments: vi.fn(),
  createBenefitPlan: vi.fn(),
  updateBenefitPlan: vi.fn(),
}));

vi.mock("@/lib/graphql/benefits", () => ({
  ...mocks,
  BENEFIT_PLANS_KEY: "benefit-plans",
  BENEFIT_COSTS_KEY: "benefit-costs",
  BENEFIT_ENROLLMENT_LIST_KEY: "benefit-enrollment-list",
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("../benefit-plan-dialog", () => ({ BenefitPlanDialog: () => null }));

vi.mock("@number-flow/react", () => ({
  default: ({ value, className, ...rest }: { value: number; className?: string }) => (
    <span className={className} {...rest}>
      {value}
    </span>
  ),
}));

const DAY = 86_400;
const NOW = Date.UTC(2026, 8, 7, 15, 30) / 1000;

const ADA = {
  id: "wrk_ada",
  firstName: "Ada",
  lastName: "Byrne",
  profilePicUrl: "",
  fleetCode: { id: "fc_1", code: "SOUTH", color: "#2563eb" },
};
const BEN = {
  id: "wrk_ben",
  firstName: "Ben",
  lastName: "Cole",
  profilePicUrl: "",
  fleetCode: null,
};

function plan(over: Record<string, unknown> = {}) {
  return {
    id: "bpl_med",
    status: "Active",
    code: "MED-26",
    name: "Medical PPO",
    description: null,
    planType: "Medical",
    carrier: "Blue Shield",
    policyNumber: "P-100",
    payCodeId: "pc_1",
    planYear: 2026,
    employeeCostMinor: 12_000,
    employerCostMinor: 48_000,
    currencyCode: "USD",
    waitingPeriodDays: 30,
    version: 1,
    payCode: { id: "pc_1", code: "BEN-MED", description: "Medical" },
    ...over,
  };
}

function cost(over: Record<string, unknown> = {}) {
  return {
    planId: "bpl_med",
    planName: "Medical PPO",
    planType: "Medical",
    enrolled: 10,
    waived: 2,
    employeeCostMinor: 120_000,
    employerCostMinor: 480_000,
    ...over,
  };
}

function enrollment(over: Record<string, unknown> = {}) {
  return {
    id: "ben_1",
    workerId: ADA.id,
    benefitPlanId: "bpl_med",
    status: "Active",
    coverageTier: "Employee",
    effectiveFrom: NOW - 100 * DAY,
    effectiveTo: null,
    employeeCostMinor: 12_000,
    employerCostMinor: 48_000,
    waivedReason: null,
    notes: null,
    version: 1,
    benefitPlan: {
      id: "bpl_med",
      code: "MED-26",
      name: "Medical PPO",
      planType: "Medical",
      planYear: 2026,
      currencyCode: "USD",
    },
    worker: ADA,
    ...over,
  };
}

type Fixtures = {
  plans?: unknown[];
  costs?: Record<string, unknown[]>;
  open?: unknown[];
  declined?: unknown[];
  byPlan?: Record<string, unknown[]>;
};

function renderConsole(fixtures: Fixtures = {}) {
  mocks.fetchBenefitPlans.mockResolvedValue(fixtures.plans ?? [plan()]);
  mocks.fetchBenefitCosts.mockImplementation(
    async (planYear?: number) => fixtures.costs?.[String(planYear)] ?? [],
  );
  mocks.fetchBenefitEnrollments.mockImplementation(
    async (args: { planId?: string | null; statuses?: string[]; openOnly?: boolean }) => {
      if (args.planId) return fixtures.byPlan?.[args.planId] ?? [];
      if (args.statuses?.includes("Waived")) return fixtures.declined ?? [];
      return fixtures.open ?? [];
    },
  );
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <BenefitsConsole />
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

describe("BenefitsConsole", () => {
  // The loading state is the loaded page drawn in grey: the KPI strip, the
  // plan groups and the aside, at the sizes they take once the plans land, so
  // the page does not reflow when they do.
  it("draws the page's own shape while the plans are still being read", () => {
    mocks.fetchBenefitPlans.mockReturnValue(new Promise(() => {}));
    mocks.fetchBenefitCosts.mockReturnValue(new Promise(() => {}));
    mocks.fetchBenefitEnrollments.mockReturnValue(new Promise(() => {}));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <MemoryRouter>
        <QueryClientProvider client={client}>
          <BenefitsConsole />
        </QueryClientProvider>
      </MemoryRouter>,
    );

    const loading = screen.getByLabelText("Loading benefits");
    expect(loading).toHaveAttribute("aria-busy", "true");
    const hidden = { hidden: true } as const;
    const medical = within(loading).getByRole("region", { name: "Medical", ...hidden });
    expect(within(medical).getAllByRole("listitem", hidden).length).toBeGreaterThan(0);
    for (const name of ["Starting soon", "Ending soon", "Recently declined"]) {
      expect(within(loading).getByRole("region", { name, ...hidden })).toBeInTheDocument();
    }
  });

  it("heads the page with who is covered, what is on offer, and who pays", async () => {
    renderConsole({
      plans: [
        plan(),
        plan({
          id: "bpl_den",
          code: "DEN-26",
          name: "Dental",
          planType: "Dental",
          employeeCostMinor: 2_000,
          employerCostMinor: 3_000,
        }),
        plan({ id: "bpl_old", code: "MED-25", planYear: 2025 }),
      ],
      costs: {
        "2026": [
          cost(),
          cost({
            planId: "bpl_den",
            planName: "Dental",
            planType: "Dental",
            enrolled: 4,
            waived: 0,
            employeeCostMinor: 8_000,
            employerCostMinor: 12_000,
          }),
        ],
      },
      open: [
        enrollment(),
        enrollment({ id: "ben_2", workerId: BEN.id, worker: BEN, effectiveFrom: NOW + 5 * DAY }),
      ],
      declined: [enrollment({ id: "ben_3", status: "Waived", waivedReason: "Covered by spouse" })],
    });

    expect(await screen.findByLabelText("People covered", { selector: "span" })).toHaveTextContent(
      "14",
    );
    expect(await screen.findByText("2 declined · 1 starting soon")).toBeInTheDocument();
    expect(screen.getByLabelText("Plans on offer", { selector: "span" })).toHaveTextContent("2");
    expect(screen.getByText("For 2026")).toBeInTheDocument();
    expect(screen.getByLabelText("Employer puts in")).toHaveTextContent("$4,920.00");
    expect(screen.getByLabelText("Off settlements")).toHaveTextContent("$1,280.00");
    expect(screen.getByRole("img", { name: /Who pays/ })).toHaveAccessibleName(
      "Who pays, per settlement period: Employer $4,920.00, Employees $1,280.00",
    );
  });

  // Next year's plans are often priced before this year is over; the page
  // opens on the year being administered and can be switched.
  it("opens on this year's plans and switches years", async () => {
    const user = userEvent.setup();
    renderConsole({
      plans: [
        plan(),
        plan({ id: "bpl_next", code: "MED-27", name: "Medical PPO 2027", planYear: 2027 }),
      ],
      costs: {
        "2026": [cost()],
        "2027": [
          cost({
            planId: "bpl_next",
            planName: "Medical PPO 2027",
            enrolled: 1,
            waived: 0,
            employeeCostMinor: 13_000,
            employerCostMinor: 50_000,
          }),
        ],
      },
    });

    expect(await screen.findByRole("article", { name: "Medical PPO" })).toBeInTheDocument();
    expect(screen.queryByRole("article", { name: "Medical PPO 2027" })).not.toBeInTheDocument();
    expect(mocks.fetchBenefitCosts).toHaveBeenCalledWith(2026, expect.anything());

    await user.click(
      within(screen.getByRole("radiogroup", { name: "Plan year" })).getByRole("radio", {
        name: "2027",
      }),
    );

    expect(await screen.findByRole("article", { name: "Medical PPO 2027" })).toBeInTheDocument();
    expect(screen.queryByRole("article", { name: "Medical PPO" })).not.toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByLabelText("People covered", { selector: "span" })).toHaveTextContent("1"),
    );
  });

  it("groups plans by kind with the kind's totals and each plan's share", async () => {
    renderConsole({
      plans: [
        plan(),
        plan({ id: "bpl_hmo", code: "HMO-26", name: "Medical HMO" }),
        plan({ id: "bpl_den", code: "DEN-26", name: "Dental", planType: "Dental" }),
      ],
      costs: {
        "2026": [
          cost({ enrolled: 6, waived: 1 }),
          cost({ planId: "bpl_hmo", planName: "Medical HMO", enrolled: 3, waived: 0 }),
          cost({
            planId: "bpl_den",
            planName: "Dental",
            planType: "Dental",
            enrolled: 1,
            waived: 0,
            employerCostMinor: 3_000,
          }),
        ],
      },
    });

    const medical = await screen.findByRole("region", { name: "Medical" });
    expect(
      await within(medical).findByText("9 covered · 1 declined · $9,600.00 employer"),
    ).toBeInTheDocument();
    const rows = within(medical).getAllByRole("article");
    expect(rows.map((row) => row.getAttribute("aria-label"))).toEqual([
      "Medical PPO",
      "Medical HMO",
    ]);
    expect(
      within(rows[0]).getByRole("img", { name: "Medical PPO: 6 of 10 covered people" }),
    ).toBeInTheDocument();
    expect(
      within(rows[0]).getByText("Blue Shield · Policy P-100 · Pay code BEN-MED · 30-day wait"),
    ).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Dental" })).toBeInTheDocument();
  });

  it("names active plans nobody has joined", async () => {
    renderConsole({
      plans: [plan(), plan({ id: "bpl_vis", code: "VIS-26", name: "Vision", planType: "Vision" })],
      costs: { "2026": [cost()] },
    });

    expect(await screen.findByText(/1 active plan has nobody on it/)).toBeInTheDocument();
  });

  // Declines and ended cover stay on the plan's list: "who declined this" is
  // asked at audit as often as "who is on it".
  it("opens who is on a plan, standing first, and finds a name", async () => {
    const user = userEvent.setup();
    renderConsole({
      plans: [plan()],
      costs: { "2026": [cost()] },
      byPlan: {
        bpl_med: [
          enrollment(),
          enrollment({
            id: "ben_2",
            workerId: BEN.id,
            worker: BEN,
            status: "Waived",
            waivedReason: "Covered by spouse",
            employeeCostMinor: 0,
          }),
          enrollment({
            id: "ben_3",
            workerId: "wrk_cal",
            worker: { ...BEN, id: "wrk_cal", firstName: "Cal", lastName: "Diaz" },
            effectiveTo: NOW + 4 * DAY,
          }),
        ],
      },
    });

    await user.click(await screen.findByRole("button", { name: "Who is on it" }));

    const list = await screen.findByRole("list", { name: "People on the plan" });
    const items = within(list).getAllByRole("listitem");
    expect(items.map((item) => within(item).getByText(/Byrne|Cole|Diaz/).textContent)).toEqual([
      "Cal Diaz",
      "Ada Byrne",
      "Ben Cole",
    ]);
    expect(within(items[0]).getByText("Ending soon")).toBeInTheDocument();
    expect(within(items[1]).getByText("Covered")).toBeInTheDocument();
    expect(within(items[1]).getByText("$120.00")).toBeInTheDocument();
    expect(within(items[2]).getByText("Declined")).toBeInTheDocument();
    expect(within(items[2]).getByText(/Covered by spouse/)).toBeInTheDocument();
    expect(screen.getByText("2 covered · 3 on record")).toBeInTheDocument();

    await user.type(screen.getByLabelText("Find on this plan"), "south");
    await waitFor(() => expect(within(list).getAllByRole("listitem")).toHaveLength(1));
    expect(within(list).getByText("Ada Byrne")).toBeInTheDocument();
  });

  it("shows what is starting, ending and declined beside the plans", async () => {
    renderConsole({
      plans: [plan()],
      costs: { "2026": [cost()] },
      open: [
        enrollment(),
        enrollment({ id: "ben_2", workerId: BEN.id, worker: BEN, effectiveFrom: NOW + 5 * DAY }),
        enrollment({
          id: "ben_3",
          workerId: "wrk_cal",
          worker: { ...BEN, id: "wrk_cal", firstName: "Cal", lastName: "Diaz" },
          effectiveTo: NOW + 4 * DAY,
        }),
      ],
      declined: [
        enrollment({
          id: "ben_4",
          workerId: "wrk_dee",
          worker: { ...BEN, id: "wrk_dee", firstName: "Dee", lastName: "Ellis" },
          status: "Waived",
          waivedReason: "Covered by spouse",
        }),
      ],
    });

    const starting = await screen.findByRole("region", { name: "Starting soon" });
    expect(await within(starting).findByText("Ben Cole")).toBeInTheDocument();
    expect(within(starting).queryByText("Ada Byrne")).not.toBeInTheDocument();
    const ending = screen.getByRole("region", { name: "Ending soon" });
    expect(within(ending).getByText("Cal Diaz")).toBeInTheDocument();
    const declined = screen.getByRole("region", { name: "Recently declined" });
    expect(await within(declined).findByText("Dee Ellis")).toBeInTheDocument();
    expect(within(declined).getByText(/Covered by spouse/)).toBeInTheDocument();
  });

  it("says how to start when there are no plans", async () => {
    renderConsole({ plans: [] });

    expect(await screen.findByText("No plans yet")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Add a plan/ })).toBeInTheDocument();
  });
});
