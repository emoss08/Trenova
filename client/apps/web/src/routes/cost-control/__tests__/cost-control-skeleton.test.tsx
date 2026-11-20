import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { afterEach, describe, expect, it, vi } from "vitest";
import { clearPermissions } from "@/test/permissions";
import { seedPlan, unavailableFeatures } from "@/test/plan";
import CostControlForm from "../_components/cost-control-form";
import { CostControlSkeleton } from "../skeleton";
import { savedCostControl } from "./cost-control-fixture";

vi.mock("@/lib/queries/cost-control", () => ({
  costControl: {
    get: Object.assign(
      () => ({ queryKey: ["cost-control"], queryFn: async () => savedCostControl }),
      { _def: ["cost-control"] },
    ),
    resolvedProfile: Object.assign(() => ({}), { _def: ["cost-control", "resolved"] }),
  },
}));

vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: () => ({ mutateAsync: vi.fn() }),
}));

const GRANTS = { [Resource.ProtectedMargins]: Operation.Read };

type Outline = { tabs: number; sections: number; rows: number };

function skeletonOutline(search = ""): Outline {
  const { container } = render(
    <TestRouter search={search}>
      <CostControlSkeleton />
    </TestRouter>,
  );
  return {
    tabs: container.querySelectorAll('[data-slot="tab-skeleton"]').length,
    sections: container.querySelectorAll('[data-slot="settings-section-skeleton"]').length,
    rows: container.querySelectorAll('[data-slot="field-skeleton"]').length,
  };
}

async function formOutline(search = ""): Promise<Outline> {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const { container } = render(
    <TestRouter search={search}>
      <QueryClientProvider client={queryClient}>
        <CostControlForm />
      </QueryClientProvider>
    </TestRouter>,
  );
  await screen.findByText("Variable costs");
  const open = '[data-slot="tabs-content"]:not([hidden])';
  return {
    tabs: container.querySelectorAll('[role="tab"]').length,
    sections: container.querySelectorAll(`${open} [data-slot="settings-section"]`).length,
    rows: container.querySelectorAll(`${open} [data-slot="setting-row"]`).length,
  };
}

describe("CostControlSkeleton", () => {
  afterEach(() => {
    cleanup();
    clearPermissions();
  });

  // The fallback has to be the page the organization will see, or the layout jumps when
  // the form lands.
  it.each([
    ["", { tabs: 3, sections: 2, rows: 4 }],
    ["?tab=rates", { tabs: 3, sections: 2, rows: 20 }],
    ["?tab=actuals", { tabs: 3, sections: 1, rows: 1 }],
  ])("draws the tab the address %s opens, as the form does", async (search, expected) => {
    seedPlan(GRANTS, []);

    const outline = skeletonOutline(search);
    expect(outline).toEqual(expected);
    cleanup();

    expect(await formOutline(search)).toEqual(outline);
  });

  it("leaves out GL actuals when the plan has no accounting, as the form does", async () => {
    seedPlan(GRANTS, unavailableFeatures("tms.accounting"));

    const outline = skeletonOutline();
    expect(outline).toEqual({ tabs: 2, sections: 2, rows: 4 });
    cleanup();

    expect(await formOutline()).toEqual(outline);
  });

  it("is announced once as loading", () => {
    seedPlan(GRANTS, []);

    render(
      <TestRouter>
        <CostControlSkeleton />
      </TestRouter>,
    );

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});
