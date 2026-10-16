import type {
  AccountingAppSettings,
  AccountingConnection,
  AccountingPreparedCertificate,
  AccountingSyncStatus,
} from "@/lib/graphql/accounting-sync";
import { netSuiteProfile } from "@/test/accounting-profiles";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AccountingIntegrationModal } from "../accounting-integration-modal";
import { netSuiteVendor } from "../accounting-vendors";

const mocks = vi.hoisted(() => ({
  fetchAccountingSyncStatus: vi.fn(),
  startAccountingAuthorization: vi.fn(),
  prepareAccountingCertificate: vi.fn(),
  connectAccountingWithCertificate: vi.fn(),
  renewAccountingCertificate: vi.fn(),
  chooseAccountingCompany: vi.fn(),
  checkAccountingConnection: vi.fn(),
  disconnectAccountingSystem: vi.fn(),
  downloadTextFile: vi.fn(),
  granted: new Set<string>(),
}));

vi.mock("@/lib/graphql/accounting-sync", () => ({
  fetchAccountingSyncStatus: mocks.fetchAccountingSyncStatus,
  startAccountingAuthorization: mocks.startAccountingAuthorization,
  prepareAccountingCertificate: mocks.prepareAccountingCertificate,
  connectAccountingWithCertificate: mocks.connectAccountingWithCertificate,
  renewAccountingCertificate: mocks.renewAccountingCertificate,
  chooseAccountingCompany: mocks.chooseAccountingCompany,
  checkAccountingConnection: mocks.checkAccountingConnection,
  disconnectAccountingSystem: mocks.disconnectAccountingSystem,
  fetchAccountingMappingSummary: vi.fn(),
  fetchAccountingMappings: vi.fn(),
  searchAccountingReferenceObjects: vi.fn(),
  confirmAccountingMappings: vi.fn(),
  completeAccountingSetup: vi.fn(),
  refreshAccountingReferenceData: vi.fn(),
  saveAccountingApp: vi.fn(),
  removeAccountingApp: vi.fn(),
  rejectAccountingMapping: vi.fn(),
  setAccountingMapping: vi.fn(),
  clearAccountingMapping: vi.fn(),
  createAccountingReferenceRecord: vi.fn(),
}));

vi.mock("@/lib/graphql/accounting-sync-ledger", () => ({
  enableAccountingSync: vi.fn(),
  chooseAccountingSyncMode: vi.fn(),
  updateAccountingSyncSettings: vi.fn(),
}));

vi.mock("@trenova/shared/lib/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/utils")>()),
  downloadTextFile: mocks.downloadTextFile,
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

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}));

const READ = `${Resource.AccountingIntegration}:${Operation.Read}`;
const UPDATE = `${Resource.AccountingIntegration}:${Operation.Update}`;
const MANAGE = `${Resource.AccountingIntegration}:${Operation.Manage}`;
const DAY = 24 * 60 * 60;
const NOW = Math.floor(Date.now() / 1000);

const instanceApp: AccountingAppSettings = {
  activeSource: "Instance",
  instanceAppAvailable: true,
  instanceEnvironment: "Production",
  redirectUrl: "https://app.example.com/admin/integrations/netsuite/callback",
  webhookPath: "",
  tenantApp: null,
};

const pending: AccountingPreparedCertificate = {
  integrationType: "NetSuite",
  certificatePem: "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n",
  expiresAt: 1_861_920_000,
  preparedAt: 1_798_848_000,
};

const signedIn: AccountingConnection = {
  id: "acctc_ns",
  integrationType: "NetSuite",
  status: "Connected",
  appSource: "Instance",
  appEnvironment: "Production",
  externalCompanyName: "Peak Freight US · 1234567",
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
  refreshTokenAbsoluteExpiresAt: NOW + 6 * DAY,
  authMethod: "Authorization",
  certificateId: null,
  certificateExpiresAt: null,
  connectedAt: NOW - DAY,
  disconnectedAt: null,
  setupStep: "Complete",
  syncMode: "Document",
  ledgerGranularity: null,
  ledgerOpeningBalancesSentAt: null,
  externalFiscalYearStartMonth: 1,
  syncStartDate: NOW - DAY,
  syncEnabledAt: NOW - DAY,
  autoSync: true,
  driverSettlementsEnabledAt: null,
  syncsDriverSettlements: false,
  inboundPaymentPolicy: "Propose",
  changesReadAt: null,
  changesErrorMessage: "",
  pausedAt: null,
  pausedBy: null,
  pausedReason: "",
  referenceRefreshStartedAt: null,
  referenceRefreshedAt: NOW - DAY,
  referenceRefreshError: "",
  version: 1,
  updatedAt: NOW - DAY,
};

