import type { AccountingConnection, AccountingSyncStatus } from "@/lib/graphql/accounting-sync";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { QuickBooksIntegrationModal } from "../accounting-integration-modal";

const mocks = vi.hoisted(() => ({
  fetchAccountingSyncStatus: vi.fn(),
  startAccountingAuthorization: vi.fn(),
  checkAccountingConnection: vi.fn(),
  disconnectAccountingSystem: vi.fn(),
  completeAccountingAuthorization: vi.fn(),
  granted: new Set<string>(),
  assign: vi.fn(),
}));

vi.mock("@/lib/graphql/accounting-sync", () => ({
  fetchAccountingSyncStatus: mocks.fetchAccountingSyncStatus,
  startAccountingAuthorization: mocks.startAccountingAuthorization,
  checkAccountingConnection: mocks.checkAccountingConnection,
  disconnectAccountingSystem: mocks.disconnectAccountingSystem,
  completeAccountingAuthorization: mocks.completeAccountingAuthorization,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: number) => ({
    allowed: mocks.granted.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

vi.mock("@trenova/shared/components/theme-provider", () => ({
  useTheme: () => ({ theme: "light" }),
}));

vi.mock("@/components/image", () => ({
  LazyImage: ({ alt }: { alt: string }) => <span>{alt}</span>,
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}));

const READ = `${Resource.AccountingIntegration}:${Operation.Read}`;
const UPDATE = `${Resource.AccountingIntegration}:${Operation.Update}`;
const MANAGE = `${Resource.AccountingIntegration}:${Operation.Manage}`;

const connected: AccountingConnection = {
  id: "acctc_1",
  integrationType: "QuickBooksOnline",
  status: "Connected",
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
  version: 1,
  updatedAt: 1_780_000_000,
};

function status(overrides: Partial<AccountingSyncStatus> = {}): AccountingSyncStatus {
  return {
    integrationType: "QuickBooksOnline",
    providerName: "QuickBooks Online",
    available: true,
    connection: null,
    ...overrides,
  };
}

function renderModal(props: { justConnected?: boolean } = {}) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <QuickBooksIntegrationModal
        open
        onOpenChange={() => undefined}
        justConnected={props.justConnected ?? false}
        onReviewed={() => undefined}
      />
    </QueryClientProvider>,
  );
}

describe("QuickBooksIntegrationModal", () => {
  beforeEach(() => {
    for (const mock of Object.values(mocks)) {
      if (typeof mock === "function") mock.mockReset();
    }
    mocks.granted = new Set([READ, UPDATE, MANAGE]);
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { ...window.location, assign: mocks.assign },
    });
  });

  it("starts at the connect step before the organization has ever connected", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status());

    renderModal();

    const connect = await screen.findByRole("button", { name: /Connect to QuickBooks Online/ });
    expect(connect).toBeEnabled();
    expect(screen.getByRole("list", { name: "QuickBooks Online setup" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Check now" })).not.toBeInTheDocument();
  });

  it("goes back to the connect step after a disconnect and names the company it was connected to", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(
      status({ connection: { ...connected, status: "Disconnected" } }),
    );

    renderModal();

    expect(
      await screen.findByText(
        "This organization was connected to Peak Freight before. Connecting the same company picks up where it left off.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Connect to QuickBooks Online/ })).toBeEnabled();
  });

  it("cannot connect while the server has no QuickBooks app credentials", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ available: false }));

    renderModal();

    expect(
      await screen.findByRole("button", { name: /Connect to QuickBooks Online/ }),
    ).toBeDisabled();
    expect(screen.getByText(/is not set up on this Trenova server yet/)).toBeInTheDocument();
  });

  it("cannot connect without manage access", async () => {
    mocks.granted = new Set([READ, UPDATE]);
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status());

    renderModal();

    expect(
      await screen.findByRole("button", { name: /Connect to QuickBooks Online/ }),
    ).toBeDisabled();
    expect(
      screen.getByText(
        "Connecting QuickBooks Online needs manage access to the accounting integration.",
      ),
    ).toBeInTheDocument();
  });

  it("sends the person to Intuit's authorize page", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status());
    mocks.startAccountingAuthorization.mockResolvedValue({
      authorizeUrl: "https://appcenter.intuit.com/connect/oauth2?state=s1",
      expiresAt: 1,
    });

    renderModal();
    await userEvent.click(
      await screen.findByRole("button", { name: /Connect to QuickBooks Online/ }),
    );

    await waitFor(() =>
      expect(mocks.assign).toHaveBeenCalledWith(
        "https://appcenter.intuit.com/connect/oauth2?state=s1",
      ),
    );
  });

  it("never opens an authorize address that is not Intuit's", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status());
    mocks.startAccountingAuthorization.mockResolvedValue({
      authorizeUrl: "https://appcenter.intuit.com.example.org/connect",
      expiresAt: 1,
    });

    renderModal();
    await userEvent.click(
      await screen.findByRole("button", { name: /Connect to QuickBooks Online/ }),
    );

    await waitFor(() => expect(mocks.startAccountingAuthorization).toHaveBeenCalledTimes(1));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /Connect to QuickBooks Online/ })).toBeEnabled(),
    );
    expect(mocks.assign).not.toHaveBeenCalled();
  });

  it("opens at the review step straight after the provider sends the person back", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ connection: connected }));

    renderModal({ justConnected: true });

    expect(await screen.findByText("Connected to Peak Freight")).toBeInTheDocument();
    expect(screen.getByText("Peak Freight LLC")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument();
  });

  it("shows the connection's health, with Check now and Disconnect, once connected", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ connection: connected }));

    renderModal();

    expect(await screen.findByRole("button", { name: "Check now" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Disconnect" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reconnect" })).not.toBeInTheDocument();
    expect(screen.getByText("No closing date")).toBeInTheDocument();
  });

  it("offers Reconnect, not Check now, once the provider revoked the authorization", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(
      status({ connection: { ...connected, status: "Revoked" } }),
    );

    renderModal();

    expect(await screen.findByRole("button", { name: "Reconnect" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Check now" })).not.toBeInTheDocument();
    expect(screen.getByText("Needs reconnecting")).toBeInTheDocument();
  });

  it("shows the provider's plain-language reason when calls are failing", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(
      status({
        connection: {
          ...connected,
          status: "Failing",
          consecutiveFailures: 3,
          lastErrorMessage: "QuickBooks Online did not answer in time.",
        },
      }),
    );

    renderModal();

    expect(await screen.findByText("3 calls in a row have failed")).toBeInTheDocument();
    expect(screen.getByText("QuickBooks Online did not answer in time.")).toBeInTheDocument();
  });

  it("hides the manage actions from someone who may only check the connection", async () => {
    mocks.granted = new Set([READ, UPDATE]);
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ connection: connected }));

    renderModal();

    expect(await screen.findByRole("button", { name: "Check now" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Disconnect" })).not.toBeInTheDocument();
  });

  it("says so instead of querying when the person may not read the integration", async () => {
    mocks.granted = new Set();

    renderModal();

    expect(
      await screen.findByText("You do not have permission to view the accounting integration."),
    ).toBeInTheDocument();
    expect(mocks.fetchAccountingSyncStatus).not.toHaveBeenCalled();
  });
});
