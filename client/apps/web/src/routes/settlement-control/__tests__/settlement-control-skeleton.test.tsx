import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import SettlementControlForm from "../_components/settlement-control-form";
import { SettlementControlSkeleton } from "../skeleton";
import { savedSettlementControl } from "./settlement-control-fixture";

vi.mock("@/lib/queries/settlement-control", () => ({
  settlementControl: {
    get: Object.assign(
      () => ({ queryKey: ["settlement-control"], queryFn: async () => savedSettlementControl }),
      { _def: ["settlement-control"] },
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
      <SettlementControlSkeleton />
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
        <SettlementControlForm />
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

describe("SettlementControlSkeleton", () => {
  afterEach(cleanup);

  // The fallback has to be the page the form draws, or the layout jumps when the form lands.
  it.each([
    ["", { tabs: 3, sections: 2, rows: 6 }],
    ["?tab=review", { tabs: 3, sections: 2, rows: 5 }],
    ["?tab=deductions", { tabs: 3, sections: 2, rows: 3 }],
  ])("draws the tab the address %j opens, as the form does", async (search, expected) => {
    const outline = skeletonOutline(search);
    expect(outline).toEqual(expected);
    cleanup();

    expect(await formOutline(search)).toEqual(outline);
  });

  it("is announced once as loading", () => {
    render(
      <TestRouter>
        <SettlementControlSkeleton />
      </TestRouter>,
    );

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});
