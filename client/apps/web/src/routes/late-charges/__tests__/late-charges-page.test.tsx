import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { LateChargesPage } from "../page";

const mocks = vi.hoisted(() => ({
  preview: vi.fn(),
  assess: vi.fn(),
  billingControl: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/lib/graphql/accounts-receivable", () => ({
  fetchLateChargePreview: mocks.preview,
  assessLateCharges: mocks.assess,
}));
vi.mock("@/services/api", () => ({
  apiService: { billingControlService: { get: mocks.billingControl } },
}));
vi.mock("sonner", () => ({ toast: { success: mocks.toastSuccess, error: mocks.toastError } }));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));
vi.mock("@/components/autocomplete-fields", () => ({
  CustomerAutocompleteField: () => <input aria-label="Customer" />,
}));
vi.mock("@/components/navigation/sidebar-layout", () => ({
  PageLayout: ({
    children,
    pageHeaderProps,
  }: {
    children: React.ReactNode;
    pageHeaderProps: { title: string; actions?: React.ReactNode };
  }) => (
    <div>
      <h1>{pageHeaderProps.title}</h1>
      {pageHeaderProps.actions}
      {children}
    </div>
  ),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const NOW = 1_789_325_147;

const preview = {
  asOfDate: NOW,
  preview: true,
  mode: "Preview",
  memosCreated: 0,
  memosPosted: 0,
  customersSkipped: 1,
  totalChargeMinor: 1800,
  customers: [
    {
      customerId: "cus_amd",
      customerName: "AMD",
      currencyCode: "USD",
      totalChargeMinor: 1800,
      debitMemoId: null,
      debitMemoNumber: "",
      posted: false,
      skipped: false,
      skipReason: "",
      lines: [
        {
          invoiceId: "inv_1",
          invoiceNumber: "INV-1",
          periodIndex: 2,
          periodStart: NOW - 30 * 86400,
          periodEnd: NOW - 1,
          basisOpenBalanceMinor: 120_000,
          ratePercent: "1.5",
          chargeMinor: 1800,
        },
      ],
    },
    {
      customerId: "cus_intel",
      customerName: "Intel",
      currencyCode: "USD",
      totalChargeMinor: 40,
      debitMemoId: null,
      debitMemoNumber: "",
      posted: false,
      skipped: true,
      skipReason: "Total late charge 0.40 is below the organization minimum 5.00",
      lines: [],
    },
  ],
};

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <LateChargesPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  mocks.preview.mockResolvedValue(preview);
  mocks.billingControl.mockResolvedValue({ lateChargeAssessmentMode: "Preview" });
  mocks.assess.mockResolvedValue({
    ...preview,
    preview: false,
    memosCreated: 1,
    customers: [
      { ...preview.customers[0], debitMemoId: "inv_dm", debitMemoNumber: "DM-1", posted: false },
      preview.customers[1],
    ],
  });
});

describe("LateChargesPage", () => {
  it("previews what the run would raise, customer by customer, and says who is skipped", async () => {
    renderPage();

    expect(await screen.findByText("AMD")).toBeInTheDocument();
    expect(screen.getAllByText("$18.00").length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText("INV-1")).toBeInTheDocument();
    expect(
      screen.getByText("Total late charge 0.40 is below the organization minimum 5.00"),
    ).toBeInTheDocument();
  });

  it("explains that the nightly run only previews while the mode is Preview", async () => {
    renderPage();

    expect(await screen.findByText(/nightly run only previews/)).toBeInTheDocument();
  });

  it("assesses the selected customers after confirming", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("checkbox", { name: "Select AMD" }));
    await user.click(screen.getByRole("button", { name: "Assess now" }));
    await user.click(await screen.findByRole("button", { name: "Assess late charges" }));

    await waitFor(() =>
      expect(mocks.assess).toHaveBeenCalledWith({ customerIds: ["cus_amd"], asOfDate: null }),
    );
    expect(await screen.findByText("DM-1")).toBeInTheDocument();
  });

  it("keeps the run button off until something is selected", async () => {
    renderPage();

    expect(await screen.findByRole("button", { name: "Assess now" })).toBeDisabled();
  });
});
