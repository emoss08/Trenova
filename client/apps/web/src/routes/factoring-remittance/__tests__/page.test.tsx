import type {
  FactoringRemittance,
  FactoringRemittanceLine,
  FactoringRemittancePlan,
} from "@/lib/graphql/factoring-remittance";
import { factoring } from "@/lib/queries/factoring";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FactoringRemittancePage, prefetch } from "../page";

const mocks = vi.hoisted(() => ({
  fetchFactoringRemittance: vi.fn(),
  listFactoringRemittanceLines: vi.fn(),
  fetchFactoringRemittancePlan: vi.fn(),
  applyFactoringRemittance: vi.fn(),
  updateFactoringRemittanceLine: vi.fn(),
  stageFactoringRemittance: vi.fn(),
  rematchFactoringRemittance: vi.fn(),
  discardFactoringRemittance: vi.fn(),
  granted: new Set<number>(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/lib/graphql/factoring-remittance", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/factoring-remittance")>()),
  fetchFactoringRemittance: mocks.fetchFactoringRemittance,
  listFactoringRemittanceLines: mocks.listFactoringRemittanceLines,
  fetchFactoringRemittancePlan: mocks.fetchFactoringRemittancePlan,
  applyFactoringRemittance: mocks.applyFactoringRemittance,
  updateFactoringRemittanceLine: mocks.updateFactoringRemittanceLine,
  stageFactoringRemittance: mocks.stageFactoringRemittance,
  rematchFactoringRemittance: mocks.rematchFactoringRemittance,
  discardFactoringRemittance: mocks.discardFactoringRemittance,
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
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { success: mocks.toastSuccess, error: mocks.toastError, info: vi.fn() },
}));

const UPDATE = 1 << 2;
const IMPORT = 1 << 5;
const CANCEL = 1 << 15;

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  mocks.granted.clear();
});

function remittance(overrides: Partial<FactoringRemittance> = {}): FactoringRemittance {
  return {
    id: "facrem_1",
    factoringCompanyId: "fac_rts",
    documentId: "doc_1",
    fileName: "rts-2027-01-15.csv",
    source: "Upload",
    inboundMessageId: null,
    sourceFormat: "Csv",
    status: "Parsed",
    defaultKind: "Funding",
    remittedOn: 1_800_000_000,
    currencyCode: "USD",
    mapping: [
      { field: "InvoiceNumber", column: 0 },
      { field: "Advance", column: 1 },
    ],
    headers: ["Invoice Number", "Advance Amount"],
    unmappedHeaders: [],
    lineCount: 2,
    matchedCount: 1,
    exceptionCount: 1,
    appliedCount: 0,
    totalCashMinor: 102_345,
    error: null,
    stagedAt: 1_800_000_100,
    appliedAt: null,
    discardedAt: null,
    discardReason: null,
    version: 3,
    createdAt: 1_800_000_000,
    factoringCompany: { id: "fac_rts", code: "RTS", name: "RTS Financial" },
    ...overrides,
  };
}

function line(overrides: Partial<FactoringRemittanceLine> = {}): FactoringRemittanceLine {
  return {
    id: "facrml_1",
    remittanceId: "facrem_1",
    rowNumber: 2,
    cells: ["INV-100", "900.00"],
    kind: "Funding",
    invoiceNumber: "INV-100",
    factorReference: null,
    debtorName: null,
    faceMinor: null,
    advanceMinor: 90_000,
    feeMinor: null,
    reserveMinor: null,
    cashMinor: null,
    occurredOn: 1_800_000_000,
    matchStatus: "Matched",
    matchedBy: "Auto",
    submissionItemId: "facsit_1",
    matchedInvoiceNumber: "INV-100",
    matchedSubmissionId: "facsub_1",
    score: 100,
    reason: "InvoiceAmountDate",
    suggestions: [],
    error: null,
    ignoredReason: null,
    reserveEntryId: null,
    ...overrides,
  };
}

