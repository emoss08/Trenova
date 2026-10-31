import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BillingQueueFilters } from "../billing-queue-filters";

vi.mock("@/services/api", () => ({
  apiService: {
    billingQueueService: {
      listFilterPresets: vi.fn().mockResolvedValue([]),
      deleteFilterPreset: vi.fn(),
    },
  },
}));
vi.mock("@/lib/graphql/select-options", () => ({
  fetchGraphQLSelectOptions: vi.fn().mockResolvedValue({ count: 0, results: [] }),
}));

afterEach(cleanup);

function renderFilters(search = "") {
  render(
    <NuqsTestingAdapter searchParams={search}>
      <QueryClientProvider client={new QueryClient()}>
        <BillingQueueFilters />
      </QueryClientProvider>
    </NuqsTestingAdapter>,
  );
}

describe("BillingQueueFilters", () => {
  it("names the status and bill type it shows in words, not their values", async () => {
    const user = userEvent.setup();
    renderFilters();

    await user.click(screen.getByRole("button", { name: /Filter/u }));

    expect(await screen.findByRole("combobox", { name: "Status" })).toHaveTextContent(
      "All statuses",
    );
    expect(screen.getByRole("combobox", { name: "Bill type" })).toHaveTextContent("All bill types");
  });

  it("shows the status and bill type in force by their labels", async () => {
    const user = userEvent.setup();
    renderFilters("?status=InReview&billType=CreditMemo");

    await user.click(screen.getByRole("button", { name: /Filter/u }));

    expect(await screen.findByRole("combobox", { name: "Status" })).toHaveTextContent("In review");
    expect(screen.getByRole("combobox", { name: "Bill type" })).toHaveTextContent("Credit memo");
  });
});
