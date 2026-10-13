import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SendReminderDialog } from "../send-reminder-dialog";

const { sendReminder, success } = vi.hoisted(() => ({
  sendReminder: vi.fn(),
  success: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: { collectionService: { sendReminder } },
}));
vi.mock("sonner", () => ({ toast: { success, error: vi.fn() } }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("SendReminderDialog", () => {
  it("sends the reminder with the note and says who it went to", async () => {
    sendReminder.mockResolvedValue({
      reminders: [
        { sentTo: ["ap@halstead.test"], invoiceNumbers: ["INV-1"], totalOverdue: "1250.00 USD" },
      ],
    });
    const onOpenChange = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient()}>
        <SendReminderDialog
          customerId="cus_1"
          customerName="Halstead Grocery"
          open
          onOpenChange={onOpenChange}
        />
      </QueryClientProvider>,
    );

    await userEvent.type(screen.getByLabelText("Note to the customer"), "  As discussed today ");
    await userEvent.click(screen.getByRole("button", { name: "Send reminder" }));

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(sendReminder).toHaveBeenCalledWith("cus_1", "As discussed today");
    expect(success).toHaveBeenCalledWith("Reminder sent to ap@halstead.test", {
      description: "1250.00 USD",
    });
  });
});
