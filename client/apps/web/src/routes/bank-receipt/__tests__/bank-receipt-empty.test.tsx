import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter, type OnUrlUpdateFunction } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BankReceiptPage } from "../page";

const mocks = vi.hoisted(() => ({ getExceptions: vi.fn(), getSummary: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: {
    bankReceiptService: {
      getExceptions: mocks.getExceptions,
      getSummary: mocks.getSummary,
      getById: vi.fn(),
      getSuggestions: vi.fn(),
      match: vi.fn(),
    },
  },
}));
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));
vi.mock("@/components/billing/billing-workspace-layout", () => ({
  BillingWorkspaceLayout: ({
    toolbar,
    sidebar,
    detail,
  }: {
    toolbar?: ReactNode;
    sidebar: ReactNode;
    detail: ReactNode;
  }) => (
    <div>
      {toolbar}
      <aside>{sidebar}</aside>
      <main>{detail}</main>
    </div>
  ),
}));

function renderPage(searchParams = "", onUrlUpdate?: OnUrlUpdateFunction) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter searchParams={searchParams} onUrlUpdate={onUrlUpdate}>
          <BankReceiptPage />
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  );
  mocks.getExceptions.mockResolvedValue([]);
  mocks.getSummary.mockResolvedValue({});
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

describe("bank receipts empty states", () => {
  it("says where receipts come from when there are none, and draws the detail pane", async () => {
    renderPage();

    expect(await screen.findByText("No receipts yet")).toBeInTheDocument();
    expect(screen.getByText(/Import Batches/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument();
    expect(screen.getByText("Nothing open")).toBeInTheDocument();
  });

  it("offers to clear the filters when they are what emptied the list", async () => {
    const onUrlUpdate = vi.fn();
    const user = userEvent.setup();
    renderPage("?status=Matched&query=ACH", onUrlUpdate);

    expect(await screen.findByText("Nothing matches")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));

    const last = onUrlUpdate.mock.calls.at(-1)?.[0] as { searchParams: URLSearchParams };
    expect(last.searchParams.get("status")).toBeNull();
    expect(last.searchParams.get("query")).toBeNull();
  });
});