const certificateConnection: AccountingConnection = {
  ...signedIn,
  authMethod: "ClientCredentials",
  certificateId: "Zm9vYmFy",
  certificateExpiresAt: NOW + 10 * DAY,
  refreshTokenAbsoluteExpiresAt: NOW + 10 * DAY,
};

function status(overrides: Partial<AccountingSyncStatus> = {}): AccountingSyncStatus {
  return {
    integrationType: "NetSuite",
    providerName: "NetSuite",
    profile: netSuiteProfile,
    available: true,
    app: instanceApp,
    connection: null,
    webhookSubscriptions: null,
    pendingCertificate: null,
    ...overrides,
  };
}

function LocationProbe() {
  const location = useLocation();
  return <span data-testid="location">{`${location.pathname}${location.search}`}</span>;
}

function renderModal() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <MemoryRouter initialEntries={["/admin/integrations?type=NetSuite"]}>
      <QueryClientProvider client={client}>
        <Routes>
          <Route
            path="*"
            element={
              <>
                <AccountingIntegrationModal
                  vendor={netSuiteVendor}
                  open
                  onOpenChange={() => undefined}
                  justConnected={false}
                  onReviewed={() => undefined}
                />
                <LocationProbe />
              </>
            }
          />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  );
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

describe("NetSuite connections", () => {
  beforeEach(() => {
    for (const mock of Object.values(mocks)) {
      if (typeof mock === "function") mock.mockReset();
    }
    mocks.granted = new Set([READ, UPDATE, MANAGE]);
  });

  it("offers a sign-in and a certificate, starting with the sign-in", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status());

    renderModal();

    const signIn = await screen.findByRole("radio", { name: /Sign in to NetSuite/ });
    expect(signIn).toHaveAttribute("aria-checked", "true");
    expect(screen.getByRole("radio", { name: /Connect with a certificate/ })).toHaveAttribute(
      "aria-checked",
      "false",
    );
    expect(screen.getByRole("button", { name: /Connect to NetSuite/ })).toBeEnabled();
  });

  it("starts on the certificate when one is already waiting to be connected", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ pendingCertificate: pending }));

    renderModal();

    expect(
      await screen.findByRole("radio", { name: /Connect with a certificate/ }),
    ).toHaveAttribute("aria-checked", "true");
    expect(screen.getByText("Certificate ready to upload")).toBeInTheDocument();
  });

  it("generates a certificate on request and refuses to connect before one exists", async () => {
    const user = userEvent.setup();
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status());
    mocks.prepareAccountingCertificate.mockResolvedValue(pending);

    renderModal();
    await user.click(await screen.findByRole("radio", { name: /Connect with a certificate/ }));

    expect(screen.getByRole("button", { name: "Connect with the certificate" })).toBeDisabled();
    const fetchedBefore = mocks.fetchAccountingSyncStatus.mock.calls.filter(
      ([system]) => system === "NetSuite",
    ).length;
    await user.click(screen.getByRole("button", { name: "Generate certificate" }));

    await waitFor(() =>
      expect(mocks.prepareAccountingCertificate).toHaveBeenCalledWith("NetSuite"),
    );
    await waitFor(() =>
      expect(
        mocks.fetchAccountingSyncStatus.mock.calls.filter(([system]) => system === "NetSuite")
          .length,
      ).toBeGreaterThan(fetchedBefore),
    );
  });

  it("downloads the waiting certificate as a PEM file named for the day it was made", async () => {
    const user = userEvent.setup();
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ pendingCertificate: pending }));

    renderModal();
    await user.click(await screen.findByRole("button", { name: "Download certificate" }));

    expect(mocks.downloadTextFile).toHaveBeenCalledWith(
      "trenova-netsuite-certificate-2027-01-02.pem",
      pending.certificatePem,
      "application/x-pem-file",
    );
  });

  it("refuses an account id that is not a NetSuite account id", async () => {
    const user = userEvent.setup();
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ pendingCertificate: pending }));

    renderModal();
    await user.type(await field("accountId"), "1234567.app.netsuite.com");
    await user.type(await field("certificateId"), "Zm9vYmFy");
    await user.click(screen.getByRole("button", { name: "Connect with the certificate" }));

    expect(
      await screen.findByText("Account ID looks like 1234567, or 1234567_SB1 for a sandbox"),
    ).toBeInTheDocument();
    expect(mocks.connectAccountingWithCertificate).not.toHaveBeenCalled();
  });

  it("connects with the account id as NetSuite writes it and opens the review step", async () => {
    const user = userEvent.setup();
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ pendingCertificate: pending }));
    mocks.connectAccountingWithCertificate.mockResolvedValue({
      kind: "connected",
      connection: certificateConnection,
    });

    renderModal();
    await user.type(await field("accountId"), " 1234567_sb1 ");
    await user.type(await field("certificateId"), "Zm9vYmFy");
    await user.click(screen.getByRole("button", { name: "Connect with the certificate" }));

    await waitFor(() =>
      expect(mocks.connectAccountingWithCertificate).toHaveBeenCalledWith({
        integrationType: "NetSuite",
        accountId: "1234567_SB1",
        certificateId: "Zm9vYmFy",
      }),
    );
    await waitFor(() =>
      expect(screen.getByTestId("location")).toHaveTextContent(
        "/admin/integrations?type=NetSuite&setup=connected",
      ),
    );
  });

  it("offers the subsidiaries when the account has several and connects the chosen one", async () => {
    const user = userEvent.setup();
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ pendingCertificate: pending }));
    mocks.connectAccountingWithCertificate.mockResolvedValue({
      kind: "choose",
      companies: [
        { id: "1234567_SB1-s1", name: "Peak Freight US · 1234567_SB1 (Sandbox)" },
        { id: "1234567_SB1-s3", name: "Peak Freight CA · 1234567_SB1 (Sandbox)" },
      ],
      choiceToken: "choice-1",
      choiceExpiresAt: NOW + 600,
    });
    mocks.chooseAccountingCompany.mockResolvedValue(certificateConnection);

    renderModal();
    await user.type(await field("accountId"), "1234567_SB1");
    await user.type(await field("certificateId"), "Zm9vYmFy");
    await user.click(screen.getByRole("button", { name: "Connect with the certificate" }));

    await user.click(
      await screen.findByRole("radio", { name: "Peak Freight CA · 1234567_SB1 (Sandbox)" }),
    );
    await user.click(screen.getByRole("button", { name: "Connect this organisation" }));

    await waitFor(() =>
      expect(mocks.chooseAccountingCompany).toHaveBeenCalledWith({
        integrationType: "NetSuite",
        choiceToken: "choice-1",
        companyId: "1234567_SB1-s3",
      }),
    );
  });

  it("warns two days before a weekly sign-in runs out, not fourteen", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(status({ connection: signedIn }));

    renderModal();

    expect(await screen.findByText("Reconnect by")).toBeInTheDocument();
    expect(screen.queryByText(/ends this authorization on/)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reconnect" })).not.toBeInTheDocument();
    expect(screen.queryByText("Last notice from NetSuite")).not.toBeInTheDocument();
  });

  it("asks for a reconnect once a sign-in is within its two days", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(
      status({
        connection: { ...signedIn, refreshTokenAbsoluteExpiresAt: NOW + DAY },
      }),
    );

    renderModal();

    expect(await screen.findByText(/NetSuite ends this authorization on/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reconnect" })).toBeInTheDocument();
  });

  it("shows when a certificate expires and offers to renew it", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(
      status({ connection: certificateConnection }),
    );

    renderModal();

    expect(await screen.findByText("Certificate expires")).toBeInTheDocument();
    expect(screen.queryByText("Reconnect by")).not.toBeInTheDocument();
    expect(screen.getByText("A certificate")).toBeInTheDocument();
    expect(
      screen.getByText(/The certificate Trenova connects to NetSuite with expires on/),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Renew certificate" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reconnect" })).not.toBeInTheDocument();
  });

  it("renews a certificate with the new certificate id", async () => {
    const user = userEvent.setup();
    mocks.fetchAccountingSyncStatus.mockResolvedValue(
      status({ connection: certificateConnection, pendingCertificate: pending }),
    );
    mocks.renewAccountingCertificate.mockResolvedValue(certificateConnection);

    renderModal();
    await user.click(await screen.findByRole("button", { name: "Renew certificate" }));
    await user.type(await field("certificateId"), "bmV3");
    const dialogs = await screen.findAllByRole("dialog");
    const renew = dialogs[dialogs.length - 1];
    await user.click(
      Array.from(renew.querySelectorAll("button")).find(
        (button) => button.textContent === "Renew certificate",
      ) as HTMLElement,
    );

    await waitFor(() =>
      expect(mocks.renewAccountingCertificate).toHaveBeenCalledWith({
        connectionId: "acctc_ns",
        certificateId: "bmV3",
      }),
    );
  });

  it("tells a refused certificate apart from a revoked sign-in", async () => {
    mocks.fetchAccountingSyncStatus.mockResolvedValue(
      status({ connection: { ...certificateConnection, status: "Revoked" } }),
    );

    renderModal();

    expect(await screen.findByText(/NetSuite refused Trenova's certificate/)).toBeInTheDocument();
  });
});
