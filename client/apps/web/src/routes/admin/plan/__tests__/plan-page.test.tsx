import type { ManagedPlan } from "@/lib/graphql/managed-plan";
import { PageLayoutStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PlanPage } from "../page";

const mocks = vi.hoisted(() => ({ fetchManagedPlan: vi.fn() }));

vi.mock("@/lib/graphql/managed-plan", () => ({ fetchManagedPlan: mocks.fetchManagedPlan }));
vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));

// The shape of the managedPlan query (services/tms/internal/api/graphql/schema/managed_plan.graphqls).
function plan(overrides: Partial<ManagedPlan> = {}): ManagedPlan {
  return {
    managed: true,
    base: [
      { key: "tms.core", name: "Freight" },
      { key: "tms.dispatch", name: "Dispatch" },
    ],
    included: [
      {
        key: "ACC",
        name: "Accounting & Cash",
        features: [
          { key: "tms.accounting", name: "Accounting" },
          { key: "tms.factoring", name: "Factoring" },
        ],
      },
    ],
    available: [
      {
        key: "DSP",
        name: "Dispatch Intelligence",
        features: [{ key: "tms.dispatch_intelligence", name: "Dispatch intelligence" }],
      },
    ],
    changes: [
      {
        changedAt: 1_794_650_400,
        starting: false,
        left: [{ key: "DSP", name: "Dispatch Intelligence" }],
        joined: [{ key: "ACC", name: "Accounting & Cash" }],
      },
      {
        changedAt: 1_794_000_000,
        starting: true,
        left: [],
        joined: [{ key: "DSP", name: "Dispatch Intelligence" }],
      },
    ],
    ...overrides,
  } as ManagedPlan;
}

afterEach(() => {
  cleanup();
  mocks.fetchManagedPlan.mockReset();
});

describe("Your plan", () => {
  it("lists what the plan includes, the add-ons on it and off it, and its changes", async () => {
    mocks.fetchManagedPlan.mockResolvedValue(plan());

    renderAccountingPage(<PlanPage />);

    expect(await screen.findByText("Freight and Dispatch")).toBeInTheDocument();
    expect(screen.getByText("Adds Accounting and Factoring.")).toBeInTheDocument();
    expect(screen.getByText("On your plan")).toBeInTheDocument();
    expect(
      screen.getByText("Would add Dispatch intelligence. Add it from your Trenova account."),
    ).toBeInTheDocument();
    expect(screen.getByText("Joined your plan: Accounting & Cash.")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Left your plan: Dispatch Intelligence. Everything you entered with it is kept and comes back if you add it again.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Your plan started with Dispatch Intelligence.")).toBeInTheDocument();
    expect(screen.queryByText("DSP")).not.toBeInTheDocument();
    expect(screen.queryByText("tms.accounting")).not.toBeInTheDocument();
  });

  it("says so when the plan has no add-ons and has never changed", async () => {
    mocks.fetchManagedPlan.mockResolvedValue(plan({ included: [], changes: [] }));

    renderAccountingPage(<PlanPage />);

    expect(
      await screen.findByText("Your plan has no add-ons. It includes everything listed above."),
    ).toBeInTheDocument();
    expect(screen.getByText("Your plan has not changed yet.")).toBeInTheDocument();
  });

  it("says every add-on is on the plan when none is left to add", async () => {
    mocks.fetchManagedPlan.mockResolvedValue(plan({ available: [] }));

    renderAccountingPage(<PlanPage />);

    expect(await screen.findByText("Every add-on is on your plan.")).toBeInTheDocument();
  });

  it("shows a failed read as a failure, not as an empty plan", async () => {
    mocks.fetchManagedPlan.mockRejectedValue(new TypeError("Failed to fetch"));

    renderAccountingPage(<PlanPage />);

    expect(await screen.findByText("Your plan could not be loaded")).toBeInTheDocument();
    expect(screen.queryByText("Your plan has not changed yet.")).not.toBeInTheDocument();
  });
});
