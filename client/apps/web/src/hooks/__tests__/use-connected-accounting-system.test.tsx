import type { AccountingConnection } from "@/lib/graphql/accounting-sync";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useConnectedAccountingSystem } from "../use-connected-accounting-system";

const mocks = vi.hoisted(() => ({
  fetchAccountingSyncSummary: vi.fn(),
  fetchAccountingSyncStatus: vi.fn(),
}));

vi.mock("@/lib/graphql/accounting-sync-ledger", () => ({
  fetchAccountingSyncSummary: mocks.fetchAccountingSyncSummary,
  fetchAccountingBackfills: vi.fn(),
  fetchAccountingSyncAttempts: vi.fn(),
  fetchAccountingSyncObjectStates: vi.fn(),
  fetchAccountingSyncRecord: vi.fn(),
}));

vi.mock("@/lib/graphql/accounting-sync", () => ({
  fetchAccountingSyncStatus: mocks.fetchAccountingSyncStatus,
  fetchAccountingMappingSummary: vi.fn(),
  searchAccountingReferenceObjects: vi.fn(),
}));

function connection(overrides: Partial<AccountingConnection>): AccountingConnection {
  return {
    status: "Connected",
    connectedAt: 1_780_000_000,
    ...overrides,
  } as AccountingConnection;
}

function renderResolution(source: "syncSummary" | "status" = "syncSummary", enabled = true) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return renderHook(() => useConnectedAccountingSystem(source, enabled), { wrapper });
}

function summariesBySystem(bySystem: Record<string, AccountingConnection | null | Error>) {
  mocks.fetchAccountingSyncSummary.mockImplementation((system: string) => {
    const value = bySystem[system] ?? null;
    return value instanceof Error ? Promise.reject(value) : Promise.resolve({ connection: value });
  });
}

describe("useConnectedAccountingSystem", () => {
  beforeEach(() => {
    mocks.fetchAccountingSyncSummary.mockReset();
    mocks.fetchAccountingSyncStatus.mockReset();
  });

  it("names no system while the connections are still being read", () => {
    mocks.fetchAccountingSyncSummary.mockReturnValue(new Promise(() => undefined));

    const { result } = renderResolution();

    expect(result.current).toMatchObject({ system: null, isLoading: true, isError: false });
  });

  it("asks every accounting system and picks the one that is connected", async () => {
    summariesBySystem({ QuickBooksOnline: null, Xero: connection({ integrationType: "Xero" }) });

    const { result } = renderResolution();

    await waitFor(() => expect(result.current.system).toBe("Xero"));
    expect(result.current.connected).toBe(true);
    expect(mocks.fetchAccountingSyncSummary).toHaveBeenCalledWith(
      "QuickBooksOnline",
      expect.anything(),
    );
    expect(mocks.fetchAccountingSyncSummary).toHaveBeenCalledWith("Xero", expect.anything());
  });

  it("falls back to QuickBooks Online when nothing is connected", async () => {
    summariesBySystem({ QuickBooksOnline: null, Xero: null });

    const { result } = renderResolution();

    await waitFor(() => expect(result.current.system).toBe("QuickBooksOnline"));
    expect(result.current.connected).toBe(false);
  });

  it("still finds the connected system when another system cannot be read", async () => {
    summariesBySystem({ QuickBooksOnline: new Error("offline"), Xero: connection({}) });

    const { result } = renderResolution();

    await waitFor(() => expect(result.current.system).toBe("Xero"));
    expect(result.current.isError).toBe(false);
  });

  it("reports an error rather than guessing when an unread system might be the connected one", async () => {
    summariesBySystem({ QuickBooksOnline: null, Xero: new Error("offline") });

    const { result } = renderResolution();

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.system).toBeNull();

    summariesBySystem({ QuickBooksOnline: null, Xero: connection({}) });
    act(() => result.current.retry());

    await waitFor(() => expect(result.current.system).toBe("Xero"));
    expect(result.current.isError).toBe(false);
  });

  it("reads nothing while disabled", () => {
    const { result } = renderResolution("status", false);

    expect(result.current).toMatchObject({ system: null, isLoading: false, isError: false });
    expect(mocks.fetchAccountingSyncStatus).not.toHaveBeenCalled();
  });
});
