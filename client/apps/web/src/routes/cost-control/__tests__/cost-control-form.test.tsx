import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import type { CostingControl } from "@/lib/graphql/cost-control";
import { clearPermissions } from "@/test/permissions";
import { seedPlan, unavailableFeatures } from "@/test/plan";
import CostControlForm from "../_components/cost-control-form";
import { savedCostControl } from "./cost-control-fixture";

const served = vi.hoisted(() => ({
  control: null as unknown,
  saves: [] as unknown[],
  requests: [] as unknown[],
}));

vi.mock("@/lib/graphql/cost-control", () => ({
  getCostingControlGraphQL: async () => served.control,
  saveCostingControlGraphQL: async (input: unknown) => {
    served.requests.push(input);
    return { id: "cstc_01", version: 3 };
  },
}));

vi.mock("@/lib/queries/cost-control", () => ({
  costControl: {
    get: Object.assign(
      () => ({ queryKey: ["cost-control"], queryFn: async () => served.control }),
      { _def: ["cost-control"] },
    ),
    resolvedProfile: Object.assign(() => ({}), { _def: ["cost-control", "resolved"] }),
  },
}));

vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: (options: { mutationFn: (values: unknown) => Promise<unknown> }) => ({
    mutateAsync: async (values: unknown) => {
      served.saves.push(values);
      return options.mutationFn(values);
    },
  }),
}));

const GRANTS = { [Resource.ProtectedMargins]: Operation.Read };

function withGLActuals(change: Partial<CostingControl["categories"][number]> = {}): CostingControl {
  return {
    ...savedCostControl,
    glActualsEnabled: true,
    categories: savedCostControl.categories.map((category, index) =>
      index === 0 ? { ...category, ...change } : category,
    ),
  };
}

function renderForm(search = "") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <TestRouter search={search}>
      <LazyMotion features={domMax}>
        <QueryClientProvider client={queryClient}>
          <CostControlForm />
        </QueryClientProvider>
      </LazyMotion>
    </TestRouter>,
  );
}

/** Each active category's rate choice, on its benchmark. */
function rateButtons() {
  return screen.getAllByRole("button", { name: "Industry benchmark" });
}

beforeAll(() => {
  MotionGlobalConfig.skipAnimations = true;
});
afterAll(() => {
  MotionGlobalConfig.skipAnimations = false;
});

afterEach(() => {
  served.control = savedCostControl;
  served.saves = [];
  served.requests = [];
  cleanup();
  clearPermissions();
});

describe("cost control tabs", () => {
  it("offers the cost basis, the rates and GL actuals, in that order", async () => {
    served.control = savedCostControl;
    seedPlan(GRANTS, []);

    renderForm();
    await screen.findByRole("heading", { name: "Fuel" });

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "Cost basis",
      "Cost rates",
      "GL actuals",
    ]);
  });

  it("leaves GL actuals out, and offers no ledger rate, when the plan has no accounting", async () => {
    served.control = withGLActuals();
    seedPlan(GRANTS, unavailableFeatures("tms.accounting"));
    const user = userEvent.setup();

    renderForm("?tab=rates");
    await screen.findByText("Variable costs");

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "Cost basis",
      "Cost rates",
    ]);
    await user.click(rateButtons()[0]!);
    expect(screen.getByRole("option", { name: /GL actuals/ })).toHaveAttribute(
      "aria-disabled",
      "true",
    );
  });
});

