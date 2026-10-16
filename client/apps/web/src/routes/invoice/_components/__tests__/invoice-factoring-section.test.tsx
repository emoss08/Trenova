import type { FactoringInvoiceState, InvoiceFactoring } from "@/lib/graphql/factoring";
import type { Invoice } from "@trenova/shared/types/invoice";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { InvoiceFactoringSection } from "../invoice-factoring-section";

const mocks = vi.hoisted(() => ({
  fetchInvoiceFactoring: vi.fn(),
  setInvoiceFactoringOverride: vi.fn(),
  requestMissingFactoringDocuments: vi.fn(),
  listFactoringReserveEntries: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/lib/graphql/factoring", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/factoring")>()),
  fetchInvoiceFactoring: mocks.fetchInvoiceFactoring,
  setInvoiceFactoringOverride: mocks.setInvoiceFactoringOverride,
  requestMissingFactoringDocuments: mocks.requestMissingFactoringDocuments,
  listFactoringReserveEntries: mocks.listFactoringReserveEntries,
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { success: mocks.toastSuccess, error: mocks.toastError, info: vi.fn() },
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

beforeEach(() => {
  mocks.listFactoringReserveEntries.mockResolvedValue({ entries: [], totalCount: 0 });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const NOW = 1_789_325_147;

const invoice = {
  id: "inv_1",
  number: "INV-1",
  status: "Posted",
  billType: "Invoice",
  customerId: "cus_1",
  currencyCode: "USD",
  factoringCompanyId: "fac_1",
  factoringOverride: "Default",
} as Invoice;

function state(overrides: Partial<FactoringInvoiceState> = {}): FactoringInvoiceState {
  return {
    invoiceId: "inv_1",
    factoringCompanyId: "fac_1",
    customerId: "cus_1",
    verdict: "NotReady",
    failedChecks: 2,
    openBalanceMinor: 125_000,
    readySince: null,
    evaluatedAt: NOW,
    submission: null,
    checks: [
      {
        code: "Eligible",
        passed: true,
        message: "Posted with an open balance",
        fix: null,
        documentTypeId: null,
        documentTypeName: null,
        shipmentId: null,
        shipmentPro: null,
        detail: null,
      },
      {
        code: "Dispute",
        passed: false,
        message: "The customer is disputing the invoice",
        fix: "ResolveDispute",
        documentTypeId: null,
        documentTypeName: null,
        shipmentId: null,
        shipmentPro: null,
        detail: null,
      },
      {
        code: "AgreementDocument",
        passed: false,
        message: "Proof of delivery is missing",
        fix: "RequestDocument",
        documentTypeId: "dt_pod",
        documentTypeName: "Proof of delivery",
        shipmentId: "shp_1",
        shipmentPro: "PRO-1",
        detail: null,
      },
    ],
    invoice: {
      id: "inv_1",
      number: "INV-1",
      status: "Posted",
      currencyCode: "USD",
      billToName: "Acme",
    },
    customer: { id: "cus_1", code: "ACME", name: "Acme" },
    factoringCompany: { id: "fac_1", code: "RTS", name: "RTS Financial" },
    ...overrides,
  };
}

function factoring(overrides: Partial<InvoiceFactoring> = {}): InvoiceFactoring {
  return {
    id: "inv_1",
    number: "INV-1",
    status: "Posted",
    billType: "Invoice",
    currencyCode: "USD",
    factoringOverride: "Default",
    factoringOverrideReason: null,
    factoringAgreementId: "facagr_1",
    factoringCompany: { id: "fac_1", code: "RTS", name: "RTS Financial" },
    factoringState: state(),
    ...overrides,
  };
}

async function loadedSection() {
  await screen.findByText("Agreement");
  return screen.getByRole("region", { name: "Factoring" });
}

function renderSection(value: Invoice = invoice) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <InvoiceFactoringSection invoice={value} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("invoice factoring section", () => {
  it("stays out of the way of an invoice that was never assigned to a factor", () => {
    renderSection({ ...invoice, factoringCompanyId: null } as Invoice);

    expect(screen.queryByRole("region", { name: "Factoring" })).toBeNull();
    expect(mocks.fetchInvoiceFactoring).not.toHaveBeenCalled();
  });

  it("names the factor and agreement and lists only the failed checks with their fixes", async () => {
    mocks.fetchInvoiceFactoring.mockResolvedValue(factoring());
    renderSection();

    const section = await loadedSection();
    expect(within(section).getByText("Not ready")).toBeInTheDocument();
    expect(within(section).getByRole("link", { name: /RTS Financial/ })).toHaveAttribute(
      "href",
      "/billing/configuration-files/factoring-companies?panelType=edit&panelEntityId=fac_1",
    );
    expect(within(section).getByRole("link", { name: /View agreement/ })).toHaveAttribute(
      "href",
      "/billing/configuration-files/factoring-companies?section=agreements&agreement=facagr_1",
    );
    expect(within(section).getByText("The customer is disputing the invoice")).toBeInTheDocument();
    expect(within(section).queryByText("Posted with an open balance")).toBeNull();
    expect(within(section).getByRole("link", { name: /Open the disputes/ })).toHaveAttribute(
      "href",
      "/billing/invoices?item=inv_1&tab=disputes",
    );
    expect(within(section).getByRole("button", { name: /Request documents/ })).toBeInTheDocument();
  });

  it("links the schedule an invoice is on while it is with the factor", async () => {
    mocks.fetchInvoiceFactoring.mockResolvedValue(
      factoring({
        factoringState: state({
          verdict: "Submitted",
          failedChecks: 0,
          submission: {
            submissionId: "facsub_1",
            submissionNumber: "FS2610-00007",
            submissionStatus: "Failed",
            itemId: "facsit_1",
            itemStatus: "Pending",
          },
        }),
      }),
    );
    renderSection();

    const section = await loadedSection();
    expect(within(section).getByText("Submitted")).toBeInTheDocument();
    expect(within(section).getByRole("link", { name: "FS2610-00007" })).toHaveAttribute(
      "href",
      "/billing/factoring/submissions/facsub_1",
    );
    expect(within(section).getByText("Failed to send")).toBeInTheDocument();
  });

  it("shows where a funded invoice stands with the factor and every movement booked for it", async () => {
    mocks.fetchInvoiceFactoring.mockResolvedValue(
      factoring({
        factoringState: state({
          verdict: "Funded",
          failedChecks: 0,
          submission: {
            submissionId: "facsub_1",
            submissionNumber: "FS2610-00007",
            submissionStatus: "PartiallyFunded",
            itemId: "facsit_1",
            itemStatus: "ChargedBack",
          },
        }),
      }),
    );
    const entry = {
      factoringCompanyId: "fac_1",
      agreementId: "facagr_1",
      submissionId: "facsub_1",
      submissionItemId: "facsit_1",
      invoiceId: "inv_1",
      customerId: "cus_1",
      treatment: "Sale" as const,
      currencyCode: "USD",
      liabilityMinor: 0,
      reversesEntryId: null,
      createdAt: NOW,
      invoice: { id: "inv_1", number: "INV-1" },
      customer: null,
      submission: { id: "facsub_1", number: "FS2610-00007" },
    };
    mocks.listFactoringReserveEntries.mockResolvedValue({
      entries: [
        {
          ...entry,
          id: "facres_1",
          kind: "Funding",
          cashMinor: 112_500,
          feeMinor: 3_750,
          reserveMinor: 8_750,
          arMinor: -125_000,
          journalEntryId: "je_1",
          accountingDate: NOW,
          occurredOn: NOW,
          memo: "Remittance 4471",
        },
        {
          ...entry,
          id: "facres_2",
          kind: "Chargeback",
          cashMinor: -116_250,
          feeMinor: 0,
          reserveMinor: -8_750,
          arMinor: 125_000,
          journalEntryId: "je_2",
          accountingDate: NOW + 86_400,
          occurredOn: NOW + 86_400,
          memo: "Debtor did not pay",
        },
      ],
      totalCount: 2,
    });
    renderSection();

    const section = await loadedSection();
    expect(within(section).getByText("Charged back")).toBeInTheDocument();
    expect(mocks.listFactoringReserveEntries).toHaveBeenCalledWith(
      { invoiceId: "inv_1" },
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
    const funding = await within(section).findByRole("row", { name: /Funding/ });
    expect(within(funding).getByText("$1,125.00")).toBeInTheDocument();
    expect(within(funding).getByText("Remittance 4471")).toBeInTheDocument();
    const chargeback = within(section).getByRole("row", { name: /Chargeback/ });
    expect(within(chargeback).getByText("-$1,162.50")).toBeInTheDocument();
    expect(within(chargeback).getByRole("link", { name: "View" })).toHaveAttribute(
      "href",
      expect.stringContaining("je_2"),
    );
  });

  it("asks for the one missing document the check names", async () => {
    mocks.fetchInvoiceFactoring.mockResolvedValue(factoring());
    mocks.requestMissingFactoringDocuments.mockResolvedValue({
      recipient: "driver@example.com",
      recipientKind: "Driver",
      documentTypes: ["Proof of delivery"],
      shipmentIds: ["shp_1"],
      sent: true,
    });
    const user = userEvent.setup();
    renderSection();

    await loadedSection();
    await user.click(screen.getByRole("button", { name: /Request documents/ }));
    await user.click(screen.getByRole("button", { name: "Send request" }));

    await waitFor(() =>
      expect(mocks.requestMissingFactoringDocuments).toHaveBeenCalledWith({
        invoiceId: "inv_1",
        documentTypeIds: ["dt_pod"],
        recipient: null,
        message: null,
      }),
    );
  });

  it("refuses to keep an invoice out of factoring without a reason, then sends the reason", async () => {
    mocks.fetchInvoiceFactoring.mockResolvedValue(factoring());
    mocks.setInvoiceFactoringOverride.mockResolvedValue(state({ verdict: "Excluded" }));
    const user = userEvent.setup();
    renderSection();

    const section = await loadedSection();
    await user.click(within(section).getByRole("button", { name: /Follow the customer/ }));
    await user.click(await screen.findByText("Keep out of factoring"));
    await user.click(within(section).getByRole("button", { name: "Save" }));

    expect(await screen.findByText("Say why this invoice stays out of factoring")).toBeVisible();
    expect(mocks.setInvoiceFactoringOverride).not.toHaveBeenCalled();

    await user.type(within(section).getByLabelText(/Reason/), "Customer pays us directly");
    await user.click(within(section).getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(mocks.setInvoiceFactoringOverride).toHaveBeenCalledWith({
        invoiceId: "inv_1",
        override: "Exclude",
        reason: "Customer pays us directly",
      }),
    );
  });

  it("shows an invoice kept out of factoring so it can be returned to its customer's factor", async () => {
    mocks.fetchInvoiceFactoring.mockResolvedValue(
      factoring({
        factoringOverride: "Exclude",
        factoringOverrideReason: "Customer pays us directly",
        factoringCompany: null,
        factoringAgreementId: null,
        factoringState: state({ verdict: "Excluded", failedChecks: 0, factoringCompany: null }),
      }),
    );
    renderSection({
      ...invoice,
      factoringCompanyId: null,
      factoringOverride: "Exclude",
    } as Invoice);

    const section = await loadedSection();
    expect(within(section).getByText("Not factored")).toBeInTheDocument();
    expect(within(section).queryByText("The customer is disputing the invoice")).toBeNull();
    expect(within(section).getByText("Why it stays out")).toBeInTheDocument();
    expect(within(section).getByText("Customer pays us directly")).toBeInTheDocument();
  });
});
