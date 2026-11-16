import type { ManagedPlan } from "@/lib/graphql/managed-plan";
import { PageLayoutStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import { cleanup, fireEvent, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PlanPage } from "../page";

const mocks = vi.hoisted(() => ({ fetchManagedPlan: vi.fn() }));

vi.mock("@/lib/graphql/managed-plan", () => ({ fetchManagedPlan: mocks.fetchManagedPlan }));
vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));

const MANAGE_URL =
  "https://console.trenova.cloud/instances/0193a1b2-0000-7000-8000-000000000001/plan";
const NOV_14 = 1_794_657_600;
const NOV_15 = 1_794_744_000;
const DEC_14 = 1_797_249_600;
const GIB = 1024 ** 3;

type Meter = NonNullable<ManagedPlan["usage"]>["meters"][number];

const SHIPMENTS: Meter = {
  key: "shipments.total",
  name: "Shipments",
  unit: "COUNT",
  used: 312,
  included: 1_500,
  hardCap: false,
  stopped: false,
};

const STORAGE: Meter = {
  key: "documents.storage_bytes",
  name: "Document storage",
  unit: "BYTES",
  used: Math.round(18.4 * GIB),
  included: 100 * GIB,
  hardCap: false,
  stopped: false,
};

const ORGANIZATIONS: Meter = {
  key: "organizations.total",
  name: "Organizations",
  unit: "COUNT",
  used: 1,
  included: 1,
  hardCap: true,
  stopped: false,
};

// The shape of the managedPlan query (services/tms/internal/api/graphql/schema/managed_plan.graphqls).
function plan(overrides: Partial<ManagedPlan> = {}): ManagedPlan {
  return {
    managed: true,
    manageUrl: MANAGE_URL,
    endsAt: null,
    finalBackupDays: 30,
    usage: {
      periodStart: NOV_14,
      periodEnd: DEC_14,
      meters: [SHIPMENTS, STORAGE, ORGANIZATIONS],
    },
    base: [
      { key: "tms.core", name: "Freight" },
      { key: "tms.dispatch", name: "Dispatch" },
    ],
    included: [
      {
        key: "ACC",
        name: "Accounting & Cash",
        endsAt: DEC_14,
        features: [
          { key: "tms.accounting", name: "Accounting" },
          { key: "tms.factoring", name: "Factoring" },
        ],
      },
      {
        key: "VIS",
        name: "Visibility & Tracking",
        endsAt: null,
        features: [{ key: "tms.customer_tracking", name: "Customer tracking" }],
      },
    ],
    available: [
      {
        key: "DSP",
        name: "Dispatch Intelligence",
        endsAt: null,
        features: [{ key: "tms.dispatch_intelligence", name: "Dispatch intelligence" }],
      },
    ],
    changes: [
      {
        changedAt: NOV_15,
        starting: false,
        left: [],
        joined: [],
        ending: [{ endsAt: DEC_14, addOn: { key: "ACC", name: "Accounting & Cash" } }],
        kept: [],
      },
      {
        changedAt: NOV_15,
        starting: false,
        left: [{ key: "DSP", name: "Dispatch Intelligence" }],
        joined: [{ key: "VIS", name: "Visibility & Tracking" }],
        ending: [],
        kept: [{ key: "AI", name: "Trenova AI" }],
      },
      {
        changedAt: NOV_14,
        starting: true,
        left: [],
        joined: [{ key: "DSP", name: "Dispatch Intelligence" }],
        ending: [],
        kept: [],
      },
    ],
    ...overrides,
  } as ManagedPlan;
}

function panel(title: string): HTMLElement {
  return screen.getByRole("region", { name: title });
}

afterEach(() => {
  cleanup();
  mocks.fetchManagedPlan.mockReset();
});

