import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { ApiRequestError } from "@trenova/shared/lib/api";
import type { PublicTrackingView } from "@trenova/shared/types/carrier-tracking";
import { CarrierTrackingPublicPage } from "./page";

const mocks = vi.hoisted(() => ({
  getPublicTracking: vi.fn(),
  postCheckCall: vi.fn(),
}));

vi.mock("@/services/carrier-tracking", () => ({
  CarrierTrackingService: class {
    getPublicTracking = mocks.getPublicTracking;
    postCheckCall = mocks.postCheckCall;
  },
}));

function view(overrides: Partial<PublicTrackingView> = {}): PublicTrackingView {
  return {
    companyName: "Acme Brokerage",
    carrierName: "Redline Freight",
    shipmentProNumber: "PRO-77",
    closed: false,
    expiresAt: 1_900_000_000,
    lastCheckCallAt: null,
    stops: [
      {
        id: "stp_pickup",
        sequence: 0,
        type: "Pickup",
        locationName: "DC 12",
        city: "Dallas",
        stateCode: "TX",
        windowStart: 1_800_000_000,
        windowEnd: null,
        arrivedAt: 1_800_000_100,
        departedAt: 1_800_000_200,
      },
      {
        id: "stp_delivery",
        sequence: 1,
        type: "Delivery",
        locationName: "Store 44",
        city: "Tulsa",
        stateCode: "OK",
        windowStart: 1_800_050_000,
        windowEnd: null,
        arrivedAt: null,
        departedAt: null,
      },
    ],
    ...overrides,
  };
}

function problem(status: number, type: string, errors?: Array<{ field: string; message: string }>) {
  return new ApiRequestError(status, {
    type: `https://trenova.app/problems/${type}`,
    title: "Error",
    status,
    errors,
  });
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={["/carrier-tracking/tok_live"]}>
          <Routes>
            <Route path="/carrier-tracking/:token" element={children} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    );
  }
  return render(<CarrierTrackingPublicPage />, { wrapper: Wrapper });
}

describe("CarrierTrackingPublicPage", () => {
  beforeEach(() => {
    mocks.getPublicTracking.mockReset();
    mocks.postCheckCall.mockReset();
  });

  it("shows the load and offers the arrival at the next stop", async () => {
    mocks.getPublicTracking.mockResolvedValue(view());

    renderPage();

    expect(await screen.findByText("Load PRO-77 for Acme Brokerage")).toBeInTheDocument();
    expect(mocks.getPublicTracking).toHaveBeenCalledWith("tok_live");
    expect(screen.getByText(/Store 44/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "I've arrived at stop 2" })).toBeInTheDocument();
  });

  it("posts the arrival once under one event id and confirms it", async () => {
    mocks.getPublicTracking.mockResolvedValue(view());
    mocks.postCheckCall.mockResolvedValue({ outcome: "Applied", replayed: false });
    const user = userEvent.setup();

    renderPage();

    await user.click(await screen.findByRole("button", { name: "I've arrived at stop 2" }));

    expect(await screen.findByText("Arrival sent. Thank you.")).toBeInTheDocument();
    expect(mocks.postCheckCall).toHaveBeenCalledTimes(1);
    const [token, payload] = mocks.postCheckCall.mock.calls[0];
    expect(token).toBe("tok_live");
    expect(payload).toMatchObject({ kind: "Arrival", stopId: "stp_delivery" });
    expect(typeof payload.clientEventId).toBe("string");
    expect(payload.clientEventId.length).toBeGreaterThan(0);
  });

  it("retries a failed post under the same event id", async () => {
    mocks.getPublicTracking.mockResolvedValue(view());
    mocks.postCheckCall
      .mockRejectedValueOnce(problem(503, "internal-error"))
      .mockResolvedValueOnce({ outcome: "Applied", replayed: true });
    const user = userEvent.setup();

    renderPage();

    const arrive = await screen.findByRole("button", { name: "I've arrived at stop 2" });
    await user.click(arrive);
    expect(await screen.findByText(/temporary problem/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "I've arrived at stop 2" }));

    await waitFor(() => expect(mocks.postCheckCall).toHaveBeenCalledTimes(2));
    expect(mocks.postCheckCall.mock.calls[1][1].clientEventId).toBe(
      mocks.postCheckCall.mock.calls[0][1].clientEventId,
    );
  });

  it("shows what the carrier must fix without calling the link dead", async () => {
    mocks.getPublicTracking.mockResolvedValue(view());
    mocks.postCheckCall.mockRejectedValue(
      problem(400, "validation-error", [
        { field: "reportedAt", message: "The reported time cannot be in the future" },
      ]),
    );
    const user = userEvent.setup();

    renderPage();

    await user.click(await screen.findByRole("button", { name: "Send update" }));

    expect(
      await screen.findByText("The reported time cannot be in the future"),
    ).toBeInTheDocument();
    expect(screen.queryByText("This tracking link is no longer valid")).not.toBeInTheDocument();
  });

  it("asks why before sending a late update", async () => {
    mocks.getPublicTracking.mockResolvedValue(view());
    const user = userEvent.setup();

    renderPage();

    await user.click(await screen.findByRole("checkbox", { name: "Running late" }));
    await user.click(screen.getByRole("button", { name: "Send update" }));

    expect(await screen.findByText("Say why the load is delayed")).toBeInTheDocument();
    expect(mocks.postCheckCall).not.toHaveBeenCalled();
  });

  it("shows the dead-link state when the token is refused", async () => {
    mocks.getPublicTracking.mockRejectedValue(
      problem(400, "validation-error", [
        { field: "token", message: "This tracking link is no longer valid" },
      ]),
    );

    renderPage();

    expect(await screen.findByText("This tracking link is no longer valid")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Send update" })).not.toBeInTheDocument();
  });

  it("shows a finished load as read-only", async () => {
    mocks.getPublicTracking.mockResolvedValue(view({ closed: true }));

    renderPage();

    expect(await screen.findByText("This load is no longer being tracked")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Send update" })).not.toBeInTheDocument();
  });
});
