import type {
  FactoringFundingPlan,
  FactoringPlannedMovement,
  FactoringReserveMovementPlan,
  FactoringSubmissionDetail,
} from "@/lib/graphql/factoring";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { factoring } from "@/lib/queries/factoring";
import { FactoringSubmissionPage, prefetch } from "../page";

const mocks = vi.hoisted(() => ({
  fetchFactoringSubmission: vi.fn(),
  retryFactoringSubmission: vi.fn(),
  cancelFactoringSubmission: vi.fn(),
  listFactoringAgreements: vi.fn(),
  fetchFactoringFundingPlan: vi.fn(),
  recordFactoringFunding: vi.fn(),
  fetchFactoringReserveMovementPlan: vi.fn(),
  bookFactoringReserveMovement: vi.fn(),
  granted: new Set<number>(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/lib/graphql/factoring", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/factoring")>()),
  fetchFactoringSubmission: mocks.fetchFactoringSubmission,
  retryFactoringSubmission: mocks.retryFactoringSubmission,
  cancelFactoringSubmission: mocks.cancelFactoringSubmission,
  listFactoringAgreements: mocks.listFactoringAgreements,
  fetchFactoringFundingPlan: mocks.fetchFactoringFundingPlan,
  recordFactoringFunding: mocks.recordFactoringFunding,
  fetchFactoringReserveMovementPlan: mocks.fetchFactoringReserveMovementPlan,
  bookFactoringReserveMovement: mocks.bookFactoringReserveMovement,
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: (_resource: string, operation: number) => ({
    allowed: mocks.granted.has(operation),
    isLoading: false,
  }),
}));
vi.mock("@/hooks/use-breadcrumb-label", () => ({ useBreadcrumbLabel: vi.fn() }));
vi.mock("@/components/navigation/sidebar-layout", () => ({
  PageLayout: ({
    pageHeaderProps,
    children,
  }: {
    pageHeaderProps: { title: string; description?: string; actions?: React.ReactNode };
    children: React.ReactNode;
  }) => (
    <main>
      <h1>{pageHeaderProps.title}</h1>
      <p>{pageHeaderProps.description}</p>
      <div>{pageHeaderProps.actions}</div>
      {children}
    </main>
  ),
}));
vi.mock("sonner", () => ({
  toast: { success: mocks.toastSuccess, error: mocks.toastError, info: vi.fn() },
}));

const CREATE = 1 << 1;
const UPDATE = 1 << 2;
const CANCEL = 1 << 15;

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  mocks.granted.clear();
});

