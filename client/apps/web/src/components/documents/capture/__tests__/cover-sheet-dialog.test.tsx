import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CoverSheetDialog } from "../cover-sheet-dialog";

function renderDialog(open: boolean, client: QueryClient) {
  return (
    <QueryClientProvider client={client}>
      <CoverSheetDialog open={open} onOpenChange={() => {}} kind="shipment" recordId="shp_1" />
    </QueryClientProvider>
  );
}

describe("CoverSheetDialog", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("starts from one sheet each time it opens, not from the last run", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("{}", { status: 200 })),
    );
    const user = userEvent.setup();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { rerender } = render(renderDialog(true, client));

    const copies = await screen.findByDisplayValue("1");
    await user.clear(copies);
    await user.type(copies, "3");
    expect(screen.getByDisplayValue("3")).toBeInTheDocument();

    rerender(renderDialog(false, client));
    rerender(renderDialog(true, client));

    await waitFor(() => {
      expect(screen.getByDisplayValue("1")).toBeInTheDocument();
    });
    expect(screen.queryByDisplayValue("3")).toBeNull();
  });
});
