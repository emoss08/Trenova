import type {
  AccountingSyncObjectState,
  AccountingSyncRecord,
} from "@/lib/graphql/accounting-sync-ledger";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AccountingSyncStateLine } from "../sync-state-line";

const mocks = vi.hoisted(() => ({
  fetchAccountingSyncObjectStates: vi.fn(),
  allowed: true,
}));

vi.mock("@/lib/graphql/accounting-sync-ledger", () => ({
  fetchAccountingSyncObjectStates: mocks.fetchAccountingSyncObjectStates,
  fetchAccountingSyncSummary: vi.fn(),
  fetchAccountingSyncRecord: vi.fn(),
  fetchAccountingSyncAttempts: vi.fn(),
  fetchAccountingBackfills: vi.fn(),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.allowed, isLoading: false }),
}));

const INVOICE_ID = "inv_01";

function record(overrides: Partial<AccountingSyncRecord> = {}): AccountingSyncRecord {
  return {
    id: "acctsr_1",
    objectType: "Invoice",
    objectId: INVOICE_ID,
    objectNumber: "INV-1042",
    operation: "Create",
    sourceEvent: "InvoicePosted",
    revision: 1,
    documentDate: 1_780_000_000,
    dependsOnRecordId: null,
    status: "Synced",
    attemptCount: 1,
    nextAttemptAt: null,
    externalId: "145",
    externalDocNumber: "1042",
    externalUrl: "https://app.qbo.intuit.com/app/invoice?txnId=145",
    errorCategory: null,
    errorCode: "",
    errorMessage: "",
    resolution: "",
    queuedAt: 1_780_000_000,
    startedAt: 1_780_000_010,
    syncedAt: 1_780_000_020,
    skippedBy: null,
    skippedReason: "",
    version: 2,
    updatedAt: 1_780_000_020,
    ...overrides,
  };
}

function state(overrides: Partial<AccountingSyncRecord> = {}): AccountingSyncObjectState {
  const rec = record(overrides);
  return {
    objectType: rec.objectType,
    objectId: rec.objectId,
    providerName: "QuickBooks Online",
    record: rec,
  };
}

function renderLine(objectId: string | null = INVOICE_ID) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const view = render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <AccountingSyncStateLine objectId={objectId} />
      </QueryClientProvider>
    </MemoryRouter>,
  );
  const settled = () =>
    waitFor(() => {
      expect(mocks.fetchAccountingSyncObjectStates).toHaveBeenCalled();
      expect(client.isFetching()).toBe(0);
    });
  return { ...view, settled };
}

describe("AccountingSyncStateLine", () => {
  beforeEach(() => {
    mocks.fetchAccountingSyncObjectStates.mockReset();
    mocks.allowed = true;
  });

  it("links a synced document to QuickBooks and to its sync history", async () => {
    mocks.fetchAccountingSyncObjectStates.mockResolvedValue([state()]);

    renderLine();

    expect(await screen.findByText("In QuickBooks Online")).toBeInTheDocument();
    expect(screen.getByText("Open 1042 in QuickBooks Online")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Sync history" })).toHaveAttribute(
      "href",
      "/accounting/sync?panelType=edit&panelEntityId=acctsr_1",
    );
    expect(mocks.fetchAccountingSyncObjectStates).toHaveBeenCalledWith(
      [INVOICE_ID],
      expect.anything(),
    );
  });

  it("says what to fix when the document is held", async () => {
    mocks.fetchAccountingSyncObjectStates.mockResolvedValue([
      state({
        status: "Blocked",
        errorCategory: "Mapping",
        resolution: "Map charge code DET to a QuickBooks Online item",
        externalUrl: "",
        syncedAt: null,
      }),
    ]);

    renderLine();

    expect(await screen.findByText("Held from QuickBooks Online")).toBeInTheDocument();
    expect(screen.getByText("Map charge code DET to a QuickBooks Online item")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Resolve in the sync ledger" })).toBeInTheDocument();
    expect(screen.queryByText(/Open .* in QuickBooks Online/)).not.toBeInTheDocument();
  });

  it("falls back to the provider's message when there is no resolution", async () => {
    mocks.fetchAccountingSyncObjectStates.mockResolvedValue([
      state({ status: "Blocked", resolution: "", errorMessage: "Business Validation Error" }),
    ]);

    renderLine();

    expect(await screen.findByText("Business Validation Error")).toBeInTheDocument();
  });

  it("counts the tries once Trenova gave up", async () => {
    mocks.fetchAccountingSyncObjectStates.mockResolvedValue([
      state({ status: "DeadLettered", attemptCount: 8, externalUrl: "" }),
    ]);

    renderLine();

    expect(await screen.findByText("Not sent to QuickBooks Online")).toBeInTheDocument();
    expect(screen.getByText("gave up after 8 tries")).toBeInTheDocument();
  });

  it("shows why a person skipped it", async () => {
    mocks.fetchAccountingSyncObjectStates.mockResolvedValue([
      state({ status: "Skipped", skippedReason: "Entered in QuickBooks by hand", externalUrl: "" }),
    ]);

    renderLine();

    expect(await screen.findByText("Entered in QuickBooks by hand")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Sync history" })).toBeInTheDocument();
  });

  it("offers the link without a document number when QuickBooks gave none", async () => {
    mocks.fetchAccountingSyncObjectStates.mockResolvedValue([state({ externalDocNumber: "" })]);

    renderLine();

    expect(await screen.findByText("Open in QuickBooks Online")).toBeInTheDocument();
  });

  it("renders nothing for a document that was never queued", async () => {
    mocks.fetchAccountingSyncObjectStates.mockResolvedValue([]);

    const { container, settled } = renderLine();

    await settled();
    expect(container).toBeEmptyDOMElement();
  });

  it("ignores a state for another document", async () => {
    mocks.fetchAccountingSyncObjectStates.mockResolvedValue([
      { ...state(), objectId: "inv_other", record: record({ objectId: "inv_other" }) },
    ]);

    const { container, settled } = renderLine();

    await settled();
    expect(container).toBeEmptyDOMElement();
  });

  it("does not ask without permission to read the ledger", () => {
    mocks.allowed = false;

    const { container } = renderLine();

    expect(container).toBeEmptyDOMElement();
    expect(mocks.fetchAccountingSyncObjectStates).not.toHaveBeenCalled();
  });

  it("does not ask before there is a document", () => {
    const { container } = renderLine(null);

    expect(container).toBeEmptyDOMElement();
    expect(mocks.fetchAccountingSyncObjectStates).not.toHaveBeenCalled();
  });
});
