import type { InvoiceArContext } from "@/lib/graphql/invoice";
import type { Invoice } from "@trenova/shared/types/invoice";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { InvoiceActionsMenu } from "../invoice-actions-menu";

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
  usePermissions: () => ({ canRead: true, canCreate: true, canUpdate: true, isLoading: false }),
}));
vi.mock("../invoice-void-dialog", () => ({
  InvoiceVoidDialog: ({ open }: { open: boolean }) => (open ? <p>Void dialog open</p> : null),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const invoice = {
  id: "inv_1",
  number: "INV-1",
  status: "Posted",
  billType: "Invoice",
  customerId: "cus_1",
  appliedAmount: "0",
  lines: [],
} as unknown as Invoice;

function arContext(overrides: Partial<InvoiceArContext> = {}): InvoiceArContext {
  return {
    id: "inv_1",
    paymentApplications: [],
    creditApplications: [],
    disputes: [],
    openDispute: null,
    lateChargeAssessments: [],
    relatedInvoices: [],
    referenceInvoice: null,
    ediSendPlan: {
      invoiceId: "inv_1",
      enabled: true,
      autoSend: false,
      partnerId: "edip_1",
      partnerName: "AMD EDI",
      documentProfileId: "edidp_1",
      communicationMethod: "AS2",
      status: "NotSent",
      lastMessageId: null,
      lastError: "",
      sentAt: null,
      blockers: [],
    },
    ...overrides,
  } as InvoiceArContext;
}

function renderMenu(context: InvoiceArContext | null) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <InvoiceActionsMenu invoice={invoice} arContext={context} />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("InvoiceActionsMenu", () => {
  it("opens the void dialog for an invoice nothing has settled against", async () => {
    const user = userEvent.setup();
    renderMenu(arContext());

    await user.click(screen.getByRole("button", { name: "Invoice actions" }));
    await user.click(await screen.findByRole("menuitem", { name: /Void invoice/ }));

    expect(screen.getByText("Void dialog open")).toBeInTheDocument();
  });

  it("disables voiding while a payment or credit is applied, and says why", async () => {
    const user = userEvent.setup();
    renderMenu(
      arContext({
        paymentApplications: [
          {
            id: "cpa_1",
            customerPaymentId: "cp_1",
            invoiceId: "inv_1",
            appliedAmountMinor: 5000,
            shortPayAmountMinor: 0,
            lineNumber: 1,
            createdAt: 1,
            payment: null,
          },
        ],
      }),
    );

    await user.click(screen.getByRole("button", { name: "Invoice actions" }));

    const item = await screen.findByRole("menuitem", { name: /Void invoice/ });
    expect(item).toHaveAttribute("aria-disabled", "true");
    expect(
      screen.getByText(
        "Unapply the customer payments and credit memos on this invoice before voiding it.",
      ),
    ).toBeInTheDocument();
  });

  it("offers the EDI send when the plan allows it and never a memo, which belongs to Adjust Invoice", async () => {
    const user = userEvent.setup();
    renderMenu(arContext());

    await user.click(screen.getByRole("button", { name: "Invoice actions" }));
    expect(await screen.findByRole("menuitem", { name: /Send EDI/ })).not.toHaveAttribute(
      "aria-disabled",
      "true",
    );

    expect(screen.queryByRole("menuitem", { name: /memo/i })).toBeNull();
    expect(screen.queryByText("Memos")).toBeNull();
  });

  it("disables the EDI send and names the blocker when the customer takes no EDI", async () => {
    const user = userEvent.setup();
    renderMenu(
      arContext({
        ediSendPlan: {
          invoiceId: "inv_1",
          enabled: false,
          autoSend: false,
          partnerId: null,
          partnerName: "",
          documentProfileId: null,
          communicationMethod: "",
          status: "NotConfigured",
          lastMessageId: null,
          lastError: "",
          sentAt: null,
          blockers: ["EDI invoicing is not enabled on the customer's billing profile"],
        },
      }),
    );

    await user.click(screen.getByRole("button", { name: "Invoice actions" }));

    expect(await screen.findByRole("menuitem", { name: /Send EDI/ })).toHaveAttribute(
      "aria-disabled",
      "true",
    );
  });
});
