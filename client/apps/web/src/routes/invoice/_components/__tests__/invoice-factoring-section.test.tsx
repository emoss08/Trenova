import type { FactoringInvoiceState, InvoiceFactoring } from "@/lib/graphql/factoring";
import type { Invoice } from "@trenova/shared/types/invoice";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { InvoiceFactoringSection } from "../invoice-factoring-section";

const mocks = vi.hoisted(() => ({
  fetchInvoiceFactoring: vi.fn(),
  setInvoiceFactoringOverride: vi.fn(),
  requestMissingFactoringDocuments: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/lib/graphql/factoring", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/factoring")>()),
  fetchInvoiceFactoring: mocks.fetchInvoiceFactoring,
  setInvoiceFactoringOverride: mocks.setInvoiceFactoringOverride,
  requestMissingFactoringDocuments: mocks.requestMissingFactoringDocuments,
}));
vi.mock("sonner", () => ({
  toast: { success: mocks.toastSuccess, error: mocks.toastError, info: vi.fn() },
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

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
