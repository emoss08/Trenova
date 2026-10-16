import type {
  FactoringInvoiceStateRow,
  FactoringSubmission,
  FactoringSubmissionPlan,
} from "@/lib/graphql/factoring";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SubmitToFactorDialog } from "../submit-to-factor-dialog";

const mocks = vi.hoisted(() => ({
  fetchFactoringSubmissionPlan: vi.fn(),
  submitInvoicesToFactor: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/lib/graphql/factoring", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/factoring")>()),
  fetchFactoringSubmissionPlan: mocks.fetchFactoringSubmissionPlan,
  submitInvoicesToFactor: mocks.submitInvoicesToFactor,
}));
vi.mock("sonner", () => ({
  toast: { success: mocks.toastSuccess, error: mocks.toastError, info: vi.fn() },
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const REFUSAL =
  "These invoices cannot be submitted: INV-1 is on schedule FS2610-00003 with the factor";

function row(invoiceId: string, factorId: string | null, factorName = "RTS Financial") {
  return {
    id: invoiceId,
    invoiceId,
    factoringCompanyId: factorId,
    customerId: "cus_1",
    verdict: "Ready",
    failedChecks: 0,
    openBalanceMinor: 100_000,
    readySince: 1_789_000_000,
    evaluatedAt: 1_789_000_000,
    submission: null,
    checks: [],
    invoice: {
      id: invoiceId,
      number: invoiceId.toUpperCase(),
      status: "Posted",
      currencyCode: "USD",
      billToName: "Acme",
    },
    customer: { id: "cus_1", code: "ACME", name: "Acme Foods" },
    factoringCompany: factorId ? { id: factorId, code: factorId, name: factorName } : null,
  } as unknown as FactoringInvoiceStateRow;
}

function plan(overrides: Partial<FactoringSubmissionPlan> = {}): FactoringSubmissionPlan {
  return {
    factoringCompanyId: "fac_rts",
    factoringCompanyName: "RTS Financial",
    agreementId: "facagr_1",
    transportMethod: "Email",
    transportTarget: "schedules@rtsfinancial.com",
    attachPackets: true,
    currencyCode: "USD",
    invoiceCount: 2,
    faceMinor: 250_000,
    expectedAdvanceMinor: 237_500,
    expectedFeeMinor: 6_250,
    expectedReserveMinor: 6_250,
    items: [
      {
        invoiceId: "inv_1",
        invoiceNumber: "INV-1",
        customerId: "cus_1",
        customerName: "Acme Foods",
        invoiceDate: 1_789_000_000,
        dueDate: 1_791_592_000,
        funding: {
          faceMinor: 100_000,
          advanceMinor: 95_000,
          feeMinor: 2_500,
          reserveMinor: 2_500,
          restOnReleaseMinor: 2_500,
          days: 30,
          feeRateBp: 250,
        },
      },
      {
        invoiceId: "inv_2",
        invoiceNumber: "INV-2",
        customerId: "cus_1",
        customerName: "Acme Foods",
        invoiceDate: 1_789_000_000,
        dueDate: null,
        funding: {
          faceMinor: 150_000,
          advanceMinor: 142_500,
          feeMinor: 3_750,
          reserveMinor: 3_750,
          restOnReleaseMinor: 3_750,
          days: 45,
          feeRateBp: 250,
        },
      },
    ],
    ...overrides,
  };
}

function renderDialog(rows: FactoringInvoiceStateRow[], onOpenChange = vi.fn()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/billing/factoring"]}>
        <Routes>
          <Route
            path="/billing/factoring"
            element={<SubmitToFactorDialog open rows={rows} onOpenChange={onOpenChange} />}
          />
          <Route path="/billing/factoring/submissions/:id" element={<p>schedule page</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { onOpenChange };
}

describe("submit to factor dialog", () => {
  it("plans the selected invoices with their factor and shows the funding expected", async () => {
    mocks.fetchFactoringSubmissionPlan.mockResolvedValue(plan());
    renderDialog([row("inv_1", "fac_rts"), row("inv_2", "fac_rts")]);

    expect(await screen.findByText("$2,375.00")).toBeInTheDocument();
    expect(mocks.fetchFactoringSubmissionPlan).toHaveBeenCalledWith(
      { factoringCompanyId: "fac_rts", invoiceIds: ["inv_1", "inv_2"] },
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
    expect(
      screen.getByText(
        "Emailed to RTS Financial at schedules@rtsfinancial.com, with each invoice's packet attached.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("INV-1")).toBeInTheDocument();
    expect(screen.getByText("INV-2")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send 2 invoices" })).toBeEnabled();
  });

  it("plans only the factor most of a mixed selection is with, and says why", async () => {
    mocks.fetchFactoringSubmissionPlan.mockResolvedValue(
      plan({ invoiceCount: 1, items: plan().items.slice(0, 1) }),
    );
    renderDialog([
      row("inv_1", "fac_apex", "Apex Capital"),
      row("inv_2", "fac_rts"),
      row("inv_3", "fac_rts"),
      row("inv_4", null),
    ]);

    expect(
      await screen.findByText(
        "The 4 selected invoices are with 2 factors. A schedule goes to one factor; send the others on their own schedule.",
      ),
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(mocks.fetchFactoringSubmissionPlan).toHaveBeenCalledWith(
        { factoringCompanyId: "fac_rts", invoiceIds: ["inv_2", "inv_3"] },
        expect.anything(),
      ),
    );
  });

  it("refuses a selection with no factor without asking the server", () => {
    renderDialog([row("inv_1", null)]);

    expect(
      screen.getByText("None of the selected invoices is assigned to a factor."),
    ).toBeInTheDocument();
    expect(mocks.fetchFactoringSubmissionPlan).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();
  });

  it("shows the server's reason when the schedule cannot be planned", async () => {
    mocks.fetchFactoringSubmissionPlan.mockRejectedValue(
      new GraphQLRequestError({
        kind: "graphql",
        status: 200,
        message: REFUSAL,
        graphQLErrors: [
          {
            message: REFUSAL,
            extensions: { type: "https://trenova.app/problems/business-rule-violation" },
            code: "BUSINESS_RULE_VIOLATION",
            type: "https://trenova.app/problems/business-rule-violation",
          },
        ],
      }),
    );
    renderDialog([row("inv_1", "fac_rts")]);

    expect(await screen.findByText(REFUSAL)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();
  });

  it("sends the planned invoices and opens the schedule it made", async () => {
    mocks.fetchFactoringSubmissionPlan.mockResolvedValue(plan());
    mocks.submitInvoicesToFactor.mockResolvedValue({
      id: "facsub_1",
      number: "FS2610-00001",
      invoiceCount: 2,
      factoringCompany: { id: "fac_rts", code: "RTS", name: "RTS Financial" },
    } as FactoringSubmission);
    const { onOpenChange } = renderDialog([row("inv_1", "fac_rts"), row("inv_2", "fac_rts")]);

    await userEvent.click(await screen.findByRole("button", { name: "Send 2 invoices" }));

    expect(await screen.findByText("schedule page")).toBeInTheDocument();
    expect(mocks.submitInvoicesToFactor).toHaveBeenCalledWith({
      factoringCompanyId: "fac_rts",
      invoiceIds: ["inv_1", "inv_2"],
    });
    expect(mocks.toastSuccess).not.toHaveBeenCalled();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
