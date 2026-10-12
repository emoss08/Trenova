import type { AccountingSyncStatus } from "@/lib/graphql/accounting-sync";
import { quickBooksProfile, xeroProfile } from "@/test/accounting-profiles";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { StrictMode } from "react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AccountingAuthorizationCallback } from "../accounting-authorization-callback";
import { quickBooksVendor, xeroVendor, type AccountingVendor } from "../accounting-vendors";

const mocks = vi.hoisted(() => ({
  finishAccountingAuthorization: vi.fn(),
  chooseAccountingCompany: vi.fn(),
  fetchAccountingSyncStatus: vi.fn(),
  toastSuccess: vi.fn(),
}));

vi.mock("@/lib/graphql/accounting-sync", () => ({
  finishAccountingAuthorization: mocks.finishAccountingAuthorization,
  chooseAccountingCompany: mocks.chooseAccountingCompany,
  fetchAccountingSyncStatus: mocks.fetchAccountingSyncStatus,
}));

vi.mock("sonner", () => ({ toast: { success: mocks.toastSuccess, error: vi.fn() } }));

const QUICKBOOKS_CALLBACK = "/admin/integrations/quickbooks/callback";
const XERO_CALLBACK = "/admin/integrations/xero/callback";

function Landing() {
  const location = useLocation();
  return <p data-testid="landed">{`${location.pathname}${location.search}`}</p>;
}

function AddressBar() {
  const location = useLocation();
  return <p data-testid="address">{`${location.pathname}${location.search}`}</p>;
}

function renderCallback(
  search: string,
  vendor: AccountingVendor = quickBooksVendor,
  callbackPath = QUICKBOOKS_CALLBACK,
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <StrictMode>
      <QueryClientProvider client={client}>
        <MemoryRouter initialEntries={[`${callbackPath}${search}`]}>
          <Routes>
            <Route
              path={callbackPath}
              element={
                <>
                  <AccountingAuthorizationCallback vendor={vendor} />
                  <AddressBar />
                </>
              }
            />
            <Route path="/admin/integrations" element={<Landing />} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    </StrictMode>,
  );
}

function renderXero(search: string) {
  return renderCallback(search, xeroVendor, XERO_CALLBACK);
}

function statusFor(system: "QuickBooksOnline" | "Xero"): AccountingSyncStatus {
  return {
    integrationType: system,
    providerName: system === "Xero" ? "Xero" : "QuickBooks Online",
    profile: system === "Xero" ? xeroProfile : quickBooksProfile,
    available: true,
    app: {
      activeSource: "Instance",
      instanceAppAvailable: true,
      instanceEnvironment: "Production",
      redirectUrl: `http://localhost:5173${system === "Xero" ? XERO_CALLBACK : QUICKBOOKS_CALLBACK}`,
      webhookPath: "",
      tenantApp: null,
    },
    connection: null,
    webhookSubscriptions: null,
  };
}

function businessError(message: string) {
  return new GraphQLRequestError({
    graphQLErrors: [{ message, code: "BUSINESS_LOGIC", extensions: { code: "BUSINESS_LOGIC" } }],
    kind: "graphql",
    message,
    status: 200,
  });
}

const REALM_TAKEN =
  "This QuickBooks Online company is already connected to another Trenova organization. Disconnect it there first.";

const connection = { id: "acctc_1", status: "Connected", externalCompanyName: "Peak Freight" };
const connected = { kind: "connected", connection };

const NOW = 1_800_000_000;
const choice = {
  kind: "choose",
  companies: [
    { id: "tenant-a", name: "Peak Freight Ltd" },
    { id: "tenant-b", name: "Peak Freight Holdings" },
  ],
  choiceToken: "choice-token-1",
  choiceExpiresAt: NOW + 600,
};

