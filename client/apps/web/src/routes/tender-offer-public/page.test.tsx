import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ApiRequestError } from "@trenova/shared/lib/api";
import type { PublicTenderOffer } from "@trenova/shared/types/tender";
import type { ReactNode } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TenderOfferPublicPage } from "./page";

const mocks = vi.hoisted(() => ({
  getPublicOffer: vi.fn(),
  acceptPublicOffer: vi.fn(),
  declinePublicOffer: vi.fn(),
}));

vi.mock("@/services/tender", () => ({
  TenderService: class {
    getPublicOffer = mocks.getPublicOffer;
    acceptPublicOffer = mocks.acceptPublicOffer;
    declinePublicOffer = mocks.declinePublicOffer;
  },
}));

const NO_LONGER_VALID = "This offer link is no longer valid";

function publicOffer(overrides: Partial<PublicTenderOffer> = {}): PublicTenderOffer {
  return {
    carrierName: "Blue Ridge Freight",
    shipmentProNumber: "PRO-2026-1042",
    originSummary: "Dallas, TX 75201",
    destinationSummary: "Tulsa, OK 74101",
    pickupWindow: "Jun 1, 08:00 – 12:00 UTC",
    deliveryWindow: "Jun 2, 06:00 – 10:00 UTC",
    equipmentSummary: "53' Dry Van",
    weightSummary: "38,400 lbs",
    rateAmount: "$1,850.00",
    rateMethodLabel: "Flat Rate",
    expiresAt: 1_760_000_000,
    responded: false,
    ...overrides,
  };
}

function apiError(status: number): ApiRequestError {
  return new ApiRequestError(status, { type: "about:blank", title: "Error", status });
}

function renderPage(path = "/tender-offer/tok_123") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[path]}>
          <Routes>
            <Route path="/tender-offer/:token" element={children} />
            <Route path="/tender-offer/:token/accept" element={children} />
            <Route path="/tender-offer/:token/decline" element={children} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    );
  }
  return render(<TenderOfferPublicPage />, { wrapper: Wrapper });
}

