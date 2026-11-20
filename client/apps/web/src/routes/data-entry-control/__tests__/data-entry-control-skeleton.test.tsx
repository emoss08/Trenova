import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import DataEntryControlForm from "../_components/data-entry-control-form";
import { DataEntryControlSkeleton } from "../skeleton";
import { savedDataEntryControl } from "./data-entry-control-fixture";

vi.mock("@/lib/queries/data-entry-control", () => ({
  dataEntryControl: {
    get: Object.assign(
      () => ({ queryKey: ["data-entry-control"], queryFn: async () => savedDataEntryControl }),
      { _def: ["data-entry-control"] },
    ),
  },
}));

vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: () => ({ mutateAsync: vi.fn() }),
}));

type Outline = { tabs: number; sections: number; rows: number };

function skeletonOutline(): Outline {
  const { container } = render(
    <TestRouter>
      <DataEntryControlSkeleton />
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
        <DataEntryControlForm />
      </QueryClientProvider>
    </TestRouter>,
  );
  await screen.findByText("Case formatting");
  return {
    tabs: container.querySelectorAll('[role="tab"]').length,
    sections: container.querySelectorAll('[data-slot="settings-section"]').length,
    rows: container.querySelectorAll('[data-slot="setting-row"]').length,
  };
}

describe("DataEntryControlSkeleton", () => {
  afterEach(() => {
    cleanup();
  });

  // The fallback has to be the page that lands, or the layout jumps: one section of four
  // rows and, with one tab, no tab row.
  it("draws the one section and its four rows, with no tab row, as the form does", async () => {
    const outline = skeletonOutline();
    expect(outline).toEqual({ tabs: 0, sections: 1, rows: 4 });
    cleanup();

    expect(await formOutline()).toEqual(outline);
  });

  it("is announced once as loading", () => {
    render(
      <TestRouter>
        <DataEntryControlSkeleton />
      </TestRouter>,
    );

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});
