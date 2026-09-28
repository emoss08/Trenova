import {
  json,
  networkFailure,
  renderWithClient,
  resetGraphQL,
  resolverError,
  stubGraphQL,
} from "@/components/capture/__tests__/capture-graphql-server";
import { PageLayoutStub } from "@/test/accounting-page-mocks";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CapturePairPage } from "../page";

vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const PREVIEW = {
  userCode: "ABCD-EFGH",
  machineName: "OFFICE-PC",
  windowsUser: "jdoe",
  agentVersion: "1.4.0",
  architecture: "X64",
  osVersion: "Windows 10.0.22631",
  clientIp: "203.0.113.7",
  expiresAt: 1_900_000_000,
};

const access = () => json({ data: { myCaptureAccess: { enabled: true, canCapture: true } } });
const preview = () => json({ data: { captureDevicePairing: PREVIEW } });

const EXPIRED_TEXT =
  "That code is not valid or has expired. Codes last ten minutes; start again from Trenova Capture's tray icon for a new one.";
const UNREACHABLE_TEXT = "Trenova could not look the code up. Try again in a moment.";
const FORBIDDEN_TEXT =
  "You do not have permission to pair computers. Ask an administrator for scanning access.";

function renderPage(code = "ABCDEFGH") {
  return renderWithClient(<CapturePairPage />, { route: `/capture/pair?code=${code}` });
}

describe("CapturePairPage", () => {
  afterEach(resetGraphQL);

  it("says a code is not valid only when the server could not find it", async () => {
    stubGraphQL({
      MyCaptureAccess: access,
      CaptureDevicePairing: () =>
        resolverError(
          "captureDevicePairing",
          "resource-not-found",
          "NOT_FOUND",
          "That code is not valid or has expired",
        ),
    });

    renderPage();

    expect(await screen.findByText(EXPIRED_TEXT)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Try again" })).toBeNull();
  });

  it("says the person lacks permission rather than blaming the code", async () => {
    stubGraphQL({
      MyCaptureAccess: access,
      CaptureDevicePairing: () =>
        resolverError(
          "captureDevicePairing",
          "authorization-error",
          "FORBIDDEN",
          "You do not have permission to create capture_batch",
        ),
    });

    renderPage();

    expect(await screen.findByText(FORBIDDEN_TEXT)).toBeInTheDocument();
    expect(screen.queryByText(EXPIRED_TEXT)).toBeNull();
  });

  it("offers a retry when Trenova could not be reached, and shows the computer once it answers", async () => {
    const user = userEvent.setup();
    stubGraphQL({ MyCaptureAccess: access, CaptureDevicePairing: [networkFailure, preview] });

    renderPage();

    expect(await screen.findByText(UNREACHABLE_TEXT)).toBeInTheDocument();
    expect(screen.queryByText(EXPIRED_TEXT)).toBeNull();

    await user.click(screen.getByRole("button", { name: "Try again" }));

    expect(await screen.findByText("OFFICE-PC")).toBeInTheDocument();
  });

  it("treats a server failure as a problem reaching Trenova, not an expired code", async () => {
    stubGraphQL({
      MyCaptureAccess: access,
      CaptureDevicePairing: () => json({ errors: [{ message: "internal" }] }, 502),
    });

    renderPage();

    expect(await screen.findByText(UNREACHABLE_TEXT)).toBeInTheDocument();
    expect(screen.queryByText(EXPIRED_TEXT)).toBeNull();
  });

  it("explains the code's format instead of silently disabling the lookup", async () => {
    const user = userEvent.setup();
    const bodies = stubGraphQL({ MyCaptureAccess: access, CaptureDevicePairing: preview });

    renderPage("");

    await user.type(screen.getByLabelText("Pairing code"), "ABC");
    await user.click(screen.getByRole("button", { name: "Look up" }));

    expect(
      await screen.findByText("Enter all eight letters of the code, like ABCD-EFGH"),
    ).toBeInTheDocument();
    expect(bodies.some((body) => body.operationName === "CaptureDevicePairing")).toBe(false);

    await user.type(screen.getByLabelText("Pairing code"), "d efgh");
    await user.click(screen.getByRole("button", { name: "Look up" }));

    expect(await screen.findByText("OFFICE-PC")).toBeInTheDocument();
    expect(bodies.find((body) => body.operationName === "CaptureDevicePairing")?.variables).toEqual(
      { userCode: "ABCDEFGH" },
    );
  });

  it("puts a name the server rejects on the name field", async () => {
    const user = userEvent.setup();
    stubGraphQL({
      MyCaptureAccess: access,
      CaptureDevicePairing: preview,
      ApproveCaptureDevicePairing: () =>
        resolverError(
          "approveCaptureDevicePairing",
          "validation-error",
          "INVALID",
          "Device name must be at most 100 characters",
          [
            {
              field: "deviceName",
              code: "INVALID",
              message: "Device name must be at most 100 characters",
            },
          ],
        ),
    });

    renderPage();

    const name = await screen.findByLabelText("Name it");
    await user.clear(name);
    await user.type(name, "Front desk");
    await user.click(screen.getByRole("button", { name: "Approve" }));

    expect(
      await screen.findByText("Device name must be at most 100 characters"),
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByLabelText("Name it")).toHaveAttribute("aria-invalid", "true"),
    );
  });

  it("sends the edited name when approving and confirms the pairing", async () => {
    const user = userEvent.setup();
    const bodies = stubGraphQL({
      MyCaptureAccess: access,
      CaptureDevicePairing: preview,
      ApproveCaptureDevicePairing: () => json({ data: { approveCaptureDevicePairing: true } }),
    });

    renderPage();

    const name = await screen.findByLabelText("Name it");
    await user.clear(name);
    await user.type(name, "  Front desk  ");
    await user.click(screen.getByRole("button", { name: "Approve" }));

    expect(await screen.findByText("Front desk is paired")).toBeInTheDocument();
    expect(
      bodies.find((body) => body.operationName === "ApproveCaptureDevicePairing")?.variables,
    ).toEqual({ userCode: "ABCDEFGH", deviceName: "Front desk" });
  });
});
