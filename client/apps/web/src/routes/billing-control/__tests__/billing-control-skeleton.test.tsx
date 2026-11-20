import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PermissionManifest } from "@trenova/shared/types/permission";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PlanFeature } from "@/lib/plan-feature";
import BillingControlForm from "../_components/billing-control-form";
import { BillingControlSkeleton } from "../skeleton";
import { savedBillingControl } from "./billing-control-fixture";

vi.mock("@/lib/queries/billing-control", () => ({
  billingControl: {
    get: Object.assign(
      () => ({ queryKey: ["billing-control"], queryFn: async () => savedBillingControl }),
      { _def: ["billing-control"] },
    ),
  },
}));

vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: () => ({ mutateAsync: vi.fn() }),
}));

function planWithout(...features: string[]) {
  usePermissionStore.setState({
    manifest: {
      unavailableFeatures: features.map((feature) => ({
        feature,
        featureName: feature,
        addOnName: feature,
      })),
    } as unknown as PermissionManifest,
  });
}

type Outline = { tabs: number; sections: number; rows: number };

function skeletonOutline(search = ""): Outline {
  const { container } = render(
    <TestRouter search={search}>
      <BillingControlSkeleton />
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
        <BillingControlForm />
      </QueryClientProvider>
    </TestRouter>,
  );
  await screen.findByText("Terms");
  const open = '[data-slot="tabs-content"]:not([hidden])';
  return {
    tabs: container.querySelectorAll('[role="tab"]').length,
    sections: container.querySelectorAll(`${open} [data-slot="settings-section"]`).length,
    rows: container.querySelectorAll(`${open} [data-slot="setting-row"]`).length,
  };
}

afterEach(() => {
  cleanup();
  usePermissionStore.getState().clearPermissions();
});

describe("BillingControlSkeleton", () => {
  // The fallback has to be the page the organization will see, or the layout jumps when
  // the form lands.
  it.each([
    ["invoices", { tabs: 5, sections: 2, rows: 5 }],
    ["automation", { tabs: 5, sections: 3, rows: 6 }],
    ["exceptions", { tabs: 5, sections: 2, rows: 3 }],
    ["rating", { tabs: 5, sections: 2, rows: 3 }],
    ["collections", { tabs: 5, sections: 3, rows: 4 }],
  ])("draws the %s tab the address opens, as the form does", async (tab, expected) => {
    const search = tab === "invoices" ? "" : `?tab=${tab}`;

    const outline = skeletonOutline(search);
    expect(outline).toEqual(expected);
    cleanup();

    expect(await formOutline(search)).toEqual(outline);
  });

  it("leaves out Nova's review when the plan does not include it, as the form does", async () => {
    planWithout(PlanFeature.AgentAutomation);

    const outline = skeletonOutline("?tab=automation");
    expect(outline).toEqual({ tabs: 5, sections: 3, rows: 5 });
    cleanup();

    expect(await formOutline("?tab=automation")).toEqual(outline);
  });

  it("leaves out margin floors when the plan has no rating, as the form does", async () => {
    planWithout(PlanFeature.Rating);

    const outline = skeletonOutline("?tab=rating");
    expect(outline).toEqual({ tabs: 5, sections: 2, rows: 2 });
    cleanup();

    expect(await formOutline("?tab=rating")).toEqual(outline);
  });

  it("is announced once as loading", () => {
    render(
      <TestRouter>
        <BillingControlSkeleton />
      </TestRouter>,
    );

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});
