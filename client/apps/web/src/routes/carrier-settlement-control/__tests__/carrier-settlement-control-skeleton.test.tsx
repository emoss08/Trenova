import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { afterEach, describe, expect, it, vi } from "vitest";
import { seedPlan, unavailableFeatures } from "@/test/plan";
import CarrierSettlementControlForm from "../_components/carrier-settlement-control-form";
import { CarrierSettlementControlSkeleton } from "../skeleton";
import { savedCarrierSettlementControl } from "./carrier-settlement-control-fixture";

vi.mock("@/lib/queries/carrier-settlement-control", () => ({
  carrierSettlementControl: {
    get: Object.assign(
      () => ({
        queryKey: ["carrier-settlement-control"],
        queryFn: async () => savedCarrierSettlementControl,
      }),
      { _def: ["carrier-settlement-control"] },
    ),
  },
}));

vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: () => ({ mutateAsync: vi.fn() }),
}));

type Outline = { tabs: number; sections: number; rows: number };

function skeletonOutline(search = ""): Outline {
  const { container } = render(
    <TestRouter search={search}>
      <CarrierSettlementControlSkeleton />
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
        <CarrierSettlementControlForm />
      </QueryClientProvider>
    </TestRouter>,
  );
  await screen.findByText("Schedule");
  const open = '[data-slot="tabs-content"]:not([hidden])';
  return {
    tabs: container.querySelectorAll('[role="tab"]').length,
    sections: container.querySelectorAll(`${open} [data-slot="settings-section"]`).length,
    rows: container.querySelectorAll(`${open} [data-slot="setting-row"]`).length,
  };
}

describe("CarrierSettlementControlSkeleton", () => {
  afterEach(() => {
    cleanup();
    usePermissionStore.setState({ manifest: null });
  });

  // The fallback has to be the page the form draws, or the layout jumps when the form lands.
  it.each([
    ["", { tabs: 3, sections: 2, rows: 5 }],
    ["?tab=matching", { tabs: 3, sections: 2, rows: 3 }],
    ["?tab=approval", { tabs: 3, sections: 2, rows: 4 }],
  ])("draws the tab the address %s opens, as the form does", async (search, expected) => {
    const outline = skeletonOutline(search);
    expect(outline).toEqual(expected);
    cleanup();

    expect(await formOutline(search)).toEqual(outline);
  });

  it("leaves out matching invoices as they arrive on a plan without EDI, as the form does", async () => {
    seedPlan({}, unavailableFeatures("integrations.edi"));

    const outline = skeletonOutline("?tab=matching");
    expect(outline).toEqual({ tabs: 3, sections: 2, rows: 2 });
    cleanup();

    expect(await formOutline("?tab=matching")).toEqual(outline);
  });

  it("is announced once as loading", () => {
    render(
      <TestRouter>
        <CarrierSettlementControlSkeleton />
      </TestRouter>,
    );

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});
