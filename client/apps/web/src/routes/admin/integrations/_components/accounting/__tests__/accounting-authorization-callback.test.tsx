import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { StrictMode } from "react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AccountingAuthorizationCallback } from "../accounting-authorization-callback";
import { quickBooksVendor } from "../accounting-vendors";

const mocks = vi.hoisted(() => ({
  completeAccountingAuthorization: vi.fn(),
  toastSuccess: vi.fn(),
}));

vi.mock("@/lib/graphql/accounting-sync", () => ({
  completeAccountingAuthorization: mocks.completeAccountingAuthorization,
}));

vi.mock("sonner", () => ({ toast: { success: mocks.toastSuccess, error: vi.fn() } }));

const CALLBACK = "/admin/integrations/quickbooks/callback";

function Landing() {
  const location = useLocation();
  return <p data-testid="landed">{`${location.pathname}${location.search}`}</p>;
}

function AddressBar() {
  const location = useLocation();
  return <p data-testid="address">{`${location.pathname}${location.search}`}</p>;
}

function renderCallback(search: string) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return render(
    <StrictMode>
      <QueryClientProvider client={client}>
        <MemoryRouter initialEntries={[`${CALLBACK}${search}`]}>
          <Routes>
            <Route
              path={CALLBACK}
              element={
                <>
                  <AccountingAuthorizationCallback vendor={quickBooksVendor} />
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

const REALM_TAKEN =
  "This QuickBooks Online company is already connected to another Trenova organization. Disconnect it there first.";

const connection = { id: "acctc_1", status: "Connected", externalCompanyName: "Peak Freight" };

describe("AccountingAuthorizationCallback", () => {
  beforeEach(() => {
    mocks.completeAccountingAuthorization.mockReset();
    mocks.toastSuccess.mockReset();
  });

  it("finishes the connection once, even under StrictMode's double effects, and opens the review step", async () => {
    mocks.completeAccountingAuthorization.mockResolvedValue(connection);

    renderCallback("?code=c1&state=s1&realmId=9130348");

    expect(await screen.findByTestId("landed")).toHaveTextContent(
      "/admin/integrations?type=QuickBooksOnline&setup=connected",
    );
    expect(mocks.completeAccountingAuthorization).toHaveBeenCalledTimes(1);
    expect(mocks.completeAccountingAuthorization).toHaveBeenCalledWith({
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
    expect(mocks.completeAccountingAuthorization).not.toHaveBeenCalled();
  });

  it("names the provider's error code for any other refusal", async () => {
    renderCallback("?error=invalid_scope&state=s1");

    expect(
      await screen.findByText(
        "QuickBooks Online returned an error (invalid_scope). Nothing was connected.",
      ),
    ).toBeInTheDocument();
    expect(mocks.completeAccountingAuthorization).not.toHaveBeenCalled();
  });

  it("refuses a return that is missing the realm, without calling the server", async () => {
    renderCallback("?code=c1&state=s1");

    expect(
      await screen.findByText(
        "This page was opened without everything QuickBooks Online sends back. Start the connection again from Integrations.",
      ),
    ).toBeInTheDocument();
    expect(mocks.completeAccountingAuthorization).not.toHaveBeenCalled();
  });

  it("shows the server's reason when it refuses the connection and stays on the page", async () => {
    mocks.completeAccountingAuthorization.mockRejectedValue(
      new GraphQLRequestError({
        graphQLErrors: [
          {
            message: REALM_TAKEN,
            code: "BUSINESS_LOGIC",
            extensions: { code: "BUSINESS_LOGIC" },
          },
        ],
        kind: "graphql",
        message: REALM_TAKEN,
        status: 200,
      }),
    );

    renderCallback("?code=c1&state=s1&realmId=9130348");

    expect(await screen.findByText(REALM_TAKEN)).toBeInTheDocument();
    expect(screen.queryByTestId("landed")).not.toBeInTheDocument();
    await waitFor(() => expect(mocks.completeAccountingAuthorization).toHaveBeenCalledTimes(1));
  });

  it("removes the one-time code from the address bar while it works", async () => {
    let resolve: (value: unknown) => void = () => undefined;
    mocks.completeAccountingAuthorization.mockReturnValue(
      new Promise((r) => {
        resolve = r;
      }),
    );

    renderCallback("?code=c1&state=s1&realmId=9130348");

    expect(
      await screen.findByText("Finishing the connection to QuickBooks Online..."),
    ).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId("address")).toHaveTextContent(CALLBACK));
    expect(screen.getByTestId("address").textContent).toBe(CALLBACK);
    expect(mocks.completeAccountingAuthorization).toHaveBeenCalledTimes(1);
    resolve(connection);
    expect(await screen.findByTestId("landed")).toBeInTheDocument();
  });
});
