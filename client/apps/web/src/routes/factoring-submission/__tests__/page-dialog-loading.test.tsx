import type { FactoringSubmissionDetail, FactoringSubmissionItem } from "@/lib/graphql/factoring";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FactoringSubmissionPage } from "../page";

/*
The schedule page's three action dialogs (record funding, a reserve movement,
cancel) carry forms, a debounced journal preview and their own GraphQL; a
person reading a schedule opens at most one of them. Each is read when the
pointer or focus reaches the action that opens it, never with the page.
*/

const mocks = vi.hoisted(() => ({
  loaded: [] as string[],
  fetchFactoringSubmission: vi.fn(),
  listFactoringAgreements: vi.fn(),
  granted: new Set<number>(),
}));

vi.mock("../_components/record-funding-dialog", async (importOriginal) => {
  mocks.loaded.push("record-funding");
  return importOriginal();
});
vi.mock("../_components/reserve-movement-dialog", async (importOriginal) => {
  mocks.loaded.push("reserve-movement");
  return importOriginal();
});
vi.mock("../_components/cancel-submission-dialog", async (importOriginal) => {
  mocks.loaded.push("cancel");
  return importOriginal();
});
vi.mock("@/lib/graphql/factoring", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/factoring")>()),
  fetchFactoringSubmission: mocks.fetchFactoringSubmission,
  listFactoringAgreements: mocks.listFactoringAgreements,
  fetchFactoringFundingPlan: () => new Promise<never>(() => {}),
  fetchFactoringReserveMovementPlan: () => new Promise<never>(() => {}),
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
    pageHeaderProps: { title: string; actions?: React.ReactNode };
    children: React.ReactNode;
  }) => (
    <main>
      <h1>{pageHeaderProps.title}</h1>
      <div>{pageHeaderProps.actions}</div>
      {children}
    </main>
  ),
}));

const CREATE = 1 << 1;
const CANCEL = 1 << 15;

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  mocks.granted.clear();
});

function item(status: FactoringSubmissionItem["status"]): FactoringSubmissionItem {
  return {
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
    advanceMinor: status === "Pending" ? null : 95_000,
    feeMinor: status === "Pending" ? null : 2_000,
    reserveMinor: status === "Pending" ? null : 3_000,
    packetDocumentId: null,
    status,
    releasedAt: null,
    customer: { id: "cus_1", code: "ACME", name: "Acme Foods" },
  } as FactoringSubmissionItem;
}

function submission(
  status: FactoringSubmissionDetail["status"],
  items: FactoringSubmissionItem[],
): FactoringSubmissionDetail {
  return {
    id: "facsub_1",
    number: "FS2610-00007",
    factoringCompanyId: "fac_rts",
    agreementId: "facagr_1",
    status,
    currencyCode: "USD",
    invoiceCount: items.length,
    faceMinor: 100_000,
    expectedAdvanceMinor: 95_000,
    expectedFeeMinor: 2_500,
    expectedReserveMinor: 2_500,
    schedulePdfDocumentId: null,
    scheduleCsvDocumentId: null,
    transportMethod: "Email",
    transportTarget: "schedules@rtsfinancial.com",
    transportReference: null,
    attempts: 1,
    lastError: status === "Failed" ? "Mailbox unavailable" : null,
    sentAt: status === "Failed" ? null : 1_789_000_100,
    failedAt: status === "Failed" ? 1_789_000_600 : null,
    version: 4,
    createdAt: 1_789_000_000,
    factoringCompany: { id: "fac_rts", code: "RTS", name: "RTS Financial" },
    items,
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

describe("factoring schedule action dialogs", () => {
  it("reads the cancel dialog when the pointer reaches Cancel schedule, and none other", async () => {
    mocks.granted.add(CANCEL);
    mocks.fetchFactoringSubmission.mockResolvedValue(submission("Failed", [item("Pending")]));
    renderPage();

    const cancel = await screen.findByRole("button", { name: "Cancel schedule" });
    expect(mocks.loaded).toEqual([]);

    await userEvent.hover(cancel);
    await waitFor(() => expect(mocks.loaded).toEqual(["cancel"]));
    await userEvent.click(cancel);

    const dialog = await screen.findByRole("dialog", {}, { timeout: 10_000 });
    expect(within(dialog).getByRole("heading", { name: "Cancel schedule" })).toBeInTheDocument();
    expect(mocks.loaded).toEqual(["cancel"]);
  }, 20_000);

  it("reads the funding dialog at Record funding", async () => {
    mocks.granted.add(CREATE);
    mocks.listFactoringAgreements.mockResolvedValue([]);
    mocks.fetchFactoringSubmission.mockResolvedValue(submission("Sent", [item("Pending")]));
    renderPage();

    const record = await screen.findByRole("button", { name: "Record funding" });
    expect(mocks.loaded).not.toContain("record-funding");

    await userEvent.click(record);

    const dialog = await screen.findByRole("dialog", {}, { timeout: 10_000 });
    expect(within(dialog).getByRole("heading", { name: "Record funding" })).toBeInTheDocument();
    expect(mocks.loaded).toContain("record-funding");
    expect(mocks.loaded).not.toContain("reserve-movement");
  }, 20_000);

  it("reads the reserve movement dialog when an invoice's actions are reached", async () => {
    mocks.granted.add(CREATE);
    mocks.listFactoringAgreements.mockResolvedValue([{ id: "facagr_1", recourse: "NonRecourse" }]);
    mocks.fetchFactoringSubmission.mockResolvedValue(submission("Funded", [item("Funded")]));
    renderPage();

    const actions = await screen.findByRole("button", { name: "Actions for invoice INV-1" });
    expect(mocks.loaded).not.toContain("reserve-movement");

    await userEvent.click(actions);
    await waitFor(() => expect(mocks.loaded).toContain("reserve-movement"));
    await userEvent.click(await screen.findByRole("menuitem", { name: "Release reserve" }));

    const dialog = await screen.findByRole("dialog", {}, { timeout: 10_000 });
    expect(within(dialog).getByRole("heading", { name: "Release reserve" })).toBeInTheDocument();
  }, 20_000);
});