function submission(overrides: Partial<FactoringSubmissionDetail> = {}): FactoringSubmissionDetail {
  return {
    id: "facsub_1",
    number: "FS2610-00007",
    factoringCompanyId: "fac_rts",
    agreementId: "facagr_1",
    status: "Failed",
    currencyCode: "USD",
    invoiceCount: 2,
    faceMinor: 250_000,
    expectedAdvanceMinor: 237_500,
    expectedFeeMinor: 6_250,
    expectedReserveMinor: 6_250,
    schedulePdfDocumentId: "doc_pdf",
    scheduleCsvDocumentId: null,
    transportMethod: "Sftp",
    transportTarget: "trenova@sftp.rtsfinancial.com:/incoming",
    transportReference: null,
    sendAttemptGroup: 1,
    attempts: 6,
    lastError: "ssh: handshake failed: host key mismatch",
    sentAt: null,
    failedAt: 1_789_000_600,
    cancelledAt: null,
    cancelReason: null,
    submittedById: "usr_1",
    version: 4,
    createdAt: 1_789_000_000,
    updatedAt: 1_789_000_600,
    factoringCompany: { id: "fac_rts", code: "RTS", name: "RTS Financial" },
    items: [
      {
        id: "facsit_1",
        submissionId: "facsub_1",
        invoiceId: "inv_1",
        invoiceNumber: "INV-1",
        customerId: "cus_1",
        position: 1,
        faceMinor: 100_000,
        expectedAdvanceMinor: 95_000,
        expectedFeeMinor: 2_500,
        expectedReserveMinor: 2_500,
        expectedDays: 30,
        packetDocumentId: "doc_packet_1",
        status: "Pending",
        releasedAt: null,
        customer: { id: "cus_1", code: "ACME", name: "Acme Foods" },
      },
      {
        id: "facsit_2",
        submissionId: "facsub_1",
        invoiceId: "inv_2",
        invoiceNumber: "INV-2",
        customerId: "cus_1",
        position: 2,
        faceMinor: 150_000,
        expectedAdvanceMinor: 142_500,
        expectedFeeMinor: 3_750,
        expectedReserveMinor: 3_750,
        expectedDays: 45,
        packetDocumentId: null,
        status: "Pending",
        releasedAt: null,
        customer: { id: "cus_1", code: "ACME", name: "Acme Foods" },
      },
    ],
    ...overrides,
  } as FactoringSubmissionDetail;
}

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/billing/factoring/submissions/facsub_1"]}>
        <Routes>
          <Route path="/billing/factoring/submissions/:id" element={<FactoringSubmissionPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("prefetch", () => {
  it("warms the schedule the route names and nothing without an id", () => {
    const args = (id?: string) =>
      ({
        params: id ? { id } : {},
        request: new Request("http://localhost/billing/factoring/submissions"),
        context: undefined,
      }) as unknown as Parameters<typeof prefetch>[0];

    expect(prefetch(args("facsub_1"))).toEqual([
      expect.objectContaining({ queryKey: factoring.submission("facsub_1").queryKey }),
    ]);
    expect(prefetch(args())).toEqual([]);
  });
});

describe("factoring schedule page", () => {
  it("shows why a failed schedule did not go out, with its invoices and what each is worth", async () => {
    mocks.fetchFactoringSubmission.mockResolvedValue(submission());
    renderPage();

    expect(
      await screen.findByRole("heading", { name: "Schedule FS2610-00007" }),
    ).toBeInTheDocument();
    expect(mocks.fetchFactoringSubmission).toHaveBeenCalledWith(
      "facsub_1",
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
    expect(
      screen.getByText("ssh: handshake failed: host key mismatch", { exact: false }),
    ).toBeInTheDocument();
    expect(screen.getByText("trenova@sftp.rtsfinancial.com:/incoming")).toBeInTheDocument();
    const invoices = screen.getByRole("region", { name: "Invoices" });
    expect(within(invoices).getByRole("link", { name: "INV-1" })).toHaveAttribute(
      "href",
      "/billing/invoices?item=inv_1",
    );
    expect(within(invoices).getByRole("link", { name: "Download" })).toHaveAttribute(
      "href",
      expect.stringContaining("/documents/doc_packet_1/download/"),
    );
    expect(within(invoices).getByText("Not built yet")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Schedule of accounts (PDF)" })).toHaveAttribute(
      "href",
      expect.stringContaining("/documents/doc_pdf/view/"),
    );
  });

  it("offers no retry or cancel to someone who may not change schedules", async () => {
    mocks.fetchFactoringSubmission.mockResolvedValue(submission());
    renderPage();

    await screen.findByRole("heading", { name: "Schedule FS2610-00007" });
    expect(screen.queryByRole("button", { name: "Retry" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Cancel schedule" })).toBeNull();
  });

  it("offers no retry or cancel on a schedule that was delivered", async () => {
    mocks.granted.add(UPDATE).add(CANCEL);
    mocks.fetchFactoringSubmission.mockResolvedValue(
      submission({ status: "Sent", sentAt: 1_789_000_100, failedAt: null, lastError: null }),
    );
    renderPage();

    await screen.findByRole("heading", { name: "Schedule FS2610-00007" });
    expect(screen.queryByRole("button", { name: "Retry" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Cancel schedule" })).toBeNull();
  });

  it("retries a failed schedule at the version it was read at", async () => {
    mocks.granted.add(UPDATE);
    mocks.fetchFactoringSubmission.mockResolvedValue(submission());
    mocks.retryFactoringSubmission.mockResolvedValue(
      submission({ status: "Sending", sendAttemptGroup: 2, version: 5 }),
    );
    renderPage();

    await userEvent.click(await screen.findByRole("button", { name: "Retry" }));

    await waitFor(() =>
      expect(mocks.retryFactoringSubmission).toHaveBeenCalledWith({ id: "facsub_1", version: 4 }),
    );
    expect(mocks.toastSuccess).toHaveBeenCalledWith("Schedule FS2610-00007 is being sent again", {
      description: "To trenova@sftp.rtsfinancial.com:/incoming",
    });
  });

  it("will not cancel a schedule without a reason, then sends the reason it was given", async () => {
    mocks.granted.add(CANCEL);
    mocks.fetchFactoringSubmission.mockResolvedValue(submission());
    mocks.cancelFactoringSubmission.mockResolvedValue(
      submission({ status: "Cancelled", cancelReason: "Factor closed the account", version: 5 }),
    );
    renderPage();

    await userEvent.click(await screen.findByRole("button", { name: "Cancel schedule" }));
    const dialog = await screen.findByRole("dialog", {}, { timeout: 5000 });
    await userEvent.click(within(dialog).getByRole("button", { name: "Cancel schedule" }));
    expect(
      await within(dialog).findByText("Say why the schedule is cancelled"),
    ).toBeInTheDocument();
    expect(mocks.cancelFactoringSubmission).not.toHaveBeenCalled();

    await userEvent.type(within(dialog).getByRole("textbox"), "  Factor closed the account ");
    await userEvent.click(within(dialog).getByRole("button", { name: "Cancel schedule" }));

    await waitFor(() =>
      expect(mocks.cancelFactoringSubmission).toHaveBeenCalledWith({
        id: "facsub_1",
        version: 4,
        reason: "Factor closed the account",
      }),
    );
    expect(mocks.toastSuccess).toHaveBeenCalledWith("Schedule FS2610-00007 cancelled", {
      description: "2 invoices are back in the queue.",
    });
  });
});

const FUNDED_ON = 1_789_600_000;

function movement(overrides: Partial<FactoringPlannedMovement> = {}): FactoringPlannedMovement {
  return {
    itemId: "facsit_1",
    invoiceId: "inv_1",
    invoiceNumber: "INV-1",
    customerId: "cus_1",
    movement: {
      kind: "Funding",
      treatment: "Sale",
      cashMinor: 95_000,
      feeMinor: 2_500,
      reserveMinor: 2_500,
      liabilityMinor: 0,
      arMinor: -100_000,
      invoiceEffect: "Settle",
      invoiceMinor: 100_000,
    },
    lines: [
      {
        role: "Cash",
        accountId: "gla_cash",
        account: { id: "gla_cash", accountCode: "1000", name: "Operating cash" },
        debit: 95_000,
        credit: 0,
      },
      {
        role: "FeeExpense",
        accountId: "gla_fee",
        account: { id: "gla_fee", accountCode: "6400", name: "Factoring fees" },
        debit: 2_500,
        credit: 0,
      },
      {
        role: "DueFromFactor",
        accountId: "gla_due",
        account: { id: "gla_due", accountCode: "1250", name: "Due from factor" },
        debit: 2_500,
        credit: 0,
      },
      {
        role: "AccountsReceivable",
        accountId: "gla_ar",
        account: null,
        debit: 0,
        credit: 100_000,
      },
    ],
    requestedDate: FUNDED_ON,
    accountingDate: FUNDED_ON,
    fiscalPeriodId: "fp_10",
    fiscalPeriodName: "October 2026",
    requiresApproval: false,
    ...overrides,
  };
}

function fundedItems(): FactoringSubmissionDetail["items"] {
  return submission().items.map((item, index) =>
    index === 0
      ? {
          ...item,
          status: "Funded",
          fundedOn: FUNDED_ON,
          advanceMinor: 94_000,
          feeMinor: 3_000,
          reserveMinor: 3_000,
        }
      : { ...item, status: "Rejected", rejectionReason: "Customer over its credit limit" },
  );
}

describe("factoring schedule funding", () => {
  it("records a factor's report: one invoice funded at what it paid, one rejected with why", async () => {
    mocks.granted.add(CREATE);
    mocks.fetchFactoringSubmission.mockResolvedValue(
      submission({ status: "Sent", sentAt: 1_789_000_100, failedAt: null, lastError: null }),
    );
    mocks.listFactoringAgreements.mockResolvedValue([{ id: "facagr_1", recourse: "Recourse" }]);
    mocks.fetchFactoringFundingPlan.mockResolvedValue({
      submissionId: "facsub_1",
      status: "Funded",
      fundedCount: 1,
      rejectCount: 1,
      advanceMinor: 95_000,
      feeMinor: 2_500,
      reserveMinor: 2_500,
      movements: [movement()],
    } satisfies FactoringFundingPlan);
    mocks.recordFactoringFunding.mockResolvedValue(
      submission({ status: "Funded", version: 5, items: fundedItems() }),
    );
    renderPage();

    await userEvent.click(await screen.findByRole("button", { name: "Record funding" }));
    const dialog = await screen.findByRole("dialog", {}, { timeout: 5000 });
    const outcomes = within(dialog).getAllByRole("radiogroup", { name: "Outcome" });
    expect(outcomes).toHaveLength(2);
    await userEvent.click(within(outcomes[1]).getByRole("radio", { name: "Rejected" }));
    await userEvent.type(
      within(dialog).getByRole("textbox", { name: "Why the factor rejected it" }),
      " Customer over its credit limit ",
    );

    const fundingInput = {
      submissionId: "facsub_1",
      version: 4,
      fundedOn: expect.any(Number),
      items: [
        { itemId: "facsit_1", outcome: "Funded", advanceMinor: 95_000, feeMinor: 2_500 },
        { itemId: "facsit_2", outcome: "Rejected", reason: "Customer over its credit limit" },
      ],
      memo: null,
    };
    await waitFor(
      () =>
        expect(mocks.fetchFactoringFundingPlan).toHaveBeenLastCalledWith(
          fundingInput,
          expect.objectContaining({ signal: expect.any(AbortSignal) }),
        ),
      { timeout: 2_000 },
    );
    expect(await within(dialog).findByText("1250 · Due from factor")).toBeInTheDocument();
    expect(within(dialog).getByText("gla_ar")).toBeInTheDocument();
    expect(within(dialog).getByText("Books on Sep 16, 2026.")).toBeInTheDocument();
    expect(within(dialog).getByText("Period October 2026.")).toBeInTheDocument();

    await waitFor(() =>
      expect(within(dialog).getByRole("button", { name: "Record funding" })).toBeEnabled(),
    );
    await userEvent.click(within(dialog).getByRole("button", { name: "Record funding" }));

    await waitFor(() => expect(mocks.recordFactoringFunding).toHaveBeenCalledWith(fundingInput));
    expect(mocks.toastSuccess).toHaveBeenCalledWith("Funding recorded on schedule FS2610-00007", {
      description: "Funded",
    });
  });

  it("offers no funding to someone who may not book factoring, nor on a failed schedule", async () => {
    mocks.fetchFactoringSubmission.mockResolvedValue(
      submission({ status: "Sent", sentAt: 1_789_000_100, failedAt: null, lastError: null }),
    );
    renderPage();
    await screen.findByRole("heading", { name: "Schedule FS2610-00007" });
    expect(screen.queryByRole("button", { name: "Record funding" })).toBeNull();
    cleanup();

    mocks.granted.add(CREATE);
    mocks.listFactoringAgreements.mockResolvedValue([]);
    mocks.fetchFactoringSubmission.mockResolvedValue(submission());
    renderPage();
    await screen.findByRole("heading", { name: "Schedule FS2610-00007" });
    expect(screen.queryByRole("button", { name: "Record funding" })).toBeNull();
  });

  it("shows what the factor actually paid and why it rejected an invoice", async () => {
    mocks.fetchFactoringSubmission.mockResolvedValue(
      submission({ status: "Funded", items: fundedItems() }),
    );
    renderPage();

    const invoices = await screen.findByRole("region", { name: "Invoices" });
    const funded = within(invoices).getByRole("row", { name: /INV-1/ });
    expect(within(funded).getByText("$940.00")).toBeInTheDocument();
    expect(within(funded).getAllByText("$30.00")).toHaveLength(2);
    expect(within(funded).queryByText("$950.00")).toBeNull();
    expect(within(funded).getByText("Funded")).toBeInTheDocument();
    const rejected = within(invoices).getByRole("row", { name: /INV-2/ });
    expect(within(rejected).getByText("Customer over its credit limit")).toBeInTheDocument();
  });

  it("releases a funded invoice's reserve at the factor's default, after previewing the journal", async () => {
    mocks.granted.add(CREATE);
    mocks.fetchFactoringSubmission.mockResolvedValue(
      submission({ status: "Funded", items: fundedItems() }),
    );
    mocks.listFactoringAgreements.mockResolvedValue([{ id: "facagr_1", recourse: "Recourse" }]);
    mocks.fetchFactoringReserveMovementPlan.mockResolvedValue({
      submissionId: "facsub_1",
      itemId: "facsit_1",
      itemStatus: "Settled",
      movement: movement({
        movement: {
          kind: "ReserveRelease",
          treatment: "Sale",
          cashMinor: 3_000,
          feeMinor: 0,
          reserveMinor: -3_000,
          liabilityMinor: 0,
          arMinor: 0,
          invoiceEffect: "None",
          invoiceMinor: 0,
        },
        lines: [
          {
            role: "Cash",
            accountId: "gla_cash",
            account: { id: "gla_cash", accountCode: "1000", name: "Operating cash" },
            debit: 3_000,
            credit: 0,
          },
          {
            role: "DueFromFactor",
            accountId: "gla_due",
            account: { id: "gla_due", accountCode: "1250", name: "Due from factor" },
            debit: 0,
            credit: 3_000,
          },
        ],
        accountingDate: FUNDED_ON + 86_400,
      }),
      reverses: null,
    } satisfies FactoringReserveMovementPlan);
    mocks.bookFactoringReserveMovement.mockResolvedValue(submission({ status: "Funded" }));
    renderPage();

    await userEvent.click(await screen.findByRole("button", { name: "Actions for invoice INV-1" }));
    expect(await screen.findByRole("menuitem", { name: "Charge back" })).toBeInTheDocument();
    await userEvent.click(screen.getByRole("menuitem", { name: "Release reserve" }));
    const dialog = await screen.findByRole("dialog", {}, { timeout: 5000 });

    expect(
      await within(dialog).findByText(
        "Sep 16, 2026 is in a closed period, so it books on Sep 17, 2026, the first open day.",
        { exact: false },
      ),
    ).toBeInTheDocument();
    expect(
      within(dialog).getByText("The invoice reads Settled on the schedule afterwards."),
    ).toBeInTheDocument();
    const [action, input] = mocks.fetchFactoringReserveMovementPlan.mock.calls.at(-1) ?? [];
    expect(action).toBe("release");
    expect(input).toEqual({
      submissionId: "facsub_1",
      itemId: "facsit_1",
      version: 4,
      occurredOn: expect.any(Number),
      cashMinor: null,
      reason: null,
    });

    await waitFor(() =>
      expect(within(dialog).getByRole("button", { name: "Release reserve" })).toBeEnabled(),
    );
    await userEvent.click(within(dialog).getByRole("button", { name: "Release reserve" }));
    await waitFor(() =>
      expect(mocks.bookFactoringReserveMovement).toHaveBeenCalledWith("release", input),
    );
    expect(mocks.toastSuccess).toHaveBeenCalledWith("Reserve released on invoice INV-1");
  });

  it("offers no chargeback under a non-recourse agreement", async () => {
    mocks.granted.add(CREATE);
    mocks.fetchFactoringSubmission.mockResolvedValue(
      submission({ status: "Funded", items: fundedItems() }),
    );
    mocks.listFactoringAgreements.mockResolvedValue([{ id: "facagr_1", recourse: "NonRecourse" }]);
    renderPage();

    await userEvent.click(await screen.findByRole("button", { name: "Actions for invoice INV-1" }));
    expect(await screen.findByRole("menuitem", { name: "Release reserve" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Charge back" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Actions for invoice INV-2" })).toBeNull();
  });
});
