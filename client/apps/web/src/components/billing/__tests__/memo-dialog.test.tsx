import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoDialog } from "../memo-dialog";

const mocks = vi.hoisted(() => ({
  createMemo: vi.fn(),
  navigate: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/lib/graphql/invoice", () => ({ createMemo: mocks.createMemo }));
vi.mock("sonner", () => ({ toast: { success: mocks.toastSuccess, error: mocks.toastError } }));
vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return { ...actual, useNavigate: () => mocks.navigate };
});
vi.mock("@/components/autocomplete-fields", () => ({
  CustomerAutocompleteField: ({ name }: { name: string }) => (
    <input aria-label="Customer" name={name} readOnly value="cus_1" />
  ),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function renderDialog(props: Partial<React.ComponentProps<typeof MemoDialog>> = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onOpenChange = vi.fn();
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <MemoDialog open onOpenChange={onOpenChange} billType="CreditMemo" {...props} />
      </QueryClientProvider>
    </MemoryRouter>,
  );
  return onOpenChange;
}

beforeEach(() => {
  mocks.createMemo.mockResolvedValue({
    id: "inv_cm",
    number: "CM-1",
    billType: "CreditMemo",
    status: "Draft",
  });
});

describe("MemoDialog", () => {
  it("starts blank for the customer it is opened on, with no invoice reference", () => {
    renderDialog({ customerId: "cus_1" });

    expect(screen.getAllByLabelText("Description")).toHaveLength(1);
    expect(screen.getByLabelText("Customer")).toHaveValue("cus_1");
    expect(screen.queryByText(/References/)).toBeNull();
    expect(screen.getByText("Total").nextSibling).toHaveTextContent("$0.00");
  });

  it("refuses a memo with no reason", async () => {
    const user = userEvent.setup();
    renderDialog({ customerId: "cus_1" });

    await user.click(screen.getByRole("button", { name: "Create credit memo" }));

    expect(await screen.findByText("Say why the memo is being raised")).toBeInTheDocument();
    expect(mocks.createMemo).not.toHaveBeenCalled();
  });

  it("creates the memo from the lines typed in and opens it", async () => {
    const user = userEvent.setup();
    const onOpenChange = renderDialog({ customerId: "cus_1" });

    await user.type(screen.getByLabelText("Description"), "Goodwill credit");
    const amount = screen.getByLabelText("Amount");
    await user.clear(amount);
    await user.type(amount, "100");
    await user.type(screen.getByLabelText("Reason"), "Detention waived as goodwill");
    await user.click(screen.getByRole("button", { name: "Create credit memo" }));

    await waitFor(() =>
      expect(mocks.createMemo).toHaveBeenCalledWith({
        customerId: "cus_1",
        billType: "CreditMemo",
        referenceInvoiceId: null,
        reason: "Detention waived as goodwill",
        invoiceDate: null,
        memo: "",
        autoPost: false,
        lines: [
          {
            description: "Goodwill credit",
            amount: "100",
            quantity: "1",
            accessorialChargeId: null,
          },
        ],
      }),
    );
    expect(mocks.navigate).toHaveBeenCalledWith("/billing/invoices?item=inv_cm");
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