describe("cost basis", () => {
  it("asks for a fuel index only while live fuel pricing is on", async () => {
    served.control = savedCostControl;
    seedPlan(GRANTS, []);
    const user = userEvent.setup();

    renderForm();
    await screen.findByRole("heading", { name: "Fuel" });
    expect(screen.queryByText("Fuel index")).toBeNull();

    await user.click(screen.getByRole("switch", { name: /Use live fuel price/ }));
    expect(await screen.findByText("Fuel index")).toBeInTheDocument();
  });

  it("holds the target margin back from a role without margins", async () => {
    served.control = savedCostControl;
    seedPlan({}, []);

    renderForm();

    expect(await screen.findByText("Your role does not include margins.")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Hidden")).toBeDisabled();
  });
});

describe("cost rates", () => {
  it("draws the variable and fixed categories apart, in the server's order", async () => {
    served.control = savedCostControl;
    seedPlan(GRANTS, []);

    renderForm("?tab=rates");
    const variable = (await screen.findByText("Variable costs")).closest("section")!;
    const fixed = screen.getByText("Fixed costs").closest("section")!;

    expect(within(variable).getAllByRole("switch")).toHaveLength(6);
    expect(within(fixed).getAllByRole("switch")).toHaveLength(4);
    expect(variable.textContent).toMatch(
      /Driver Wages.*Driver Benefits.*Fuel.*Repair & Maintenance.*Tires.*Tolls/s,
    );
    expect(fixed.textContent).toMatch(
      /Truck & Trailer Payments.*Insurance Premiums.*Permits & Licenses.*Overhead & Admin/s,
    );
  });

  it("starts your own rate from the category's benchmark", async () => {
    served.control = savedCostControl;
    seedPlan(GRANTS, []);
    const user = userEvent.setup();

    renderForm("?tab=rates");
    await screen.findByText("Variable costs");
    await user.click(rateButtons()[0]!);
    await user.click(screen.getByRole("option", { name: /Your own rate/ }));

    expect(await screen.findByText("Your rate")).toBeInTheDocument();
    expect(screen.getByDisplayValue(/0\.82/)).toBeInTheDocument();
  });

  it("hides an inactive category's rate", async () => {
    served.control = savedCostControl;
    seedPlan(GRANTS, []);
    const user = userEvent.setup();

    renderForm("?tab=rates");
    await screen.findByText("Variable costs");
    expect(rateButtons()).toHaveLength(10);

    await user.click(screen.getByRole("switch", { name: /Driver Wages/ }));
    await waitFor(() => expect(rateButtons()).toHaveLength(9));
  });

  it("asks for GL accounts once a category reads its rate from GL actuals", async () => {
    served.control = withGLActuals();
    seedPlan(GRANTS, []);
    const user = userEvent.setup();

    renderForm("?tab=rates");
    await screen.findByText("Variable costs");
    expect(screen.queryByText("GL accounts")).toBeNull();

    await user.click(rateButtons()[0]!);
    await user.click(screen.getByRole("option", { name: /GL actuals/ }));

    expect(await screen.findByText("GL accounts")).toBeInTheDocument();
  });
});

describe("GL actuals", () => {
  it("shows the window and planned miles only while GL actuals are on", async () => {
    served.control = savedCostControl;
    seedPlan(GRANTS, []);
    const user = userEvent.setup();

    renderForm("?tab=actuals");
    await screen.findByRole("switch", { name: /Use GL actuals/ });
    expect(screen.queryByText("Rolling window")).toBeNull();

    await user.click(screen.getByRole("switch", { name: /Use GL actuals/ }));

    expect(await screen.findByText("Rolling window")).toBeInTheDocument();
    expect(screen.getByText("Planned monthly miles")).toBeInTheDocument();
  });

  it("opens Cost rates from its pointer", async () => {
    served.control = withGLActuals();
    seedPlan(GRANTS, []);
    const user = userEvent.setup();

    renderForm("?tab=actuals");
    await user.click(await screen.findByRole("button", { name: "Open Cost rates" }));

    expect(await screen.findByRole("tab", { name: /Cost rates/, selected: true })).toBeVisible();
  });
});

describe("a refused save", () => {
  // The category on GL actuals has no account mapped; the save is refused on the rates tab,
  // which opens, marked, with the reason under the field.
  it("opens and marks Cost rates when a category on GL actuals has no account", async () => {
    served.control = withGLActuals({ rateSource: "GLActual" });
    seedPlan(GRANTS, []);
    const user = userEvent.setup();

    renderForm();
    await user.click(await screen.findByRole("switch", { name: /Include deadhead miles/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    const rates = await screen.findByRole("tab", { name: /Cost rates/, selected: true });
    expect(within(rates).getByText("Has fields to fix")).toBeInTheDocument();
    expect(
      screen.getByText("Map at least one GL account to use GL actuals for this category"),
    ).toBeVisible();
    expect(served.saves).toEqual([]);
  });

  it("saves an inactive category left on an override without a rate on its benchmark", async () => {
    served.control = {
      ...savedCostControl,
      categories: savedCostControl.categories.map(
        (category, index): CostingControl["categories"][number] =>
          index === 0
            ? { ...category, rateSource: "Override", overrideRatePerMile: "0.9" }
            : category,
      ),
    } satisfies CostingControl;
    seedPlan(GRANTS, []);
    const user = userEvent.setup();

    renderForm("?tab=rates");
    await screen.findByText("Your rate");
    await user.clear(screen.getByDisplayValue(/0\.9/));
    await user.click(screen.getByRole("switch", { name: /Driver Wages/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(served.saves).toHaveLength(1));
    const [sent] = served.saves as { categories: { rateSource: string; isActive: boolean }[] }[];
    expect(sent!.categories[0]).toMatchObject({ rateSource: "Benchmark", isActive: false });
  });
});

describe("a save", () => {
  it("sends the changed settings and the changed category in one request", async () => {
    served.control = savedCostControl;
    seedPlan(GRANTS, []);
    const user = userEvent.setup();

    renderForm("?tab=rates");
    await screen.findByText("Variable costs");
    await user.click(screen.getByRole("switch", { name: /Tolls/ }));
    await user.click(screen.getByRole("tab", { name: /Cost basis/ }));
    await user.click(await screen.findByRole("switch", { name: /Include deadhead miles/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(served.requests).toHaveLength(1));
    expect(served.requests[0]).toEqual({
      control: {
        fuelIndexId: null,
        useLiveFuelPrice: false,
        milesPerGallon: "6.5",
        includeDeadheadMiles: false,
        glActualsEnabled: false,
        glRollingMonths: 3,
        plannedMonthlyMiles: null,
        targetMarginPercent: null,
        version: savedCostControl.version,
      },
      categories: [
        {
          id: "ccat_07",
          rateSource: "Benchmark",
          overrideRatePerMile: null,
          isActive: false,
          glAccountIds: [],
          version: 1,
        },
      ],
    });
  });
});
