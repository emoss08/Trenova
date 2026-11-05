import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { YtdCard } from "./ytd-card";

const { fetchMyYtdPay } = vi.hoisted(() => ({ fetchMyYtdPay: vi.fn() }));

vi.mock("@trenova/shared/lib/graphql/driver-portal", () => ({ fetchMyYtdPay }));

function failed(status: number): ApiRequestError {
  return new ApiRequestError(status, { type: "about:blank", title: "Failed", status });
}

function renderCard() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <YtdCard />
    </QueryClientProvider>,
  );
}

describe("YtdCard", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("says pay could not be loaded and offers another try when the read fails", async () => {
    fetchMyYtdPay.mockRejectedValue(failed(500));

    renderCard();

    expect((await screen.findByRole("alert")).textContent).toContain("We couldn't load your pay.");
    expect(screen.getByRole("button", { name: "Try again" })).toBeTruthy();
  });

  it("draws nothing for a driver with no pay to show", async () => {
    fetchMyYtdPay.mockRejectedValue(failed(404));

    const { container } = renderCard();

    await vi.waitFor(() => expect(fetchMyYtdPay).toHaveBeenCalled());
    await vi.waitFor(() => expect(container.querySelector(".ui-shimmer")).toBeNull());
    expect(screen.queryByRole("alert")).toBeNull();
  });
});
