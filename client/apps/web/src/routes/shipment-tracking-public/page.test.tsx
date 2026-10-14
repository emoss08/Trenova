import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { ApiRequestError } from "@trenova/shared/lib/api";
import type { PublicShipmentView } from "@trenova/shared/types/shipment-tracking";
import { TrackingView } from "./_components/tracking-view";
import { ShipmentTrackingPublicPage } from "./page";

const mocks = vi.hoisted(() => ({
  getPublicView: vi.fn(),
}));

vi.mock("@/services/shipment-tracking", () => ({
  ShipmentTrackingService: class {
    getPublicView = mocks.getPublicView;
    proofOfDeliveryUrl = (token: string) =>
      `/api/v1/shipment-tracking-links/${token}/proof-of-delivery/`;
  },
}));

function view(overrides: Partial<PublicShipmentView> = {}): PublicShipmentView {
  return {
    companyName: "Acme Logistics",
    shipmentProNumber: "PRO-77",
    fields: ["Status", "Windows", "Actuals", "Eta", "References"],
    expiresAt: 1_900_000_000,
    closed: false,
    status: "InTransit",
    bol: "BOL-1",
    stops: [
      {
        sequence: 1,
        type: "Pickup",
        city: "Dallas",
        stateCode: "TX",
        windowStart: 1_800_000_000,
        departedAt: 1_800_000_200,
      },
      {
        sequence: 2,
        type: "Delivery",
        city: "Tulsa",
        stateCode: "OK",
        windowStart: 1_800_050_000,
        estimatedArrival: 1_800_060_000,
        running: "Late",
      },
    ],
    ...overrides,
  };
}

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={["/shipment-tracking/tok_live"]}>
          <Routes>
            <Route path="/shipment-tracking/:token" element={children} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    );
  }
  return render(<ShipmentTrackingPublicPage />, { wrapper: Wrapper });
}

const NOW = 1_800_030_000;

describe("ShipmentTrackingPublicPage", () => {
  beforeEach(() => {
    mocks.getPublicView.mockReset();
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(NOW * 1000);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("leads with where the load is going and when it is expected there", async () => {
    mocks.getPublicView.mockResolvedValue(view());
    renderPage();

    expect(
      await screen.findByRole("heading", { level: 1, name: "On its way to Tulsa." }),
    ).toBeDefined();
    expect(screen.getByText("In transit")).toBeDefined();
    expect(screen.getByText("Load PRO-77")).toBeDefined();
    expect(screen.getByText("Expected at Tulsa")).toBeDefined();
    expect(screen.getByText("in 8h 20m")).toBeDefined();
    expect(screen.getByText("BOL-1")).toBeDefined();
    expect(screen.getAllByText("Dallas, TX").length).toBeGreaterThan(0);
    expect(screen.getByText("Running late")).toBeDefined();
    expect(mocks.getPublicView).toHaveBeenCalledWith("tok_live");
  });

  it("flags an arrival expected after the appointment window", async () => {
    mocks.getPublicView.mockResolvedValue(
      view({
        stops: [
          { sequence: 1, type: "Pickup", city: "Dallas", stateCode: "TX", departedAt: NOW - 3600 },
          {
            sequence: 2,
            type: "Delivery",
            city: "Tulsa",
            stateCode: "OK",
            windowStart: NOW + 3600,
            windowEnd: NOW + 7200,
            estimatedArrival: NOW + 9000,
          },
        ],
      }),
    );
    renderPage();

    expect(await screen.findByText("About 30m after the appointment")).toBeDefined();
  });

  it("offers to put the arrival in a calendar and to copy the page's link", async () => {
    mocks.getPublicView.mockResolvedValue(view());
    renderPage();

    expect(await screen.findByRole("button", { name: "Add to calendar" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Copy link" })).toBeDefined();
  });

  it("draws nothing the view did not carry", async () => {
    mocks.getPublicView.mockResolvedValue(view({ fields: [], status: undefined, bol: undefined }));
    renderPage();

    expect(await screen.findByText("Load PRO-77")).toBeDefined();
    expect(screen.getByRole("heading", { level: 1, name: "On its way to Tulsa." })).toBeDefined();
    expect(screen.queryByText("In transit")).toBeNull();
    expect(screen.queryByText("BOL")).toBeNull();
    expect(screen.queryByText("Proof of delivery")).toBeNull();
  });

  it("offers the proof of delivery for download when it is shown and on file", async () => {
    mocks.getPublicView.mockResolvedValue(
      view({
        fields: ["ProofOfDelivery"],
        proofOfDelivery: { documentId: "doc_1", fileName: "pod.pdf", filedAt: 1_800_070_000 },
      }),
    );
    renderPage();

    expect(await screen.findByText("pod.pdf")).toBeDefined();
    const link = screen.getByRole("link", { name: "Download" });
    expect(link.getAttribute("href")).toBe(
      "/api/v1/shipment-tracking-links/tok_live/proof-of-delivery/",
    );
  });

  it("says the link is dead for a rejected token, not for an outage", async () => {
    mocks.getPublicView.mockRejectedValue(
      new ApiRequestError(400, {
        type: "https://trenova.app/problems/validation-error",
        title: "Error",
        status: 400,
        errors: [{ field: "token", message: "This tracking link is no longer valid" }],
      }),
    );
    renderPage();
    expect(await screen.findByText("This tracking link is no longer valid")).toBeDefined();
  });

  it("offers a retry when the server is unavailable", async () => {
    mocks.getPublicView.mockRejectedValue(
      new ApiRequestError(503, {
        type: "https://trenova.app/problems/unavailable",
        title: "Error",
        status: 503,
      }),
    );
    renderPage();
    expect(await screen.findByText("Temporarily unavailable")).toBeDefined();
    expect(screen.getByRole("button", { name: "Try again" })).toBeDefined();
  });
});

describe("TrackingView in preview", () => {
  it("names who the preview is for and leaves out what only a link can do", () => {
    render(
      <TrackingView
        view={view({
          fields: ["Status", "Eta", "ProofOfDelivery"],
          proofOfDelivery: { documentId: "doc_1", fileName: "pod.pdf", filedAt: 1_800_070_000 },
        })}
        now={NOW}
        mode={{ kind: "preview", audience: "Cargill Protein" }}
      />,
    );

    expect(screen.getByText("Preview")).toBeDefined();
    expect(screen.getByText("What Cargill Protein would see through a link sent now")).toBeDefined();
    expect(screen.queryByRole("button", { name: "Add to calendar" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Refresh" })).toBeNull();
    expect(screen.getByText("pod.pdf")).toBeDefined();
    expect(screen.queryByRole("link", { name: "Download" })).toBeNull();
  });
});