describe("AccountingAuthorizationCallback", () => {
  beforeEach(() => {
    mocks.finishAccountingAuthorization.mockReset();
    mocks.chooseAccountingCompany.mockReset();
    mocks.fetchAccountingSyncStatus.mockReset();
    mocks.toastSuccess.mockReset();
    mocks.fetchAccountingSyncStatus.mockImplementation((system: "QuickBooksOnline" | "Xero") =>
      Promise.resolve(statusFor(system)),
    );
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("finishes the connection once, even under StrictMode's double effects, and opens the review step", async () => {
    mocks.finishAccountingAuthorization.mockResolvedValue(connected);

    renderCallback("?code=c1&state=s1&realmId=9130348");

    expect(await screen.findByTestId("landed")).toHaveTextContent(
      "/admin/integrations?type=QuickBooksOnline&setup=connected",
    );
    expect(mocks.finishAccountingAuthorization).toHaveBeenCalledTimes(1);
    expect(mocks.finishAccountingAuthorization).toHaveBeenCalledWith({
      integrationType: "QuickBooksOnline",
      code: "c1",
      state: "s1",
      realmId: "9130348",
    });
    expect(mocks.toastSuccess).toHaveBeenCalledTimes(1);
  });

  it("says the person declined and sends nothing when the provider reports access_denied", async () => {
    renderCallback("?error=access_denied&state=s1");

    expect(
      await screen.findByText(
        "Access to QuickBooks Online was not granted, so nothing was connected.",
      ),
    ).toBeInTheDocument();
    expect(mocks.finishAccountingAuthorization).not.toHaveBeenCalled();
    expect(mocks.fetchAccountingSyncStatus).not.toHaveBeenCalled();
  });

  it("names the provider's error code for any other refusal", async () => {
    renderCallback("?error=invalid_scope&state=s1");

    expect(
      await screen.findByText(
        "QuickBooks Online returned an error (invalid_scope). Nothing was connected.",
      ),
    ).toBeInTheDocument();
    expect(mocks.finishAccountingAuthorization).not.toHaveBeenCalled();
  });

  it("refuses a QuickBooks return that is missing the realm, without finishing", async () => {
    renderCallback("?code=c1&state=s1");

    expect(
      await screen.findByText(
        "This page was opened without everything QuickBooks Online sends back. Start the connection again from Integrations.",
      ),
    ).toBeInTheDocument();
    expect(mocks.finishAccountingAuthorization).not.toHaveBeenCalled();
  });

  it("shows the server's reason when it refuses the connection and stays on the page", async () => {
    mocks.finishAccountingAuthorization.mockRejectedValue(businessError(REALM_TAKEN));

    renderCallback("?code=c1&state=s1&realmId=9130348");

    expect(await screen.findByText(REALM_TAKEN)).toBeInTheDocument();
    expect(screen.queryByTestId("landed")).not.toBeInTheDocument();
    await waitFor(() => expect(mocks.finishAccountingAuthorization).toHaveBeenCalledTimes(1));
  });

  it("removes the one-time code from the address bar while it works", async () => {
    let resolve: (value: unknown) => void = () => undefined;
    mocks.finishAccountingAuthorization.mockReturnValue(
      new Promise((r) => {
        resolve = r;
      }),
    );

    renderCallback("?code=c1&state=s1&realmId=9130348");

    expect(
      await screen.findByText("Finishing the connection to QuickBooks Online..."),
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByTestId("address")).toHaveTextContent(QUICKBOOKS_CALLBACK),
    );
    expect(screen.getByTestId("address").textContent).toBe(QUICKBOOKS_CALLBACK);
    await waitFor(() => expect(mocks.finishAccountingAuthorization).toHaveBeenCalledTimes(1));
    resolve(connected);
    expect(await screen.findByTestId("landed")).toBeInTheDocument();
  });

  it("says why when the provider's rules could not be loaded, without finishing", async () => {
    mocks.fetchAccountingSyncStatus.mockRejectedValue(
      businessError("The accounting integration is not available."),
    );

    renderXero("?code=c1&state=s1");

    expect(
      await screen.findByText("The accounting integration is not available."),
    ).toBeInTheDocument();
    expect(screen.getByText("Xero was not connected")).toBeInTheDocument();
    expect(mocks.finishAccountingAuthorization).not.toHaveBeenCalled();
  });

  it("finishes a Xero return that carries no company, sending no realm", async () => {
    mocks.finishAccountingAuthorization.mockResolvedValue(connected);

    renderXero("?code=x1&state=s1&scope=openid");

    expect(await screen.findByTestId("landed")).toHaveTextContent(
      "/admin/integrations?type=Xero&setup=connected",
    );
    expect(mocks.finishAccountingAuthorization).toHaveBeenCalledTimes(1);
    expect(mocks.finishAccountingAuthorization).toHaveBeenCalledWith({
      integrationType: "Xero",
      code: "x1",
      state: "s1",
      realmId: null,
    });
  });

  it("asks which organisation to connect when the sign-in covered several, then connects that one", async () => {
    vi.useFakeTimers({ now: NOW * 1000, toFake: ["Date"] });
    mocks.finishAccountingAuthorization.mockResolvedValue(choice);
    mocks.chooseAccountingCompany.mockResolvedValue(connection);
    const user = userEvent.setup();

    renderXero("?code=x1&state=s1");

    expect(await screen.findByText("Choose the organisation")).toBeInTheDocument();
    const group = screen.getByRole("radiogroup", { name: "Organisations in Xero" });
    expect(group).toBeInTheDocument();
    expect(
      screen.getByText(/^Choose by .+\. After that, start the connection again\.$/),
    ).toBeInTheDocument();
    const connect = screen.getByRole("button", { name: "Connect this organisation" });
    expect(connect).toBeDisabled();

    await user.click(screen.getByRole("radio", { name: "Peak Freight Holdings" }));
    expect(screen.getByRole("radio", { name: "Peak Freight Holdings" })).toHaveAttribute(
      "aria-checked",
      "true",
    );
    await user.click(connect);

    expect(await screen.findByTestId("landed")).toHaveTextContent(
      "/admin/integrations?type=Xero&setup=connected",
    );
    expect(mocks.chooseAccountingCompany).toHaveBeenCalledTimes(1);
    expect(mocks.chooseAccountingCompany).toHaveBeenCalledWith({
      integrationType: "Xero",
      choiceToken: "choice-token-1",
      companyId: "tenant-b",
    });
    expect(mocks.finishAccountingAuthorization).toHaveBeenCalledTimes(1);
  });

  it("keeps the choice open and shows the reason when the server refuses the organisation", async () => {
    vi.useFakeTimers({ now: NOW * 1000, toFake: ["Date"] });
    const taken = "This Xero organisation is already connected to another Trenova organization.";
    mocks.finishAccountingAuthorization.mockResolvedValue(choice);
    mocks.chooseAccountingCompany.mockRejectedValue(businessError(taken));
    const user = userEvent.setup();

    renderXero("?code=x1&state=s1");

    await user.click(await screen.findByRole("radio", { name: "Peak Freight Ltd" }));
    await user.click(screen.getByRole("button", { name: "Connect this organisation" }));

    expect(await screen.findByText(taken)).toBeInTheDocument();
    expect(screen.getByText("Choose the organisation")).toBeInTheDocument();
    expect(screen.queryByTestId("landed")).not.toBeInTheDocument();
  });

  it("says the choice ran out instead of offering organisations once it has expired", async () => {
    vi.useFakeTimers({ now: (NOW + 601) * 1000, toFake: ["Date"] });
    mocks.finishAccountingAuthorization.mockResolvedValue(choice);

    renderXero("?code=x1&state=s1");

    expect(
      await screen.findByText(
        "The time to choose an organisation ran out, so nothing was connected. Start the connection again from Integrations.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole("radiogroup")).not.toBeInTheDocument();
    expect(mocks.chooseAccountingCompany).not.toHaveBeenCalled();
  });

  it("says nothing was connected when the sign-in covered no organisation", async () => {
    mocks.finishAccountingAuthorization.mockResolvedValue({ kind: "none" });

    renderXero("?code=x1&state=s1");

    expect(
      await screen.findByText(
        "Xero did not give Trenova access to any organisation, so nothing was connected.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("landed")).not.toBeInTheDocument();
  });
});
