import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BankReceiptQueuePage } from "../page";

const mocks = vi.hoisted(() => ({ list: vi.fn(), getSummary: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: {
    bankReceiptWorkItemService: {
      list: mocks.list,
      getById: vi.fn(),
      assign: vi.fn(),
      startReview: vi.fn(),
      resolve: vi.fn(),
      dismiss: vi.fn(),
    },
    bankReceiptService: { getSummary: mocks.getSummary, getById: vi.fn() },
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

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <BankReceiptQueuePage />
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
  mocks.list.mockResolvedValue([]);
  mocks.getSummary.mockResolvedValue({});
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

describe("reconciliation work queue empty states", () => {
  it("says what raises a work item when nothing is waiting, and draws the detail pane", async () => {
    renderPage();

    expect(await screen.findByText("Nothing waiting")).toBeInTheDocument();
    expect(screen.getByText(/cannot be matched/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument();
    expect(screen.getByText("Nothing open")).toBeInTheDocument();
  });

  it("offers to clear the search when it is what emptied the list", async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByText("Nothing waiting");

    const search = screen.getByPlaceholderText("Search reference, ID...");
    await user.type(search, "ACH-1");
    expect(await screen.findByText("Nothing matches")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(await screen.findByText("Nothing waiting")).toBeInTheDocument();
    expect(search).toHaveValue("");
  });
});
