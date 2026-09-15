import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { InvoiceSidebar } from "../invoice-sidebar";

const mocks = vi.hoisted(() => ({ requestGraphQL: vi.fn() }));

vi.mock("@trenova/shared/lib/graphql", () => ({ requestGraphQL: mocks.requestGraphQL }));
vi.mock("@/hooks/use-post-invoice", () => ({
  usePostInvoice: () => ({ mutate: vi.fn(), isPending: false }),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

beforeEach(() => {
  mocks.requestGraphQL.mockResolvedValue({
    invoices: {
      edges: [],
      totalCount: 0,
      pageInfo: { hasNextPage: false, endCursor: null, hasPreviousPage: false, startCursor: null },
    },
  });
});

function renderSidebar(searchParams: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter searchParams={searchParams}>
          <InvoiceSidebar selectedInvoiceId={null} onSelectInvoice={vi.fn()} />
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

function lastFieldFilters() {
  const call = mocks.requestGraphQL.mock.calls.at(-1)?.[0] as {
    variables: { input: { fieldFilters: Array<{ field: string; value: string }> } };
  };
  return call.variables.input.fieldFilters;
}

describe("invoice sidebar dispute filter", () => {
  it("asks only for disputed invoices when the URL says so", async () => {
    renderSidebar("?dispute=true");

    await waitFor(() => expect(mocks.requestGraphQL).toHaveBeenCalled());
    expect(lastFieldFilters()).toContainEqual({
      field: "disputeStatus",
      operator: "eq",
      value: "Disputed",
    });
    expect(screen.getByRole("checkbox", { name: "Disputed only" })).toBeChecked();
  });

  it("filters on nothing but the status when the URL names one", async () => {
    renderSidebar("?status=Voided");

    await waitFor(() => expect(mocks.requestGraphQL).toHaveBeenCalled());
    expect(lastFieldFilters()).toEqual([{ field: "status", operator: "eq", value: "Voided" }]);
  });
});
