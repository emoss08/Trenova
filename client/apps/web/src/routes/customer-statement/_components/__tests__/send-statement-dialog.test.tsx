import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SendStatementDialog } from "../send-statement-dialog";

const { send, success } = vi.hoisted(() => ({
  send: vi.fn(),
  success: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: { customerStatementService: { send } },
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({ toast: { success, error: vi.fn() } }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("SendStatementDialog", () => {
  it("emails the statement on screen with the note and says who it went to", async () => {
    send.mockResolvedValue({
      sentTo: ["ap@halstead.test", "controller@halstead.test"],
      endingBalance: "USD 4,102.36",
      fileName: "Statement-HALSTEAD-2026-08-31.pdf",
    });
    const onOpenChange = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient()}>
        <SendStatementDialog
          customerId="cus_1"
          customerName="Halstead Grocery"
          range={{ startDate: 100, asOfDate: 200 }}
          rangeLabel="from Aug 1, 2026 to Aug 31, 2026"
          open
          onOpenChange={onOpenChange}
        />
      </QueryClientProvider>,
    );

    expect(screen.getByText(/from Aug 1, 2026 to Aug 31, 2026/)).toBeInTheDocument();
    await userEvent.type(screen.getByLabelText("Note to the customer"), "  As requested ");
    await userEvent.click(screen.getByRole("button", { name: "Send statement" }));

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(send).toHaveBeenCalledWith("cus_1", { startDate: 100, asOfDate: 200 }, "As requested");
    expect(success).toHaveBeenCalledWith(
      "Statement sent to ap@halstead.test, controller@halstead.test",
      { description: "USD 4,102.36 due" },
    );
  });
});
