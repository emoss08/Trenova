import { DataTableAsk } from "@/components/data-table/data-table-ask";
import { apiService } from "@/services/api";
import type { ComposedTableQuery } from "@/types/table-query";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

/*
The Ask input answers with filters, not with an answer.

That is the whole reason it is safe to put on a data table: what comes back
lands in the same builder and the same chips a hand-built filter does, so the
person sees every condition and can change or drop it before reading a row.

What these tests hold is the other half — that a condition the catalogue could
not express is never silently dropped. A filter that did not apply and did not
say so is the worst outcome available here: the table comes back looking
answered, and the one thing the person cared about is the one that went
missing.
*/

function composed(overrides: Partial<ComposedTableQuery> = {}): ComposedTableQuery {
  return {
    query: "",
    fieldFilters: [],
    sort: [],
    explanation: "",
    terms: [],
    unresolved: [],
    ...overrides,
  };
}

function renderAsk(onApply = vi.fn()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );

  render(
    <DataTableAsk resource="shipment" query="" fieldFilters={[]} sort={[]} onApply={onApply} />,
    { wrapper },
  );

  return onApply;
}

const composeMock = vi.fn();
const catalogueMock = vi.fn();

beforeEach(() => {
  vi.clearAllMocks();
  apiService.tableQueryService.compose = composeMock;
  apiService.tableQueryService.catalogue = catalogueMock;
  catalogueMock.mockResolvedValue({
    results: [{ resource: "shipment", entity: "shipments", summary: "", fields: [] }],
    count: 1,
  });
});

async function ask(text: string) {
  const user = userEvent.setup();
  await user.type(await screen.findByLabelText("Ask for what you want"), text);
  await user.keyboard("{Enter}");

  return user;
}

describe("DataTableAsk", () => {
  it("hands the compiled filters to the table", async () => {
    composeMock.mockResolvedValue(
      composed({
        fieldFilters: [{ field: "status", operator: "eq", value: "InTransit" }],
        explanation: "shipments where status equals InTransit",
      }),
    );
    const onApply = renderAsk();

    await ask("in transit");

    await waitFor(() => expect(onApply).toHaveBeenCalledTimes(1));
    expect(onApply.mock.calls[0][0].fieldFilters).toEqual([
      { field: "status", operator: "eq", value: "InTransit" },
    ]);
  });

  // The current state goes with the question, so "and only Acme's" narrows
  // what is on screen rather than replacing it.
  it("sends what the table already shows", async () => {
    composeMock.mockResolvedValue(composed());
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <DataTableAsk
          resource="shipment"
          query="acme"
          fieldFilters={[{ field: "status", operator: "eq", value: "InTransit" }]}
          sort={[{ field: "proNumber", direction: "asc" }]}
          onApply={vi.fn()}
        />
      </QueryClientProvider>,
    );

    await ask("and delivered");

    await waitFor(() => expect(composeMock).toHaveBeenCalled());
    expect(composeMock.mock.calls[0][1].current).toEqual({
      query: "acme",
      fieldFilters: [{ field: "status", operator: "eq", value: "InTransit" }],
      sort: [{ field: "proNumber", direction: "asc" }],
    });
  });

  it("says what it could not express, unprompted", async () => {
    composeMock.mockResolvedValue(
      composed({
        fieldFilters: [{ field: "status", operator: "eq", value: "InTransit" }],
        explanation: "shipments where status equals InTransit",
        unresolved: [{ phrase: "the profitable ones", reason: "margin is not a field here" }],
      }),
    );
    renderAsk();

    await ask("in transit and profitable");

    // Opened without being clicked: a missed condition the reader has to go
    // looking for is a missed condition they will not find.
    expect(await screen.findByText("margin is not a field here")).toBeInTheDocument();
    expect(screen.getByText("the profitable ones")).toBeInTheDocument();
  });

  // Nothing compiled means there is nothing to apply and nothing to review;
  // moving the table would be moving it for no stated reason.
  it("does not touch the table when nothing compiled", async () => {
    composeMock.mockResolvedValue(
      composed({
        unresolved: [{ phrase: "happy drivers", reason: "driverMood is not a field here" }],
      }),
    );
    const onApply = renderAsk();

    await ask("happy drivers");

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /Nothing applied/ })).toBeInTheDocument(),
    );
    expect(onApply).not.toHaveBeenCalled();
  });

  it("applies a sort with no filters", async () => {
    composeMock.mockResolvedValue(composed({ sort: [{ field: "proNumber", direction: "asc" }] }));
    const onApply = renderAsk();

    await ask("oldest first");

    await waitFor(() => expect(onApply).toHaveBeenCalledTimes(1));
  });

  it("asks nothing when the box is empty", async () => {
    renderAsk();

    const user = userEvent.setup();
    await user.click(await screen.findByLabelText("Ask for what you want"));
    await user.keyboard("{Enter}");

    expect(composeMock).not.toHaveBeenCalled();
  });

  // A table the catalogue does not cover gets no input at all, rather than one
  // whose every question comes back refused.
  it("is not there for a table the server cannot narrow", async () => {
    catalogueMock.mockResolvedValue({ results: [], count: 0 });
    renderAsk();

    await waitFor(() => expect(catalogueMock).toHaveBeenCalled());
    expect(screen.queryByLabelText("Ask for what you want")).not.toBeInTheDocument();
  });
});
