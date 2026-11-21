import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import DistanceControlForm from "../_components/distance-control-form";
import { DistanceControlSkeleton } from "../skeleton";
import { savedDistanceControl } from "./distance-control-fixture";

// Both stored mileage switches off, so city matching is closed, as the outline draws it.
const closed = { ...savedDistanceControl, storeMileage: false, autoCreateStoredMileage: false };

vi.mock("@/lib/queries/distance-control", () => ({
  distanceControl: {
    get: Object.assign(() => ({ queryKey: ["distance-control"], queryFn: async () => closed }), {
      _def: ["distance-control"],
    }),
  },
}));

vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: () => ({ mutateAsync: vi.fn() }),
}));

type Outline = { tabs: number; sections: number; rows: number };

function skeletonOutline(): Outline {
  const { container } = render(
    <TestRouter>
      <DistanceControlSkeleton />
    </TestRouter>,
  );
  return {
    tabs: container.querySelectorAll('[data-slot="tab-skeleton"]').length,
    sections: container.querySelectorAll('[data-slot="settings-section-skeleton"]').length,
    rows: container.querySelectorAll('[data-slot="field-skeleton"]').length,
  };
}

async function formOutline(): Promise<Outline> {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const { container } = render(
    <TestRouter>
      <QueryClientProvider client={queryClient}>
        <DistanceControlForm />
      </QueryClientProvider>
    </TestRouter>,
  );
  await screen.findByText("Stored mileage");
  return {
    tabs: container.querySelectorAll('[role="tab"]').length,
    sections: container.querySelectorAll('[data-slot="settings-section"]').length,
    rows: container.querySelectorAll('[data-slot="setting-row"]').length,
  };
}

describe("DistanceControlSkeleton", () => {
  afterEach(() => {
    cleanup();
  });

  // The fallback has to be the page that lands, or the layout jumps: five sections, ten
  // rows with city matching closed, and, with one tab, no tab row.
  it("draws the five sections and their rows, with no tab row, as the form does", async () => {
    const outline = skeletonOutline();
    expect(outline).toEqual({ tabs: 0, sections: 5, rows: 10 });
    cleanup();

    expect(await formOutline()).toEqual(outline);
  });

  it("is announced once as loading", () => {
    render(
      <TestRouter>
        <DistanceControlSkeleton />
      </TestRouter>,
    );

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});
