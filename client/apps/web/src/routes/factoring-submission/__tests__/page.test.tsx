import type { FactoringSubmissionDetail } from "@/lib/graphql/factoring";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FactoringSubmissionPage } from "../page";

const mocks = vi.hoisted(() => ({
  fetchFactoringSubmission: vi.fn(),
  retryFactoringSubmission: vi.fn(),
  cancelFactoringSubmission: vi.fn(),
  granted: new Set<number>(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/lib/graphql/factoring", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/factoring")>()),
  fetchFactoringSubmission: mocks.fetchFactoringSubmission,
  retryFactoringSubmission: mocks.retryFactoringSubmission,
  cancelFactoringSubmission: mocks.cancelFactoringSubmission,
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
    const dialog = await screen.findByRole("dialog");
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
