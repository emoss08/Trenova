import type { CaptureFleetDevice } from "@/lib/graphql/capture";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FleetPanel } from "../fleet-panel";

const mocks = vi.hoisted(() => ({
  fetchCaptureDevices: vi.fn(),
  revokeCaptureDevice: vi.fn(),
}));

vi.mock("@/lib/graphql/capture", () => ({
  fetchCaptureDevices: mocks.fetchCaptureDevices,
  revokeCaptureDevice: mocks.revokeCaptureDevice,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

// Both are owned by the capture components; the panel only decides what they receive.
vi.mock("@/components/capture/download-panel", () => ({
  CaptureDownloadPanel: () => null,
}));

vi.mock("@/components/capture/device-list", () => ({
  DeviceList: ({ devices }: { devices: { id: string; name: string }[] }) => (
    <ul aria-label="Paired computers">
      {devices.map((device) => (
        <li key={device.id}>{device.name}</li>
      ))}
    </ul>
  ),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() } }));

const dispatchPc = {
  id: "cdev_1",
  userId: "usr_1",
  name: "DISPATCH-01",
  machineName: "DISPATCH-01",
  windowsUser: "dispatch",
  agentVersion: "1.4.0",
  architecture: "x64",
  osVersion: "10.0.22631",
  status: "Active",
  lastSeenAt: 1_780_000_000,
  isOnline: true,
  lastIp: "10.0.0.2",
  revokedAt: null,
  revokedReason: "",
  version: 1,
  createdAt: 1_780_000_000,
  sources: [],
  user: { id: "usr_1", name: "Dana Dispatcher", emailAddress: "dana@example.com" },
} as unknown as CaptureFleetDevice;

function renderPanel() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return render(<FleetPanel />, { wrapper });
}

describe("FleetPanel", () => {
  beforeEach(() => {
    mocks.fetchCaptureDevices.mockResolvedValue([dispatchPc]);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("keeps the current list on screen while a search is fetched", async () => {
    const user = userEvent.setup();
    renderPanel();
    expect(await screen.findByText("DISPATCH-01")).toBeInTheDocument();

    mocks.fetchCaptureDevices.mockReturnValue(new Promise(() => undefined));
    await user.type(
      screen.getByRole("textbox", { name: "Search computer, user or Windows account" }),
      "dis",
    );

    await waitFor(() =>
      expect(mocks.fetchCaptureDevices).toHaveBeenLastCalledWith(
        { status: "Active", query: "dis" },
        expect.anything(),
      ),
    );
    expect(screen.getByText("DISPATCH-01")).toBeInTheDocument();
    expect(document.querySelector("[aria-busy=true]")).toBeNull();
  });

  it("says no computer is paired only when nothing narrows the list", async () => {
    mocks.fetchCaptureDevices.mockResolvedValue([]);
    renderPanel();

    expect(await screen.findByRole("heading", { name: "No computers paired" })).toBeInTheDocument();
  });

  it("says no computer is revoked when the revoked filter is empty", async () => {
    mocks.fetchCaptureDevices.mockResolvedValue([]);
    const user = userEvent.setup();
    renderPanel();
    await screen.findByRole("heading", { name: "No computers paired" });

    await user.click(screen.getByRole("radio", { name: "Revoked" }));

    expect(
      await screen.findByRole("heading", { name: "No revoked computers" }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "No computers paired" })).toBeNull();
  });

  it("says nothing matches when a search is active, whatever the filter", async () => {
    const user = userEvent.setup();
    renderPanel();
    await screen.findByText("DISPATCH-01");

    mocks.fetchCaptureDevices.mockResolvedValue([]);
    await user.click(screen.getByRole("radio", { name: "Revoked" }));
    await user.type(
      screen.getByRole("textbox", { name: "Search computer, user or Windows account" }),
      "zzz",
    );

    expect(await screen.findByRole("heading", { name: "No computer matches" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "No computers paired" })).toBeNull();
  });

  it("offers a retry when the computers cannot be loaded", async () => {
    mocks.fetchCaptureDevices.mockRejectedValueOnce(new Error("offline"));
    const user = userEvent.setup();
    renderPanel();

    await user.click(await screen.findByRole("button", { name: "Try again" }));

    expect(await screen.findByText("DISPATCH-01")).toBeInTheDocument();
  });
});
