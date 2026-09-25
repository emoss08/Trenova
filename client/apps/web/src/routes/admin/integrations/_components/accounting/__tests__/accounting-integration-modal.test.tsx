import type {
  AccountingAppSettings,
  AccountingConnection,
  AccountingMapping,
  AccountingMappingSummary,
  AccountingSyncStatus,
} from "@/lib/graphql/accounting-sync";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { render, screen, waitFor, within } from "@testing-library/react";
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
  saveAccountingApp: vi.fn(),
  removeAccountingApp: vi.fn(),
  enableAccountingSync: vi.fn(),
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
  saveAccountingApp: mocks.saveAccountingApp,
  removeAccountingApp: mocks.removeAccountingApp,
  rejectAccountingMapping: vi.fn(),
  setAccountingMapping: vi.fn(),
  clearAccountingMapping: vi.fn(),
  createAccountingReferenceRecord: vi.fn(),
}));

vi.mock("@/lib/graphql/accounting-sync-ledger", () => ({
  enableAccountingSync: mocks.enableAccountingSync,
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
  syncStartDate: 1_780_000_000,
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

const mapping = { ...connected, setupStep: "Mappings" as const };
const startDate = {
  ...connected,
  setupStep: "StartDate" as const,
  syncStartDate: null,
  syncEnabledAt: null,
};

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

const REDIRECT_URL = "http://localhost:5173/admin/integrations/quickbooks/callback";

const instanceApp: AccountingAppSettings = {
  activeSource: "Instance",
  instanceAppAvailable: true,
  instanceEnvironment: "Production",
  redirectUrl: REDIRECT_URL,
  webhookPath: "/webhooks/accounting/quickbooks/",
  tenantApp: null,
};

const noApp: AccountingAppSettings = {
  ...instanceApp,
  activeSource: null,
  instanceAppAvailable: false,
  instanceEnvironment: null,
};

const tenantApp: AccountingAppSettings = {
  ...noApp,
  activeSource: "Tenant",
  tenantApp: {
    id: "acctapp_1",
    integrationType: "QuickBooksOnline",
    environment: "Sandbox",
    clientId: "ABtenantClient",
    hasWebhookVerifier: true,
    version: 2,
    updatedAt: 1_780_000_000,
  },
};

function status(overrides: Partial<AccountingSyncStatus> = {}): AccountingSyncStatus {
  return {
    integrationType: "QuickBooksOnline",
    providerName: "QuickBooks Online",
    available: true,
    app: instanceApp,
    connection: null,
    ...overrides,
  };
}

async function field(name: string): Promise<HTMLElement> {
  return waitFor(() => {
    const element = document.getElementById(`input-${name}`);
    if (!element) {
      throw new Error(`input-${name} not rendered`);
    }
    return element;
  });
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

  it("asks for the organization's own app keys when the server has no app", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ available: false, app: noApp }));

    renderModal();

    expect(
      await screen.findByRole("button", { name: /Connect to QuickBooks Online/ }),
    ).toBeDisabled();
    expect(screen.getByText(/has no Intuit app of its own/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save keys" })).toBeInTheDocument();
    expect(screen.getByText(REDIRECT_URL)).toBeInTheDocument();
    expect(screen.getByText(/\/webhooks\/accounting\/quickbooks\/$/)).toBeInTheDocument();
    expect(screen.queryByText(/is not set up on this Trenova server yet/)).not.toBeInTheDocument();
  });

  it("saves the keys and makes connecting possible", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ available: false, app: noApp }));
    mocks.saveAccountingApp.mockResolvedValue(status({ available: true, app: tenantApp }));

    renderModal();
    await userEvent.type(await field("clientId"), "  ABtenantClient ");
    await userEvent.type(await field("clientSecret"), "s3cret");
    await userEvent.click(screen.getByRole("button", { name: "Save keys" }));

    await waitFor(() =>
      expect(mocks.saveAccountingApp).toHaveBeenCalledWith({
        integrationType: "QuickBooksOnline",
        environment: "Sandbox",
        clientId: "ABtenantClient",
        clientSecret: "s3cret",
        webhookVerifierToken: null,
        clearWebhookVerifierToken: false,
      }),
    );
    expect(
      await screen.findByRole("button", { name: /Connect to QuickBooks Online/ }),
    ).toBeEnabled();
    expect(screen.getByText("ABtenantClient")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Save keys" })).not.toBeInTheDocument();
  });

  it("will not save a new app without its secret", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ available: false, app: noApp }));

    renderModal();
    await userEvent.type(await field("clientId"), "ABtenantClient");
    await userEvent.click(screen.getByRole("button", { name: "Save keys" }));

    expect(
      await screen.findByText("Enter the client secret that goes with this client ID"),
    ).toBeInTheDocument();
    expect(mocks.saveAccountingApp).not.toHaveBeenCalled();
  });

  it("shows the server's error on the field it names", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ available: false, app: noApp }));
    mocks.saveAccountingApp.mockRejectedValue(
      new ApiRequestError(422, {
        type: "https://api.trenova.app/problems/validation-error",
        title: "Validation error",
        status: 422,
        detail: "Validation failed",
        errors: [
          {
            field: "clientSecret",
            code: "INVALID",
            message: "QuickBooks Online did not accept this client ID and secret.",
          },
        ],
      }),
    );

    renderModal();
    await userEvent.type(await field("clientId"), "ABtenantClient");
    await userEvent.type(await field("clientSecret"), "wrong");
    await userEvent.click(screen.getByRole("button", { name: "Save keys" }));

    expect(
      await screen.findByText("QuickBooks Online did not accept this client ID and secret."),
    ).toBeInTheDocument();
  });

  it("connects through the server's app and offers the organization's own instead", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status());

    renderModal();

    expect(
      await screen.findByText(/Connects through the Intuit app this server/),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Save keys" })).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Use your own app" }));
    expect(screen.getByRole("button", { name: "Save keys" })).toBeInTheDocument();
    const keys = screen.getByRole("region", { name: "Intuit app" });
    await userEvent.click(within(keys).getByRole("button", { name: "Cancel" }));
    expect(screen.queryByRole("button", { name: "Save keys" })).not.toBeInTheDocument();
  });

  it("keeps the saved secret when changing only the verifier token", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ app: tenantApp }));
    mocks.saveAccountingApp.mockResolvedValue(status({ app: tenantApp }));

    renderModal();
    await userEvent.click(await screen.findByRole("button", { name: "Change keys" }));
    expect(await field("clientId")).toHaveValue("ABtenantClient");
    await userEvent.type(await field("webhookVerifierToken"), "new-verifier");
    await userEvent.click(screen.getByRole("button", { name: "Save keys" }));

    await waitFor(() =>
      expect(mocks.saveAccountingApp).toHaveBeenCalledWith({
        integrationType: "QuickBooksOnline",
        environment: "Sandbox",
        clientId: "ABtenantClient",
        clientSecret: null,
        webhookVerifierToken: "new-verifier",
        clearWebhookVerifierToken: false,
      }),
    );
  });

  it("removes the organization's keys after confirming", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ app: tenantApp }));
    mocks.removeAccountingApp.mockResolvedValue(status());

    renderModal();
    await userEvent.click(await screen.findByRole("button", { name: "Remove" }));
    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent("Remove the Intuit app keys?");
    await userEvent.click(within(dialog).getByRole("button", { name: "Remove" }));

    await waitFor(() => expect(mocks.removeAccountingApp).toHaveBeenCalledWith("QuickBooksOnline"));
    expect(await screen.findByRole("button", { name: "Use your own app" })).toBeInTheDocument();
  });

  it("does not offer switching apps while connected through the server's app", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ connection: connected }));

    renderModal();

    expect(await screen.findByRole("button", { name: "Use your own app" })).toBeDisabled();
    expect(
      screen.getByText(/Disconnect Peak Freight before switching to your own app/),
    ).toBeInTheDocument();
  });

  it("locks the client ID but not the secret while connected through the organization's app", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(
      status({ app: tenantApp, connection: { ...connected, appSource: "Tenant" } }),
    );

    renderModal();

    expect(await screen.findByRole("button", { name: "Remove" })).toBeDisabled();
    await userEvent.click(screen.getByRole("button", { name: "Change keys" }));
    expect(await field("clientId")).toHaveAttribute("readonly");
    expect(await field("clientSecret")).not.toHaveAttribute("readonly");
  });

  it("asks someone with manage access to enter the keys", async () => {
    mocks.granted = new Set([READ, UPDATE]);
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ available: false, app: noApp }));

    renderModal();

    expect(
      await screen.findByText(/Someone with manage access to the accounting integration/),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Save keys" })).not.toBeInTheDocument();
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

  it("asks for a start date once the mappings are confirmed", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ connection: startDate }));
    mocks.enableAccountingSync.mockResolvedValue({ ...startDate, setupStep: "Complete" });

    renderModal();

    expect(await screen.findByText("Choose when sending starts")).toBeInTheDocument();
    expect(
      screen.queryByText("Also send documents already posted since the start date"),
    ).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Start sending" }));

    await waitFor(() => expect(mocks.enableAccountingSync).toHaveBeenCalledTimes(1));
    const [input] = mocks.enableAccountingSync.mock.calls[0] as [
      { integrationType: string; startDate: number; autoSync: boolean; backfill: boolean },
    ];
    expect(input.integrationType).toBe("QuickBooksOnline");
    expect(input.autoSync).toBe(true);
    expect(input.backfill).toBe(false);
    expect(input.startDate).toBeLessThanOrEqual(Math.floor(Date.now() / 1000));
    expect(Math.floor(Date.now() / 1000) - input.startDate).toBeLessThan(24 * 60 * 60 + 1);
  });

  it("offers a backfill when the start date is in the past", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(
      status({ connection: { ...startDate, syncStartDate: 1_780_000_000 } }),
    );
    mocks.enableAccountingSync.mockResolvedValue({ ...startDate, setupStep: "Complete" });

    renderModal();
    await userEvent.click(
      await screen.findByText("Also send documents already posted since the start date"),
    );
    await userEvent.click(screen.getByRole("button", { name: "Start sending" }));

    await waitFor(() =>
      expect(mocks.enableAccountingSync).toHaveBeenCalledWith({
        integrationType: "QuickBooksOnline",
        startDate: 1_780_000_000,
        autoSync: true,
        backfill: true,
      }),
    );
  });

  it("warns when the start date falls in books QuickBooks has closed", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(
      status({
        connection: {
          ...startDate,
          syncStartDate: 1_780_000_000,
          externalBooksClosedThrough: 1_780_100_000,
        },
      }),
    );

    renderModal();

    expect(await screen.findByText(/has its books closed through/)).toBeInTheDocument();
  });

  it("holds each document for release when automatic sending is off", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ connection: startDate }));
    mocks.enableAccountingSync.mockResolvedValue({ ...startDate, setupStep: "Complete" });

    renderModal();
    await userEvent.click(await screen.findByRole("switch"));
    expect(
      screen.getByText("Each document waits in the sync ledger until someone releases it."),
    ).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Start sending" }));

    await waitFor(() =>
      expect(mocks.enableAccountingSync).toHaveBeenCalledWith(
        expect.objectContaining({ autoSync: false }),
      ),
    );
  });

  it("cannot choose the start date without manage access", async () => {
    mocks.granted = new Set([READ, UPDATE]);
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ connection: startDate }));

    renderModal();

    expect(await screen.findByRole("button", { name: "Start sending" })).toBeDisabled();
    expect(
      screen.getByText(
        "Choosing the start date needs manage access to the accounting integration.",
      ),
    ).toBeInTheDocument();
  });
});
