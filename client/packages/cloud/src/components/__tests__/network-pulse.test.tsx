import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NetworkPulse } from "../network-pulse";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
}));

vi.mock("../../services/network-pulse", () => ({
  networkPulseService: { get: mocks.get },
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

function renderPulse() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <NetworkPulse />
    </QueryClientProvider>,
  );
}

describe("NetworkPulse", () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    mocks.get.mockReset();
    mocks.get.mockResolvedValue(pulse());
  });

  it("renders the instance figures and its live lanes", async () => {
    renderPulse();

    await waitFor(() => expect(screen.getByText("12,480")).toBeInTheDocument());
    expect(screen.getByText("loads in motion")).toBeInTheDocument();
    expect(screen.getByText("98.6%")).toBeInTheDocument();
    expect(screen.getByText("on-time this week")).toBeInTheDocument();

    expect(screen.getAllByText("CA").length).toBeGreaterThan(0);
    expect(screen.getAllByText("AZ").length).toBeGreaterThan(0);
    expect(screen.getAllByText(/12 in transit/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/4 loading/).length).toBeGreaterThan(0);
  });

  it("renders nothing when the pulse is unavailable", async () => {
    mocks.get.mockRejectedValue(new Error("not found"));
    const { container } = renderPulse();

    await waitFor(() => expect(mocks.get).toHaveBeenCalled());
    expect(container).toBeEmptyDOMElement();
  });

  it("drops the on-time figure when the window scored no deliveries", async () => {
    mocks.get.mockResolvedValue(pulse({ sampleSize: 0, onTimePercent: 0 }));
    renderPulse();

    await waitFor(() => expect(screen.getByText("loads in motion")).toBeInTheDocument());
    expect(screen.queryByText(/on-time/)).not.toBeInTheDocument();
    expect(screen.queryByText("0.0%")).not.toBeInTheDocument();
  });

  it("renders no lane band when the instance is running nothing", async () => {
    mocks.get.mockResolvedValue(pulse({ lanes: [] }));
    renderPulse();

    await waitFor(() => expect(screen.getByText("loads in motion")).toBeInTheDocument());
    expect(screen.queryByText(/in transit/)).not.toBeInTheDocument();
  });
});
