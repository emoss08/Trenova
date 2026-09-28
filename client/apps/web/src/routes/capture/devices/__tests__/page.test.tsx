import {
  captureDevice,
  json,
  renderWithClient,
  resetGraphQL,
  resolverError,
  stubGraphQL,
} from "@/components/capture/__tests__/capture-graphql-server";
import { PageLayoutStub } from "@/test/accounting-page-mocks";
import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CaptureDevicesPage } from "../page";

vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const NOW = 1_800_000_000;

const access =
  (enabled = true, canCapture = true) =>
  () =>
    json({ data: { myCaptureAccess: { enabled, canCapture } } });
const noRelease = () => json({ data: { captureAgentRelease: null } });

describe("CaptureDevicesPage", () => {
  afterEach(() => {
    vi.useRealTimers();
    resetGraphQL();
  });

  it("ticks 'Last seen' forward while the page stays open", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.setSystemTime(NOW * 1000);
    stubGraphQL({
      MyCaptureAccess: access(),
      CaptureAgentRelease: noRelease,
      MyCaptureDevices: () =>
        json({ data: { myCaptureDevices: [captureDevice({ lastSeenAt: NOW - 60 })] } }),
    });

    renderWithClient(<CaptureDevicesPage />);

    expect(await screen.findByText("Last seen 1m ago")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(3 * 60_000);
    });

    expect(await screen.findByText("Last seen 4m ago")).toBeInTheDocument();
  });

  it("offers to try again when the computers could not be loaded", async () => {
    const user = userEvent.setup();
    stubGraphQL({
      MyCaptureAccess: access(),
      CaptureAgentRelease: noRelease,
      MyCaptureDevices: [
        () => json({ errors: [{ message: "boom" }] }, 500),
        () => json({ data: { myCaptureDevices: [captureDevice({ name: "Front desk" })] } }),
      ],
    });

    renderWithClient(<CaptureDevicesPage />);

    expect(await screen.findByText("Your computers could not be loaded.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try again" }));

    expect(await screen.findByText("Front desk")).toBeInTheDocument();
  });

  it("says so when the person may not scan into Trenova", async () => {
    stubGraphQL({
      MyCaptureAccess: access(true, false),
      CaptureAgentRelease: noRelease,
      MyCaptureDevices: () => json({ data: { myCaptureDevices: [] } }),
    });

    renderWithClient(<CaptureDevicesPage />);

    expect(
      await screen.findByText(
        "You do not have permission to scan into Trenova. Your paired computers cannot upload until an administrator gives it to you.",
      ),
    ).toBeInTheDocument();
  });

  it("lists the computers under a titled panel", async () => {
    stubGraphQL({
      MyCaptureAccess: access(),
      CaptureAgentRelease: noRelease,
      MyCaptureDevices: () => json({ data: { myCaptureDevices: [captureDevice()] } }),
    });

    renderWithClient(<CaptureDevicesPage />);

    const panel = await screen.findByRole("region", { name: "Paired computers" });
    expect(await within(panel).findByText("Office")).toBeInTheDocument();
  });

  it("keeps the revoke dialog open with the error when the server refuses", async () => {
    const user = userEvent.setup();
    const bodies = stubGraphQL({
      MyCaptureAccess: access(),
      CaptureAgentRelease: noRelease,
      MyCaptureDevices: () => json({ data: { myCaptureDevices: [captureDevice()] } }),
      RevokeMyCaptureDevice: () =>
        resolverError(
          "revokeMyCaptureDevice",
          "resource-not-found",
          "NOT_FOUND",
          "Capture device not found",
        ),
    });

    renderWithClient(<CaptureDevicesPage />);

    await user.click(await screen.findByRole("button", { name: "Revoke" }));
    const dialog = await screen.findByRole("alertdialog");
    await user.type(within(dialog).getByLabelText("Reason"), "Replaced");
    await user.click(within(dialog).getByRole("button", { name: "Revoke" }));

    expect(
      await within(dialog).findByText(
        "This computer is no longer paired. Reload the list to see where it stands.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    await waitFor(() =>
      expect(
        bodies.find((body) => body.operationName === "RevokeMyCaptureDevice")?.variables,
      ).toEqual({ id: "cdev_office", reason: "Replaced" }),
    );
  });
});
