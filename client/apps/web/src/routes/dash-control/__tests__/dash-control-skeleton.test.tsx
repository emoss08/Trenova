import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { domMax, LazyMotion } from "motion/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { seedPlan, unavailableFeatures } from "@/test/plan";
import DashControlForm from "../_components/dash-control-form";
import { DashControlSkeleton } from "../skeleton";
import { savedDashControl } from "./dash-control-fixture";

vi.mock("@/lib/queries/dash-control", () => ({
  dashControl: {
    get: Object.assign(
      () => ({ queryKey: ["dash-control", "get"], queryFn: async () => savedDashControl }),
      { _def: ["dash-control", "get"] },
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
      <DashControlSkeleton />
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
      <LazyMotion features={domMax}>
        <QueryClientProvider client={queryClient}>
          <DashControlForm />
        </QueryClientProvider>
      </LazyMotion>
    </TestRouter>,
  );
  await screen.findByText("Assignments");
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

describe("DashControlSkeleton", () => {
  it("draws every tab, and Loads' sections and rows, on a full plan", async () => {
    const outline = skeletonOutline();
    expect(outline).toEqual({ tabs: 4, sections: 3, rows: 5 });
    cleanup();

    expect(await formOutline()).toEqual(outline);
  });

  it.each([
    ["pay", { tabs: 4, sections: 2, rows: 3 }],
    ["profile", { tabs: 4, sections: 2, rows: 3 }],
    ["reminders", { tabs: 4, sections: 1, rows: 1 }],
  ])("draws the %s tab the address opens, as the form does", async (tab, expected) => {
    const outline = skeletonOutline(`?tab=${tab}`);
    expect(outline).toEqual(expected);
    cleanup();

    expect(await formOutline(`?tab=${tab}`)).toEqual(outline);
  });

  it("leaves out pay and expenses when the plan has no driver settlements, as the form does", async () => {
    seedPlan({}, unavailableFeatures("tms.settlement"));

    const outline = skeletonOutline();
    expect(outline).toEqual({ tabs: 3, sections: 3, rows: 5 });
    cleanup();

    expect(await formOutline()).toEqual(outline);
  });

  it("is announced once as loading", () => {
    render(
      <TestRouter>
        <DashControlSkeleton />
      </TestRouter>,
    );

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});
