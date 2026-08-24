import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { ApiRequestError } from "@trenova/shared/lib/api";
import type { PublicRateConfirmation } from "@trenova/shared/types/rate-confirmation";
import { RateConfirmationPublicPage } from "./page";

const mocks = vi.hoisted(() => ({
  getPublicRateConfirmation: vi.fn(),
  confirmPublicRateConfirmation: vi.fn(),
}));

vi.mock("@/services/rate-confirmation", () => ({
  RateConfirmationService: class {
    getPublicRateConfirmation = mocks.getPublicRateConfirmation;
    confirmPublicRateConfirmation = mocks.confirmPublicRateConfirmation;
  },
}));

function publicView(overrides: Partial<PublicRateConfirmation> = {}): PublicRateConfirmation {
  return {
    carrierName: "Knight Logistics LLC",
    companyName: "Acme Brokerage",
    shipmentProNumber: "PRO-10422",
    revisionLabel: "Revision 2",
    rateMethodLabel: "Flat Rate",
    baseRateLabel: "$1,500.00",
    totalCost: "$1,650.00",
    currency: "USD",
    paymentTermsLabel: "Net 30",
    stops: [
      {
        sequence: 1,
        type: "Pickup",
        name: "DC 12",
        address: "100 Main St, Dallas, TX 75201",
        window: "Jun 1, 08:00 – 12:00 UTC",
      },
      {
        sequence: 2,
        type: "Delivery",
        name: "Store 44",
        address: "9 Elm Ave, Tulsa, OK 74101",
        window: "",
      },
    ],
    status: "Sent",
    expiresAt: 1_760_000_000,
    confirmed: false,
    ...overrides,
  };
}

function apiError(status: number): ApiRequestError {
  return new ApiRequestError(status, { type: "about:blank", title: "Error", status });
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={["/rate-confirmation/tok_123"]}>
          <Routes>
            <Route path="/rate-confirmation/:token" element={children} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    );
  }
  return render(<RateConfirmationPublicPage />, { wrapper: Wrapper });
}

describe("RateConfirmationPublicPage", () => {
  beforeEach(() => {
    mocks.getPublicRateConfirmation.mockReset();
    mocks.confirmPublicRateConfirmation.mockReset();
  });

  it("renders the frozen snapshot with signer inputs and a disabled confirm button", async () => {
    mocks.getPublicRateConfirmation.mockResolvedValue(publicView());

    renderPage();

    expect(
      await screen.findByText("Rate confirmation for Knight Logistics LLC"),
    ).toBeInTheDocument();
    expect(mocks.getPublicRateConfirmation).toHaveBeenCalledWith("tok_123");
    expect(screen.getByText(/PRO-10422/)).toBeInTheDocument();
    expect(screen.getByText("$1,650.00")).toBeInTheDocument();
    expect(screen.getByText(/DC 12/)).toBeInTheDocument();
    expect(screen.getByText(/Store 44/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Your name/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Title/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Confirm rate/ })).toBeDisabled();
  });

  it("shows the vague invalid-link state for a definitive 4xx preview rejection", async () => {
    mocks.getPublicRateConfirmation.mockRejectedValue(apiError(422));

    renderPage();

    expect(
      await screen.findByText("This rate confirmation link is no longer valid"),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText(/Your name/)).not.toBeInTheDocument();
  });

  it("shows the already-confirmed state without a sign form", async () => {
    mocks.getPublicRateConfirmation.mockResolvedValue(
      publicView({ status: "Confirmed", confirmed: true }),
    );

    renderPage();

    expect(await screen.findByText("Already confirmed")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Confirm rate/ })).not.toBeInTheDocument();
  });

  it("submits the signer and shows the success state", async () => {
    mocks.getPublicRateConfirmation.mockResolvedValue(publicView());
    mocks.confirmPublicRateConfirmation.mockResolvedValue(undefined);
    const user = userEvent.setup();

    renderPage();

    await user.type(await screen.findByLabelText(/Your name/), "Jane Smith");
    await user.type(screen.getByLabelText(/Title/), "Operations Manager");
    await user.click(screen.getByRole("button", { name: /Confirm rate/ }));

    await waitFor(() => {
      expect(mocks.confirmPublicRateConfirmation).toHaveBeenCalledWith("tok_123", {
        signerName: "Jane Smith",
        signerTitle: "Operations Manager",
      });
    });
    expect(await screen.findByText("Rate confirmed")).toBeInTheDocument();
  });

  it("shows the throttled state when the confirm endpoint rate-limits", async () => {
    mocks.getPublicRateConfirmation.mockResolvedValue(publicView());
    mocks.confirmPublicRateConfirmation.mockRejectedValue(apiError(429));
    const user = userEvent.setup();

    renderPage();

    await user.type(await screen.findByLabelText(/Your name/), "Jane Smith");
    await user.click(screen.getByRole("button", { name: /Confirm rate/ }));

    expect(await screen.findByText("Too many attempts")).toBeInTheDocument();
  });

  it("offers a retry that resubmits the same signer after a transient failure", async () => {
    mocks.getPublicRateConfirmation.mockResolvedValue(publicView());
    mocks.confirmPublicRateConfirmation
      .mockRejectedValueOnce(apiError(503))
      .mockResolvedValueOnce(undefined);
    const user = userEvent.setup();

    renderPage();

    await user.type(await screen.findByLabelText(/Your name/), "Jane Smith");
    await user.click(screen.getByRole("button", { name: /Confirm rate/ }));

    expect(await screen.findByText("Temporarily unavailable")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Try again/ }));

    expect(await screen.findByText("Rate confirmed")).toBeInTheDocument();
    expect(mocks.confirmPublicRateConfirmation).toHaveBeenCalledTimes(2);
    expect(mocks.confirmPublicRateConfirmation).toHaveBeenLastCalledWith("tok_123", {
      signerName: "Jane Smith",
      signerTitle: "",
    });
  });
});
