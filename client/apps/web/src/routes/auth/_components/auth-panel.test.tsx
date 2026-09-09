import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AuthPanel } from "./auth-panel";

const mocks = vi.hoisted(() => ({
  getVersion: vi.fn(),
  getNetworkPulse: vi.fn(),
}));

vi.mock("@/services/update", () => ({
  updateService: {
    getVersion: mocks.getVersion,
    getNetworkPulse: mocks.getNetworkPulse,
  },
}));

function pulse(overrides: Record<string, unknown> = {}) {
  return {
    loadsInMotion: 12480,
    onTimePercent: 98.6,
    sampleSize: 3421,
    windowDays: 7,
    lanes: [
      { from: "CA", to: "AZ", status: "InTransit", count: 12 },
      { from: "TX", to: "GA", status: "Assigned", count: 4 },
    ],
    ...overrides,
  };
}

function renderPanel() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <AuthPanel receipt={{ issued: false, rows: [{ key: "Identity" }] }} />
    </QueryClientProvider>,
  );
}

describe("AuthPanel", () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockClear());
    mocks.getVersion.mockResolvedValue({ version: "4.12.0", environment: "development" });
    mocks.getNetworkPulse.mockResolvedValue(pulse());
  });

  it("renders the instance figures and its live lanes", async () => {
    renderPanel();

    await waitFor(() => expect(screen.getByText("12,480")).toBeInTheDocument());
    expect(screen.getByText("loads in motion")).toBeInTheDocument();
    expect(screen.getByText("98.6%")).toBeInTheDocument();
    expect(screen.getByText("on-time this week")).toBeInTheDocument();

    // Chips repeat to fill the drifting band, so both tracks carry every lane.
    expect(screen.getAllByText("CA").length).toBeGreaterThan(0);
    expect(screen.getAllByText("AZ").length).toBeGreaterThan(0);
    expect(screen.getAllByText(/12 in transit/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/4 loading/).length).toBeGreaterThan(0);
  });

  // The endpoint 404s unless the operator enabled it. Nothing on the panel may fall
  // back to invented figures or invented lanes.
  it("renders no metrics and no lanes when the pulse is unavailable", async () => {
    mocks.getNetworkPulse.mockRejectedValue(new Error("not found"));
    renderPanel();

    await waitFor(() => expect(screen.getByText("Network operational")).toBeInTheDocument());
    expect(screen.queryByText("loads in motion")).not.toBeInTheDocument();
    expect(screen.queryByText(/on-time/)).not.toBeInTheDocument();
    expect(screen.queryByText(/in transit/)).not.toBeInTheDocument();
  });

  it("drops the on-time figure when the window scored no deliveries", async () => {
    mocks.getNetworkPulse.mockResolvedValue(pulse({ sampleSize: 0, onTimePercent: 0 }));
    renderPanel();

    await waitFor(() => expect(screen.getByText("loads in motion")).toBeInTheDocument());
    expect(screen.queryByText(/on-time/)).not.toBeInTheDocument();
    expect(screen.queryByText("0.0%")).not.toBeInTheDocument();
  });

  it("renders no lane band when the instance is running nothing", async () => {
    mocks.getNetworkPulse.mockResolvedValue(pulse({ lanes: [] }));
    renderPanel();

    await waitFor(() => expect(screen.getByText("loads in motion")).toBeInTheDocument());
    expect(screen.queryByText(/in transit/)).not.toBeInTheDocument();
  });

  it("reports the network as unreachable when the version call fails", async () => {
    mocks.getVersion.mockRejectedValue(new Error("offline"));
    renderPanel();

    await waitFor(() => expect(screen.getByText("Network unreachable")).toBeInTheDocument());
  });
});
