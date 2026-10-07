import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { queries } from "@/lib/queries";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SHIPMENT_LIST_KEY } from "../../shipment-queries";
import { BoardRefreshButton } from "../toolbar/board-refresh-button";

afterEach(cleanup);

function renderButton() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <BoardRefreshButton />
    </QueryClientProvider>,
  );
  return { queryClient, button: screen.getByRole("button", { name: "Refresh" }) };
}

function isSpinning(button: HTMLElement) {
  return button.querySelector("svg")?.classList.contains("animate-spin") ?? false;
}

async function startLoading(queryClient: QueryClient, queryKey: readonly unknown[]) {
  await act(async () => {
    void queryClient.prefetchQuery({ queryKey, queryFn: never });
    await new Promise((resolve) => setTimeout(resolve, 20));
  });
}

function never() {
  return new Promise<never>(() => {});
}

describe("BoardRefreshButton", () => {
  it("rests while nothing on the board is loading", () => {
    const { button } = renderButton();
    expect(isSpinning(button)).toBe(false);
  });

  it("spins while the shipment list loads", async () => {
    const { queryClient, button } = renderButton();
    await startLoading(queryClient, [SHIPMENT_LIST_KEY, "page"]);
    expect(isSpinning(button)).toBe(true);
  });

  it("spins while a board query loads", async () => {
    const { queryClient, button } = renderButton();
    await startLoading(queryClient, [...queries.shipmentBoard._def, "summary"]);
    expect(isSpinning(button)).toBe(true);
  });

  it("ignores loads that are not the board's", async () => {
    const { queryClient, button } = renderButton();
    await startLoading(queryClient, ["customers"]);
    expect(isSpinning(button)).toBe(false);
  });

  it("reloads the list, the board and the shipment events", () => {
    const { queryClient, button } = renderButton();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");

    fireEvent.click(button);

    expect(invalidate).toHaveBeenCalledWith({ queryKey: [SHIPMENT_LIST_KEY] });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: queries.shipmentBoard._def });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["shipment-events"] });
  });
});
