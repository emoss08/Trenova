import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { TestRouter } from "@trenova/shared/testing/test-router";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { FakeBroadcastChannel } from "@trenova/shared/testing/fake-broadcast-channel";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const navigate = vi.hoisted(() => vi.fn());

vi.mock("@tanstack/react-router", async (importActual) => ({
  ...(await importActual<typeof import("@tanstack/react-router")>()),
  useNavigate: () => navigate,
}));

vi.stubGlobal("BroadcastChannel", FakeBroadcastChannel);

const { DashLoginPage } = await import("./login");

const login = vi.fn();
const verifyMFA = vi.fn();

async function signInAt(search: string) {
  render(
    <TestRouter initialEntries={["/dash/login"]} search={search}>
      <DashLoginPage />
    </TestRouter>,
  );
  fireEvent.change(await screen.findByLabelText("Email"), {
    target: { value: "driver@example.com" },
  });
  fireEvent.change(screen.getByLabelText("Password"), { target: { value: "pw" } });
  submitFormOf(screen.getByRole("button", { name: "Sign in" }));
}

function submitFormOf(button: HTMLElement) {
  const form = button.closest("form");
  if (!form) {
    throw new Error("The button is not in a form");
  }
  fireEvent.submit(form);
}

describe("DashLoginPage", () => {
  afterEach(cleanup);

  beforeEach(() => {
    navigate.mockClear();
    login.mockReset().mockResolvedValue({ status: "signed_in" });
    verifyMFA.mockReset().mockResolvedValue(undefined);
    useAuthStore.setState({ login, verifyMFA });
  });

  it("goes into the portal when the person finishes signing in in another tab", async () => {
    render(
      <TestRouter>
        <DashLoginPage />
      </TestRouter>,
    );
    await screen.findByLabelText(/email/i);

    new FakeBroadcastChannel("trenova-session-context").postMessage({ kind: "signed-in" });

    await waitFor(() => expect(navigate).toHaveBeenCalledWith({ href: "/dash", replace: true }));
  });

  it("returns a driver to the portal page they were going to once signed in", async () => {
    await signInAt("?signed_out=expired&redirect=%2Fdash%2Fpay%2Fstl_1%3Ftab%3Ddeductions");

    await waitFor(() =>
      expect(navigate).toHaveBeenCalledWith({
        href: "/dash/pay/stl_1?tab=deductions",
        replace: true,
      }),
    );
    expect(login).toHaveBeenCalledWith({ emailAddress: "driver@example.com", password: "pw" });
  });

  it("returns a driver to the page they were going to after the second factor", async () => {
    login.mockResolvedValueOnce({
      status: "mfa_required",
      challenge: {
        mfaRequired: true,
        mfaChallengeToken: "mfa_tok",
        mfaMethods: ["totp", "recovery_code"],
        expiresAt: 1782403304,
      },
    });
    await signInAt("?redirect=%2Fdash%2Floads");

    fireEvent.change(await screen.findByLabelText("Authentication code"), {
      target: { value: "123456" },
    });
    submitFormOf(screen.getByRole("button", { name: "Verify" }));

    await waitFor(() =>
      expect(navigate).toHaveBeenCalledWith({ href: "/dash/loads", replace: true }),
    );
    expect(verifyMFA).toHaveBeenCalledWith({
      challengeToken: "mfa_tok",
      code: "123456",
      recoveryCode: "",
    });
  });

  it.each([
    ["an address on another site", "?redirect=https%3A%2F%2Fevil.com"],
    ["a protocol-relative address", "?redirect=%2F%2Fevil.com"],
    ["a backslash address", "?redirect=%2F%5Cevil.com"],
    ["a page outside the portal", "?redirect=%2Fbilling%2Finvoices"],
    ["the portal sign-in itself", "?redirect=%2Fdash%2Flogin"],
  ])("takes a driver to the portal's home instead of %s", async (_, search) => {
    await signInAt(search);

    await waitFor(() => expect(navigate).toHaveBeenCalledWith({ href: "/dash", replace: true }));
    expect(navigate).toHaveBeenCalledTimes(1);
  });

  it("tells a driver that the app and what it shows come from their carrier, and not to use it while driving", async () => {
    render(
      <TestRouter>
        <DashLoginPage />
      </TestRouter>,
    );

    expect(
      await screen.findByText(
        "Dash is provided by your carrier. Routes, assignments and times shown here come from your carrier, not from Trenova. Don't use Dash while your vehicle is moving.",
      ),
    ).toBeInTheDocument();
  });
});
