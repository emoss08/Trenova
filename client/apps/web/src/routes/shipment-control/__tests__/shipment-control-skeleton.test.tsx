import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import ShipmentControlForm from "../_components/shipment-control-form";
import { ShipmentControlSkeleton } from "../skeleton";
import { savedShipmentControl } from "./shipment-control-fixture";
import { resetPlan, withoutDetention } from "./shipment-control-plan";

vi.mock("@/lib/queries/shipment-control", () => ({
  shipmentControl: {
    get: Object.assign(
      () => ({ queryKey: ["shipment-control"], queryFn: async () => savedShipmentControl }),
      { _def: ["shipment-control"] },
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
      <ShipmentControlSkeleton />
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
        <ShipmentControlForm />
      </QueryClientProvider>
    </TestRouter>,
  );
  await screen.findByText("Entry checks");
  const open = '[data-slot="tabs-content"]:not([hidden])';
  return {
    tabs: container.querySelectorAll('[role="tab"]').length,
    sections: container.querySelectorAll(`${open} [data-slot="settings-section"]`).length,
    rows: container.querySelectorAll(`${open} [data-slot="setting-row"]`).length,
  };
}

describe("ShipmentControlSkeleton", () => {
  afterEach(() => {
    cleanup();
    resetPlan();
  });

  // The fallback has to be the page the organization will see, or the layout jumps when the
  // form lands.
  it.each([
    ["", { tabs: 3, sections: 2, rows: 4 }],
    ["?tab=automation", { tabs: 3, sections: 2, rows: 2 }],
    ["?tab=detention", { tabs: 3, sections: 2, rows: 2 }],
  ])("draws the tab the address %s opens, as the form does", async (search, expected) => {
    const outline = skeletonOutline(search);
    expect(outline).toEqual(expected);
    cleanup();

    expect(await formOutline(search)).toEqual(outline);
  });

  it("leaves out detention when the plan does, as the form does", async () => {
    withoutDetention();

    const outline = skeletonOutline("?tab=detention");
    expect(outline).toEqual({ tabs: 2, sections: 2, rows: 4 });
    cleanup();

    expect(await formOutline("?tab=detention")).toEqual(outline);
  });

  it("is announced once as loading", () => {
    render(
      <TestRouter>
        <ShipmentControlSkeleton />
      </TestRouter>,
    );

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});
