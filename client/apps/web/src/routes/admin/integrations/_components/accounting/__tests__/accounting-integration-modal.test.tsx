import type {
  AccountingConnection,
  AccountingMapping,
  AccountingMappingSummary,
  AccountingSyncStatus,
} from "@/lib/graphql/accounting-sync";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { QuickBooksIntegrationModal } from "../accounting-integration-modal";

const mocks = vi.hoisted(() => ({
  fetchAccountingSyncStatus: vi.fn(),
  startAccountingAuthorization: vi.fn(),
  checkAccountingConnection: vi.fn(),
  disconnectAccountingSystem: vi.fn(),
  completeAccountingAuthorization: vi.fn(),
  fetchAccountingMappingSummary: vi.fn(),
  fetchAccountingMappings: vi.fn(),
  searchAccountingReferenceObjects: vi.fn(),
  confirmAccountingMappings: vi.fn(),
  completeAccountingSetup: vi.fn(),
  refreshAccountingReferenceData: vi.fn(),
  granted: new Set<string>(),
  assign: vi.fn(),
}));

vi.mock("@/lib/graphql/accounting-sync", () => ({
  fetchAccountingSyncStatus: mocks.fetchAccountingSyncStatus,
  startAccountingAuthorization: mocks.startAccountingAuthorization,
  checkAccountingConnection: mocks.checkAccountingConnection,
  disconnectAccountingSystem: mocks.disconnectAccountingSystem,
  completeAccountingAuthorization: mocks.completeAccountingAuthorization,
  fetchAccountingMappingSummary: mocks.fetchAccountingMappingSummary,
  fetchAccountingMappings: mocks.fetchAccountingMappings,
  searchAccountingReferenceObjects: mocks.searchAccountingReferenceObjects,
  confirmAccountingMappings: mocks.confirmAccountingMappings,
  completeAccountingSetup: mocks.completeAccountingSetup,
  refreshAccountingReferenceData: mocks.refreshAccountingReferenceData,
  rejectAccountingMapping: vi.fn(),
  setAccountingMapping: vi.fn(),
  clearAccountingMapping: vi.fn(),
  createAccountingReferenceRecord: vi.fn(),
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
  setupStep: "Complete",
  referenceRefreshStartedAt: null,
  referenceRefreshedAt: 1_780_000_100,
  referenceRefreshError: "",
  version: 1,
  updatedAt: 1_780_000_000,
};

const mapping = { ...connected, setupStep: "Mappings" as const };

function mappingRow(overrides: Partial<AccountingMapping>): AccountingMapping {
  return {
    id: "acctm_1",
    targetType: "AccountRole",
    trenovaObjectId: null,
    trenovaKey: "ARAccount",
    targetLabel: "Accounts receivable",
    providerKind: "Account",
    externalId: "",
    externalName: "",
    state: "Unmatched",
    source: null,
    confidence: null,
    reason: "",
    required: true,
    prechecked: false,
    candidates: [],
    confirmedBy: null,
    confirmedAt: null,
    version: 1,
    updatedAt: 1_780_000_000,
    ...overrides,
  };
}

function summary(overrides: Partial<AccountingMappingSummary> = {}): AccountingMappingSummary {
  return {
    integrationType: "QuickBooksOnline",
    providerName: "QuickBooks Online",
    connection: mapping,
    groups: [],
    requiredTotal: 4,
    requiredConfirmed: 1,
    canCompleteSetup: false,
    ...overrides,
  };
}

function mappingPages(required: AccountingMapping[], proposed: AccountingMapping[]) {
  mocks.fetchAccountingMappings.mockImplementation(
    ({ filter }: { filter: { requiredOnly?: boolean } }) =>
      Promise.resolve({
        mappings: filter.requiredOnly ? required : proposed,
        hasNextPage: false,
        endCursor: null,
      }),
  );
}

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
  return renderWithRouter(props);
}

function renderWithRouter(props: { justConnected?: boolean }) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <QuickBooksIntegrationModal
          open
          onOpenChange={() => undefined}
          justConnected={props.justConnected ?? false}
          onReviewed={() => undefined}
        />
      </QueryClientProvider>
    </MemoryRouter>,
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

  it("resumes at the match step while the required mappings are not finished", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ connection: mapping }));
    mocks.fetchAccountingMappingSummary.mockResolvedValue(summary());
    mappingPages(
      [
        mappingRow({
          id: "acctm_ar",
          state: "Confirmed",
          externalName: "Accounts Receivable (A/R)",
        }),
        mappingRow({ id: "acctm_rev", targetLabel: "Revenue", trenovaKey: "RevenueAccount" }),
      ],
      [],
    );

    renderModal();

    expect(await screen.findByText("Match your records")).toBeInTheDocument();
    expect(screen.getByText("1 of 4 required mappings confirmed")).toBeInTheDocument();
    expect(await screen.findByText("Revenue")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Finish setup" })).toBeDisabled();
    expect(screen.queryByRole("button", { name: "Check now" })).not.toBeInTheDocument();
  });

  it("confirms the proposals that are ticked, starting from the ones Trenova is sure of", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ connection: mapping }));
    mocks.fetchAccountingMappingSummary.mockResolvedValue(summary());
    mocks.confirmAccountingMappings.mockResolvedValue([]);
    mappingPages(
      [],
      [
        mappingRow({
          id: "acctm_sure",
          state: "Proposed",
          source: "Suggested",
          prechecked: true,
          targetLabel: "Freight charges",
          externalId: "90",
          externalName: "Freight",
          required: false,
        }),
        mappingRow({
          id: "acctm_unsure",
          state: "Proposed",
          source: "Model",
          prechecked: false,
          targetLabel: "Acme Logistics",
          externalId: "50",
          externalName: "Acme Logistics, Inc.",
          required: false,
        }),
      ],
    );
    const user = userEvent.setup();

    renderModal();

    const confirm = await screen.findByRole("button", { name: "Confirm 1 checked" });
    await user.click(screen.getByRole("checkbox", { name: "Confirm Acme Logistics" }));
    await user.click(await screen.findByRole("button", { name: "Confirm 2 checked" }));

    await waitFor(() =>
      expect(mocks.confirmAccountingMappings).toHaveBeenCalledWith([
        { id: "acctm_sure", externalId: "90" },
        { id: "acctm_unsure", externalId: "50" },
      ]),
    );
    expect(confirm).toBeInTheDocument();
  });

  it("finishes setup once every required mapping is confirmed", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ connection: mapping }));
    mocks.fetchAccountingMappingSummary.mockResolvedValue(
      summary({ requiredConfirmed: 4, canCompleteSetup: true }),
    );
    mocks.completeAccountingSetup.mockResolvedValue({ ...connected });
    mappingPages([], []);
    const user = userEvent.setup();

    renderModal();

    await user.click(await screen.findByRole("button", { name: "Finish setup" }));
    await waitFor(() =>
      expect(mocks.completeAccountingSetup).toHaveBeenCalledWith("QuickBooksOnline"),
    );
  });

  it("says when the last read of QuickBooks failed and offers to read it again", async () => {
    const failed = { ...mapping, referenceRefreshError: "QuickBooks did not answer" };
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ connection: failed }));
    mocks.fetchAccountingMappingSummary.mockResolvedValue(summary({ connection: failed }));
    mocks.refreshAccountingReferenceData.mockResolvedValue(failed);
    mappingPages([], []);
    const user = userEvent.setup();

    renderModal();

    expect(
      await screen.findByText(
        "The last read of QuickBooks Online failed: QuickBooks did not answer",
      ),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Read QuickBooks Online again/ }));
    await waitFor(() =>
      expect(mocks.refreshAccountingReferenceData).toHaveBeenCalledWith("QuickBooksOnline"),
    );
  });

  it("continues from the company review to the match step", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ connection: mapping }));
    mocks.fetchAccountingMappingSummary.mockResolvedValue(summary());
    mappingPages([], []);

    renderModal({ justConnected: true });

    expect(await screen.findByRole("button", { name: "Continue" })).toBeInTheDocument();
  });
});