describe("TenderOfferPublicPage", () => {
  beforeEach(() => {
    mocks.getPublicOffer.mockReset();
    mocks.acceptPublicOffer.mockReset();
    mocks.declinePublicOffer.mockReset();
  });

  it("renders the lane, rate and expiry preview with both response buttons", async () => {
    mocks.getPublicOffer.mockResolvedValue(publicOffer());

    renderPage();

    expect(await screen.findByText("Load offer for Blue Ridge Freight")).toBeInTheDocument();
    expect(mocks.getPublicOffer).toHaveBeenCalledWith("tok_123");
    expect(screen.getByText(/PRO-2026-1042/)).toBeInTheDocument();
    expect(screen.getByText("Dallas, TX 75201")).toBeInTheDocument();
    expect(screen.getByText("Tulsa, OK 74101")).toBeInTheDocument();
    expect(screen.getByText("53' Dry Van")).toBeInTheDocument();
    expect(screen.getByText("38,400 lbs")).toBeInTheDocument();
    expect(screen.getByText(/\$1,850\.00/)).toBeInTheDocument();
    expect(screen.getByText("Flat Rate")).toBeInTheDocument();
    expect(screen.getByText(/Offer expires/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Accept load" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Decline" })).toBeInTheDocument();
  });

  it("hides the expiry line when the offer carries no expiry", async () => {
    mocks.getPublicOffer.mockResolvedValue(publicOffer({ expiresAt: 0 }));

    renderPage();

    expect(await screen.findByText("Load offer for Blue Ridge Freight")).toBeInTheDocument();
    expect(screen.queryByText(/Offer expires/)).not.toBeInTheDocument();
  });

  it("never auto-submits on the /accept intent URL — it only pre-highlights", async () => {
    mocks.getPublicOffer.mockResolvedValue(publicOffer());

    renderPage("/tender-offer/tok_123/accept");

    expect(await screen.findByText("Load offer for Blue Ridge Freight")).toBeInTheDocument();
    expect(mocks.acceptPublicOffer).not.toHaveBeenCalled();
    expect(mocks.declinePublicOffer).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Accept load" })).toBeEnabled();
    expect(screen.queryByLabelText(/Reason/)).not.toBeInTheDocument();
  });

  it("never auto-submits on the /decline intent URL", async () => {
    mocks.getPublicOffer.mockResolvedValue(publicOffer());

    renderPage("/tender-offer/tok_123/decline");

    expect(await screen.findByText("Load offer for Blue Ridge Freight")).toBeInTheDocument();
    expect(mocks.declinePublicOffer).not.toHaveBeenCalled();
    expect(mocks.acceptPublicOffer).not.toHaveBeenCalled();
  });

  it("pre-expands the reason textarea on the /decline intent URL", async () => {
    mocks.getPublicOffer.mockResolvedValue(publicOffer());

    renderPage("/tender-offer/tok_123/decline");

    expect(await screen.findByLabelText(/Reason/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Confirm decline" })).toBeInTheDocument();
  });

  it("shows the vague invalid-link state for a definitive 4xx preview rejection", async () => {
    mocks.getPublicOffer.mockRejectedValue(apiError(422));

    renderPage();

    expect(await screen.findByText(NO_LONGER_VALID)).toBeInTheDocument();
    expect(screen.queryByText(/Blue Ridge Freight/)).not.toBeInTheDocument();
    expect(screen.queryByText(/PRO-2026-1042/)).not.toBeInTheDocument();
    expect(screen.queryByText(/\$1,850\.00/)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Accept load" })).not.toBeInTheDocument();
  });

  it("shows the already-answered state when the preview reports a recorded response", async () => {
    mocks.getPublicOffer.mockResolvedValue(publicOffer({ responded: true }));

    renderPage();

    expect(await screen.findByText("Already answered")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Accept load" })).not.toBeInTheDocument();
  });

  it("shows the shared no-longer-valid copy when the respond call rejects as answered", async () => {
    mocks.getPublicOffer.mockResolvedValue(publicOffer());
    mocks.acceptPublicOffer.mockRejectedValue(apiError(422));
    const user = userEvent.setup();

    renderPage();

    await user.click(await screen.findByRole("button", { name: "Accept load" }));

    expect(await screen.findByText(NO_LONGER_VALID)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Accept load" })).not.toBeInTheDocument();
  });

  it("records an acceptance and shows the confirmation state", async () => {
    mocks.getPublicOffer.mockResolvedValue(publicOffer());
    mocks.acceptPublicOffer.mockResolvedValue(undefined);
    const user = userEvent.setup();

    renderPage();

    await user.click(await screen.findByRole("button", { name: "Accept load" }));

    await waitFor(() => {
      expect(mocks.acceptPublicOffer).toHaveBeenCalledWith("tok_123");
    });
    expect(await screen.findByText("Response recorded")).toBeInTheDocument();
    expect(mocks.declinePublicOffer).not.toHaveBeenCalled();
  });

  it("requires a second click to decline and sends the typed reason", async () => {
    mocks.getPublicOffer.mockResolvedValue(publicOffer());
    mocks.declinePublicOffer.mockResolvedValue(undefined);
    const user = userEvent.setup();

    renderPage();

    await user.click(await screen.findByRole("button", { name: "Decline" }));

    expect(mocks.declinePublicOffer).not.toHaveBeenCalled();
    await user.type(screen.getByLabelText(/Reason/), "No truck available in the area");
    await user.click(screen.getByRole("button", { name: "Confirm decline" }));

    await waitFor(() => {
      expect(mocks.declinePublicOffer).toHaveBeenCalledWith(
        "tok_123",
        "No truck available in the area",
      );
    });
    expect(await screen.findByText("Response recorded")).toBeInTheDocument();
  });

  it("sends an empty reason when the carrier declines without typing one", async () => {
    mocks.getPublicOffer.mockResolvedValue(publicOffer());
    mocks.declinePublicOffer.mockResolvedValue(undefined);
    const user = userEvent.setup();

    renderPage("/tender-offer/tok_123/decline");

    await user.click(await screen.findByRole("button", { name: "Confirm decline" }));

    await waitFor(() => {
      expect(mocks.declinePublicOffer).toHaveBeenCalledWith("tok_123", "");
    });
  });

  it("shows the throttled state when the respond endpoint rate-limits", async () => {
    mocks.getPublicOffer.mockResolvedValue(publicOffer());
    mocks.acceptPublicOffer.mockRejectedValue(apiError(429));
    const user = userEvent.setup();

    renderPage();

    await user.click(await screen.findByRole("button", { name: "Accept load" }));

    expect(await screen.findByText("Too many attempts")).toBeInTheDocument();
    expect(screen.queryByText(NO_LONGER_VALID)).not.toBeInTheDocument();
  });

  it("offers a retry that resubmits the same action after a transient failure", async () => {
    mocks.getPublicOffer.mockResolvedValue(publicOffer());
    mocks.declinePublicOffer
      .mockRejectedValueOnce(apiError(503))
      .mockResolvedValueOnce(undefined);
    const user = userEvent.setup();

    renderPage("/tender-offer/tok_123/decline");

    await user.type(await screen.findByLabelText(/Reason/), "Rate too low");
    await user.click(screen.getByRole("button", { name: "Confirm decline" }));

    expect(await screen.findByText("Temporarily unavailable")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Try again/ }));

    expect(await screen.findByText("Response recorded")).toBeInTheDocument();
    expect(mocks.declinePublicOffer).toHaveBeenCalledTimes(2);
    expect(mocks.declinePublicOffer).toHaveBeenLastCalledWith("tok_123", "Rate too low");
  });
});
