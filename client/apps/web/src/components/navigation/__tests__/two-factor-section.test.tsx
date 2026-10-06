import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { TwoFactorSection } from "../two-factor-section";

const mocks = vi.hoisted(() => ({
  status: vi.fn(),
  beginEnrollment: vi.fn(),
  confirmEnrollment: vi.fn(),
  disable: vi.fn(),
  regenerateRecoveryCodes: vi.fn(),
}));

vi.mock("@trenova/shared/services/mfa", () => ({
  mfaService: mocks,
}));

function renderSection(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

const OFF = {
  totpEnabled: false,
  enrollmentPending: false,
  recoveryCodesRemaining: 0,
  sessionVerified: false,
};

const ON = {
  totpEnabled: true,
  enrollmentPending: false,
  enabledAt: 1_800_000_000,
  recoveryCodesRemaining: 2,
  sessionVerified: true,
};

describe("TwoFactorSection", () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockReset());
  });

  it("enrolls with the password, a code from the app, and shows the recovery codes once", async () => {
    mocks.status.mockResolvedValueOnce(OFF).mockResolvedValue(ON);
    mocks.beginEnrollment.mockResolvedValue({
      authenticatorId: "mfa_1",
      secret: "JBSWY3DPEHPK3PXP",
      uri: "otpauth://totp/Trenova:ada@example.com?secret=JBSWY3DPEHPK3PXP",
      qrCode: "data:image/png;base64,AAAA",
    });
    mocks.confirmEnrollment.mockResolvedValue({
      recoveryCodes: ["aaaaa-bbbbb", "ccccc-ddddd"],
    });
    const user = userEvent.setup();
    renderSection(<TwoFactorSection />);

    await user.click(await screen.findByRole("button", { name: "Set up" }));
    await user.type(screen.getByPlaceholderText("Confirm it is you"), "correct-horse");
    await user.click(screen.getByRole("button", { name: "Continue" }));

    expect(await screen.findByAltText("QR code for your authenticator app")).toHaveAttribute(
      "src",
      "data:image/png;base64,AAAA",
    );
    expect(screen.getByText("JBSWY3DPEHPK3PXP")).toBeInTheDocument();
    expect(mocks.beginEnrollment).toHaveBeenCalledWith("correct-horse");

    await user.type(screen.getByPlaceholderText("123 456"), "123456");
    await user.click(screen.getByRole("button", { name: "Turn on" }));

    const list = await screen.findByRole("list", { name: "Recovery codes" });
    expect(list).toHaveTextContent("aaaaa-bbbbb");
    expect(list).toHaveTextContent("ccccc-ddddd");
    expect(mocks.confirmEnrollment).toHaveBeenCalledWith("123456");
  });

  it("warns when recovery codes run low and offers turning off", async () => {
    mocks.status.mockResolvedValue(ON);
    renderSection(<TwoFactorSection />);

    expect(await screen.findByText("On")).toBeInTheDocument();
    expect(
      screen.getByText("You are running out of recovery codes. Create new ones and store them safely."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Turn off" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "New recovery codes" })).toBeInTheDocument();
  });

  it("turns the factor off only with the password and a code", async () => {
    mocks.status.mockResolvedValue(ON);
    mocks.disable.mockResolvedValue(undefined);
    const user = userEvent.setup();
    renderSection(<TwoFactorSection />);

    await user.click(await screen.findByRole("button", { name: "Turn off" }));
    const dialog = await screen.findByRole("dialog");
    await user.type(screen.getByPlaceholderText("Enter your password"), "correct-horse");
    await user.type(screen.getByPlaceholderText("123 456"), "654321");
    const buttons = dialog.querySelectorAll("button");
    const confirm = Array.from(buttons).find((button) => button.textContent === "Turn off");
    expect(confirm).toBeDefined();
    await user.click(confirm!);

    await waitFor(() =>
      expect(mocks.disable).toHaveBeenCalledWith({ password: "correct-horse", code: "654321" }),
    );
  });
});
