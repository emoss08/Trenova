import type { IftaReturnView } from "@/lib/ifta-return";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ReopenReturnDialog } from "../reopen-return-dialog";

const { reopenIftaReturn, toastSuccess } = vi.hoisted(() => ({
  reopenIftaReturn: vi.fn(),
  toastSuccess: vi.fn(),
}));

vi.mock("@/lib/graphql/ifta-return", () => ({
  IFTA_RETURN_KEY: "ifta-return",
  IFTA_RETURN_LIST_KEY: "ifta-return-list",
  IFTA_PERIOD_KEY: "ifta-period",
  reopenIftaReturn,
}));

vi.mock("sonner", () => ({
  toast: { success: toastSuccess, error: vi.fn(), warning: vi.fn(), info: vi.fn() },
}));

function ret(over: Partial<IftaReturnView> = {}) {
  return {
    id: "ir_1",
    year: 2026,
    quarter: 2,
    status: "Finalized",
    version: 3,
    finalizedAt: 1_785_000_000,
    ...over,
  } as IftaReturnView;
}

function renderDialog(over: Partial<IftaReturnView> = {}) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const invalidateSpy = vi.spyOn(client, "invalidateQueries");
  const onOpenChange = vi.fn();
  render(
    <QueryClientProvider client={client}>
      <ReopenReturnDialog
        open
        onOpenChange={onOpenChange}
        ret={ret(over)}
        period={{ year: 2026, quarter: 2 }}
      />
    </QueryClientProvider>,
  );
  return { invalidateSpy, onOpenChange };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ReopenReturnDialog", () => {
  // The reason is the whole record of why a locked return was opened again,
  // so a few characters is not a reason and the return is left alone.
  it("refuses a reason shorter than ten characters and never calls the mutation", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.type(screen.getByLabelText("Reason"), "typo");
    await user.click(screen.getByRole("button", { name: /Reopen return/i }));

    expect(await screen.findByText(/at least ten characters/i)).toBeInTheDocument();
    expect(reopenIftaReturn).not.toHaveBeenCalled();
  });

  it("sends the return's own version with the trimmed reason", async () => {
    reopenIftaReturn.mockResolvedValue(ret({ status: "Draft", version: 4 }));
    const user = userEvent.setup();
    const { invalidateSpy, onOpenChange } = renderDialog();

    await user.type(screen.getByLabelText("Reason"), "  Oklahoma rate published late  ");
    await user.click(screen.getByRole("button", { name: /Reopen return/i }));

    await waitFor(() => {
      expect(reopenIftaReturn).toHaveBeenCalledExactlyOnceWith({
        id: "ir_1",
        version: 3,
        reason: "Oklahoma rate published late",
      });
    });
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(invalidateSpy.mock.calls.map((call) => call[0]?.queryKey)).toEqual(
      expect.arrayContaining([["ifta-return", "period", 2026, 2], ["ifta-return-list"]]),
    );
  });
});
