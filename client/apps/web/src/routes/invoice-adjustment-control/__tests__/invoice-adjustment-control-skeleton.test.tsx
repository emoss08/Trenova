import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import InvoiceAdjustmentControlForm from "../_components/invoice-adjustment-control-form";
import { InvoiceAdjustmentControlSkeleton } from "../skeleton";
import { closedInvoiceAdjustmentControl } from "./invoice-adjustment-control-fixture";

vi.mock("@/lib/queries/invoice-adjustment-control", () => ({
  invoiceAdjustmentControl: {
    get: Object.assign(
      () => ({
        queryKey: ["invoice-adjustment-control"],
        queryFn: async () => closedInvoiceAdjustmentControl,
      }),
      { _def: ["invoice-adjustment-control"] },
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
      <InvoiceAdjustmentControlSkeleton />
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
        <InvoiceAdjustmentControlForm />
      </QueryClientProvider>
    </TestRouter>,
  );
  await screen.findByText("Invoices that can be adjusted");
  const open = '[data-slot="tabs-content"]:not([hidden])';
  return {
    tabs: container.querySelectorAll('[role="tab"]').length,
    sections: container.querySelectorAll(`${open} [data-slot="settings-section"]`).length,
    rows: container.querySelectorAll(`${open} [data-slot="setting-row"]`).length,
  };
}

describe("InvoiceAdjustmentControlSkeleton", () => {
  afterEach(cleanup);

  // The fallback has to be the form's own shape, with the rows that wait on another setting
  // closed, or the layout jumps when the form lands.
  it.each([
    ["", { tabs: 3, sections: 2, rows: 4 }],
    ["?tab=approval", { tabs: 3, sections: 3, rows: 4 }],
    ["?tab=credits", { tabs: 3, sections: 2, rows: 3 }],
  ])("draws the tab the address %s opens, as the form does", async (search, expected) => {
    const outline = skeletonOutline(search);
    expect(outline).toEqual(expected);
    cleanup();

    expect(await formOutline(search)).toEqual(outline);
  });

  it("is announced once as loading", () => {
    render(
      <TestRouter>
        <InvoiceAdjustmentControlSkeleton />
      </TestRouter>,
    );

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});
