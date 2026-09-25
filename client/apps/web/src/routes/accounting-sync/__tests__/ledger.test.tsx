import type { AccountingConnection } from "@/lib/graphql/accounting-sync";
import type {
  AccountingSyncRecord,
  AccountingSyncSummary,
} from "@/lib/graphql/accounting-sync-ledger";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { NuqsTestingAdapter, type UrlUpdateEvent } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { LedgerHeaderActions } from "../_components/ledger-header-actions";
import { LedgerRecordPanel } from "../_components/ledger-record-panel";
import { LedgerNotices, LedgerSummary } from "../_components/ledger-summary";

const mocks = vi.hoisted(() => ({
  retryAccountingSync: vi.fn(),
  releaseAccountingSync: vi.fn(),
  skipAccountingSync: vi.fn(),
  pauseAccountingSync: vi.fn(),
  resumeAccountingSync: vi.fn(),
  requestAccountingBackfill: vi.fn(),
  changeAccountingBackfill: vi.fn(),
  fetchAccountingSyncRecord: vi.fn(),
  fetchAccountingSyncAttempts: vi.fn(),
  granted: new Set<string>(),
}));

vi.mock("@/lib/graphql/accounting-sync-ledger", () => ({
  retryAccountingSync: mocks.retryAccountingSync,
  releaseAccountingSync: mocks.releaseAccountingSync,
  skipAccountingSync: mocks.skipAccountingSync,
  pauseAccountingSync: mocks.pauseAccountingSync,
  resumeAccountingSync: mocks.resumeAccountingSync,
  requestAccountingBackfill: mocks.requestAccountingBackfill,
  changeAccountingBackfill: mocks.changeAccountingBackfill,
  fetchAccountingSyncRecord: mocks.fetchAccountingSyncRecord,
  fetchAccountingSyncAttempts: mocks.fetchAccountingSyncAttempts,
  fetchAccountingSyncSummary: vi.fn(),
  fetchAccountingSyncObjectStates: vi.fn(),
  fetchAccountingBackfills: vi.fn(),
  enableAccountingSync: vi.fn(),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: number) => ({
    allowed: mocks.granted.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}));

const SYNC_UPDATE = `${Resource.AccountingSync}:${Operation.Update}`;
const INTEGRATION_MANAGE = `${Resource.AccountingIntegration}:${Operation.Manage}`;

const connection: AccountingConnection = {
  id: "acctc_1",
  integrationType: "QuickBooksOnline",
  status: "Connected",
  appSource: "Instance",
  appEnvironment: "Sandbox",
  externalCompanyName: "Peak Freight",
  externalLegalName: "Peak Freight LLC",
  externalCountry: "US",
  externalHomeCurrency: "USD",
  externalMultiCurrencyEnabled: false,
  externalBooksClosedThrough: null,
  lastCheckedAt: null,
  lastSuccessAt: null,
  lastFailureAt: null,
  consecutiveFailures: 0,
  lastErrorCategory: null,
  lastErrorMessage: "",
  lastWebhookAt: null,
  refreshTokenAbsoluteExpiresAt: 4_102_444_800,
  connectedAt: 1_780_000_000,
  disconnectedAt: null,
  setupStep: "Complete",
  syncStartDate: 1_779_000_000,
  syncEnabledAt: 1_780_000_000,
  autoSync: true,
  pausedAt: null,
  pausedBy: null,
  pausedReason: "",
  referenceRefreshStartedAt: null,
  referenceRefreshedAt: 1_780_000_100,
  referenceRefreshError: "",
  version: 1,
  updatedAt: 1_780_000_000,
};

function summary(overrides: Partial<AccountingSyncSummary> = {}): AccountingSyncSummary {
  return {
    integrationType: "QuickBooksOnline",
    providerName: "QuickBooks Online",
    connection,
    counts: [
      { status: "Synced", count: 40 },
      { status: "Queued", count: 2 },
      { status: "Retrying", count: 1 },
      { status: "AwaitingApproval", count: 3 },
      { status: "Blocked", count: 4 },
      { status: "DeadLettered", count: 1 },
    ],
    attention: [],
    activeBackfill: null,
    ...overrides,
  };
}

function record(overrides: Partial<AccountingSyncRecord> = {}): AccountingSyncRecord {
  return {
    id: "acctsr_1",
    objectType: "Invoice",
    objectId: "inv_1",
    objectNumber: "INV-1042",
    operation: "Create",
    sourceEvent: "InvoicePosted",
    revision: 1,
    documentDate: 1_780_000_000,
    dependsOnRecordId: null,
    status: "Blocked",
    attemptCount: 1,
    nextAttemptAt: null,
    externalId: "",
    externalDocNumber: "",
    externalUrl: "",
    errorCategory: "Mapping",
    errorCode: "",
    errorMessage: "",
    resolution: "Map charge code DET to a QuickBooks Online item",
    queuedAt: 1_780_000_000,
    startedAt: null,
    syncedAt: null,
    skippedBy: null,
    skippedReason: "",
    version: 1,
    updatedAt: 1_780_000_000,
    ...overrides,
  };
}

function renderWith(node: ReactNode, searchParams = "") {
  const urls: UrlUpdateEvent[] = [];
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const view = render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter
          searchParams={searchParams}
          hasMemory
          onUrlUpdate={(event) => urls.push(event)}
        >
          {node}
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
  return { ...view, urls };
}

beforeEach(() => {
  for (const mock of Object.values(mocks)) {
    if (typeof mock === "function") mock.mockReset();
  }
  mocks.granted = new Set([SYNC_UPDATE, INTEGRATION_MANAGE]);
});

describe("LedgerSummary", () => {
  it("groups the counts the way people read them", () => {
    renderWith(<LedgerSummary summary={summary()} />);

    const synced = screen.getByRole("button", { name: /Synced/ });
    expect(synced).toHaveTextContent("40");
    expect(screen.getByRole("button", { name: /In progress/ })).toHaveTextContent("3");
    expect(screen.getByRole("button", { name: /Waiting for release/ })).toHaveTextContent("3");
    expect(screen.getByRole("button", { name: /Needs attention/ })).toHaveTextContent("5");
  });

  it("filters the table to a figure and clears it on a second click", async () => {
    const { urls } = renderWith(<LedgerSummary summary={summary()} />);

    await userEvent.click(screen.getByRole("button", { name: /Needs attention/ }));
    await waitFor(() => expect(urls.length).toBeGreaterThan(0));
    const filters = JSON.parse(urls.at(-1)!.searchParams.get("fieldFilters") ?? "[]");
    expect(filters).toEqual([
      { field: "status", operator: "in", value: ["Blocked", "DeadLettered"] },
    ]);
    expect(urls.at(-1)!.searchParams.get("pageIndex")).toBeNull();

    await userEvent.click(screen.getByRole("button", { name: /Needs attention/ }));
    await waitFor(() => expect(urls.at(-1)!.searchParams.get("fieldFilters") ?? "").toBe(""));
  });
});

describe("LedgerNotices", () => {
  it("says who paused sending and resumes it", async () => {
    mocks.resumeAccountingSync.mockResolvedValue(connection);
    renderWith(
      <LedgerNotices
        summary={summary({
          connection: {
            ...connection,
            pausedAt: 1_780_000_500,
            pausedBy: { id: "usr_1", name: "Dana Whitfield" },
            pausedReason: "Month-end close",
          },
        })}
      />,
    );

    expect(screen.getByText(/by Dana Whitfield: Month-end close/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Resume" }));
    await waitFor(() =>
      expect(mocks.resumeAccountingSync).toHaveBeenCalledWith("QuickBooksOnline"),
    );
  });

  it("retries every record held for one reason", async () => {
    mocks.retryAccountingSync.mockResolvedValue(4);
    renderWith(
      <LedgerNotices
        summary={summary({
          attention: [
            {
              status: "Blocked",
              errorCategory: "Mapping",
              resolution: "Map charge code DET to a QuickBooks Online item",
              count: 4,
              oldestQueuedAt: 1_780_000_000,
              sampleRecordId: "acctsr_9",
            },
          ],
        })}
      />,
    );

    expect(screen.getByText("Map charge code DET to a QuickBooks Online item")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open an example" })).toHaveAttribute(
      "href",
      "/accounting/sync?panelType=edit&panelEntityId=acctsr_9",
    );
    await userEvent.click(screen.getByRole("button", { name: "Retry these" }));
    await waitFor(() =>
      expect(mocks.retryAccountingSync).toHaveBeenCalledWith({
        integrationType: "QuickBooksOnline",
        ids: null,
        errorCategories: ["Mapping"],
      }),
    );
  });

  it("hides the retry from someone who may only read the ledger", () => {
    mocks.granted = new Set();
    renderWith(
      <LedgerNotices
        summary={summary({
          attention: [
            {
              status: "Blocked",
              errorCategory: "Mapping",
              resolution: "Map it",
              count: 1,
              oldestQueuedAt: 1_780_000_000,
              sampleRecordId: "acctsr_9",
            },
          ],
        })}
      />,
    );

    expect(screen.queryByRole("button", { name: "Retry these" })).not.toBeInTheDocument();
  });

  it("pauses and cancels a running backfill", async () => {
    mocks.changeAccountingBackfill.mockResolvedValue({});
    renderWith(
      <LedgerNotices
        summary={summary({
          activeBackfill: {
            id: "acctbf_1",
            rangeStart: 1_779_000_000,
            rangeEnd: 1_780_000_000,
            objectTypes: [],
            status: "Running",
            enqueuedCount: 12,
            alreadyQueuedCount: 3,
            startedAt: 1_780_000_010,
            completedAt: null,
            lastError: "",
            version: 1,
            createdAt: 1_780_000_000,
            updatedAt: 1_780_000_010,
          },
        })}
      />,
    );

    expect(screen.getByText(/12 queued, 3 already queued/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Pause" }));
    await waitFor(() =>
      expect(mocks.changeAccountingBackfill).toHaveBeenCalledWith({
        id: "acctbf_1",
        action: "Pause",
      }),
    );
    await userEvent.click(screen.getByRole("button", { name: "Cancel backfill" }));
    await waitFor(() =>
      expect(mocks.changeAccountingBackfill).toHaveBeenCalledWith({
        id: "acctbf_1",
        action: "Cancel",
      }),
    );
  });
});

describe("LedgerHeaderActions", () => {
  it("offers nothing before sending is turned on", () => {
    const { container } = renderWith(
      <LedgerHeaderActions
        summary={summary({ connection: { ...connection, syncEnabledAt: null } })}
      />,
    );

    expect(container).toBeEmptyDOMElement();
  });

  it("retries every failed record", async () => {
    mocks.retryAccountingSync.mockResolvedValue(5);
    renderWith(<LedgerHeaderActions summary={summary()} />);

    await userEvent.click(screen.getByRole("button", { name: "Retry failed (5)" }));
    await waitFor(() =>
      expect(mocks.retryAccountingSync).toHaveBeenCalledWith({
        integrationType: "QuickBooksOnline",
        ids: null,
        errorCategories: null,
      }),
    );
  });

  it("confirms before releasing held documents", async () => {
    mocks.releaseAccountingSync.mockResolvedValue(3);
    renderWith(<LedgerHeaderActions summary={summary()} />);

    await userEvent.click(screen.getByRole("button", { name: "Release held (3)" }));
    const dialog = await screen.findByRole("alertdialog");
    expect(mocks.releaseAccountingSync).not.toHaveBeenCalled();
    await userEvent.click(within(dialog).getByRole("button", { name: "Release" }));
    await waitFor(() =>
      expect(mocks.releaseAccountingSync).toHaveBeenCalledWith({
        integrationType: "QuickBooksOnline",
        ids: null,
      }),
    );
  });

  it("pauses with the reason given", async () => {
    mocks.pauseAccountingSync.mockResolvedValue(connection);
    renderWith(<LedgerHeaderActions summary={summary()} />);

    await userEvent.click(screen.getByRole("button", { name: "Pause sending" }));
    const dialog = await screen.findByRole("dialog");
    await userEvent.type(within(dialog).getByRole("textbox"), "Month-end close");
    await userEvent.click(within(dialog).getByRole("button", { name: "Pause sending" }));
    await waitFor(() =>
      expect(mocks.pauseAccountingSync).toHaveBeenCalledWith({
        integrationType: "QuickBooksOnline",
        reason: "Month-end close",
      }),
    );
  });

  it("resumes when paused", async () => {
    mocks.resumeAccountingSync.mockResolvedValue(connection);
    renderWith(
      <LedgerHeaderActions
        summary={summary({ connection: { ...connection, pausedAt: 1_780_000_500 } })}
      />,
    );

    expect(screen.queryByRole("button", { name: "Pause sending" })).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Resume sending" }));
    await waitFor(() => expect(mocks.resumeAccountingSync).toHaveBeenCalled());
  });

  it("starts a backfill over the range sync has not covered", async () => {
    mocks.requestAccountingBackfill.mockResolvedValue({});
    renderWith(<LedgerHeaderActions summary={summary()} />);

    await userEvent.click(screen.getByRole("button", { name: "Backfill" }));
    await userEvent.click(await screen.findByRole("button", { name: "Start backfill" }));
    await waitFor(() =>
      expect(mocks.requestAccountingBackfill).toHaveBeenCalledWith({
        integrationType: "QuickBooksOnline",
        rangeStart: 1_779_000_000,
        rangeEnd: 1_780_000_000,
        objectTypes: null,
      }),
    );
  });

  it("does not offer a second backfill while one runs, nor to someone without manage access", () => {
    mocks.granted = new Set([SYNC_UPDATE]);
    renderWith(<LedgerHeaderActions summary={summary()} />);

    expect(screen.queryByRole("button", { name: "Backfill" })).not.toBeInTheDocument();
  });

  it("offers only what fits the counts", () => {
    renderWith(
      <LedgerHeaderActions summary={summary({ counts: [{ status: "Synced", count: 3 }] })} />,
    );

    expect(screen.queryByRole("button", { name: /Retry failed/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Release held/ })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Pause sending" })).toBeInTheDocument();
  });
});

describe("LedgerRecordPanel", () => {
  function renderPanel(row: AccountingSyncRecord | null, searchParams = "") {
    mocks.fetchAccountingSyncAttempts.mockResolvedValue([
      {
        id: "acctsa_1",
        attemptNumber: 1,
        outcome: "Blocked",
        errorCategory: "Mapping",
        errorCode: "",
        errorMessage: "Charge code DET is not mapped",
        startedAt: 1_780_000_010,
        finishedAt: 1_780_000_011,
        durationMs: 420,
      },
    ]);
    return renderWith(
      <LedgerRecordPanel
        open
        onOpenChange={() => undefined}
        mode="edit"
        row={row}
        system="QuickBooksOnline"
        providerName="QuickBooks Online"
      />,
      searchParams,
    );
  }

  it("shows what to fix and every try", async () => {
    renderPanel(record());

    expect(
      await screen.findByText("Map charge code DET to a QuickBooks Online item"),
    ).toBeInTheDocument();
    expect(await screen.findByText(/Charge code DET is not mapped/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "INV-1042" })).toHaveAttribute(
      "href",
      expect.stringContaining("inv_1"),
    );
  });

  it("retries the record", async () => {
    mocks.retryAccountingSync.mockResolvedValue(1);
    renderPanel(record());

    await userEvent.click(await screen.findByRole("button", { name: "Retry now" }));
    await waitFor(() =>
      expect(mocks.retryAccountingSync).toHaveBeenCalledWith({
        integrationType: "QuickBooksOnline",
        ids: ["acctsr_1"],
        errorCategories: null,
      }),
    );
  });

  it("releases a record waiting for approval, and offers no retry for it", async () => {
    mocks.releaseAccountingSync.mockResolvedValue(1);
    renderPanel(record({ status: "AwaitingApproval", errorCategory: null, resolution: "" }));

    expect(screen.queryByRole("button", { name: "Retry now" })).not.toBeInTheDocument();
    await userEvent.click(await screen.findByRole("button", { name: "Release" }));
    await waitFor(() =>
      expect(mocks.releaseAccountingSync).toHaveBeenCalledWith({
        integrationType: "QuickBooksOnline",
        ids: ["acctsr_1"],
      }),
    );
  });

  it("will not skip without a reason", async () => {
    mocks.skipAccountingSync.mockResolvedValue(record({ status: "Skipped" }));
    renderPanel(record());

    await userEvent.click(await screen.findByRole("button", { name: "Skip" }));
    await userEvent.click(screen.getByRole("button", { name: "Skip document" }));
    expect(await screen.findByText("Say why this document is not sent")).toBeInTheDocument();
    expect(mocks.skipAccountingSync).not.toHaveBeenCalled();

    await userEvent.type(screen.getByRole("textbox"), "Entered by hand");
    await userEvent.click(screen.getByRole("button", { name: "Skip document" }));
    await waitFor(() =>
      expect(mocks.skipAccountingSync).toHaveBeenCalledWith({
        id: "acctsr_1",
        reason: "Entered by hand",
      }),
    );
  });

  it("offers no action on a synced record", async () => {
    renderPanel(
      record({
        status: "Synced",
        errorCategory: null,
        resolution: "",
        externalDocNumber: "1042",
        externalUrl: "https://app.qbo.intuit.com/app/invoice?txnId=145",
        syncedAt: 1_780_000_020,
      }),
    );

    expect(await screen.findByText("1042")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Retry now" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Skip" })).not.toBeInTheDocument();
  });

  it("hides the actions from someone who may only read the ledger", async () => {
    mocks.granted = new Set();
    renderPanel(record());

    expect(
      await screen.findByText("Map charge code DET to a QuickBooks Online item"),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Retry now" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Skip" })).not.toBeInTheDocument();
  });

  it("loads a record opened from a link when it is not on the current page", async () => {
    mocks.fetchAccountingSyncRecord.mockResolvedValue(record({ id: "acctsr_7" }));
    renderPanel(null, "?panelType=edit&panelEntityId=acctsr_7");

    expect(
      await screen.findByText("Map charge code DET to a QuickBooks Online item"),
    ).toBeInTheDocument();
    expect(mocks.fetchAccountingSyncRecord).toHaveBeenCalledWith("acctsr_7", expect.anything());
  });

  it("says so when a linked record cannot be loaded", async () => {
    mocks.fetchAccountingSyncRecord.mockRejectedValue(new Error("not found"));
    renderPanel(null, "?panelType=edit&panelEntityId=acctsr_7");

    expect(await screen.findByText("This sync record could not be loaded.")).toBeInTheDocument();
  });
});