const suggested = line({
  id: "facrml_2",
  rowNumber: 3,
  invoiceNumber: "IN-200",
  advanceMinor: 12_345,
  matchStatus: "Suggested",
  matchedBy: null,
  submissionItemId: null,
  matchedInvoiceNumber: null,
  matchedSubmissionId: null,
  score: 55,
  reason: "PartialInvoiceNearAmount",
  suggestions: [
    {
      itemId: "facsit_2",
      submissionId: "facsub_1",
      invoiceNumber: "INV-200",
      score: 55,
      reason: "PartialInvoiceNearAmount",
    },
  ],
});

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/billing/factoring/remittances/facrem_1"]}>
        <Routes>
          <Route path="/billing/factoring/remittances/:id" element={<FactoringRemittancePage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("prefetch", () => {
  it("warms the remittance the route names and nothing without an id", () => {
    const args = (id?: string) =>
      ({
        params: id ? { id } : {},
        request: new Request("http://localhost/billing/factoring/remittances"),
        context: undefined,
      }) as unknown as Parameters<typeof prefetch>[0];

    expect(prefetch(args("facrem_1"))).toEqual([
      expect.objectContaining({ queryKey: factoring.remittance("facrem_1").queryKey }),
    ]);
    expect(prefetch(args())).toEqual([]);
  });
});

describe("FactoringRemittancePage", () => {
  it("lists the exception with its suggestion and matches it on Use", async () => {
    mocks.granted.add(UPDATE);
    mocks.fetchFactoringRemittance.mockResolvedValue(remittance());
    mocks.listFactoringRemittanceLines.mockResolvedValue([line(), suggested]);
    mocks.updateFactoringRemittanceLine.mockResolvedValue(remittance({ version: 4 }));

    renderPage();
    expect(await screen.findByRole("heading", { name: "rts-2027-01-15.csv" })).toBeTruthy();
    expect(await screen.findByText("INV-200")).toBeTruthy();
    expect(screen.getByText("Invoice partly agrees, amount is close")).toBeTruthy();

    await userEvent.click(screen.getByRole("button", { name: "Use" }));
    await waitFor(() =>
      expect(mocks.updateFactoringRemittanceLine).toHaveBeenCalledWith({
        remittanceId: "facrem_1",
        version: 3,
        lineId: "facrml_2",
        action: "Match",
        itemId: "facsit_2",
        kind: null,
        reason: null,
      }),
    );
  });

  it("previews the journal and applies the remittance only with the apply permission", async () => {
    mocks.fetchFactoringRemittance.mockResolvedValue(remittance());
    mocks.listFactoringRemittanceLines.mockResolvedValue([line(), suggested]);
    const { unmount } = renderPage();
    await screen.findByRole("heading", { name: "rts-2027-01-15.csv" });
    expect(screen.queryByRole("button", { name: "Apply" })).toBeNull();
    unmount();

    mocks.granted.add(IMPORT);
    const plan: FactoringRemittancePlan = {
      remittanceId: "facrem_1",
      fundedCount: 1,
      rejectedCount: 0,
      releasedCount: 0,
      chargebackCount: 0,
      leftOpenCount: 1,
      advanceMinor: 90_000,
      feeMinor: 3_000,
      releasedMinor: 0,
      chargebackMinor: 0,
      lines: [
        {
          lineId: "facrml_1",
          rowNumber: 2,
          kind: "Funding",
          invoiceNumber: "INV-100",
          submissionId: "facsub_1",
          itemId: "facsit_1",
          cashMinor: 90_000,
          movement: null,
        },
      ],
    };
    mocks.fetchFactoringRemittancePlan.mockResolvedValue(plan);
    mocks.applyFactoringRemittance.mockResolvedValue(
      remittance({ status: "PartiallyApplied", appliedCount: 1, version: 4 }),
    );

    renderPage();
    await userEvent.click(await screen.findByRole("button", { name: "Apply" }));
    const dialog = await screen.findByRole("dialog");
    expect(await within(dialog).findByText("Row 2: Funding of invoice INV-100")).toBeTruthy();
    expect(within(dialog).getByText("1 line is still to resolve and stay open.")).toBeTruthy();
    await userEvent.click(within(dialog).getByRole("button", { name: "Apply" }));
    await waitFor(() =>
      expect(mocks.applyFactoringRemittance).toHaveBeenCalledWith({ id: "facrem_1", version: 3 }),
    );
    expect(mocks.toastSuccess).toHaveBeenCalled();
  });

  it("asks for the column mapping when the headers did not place the fields", async () => {
    mocks.granted.add(UPDATE);
    mocks.fetchFactoringRemittance.mockResolvedValue(
      remittance({
        status: "NeedsMapping",
        mapping: [],
        headers: ["Ref", "Paid Out", "Wire Dt"],
        lineCount: 0,
        matchedCount: 0,
        exceptionCount: 0,
      }),
    );
    mocks.stageFactoringRemittance.mockResolvedValue(remittance({ version: 4 }));

    renderPage();
    const read = await screen.findByRole("button", { name: "Read with this mapping" });
    expect(read).toHaveProperty("disabled", true);
    expect(
      screen.getByText(
        "Choose the column that holds the invoice number or the factor's reference.",
      ),
    ).toBeTruthy();
    expect(mocks.listFactoringRemittanceLines).not.toHaveBeenCalled();
  });

  it("offers discarding only to someone allowed to, and asks why", async () => {
    mocks.granted.add(CANCEL);
    mocks.fetchFactoringRemittance.mockResolvedValue(remittance());
    mocks.listFactoringRemittanceLines.mockResolvedValue([line()]);
    mocks.discardFactoringRemittance.mockResolvedValue(remittance({ status: "Discarded" }));

    renderPage();
    await userEvent.click(await screen.findByRole("button", { name: "Discard" }));
    const dialog = await screen.findByRole("dialog");
    await userEvent.type(within(dialog).getByRole("textbox"), "Duplicate of yesterday's report");
    await userEvent.click(within(dialog).getByRole("button", { name: "Discard" }));
    await waitFor(() =>
      expect(mocks.discardFactoringRemittance).toHaveBeenCalledWith({
        id: "facrem_1",
        version: 3,
        reason: "Duplicate of yesterday's report",
      }),
    );
  });
});
