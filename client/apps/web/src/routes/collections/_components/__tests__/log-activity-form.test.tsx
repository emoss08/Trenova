import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useController, type Control } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { LogActivityForm } from "../log-activity-form";

const { logActivity } = vi.hoisted(() => ({ logActivity: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: { collectionService: { logActivity } },
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/components/fields/date-field/date-field", () => ({
  AutoCompleteDateField: ({
    control,
    name,
    label,
  }: {
    control: Control;
    name: string;
    label: string;
  }) => {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label}
        type="number"
        value={(field.value as number) || ""}
        onChange={(event) =>
          field.onChange(event.target.value === "" ? null : Number(event.target.value))
        }
      />
    );
  },
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function renderForm() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <LogActivityForm customerId="cus_1" />
    </QueryClientProvider>,
  );
}

describe("LogActivityForm", () => {
  it("records a promise with its exact amount and date", async () => {
    logActivity.mockResolvedValue({ row: {}, activities: [], hasMore: false });
    renderForm();

    await userEvent.click(screen.getByRole("radio", { name: "Promise to pay" }));
    await userEvent.type(screen.getByLabelText("Spoke with"), "Dana in payables");
    await userEvent.type(screen.getByLabelText("Promised amount"), "1250.50");
    await userEvent.type(screen.getByLabelText("Will pay by"), "1700000000");
    await userEvent.click(screen.getByRole("button", { name: "Record promise" }));

    await waitFor(() => expect(logActivity).toHaveBeenCalledTimes(1));
    expect(logActivity).toHaveBeenCalledWith("cus_1", {
      kind: "PromiseToPay",
      summary: "",
      contactName: "Dana in payables",
      invoiceId: null,
      promisedAmount: "1250.50",
      promisedOn: 1_700_000_000,
      followUpOn: null,
    });
  });

  it("refuses a promise amount with more than two decimal places", async () => {
    renderForm();

    await userEvent.click(screen.getByRole("radio", { name: "Promise to pay" }));
    await userEvent.type(screen.getByLabelText("Promised amount"), "12.345");
    await userEvent.type(screen.getByLabelText("Will pay by"), "1700000000");
    await userEvent.click(screen.getByRole("button", { name: "Record promise" }));

    expect(
      await screen.findByText("Enter the promised amount, with at most two decimal places"),
    ).toBeInTheDocument();
    expect(logActivity).not.toHaveBeenCalled();
  });

  it("asks what happened on a call, and sends no promise fields", async () => {
    logActivity.mockResolvedValue({ row: {}, activities: [], hasMore: false });
    renderForm();

    await userEvent.click(screen.getByRole("button", { name: "Log it" }));
    expect(await screen.findByText("Say what happened")).toBeInTheDocument();

    await userEvent.type(screen.getByLabelText("What happened"), "  Left a voicemail ");
    await userEvent.type(screen.getByLabelText("Follow up on"), "1700086400");
    await userEvent.click(screen.getByRole("button", { name: "Log it" }));

    await waitFor(() => expect(logActivity).toHaveBeenCalledTimes(1));
    expect(logActivity).toHaveBeenCalledWith("cus_1", {
      kind: "Call",
      summary: "Left a voicemail",
      contactName: "",
      invoiceId: null,
      promisedAmount: null,
      promisedOn: null,
      followUpOn: 1_700_086_400,
    });
  });
});