describe("Your plan", () => {
  it("links Manage plan to the plan's page in the Trenova account, in a new tab", async () => {
    mocks.fetchManagedPlan.mockResolvedValue(plan());

    renderAccountingPage(<PlanPage />);

    const manage = await screen.findByRole("link", { name: "Manage plan" });
    expect(manage).toHaveAttribute("href", MANAGE_URL);
    expect(manage).toHaveAttribute("target", "_blank");
    expect(manage).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("says when a plan set to end ends, with Keep plan opening the Trenova account", async () => {
    mocks.fetchManagedPlan.mockResolvedValue(plan({ endsAt: DEC_14, finalBackupDays: 7 }));

    renderAccountingPage(<PlanPage />);

    const notice = await screen.findByRole("alert");
    expect(within(notice).getByText("Your plan ends on Dec 14, 2026.")).toBeInTheDocument();
    expect(
      within(notice).getByText(
        "Trenova stays available until then, with everything your plan includes. After that, this workspace is taken down and a final backup is kept for 7 days.",
      ),
    ).toBeInTheDocument();
    const keep = within(notice).getByRole("link", { name: "Keep plan" });
    expect(keep).toHaveAttribute("href", MANAGE_URL);
    expect(keep).toHaveAttribute("target", "_blank");
    expect(keep).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("says when the plan ends without a button to whoever has no account link", async () => {
    mocks.fetchManagedPlan.mockResolvedValue(plan({ endsAt: DEC_14, manageUrl: null }));

    renderAccountingPage(<PlanPage />);

    const notice = await screen.findByRole("alert");
    expect(within(notice).getByText("Your plan ends on Dec 14, 2026.")).toBeInTheDocument();
    expect(within(notice).queryByRole("link")).not.toBeInTheDocument();
  });

  it("shows no plan-end notice while the plan is not ending", async () => {
    mocks.fetchManagedPlan.mockResolvedValue(plan());

    renderAccountingPage(<PlanPage />);

    await screen.findByText("Shipments");
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Keep plan" })).not.toBeInTheDocument();
  });

  it("shows this period's usage against what the plan includes", async () => {
    mocks.fetchManagedPlan.mockResolvedValue(plan());

    renderAccountingPage(<PlanPage />);

    await screen.findByText("Shipments");
    const usage = panel("Usage");
    expect(within(usage).getByText("Nov 14 to Dec 14")).toBeInTheDocument();
    expect(within(usage).getByText("312").parentElement).toHaveTextContent("312 of 1,500");
    expect(within(usage).getByText("Document storage")).toBeInTheDocument();
    expect(within(usage).getByText("18.4 GB").parentElement).toHaveTextContent("18.4 GB of 100 GB");
    expect(within(usage).getByText("Your plan includes one organization.")).toBeInTheDocument();
    expect(within(usage).getAllByRole("progressbar")).toHaveLength(3);
    expect(within(usage).queryByText("shipments.total")).not.toBeInTheDocument();
  });

  it("marks an add-on on the plan as on, or as ending on its day", async () => {
    mocks.fetchManagedPlan.mockResolvedValue(plan());

    renderAccountingPage(<PlanPage />);

    await screen.findByText("Shipments");
    const on = panel("On your plan");
    expect(within(on).getByText("Ends Dec 14")).toBeInTheDocument();
    expect(within(on).getByText("On")).toBeInTheDocument();
    expect(within(on).getByText("Accounting and Factoring")).toBeInTheDocument();
    expect(within(on).queryByRole("link")).not.toBeInTheDocument();
  });

  it("opens an add-on's full feature list from its name", async () => {
    mocks.fetchManagedPlan.mockResolvedValue(plan());

    renderAccountingPage(<PlanPage />);

    const name = await screen.findByRole("button", { name: "Accounting & Cash" });
    expect(name).toHaveAttribute("aria-expanded", "false");
    fireEvent.click(name);
    expect(name).toHaveAttribute("aria-expanded", "true");
    fireEvent.click(name);
    expect(name).toHaveAttribute("aria-expanded", "false");
  });

  it("links each add-on that can be added to the account with it switched on", async () => {
    mocks.fetchManagedPlan.mockResolvedValue(plan());

    renderAccountingPage(<PlanPage />);

    await screen.findByText("Shipments");
    const add = within(panel("You can add")).getByRole("link", {
      name: "Add Dispatch Intelligence from your Trenova account",
    });
    expect(add).toHaveAttribute("href", `${MANAGE_URL}?add=DSP`);
    expect(add).toHaveAttribute("target", "_blank");
    expect(add).toHaveAttribute("rel", "noopener noreferrer");
    expect(screen.queryByText("DSP")).not.toBeInTheDocument();
  });

  it("lists what every plan includes and the changes, including endings and add-ons that stay", async () => {
    mocks.fetchManagedPlan.mockResolvedValue(plan());

    renderAccountingPage(<PlanPage />);

    await screen.findByText("Shipments");
    expect(
      within(panel("Included in every plan")).getByText("Freight and Dispatch"),
    ).toBeInTheDocument();
    const changes = panel("Recent changes");
    expect(
      within(changes).getByText(
        "Accounting & Cash was set to end on Dec 14. Everything entered with it is kept and comes back if you add it again.",
      ),
    ).toBeInTheDocument();
    expect(
      within(changes).getByText("Visibility & Tracking joined your plan."),
    ).toBeInTheDocument();
    expect(
      within(changes).getByText(
        "Dispatch Intelligence left your plan. Everything entered with it is kept and comes back if you add it again.",
      ),
    ).toBeInTheDocument();
    expect(within(changes).getByText("Trenova AI stays on your plan.")).toBeInTheDocument();
    expect(
      within(changes).getByText("Your plan started with Dispatch Intelligence."),
    ).toBeInTheDocument();
    expect(within(changes).getAllByText("Nov 15, 2026")).toHaveLength(2);
  });

  it("leaves out Manage plan, the Add links and usage until the control plane sends them", async () => {
    mocks.fetchManagedPlan.mockResolvedValue(plan({ manageUrl: null, usage: null }));

    renderAccountingPage(<PlanPage />);

    expect(await screen.findByText("Dispatch Intelligence")).toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Usage" })).not.toBeInTheDocument();
  });

  it("draws an allowance that is nearly used in amber, and a reached hard cap in neutral", async () => {
    mocks.fetchManagedPlan.mockResolvedValue(
      plan({
        usage: {
          periodStart: NOV_14,
          periodEnd: DEC_14,
          meters: [{ ...SHIPMENTS, used: 1_450 }, ORGANIZATIONS],
        },
      }),
    );

    renderAccountingPage(<PlanPage />);

    await screen.findByText("Shipments");
    expect(screen.getByRole("progressbar", { name: "Shipments" }).firstElementChild).toHaveClass(
      "bg-warning",
    );
    expect(
      screen.getByRole("progressbar", { name: "Organizations" }).firstElementChild,
    ).not.toHaveClass("bg-warning");
  });

  it("says so when the plan has no add-ons and has never changed", async () => {
    mocks.fetchManagedPlan.mockResolvedValue(plan({ included: [], changes: [] }));

    renderAccountingPage(<PlanPage />);

    expect(await screen.findByText("Your plan has no add-ons.")).toBeInTheDocument();
    expect(screen.getByText("Your plan has not changed yet.")).toBeInTheDocument();
  });

  it("says every add-on is on the plan when none is left to add", async () => {
    mocks.fetchManagedPlan.mockResolvedValue(plan({ available: [] }));

    renderAccountingPage(<PlanPage />);

    expect(await screen.findByText("Every add-on is on your plan.")).toBeInTheDocument();
  });

  it("says a self-hosted install has every feature, with no link to an account", async () => {
    mocks.fetchManagedPlan.mockResolvedValue({
      managed: false,
      manageUrl: null,
      endsAt: null,
      finalBackupDays: 30,
      usage: null,
      base: [],
      included: [],
      available: [],
      changes: [],
    } satisfies ManagedPlan);

    renderAccountingPage(<PlanPage />);

    expect(
      await screen.findByText(
        "This installation is not on a Managed Trenova plan, so every feature is on.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Manage plan" })).not.toBeInTheDocument();
  });

  it("shows a failed read as a failure, not as an empty plan", async () => {
    mocks.fetchManagedPlan.mockRejectedValue(new TypeError("Failed to fetch"));

    renderAccountingPage(<PlanPage />);

    expect(await screen.findByText("Your plan could not be loaded")).toBeInTheDocument();
    expect(screen.queryByText("Your plan has not changed yet.")).not.toBeInTheDocument();
  });
});
