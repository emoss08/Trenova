import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ForgotPasswordForm } from "./forgot-password-form";
import { AuthStageContext, type AuthStageControls } from "./stage/auth-stage-context";

const mocks = vi.hoisted(() => ({
  forgotPassword: vi.fn(),
  onBack: vi.fn(),
}));

vi.mock("@trenova/shared/services/auth", () => ({
  authService: { forgotPassword: mocks.forgotPassword },
}));

function renderForm(defaultEmail?: string, wrap: (ui: ReactNode) => ReactNode = (ui) => ui) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      {wrap(<ForgotPasswordForm defaultEmail={defaultEmail} onBack={mocks.onBack} />)}
    </QueryClientProvider>,
  );
}

describe("ForgotPasswordForm", () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockClear());
    mocks.forgotPassword.mockResolvedValue({ message: "ok" });
  });

  it("carries over the address already typed on the sign-in step", () => {
    renderForm("dana@example.com");

    expect(screen.getByLabelText("Email")).toHaveValue("dana@example.com");
  });

  it("requests a link for the address given", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.type(screen.getByLabelText("Email"), "dana@example.com");
    await user.click(screen.getByRole("button", { name: /send reset link/i }));

    await waitFor(() => expect(mocks.forgotPassword).toHaveBeenCalledWith("dana@example.com"));
  });

  // The server answers identically whether or not the address is registered. A
  // confirmation that said "sent" only for real accounts would hand back exactly the
  // distinction the server refuses to make.
  it("confirms non-committally, naming the address without claiming an account exists", async () => {
    const user = userEvent.setup();
    renderForm("nobody@example.com");

    await user.click(screen.getByRole("button", { name: /send reset link/i }));

    expect(await screen.findByRole("heading", { name: "Check your inbox" })).toBeInTheDocument();
    const address = screen.getByText("nobody@example.com");
    expect(address.tagName).toBe("B");
    expect(address.parentElement).toHaveTextContent(
      "If nobody@example.com has an account, a reset link is on its way.",
    );
    expect(screen.queryByText(/we sent you/i)).not.toBeInTheDocument();
  });

  it("goes back to the form, address kept, to use a different one", async () => {
    const user = userEvent.setup();
    renderForm("nobody@example.com");

    await user.click(screen.getByRole("button", { name: /send reset link/i }));
    await user.click(await screen.findByRole("button", { name: "Use a different address" }));

    expect(screen.getByLabelText("Email")).toHaveValue("nobody@example.com");
    expect(screen.queryByRole("heading", { name: "Check your inbox" })).not.toBeInTheDocument();
  });

  it("returns to sign in from the confirmation", async () => {
    const user = userEvent.setup();
    renderForm("nobody@example.com");

    await user.click(screen.getByRole("button", { name: /send reset link/i }));
    await user.click(await screen.findByRole("button", { name: "Back to sign in" }));

    expect(mocks.onBack).toHaveBeenCalledTimes(1);
  });

  it("refuses a malformed address before calling the server", async () => {
    const user = userEvent.setup();
    const stage: AuthStageControls = { burst: vi.fn(), setDone: vi.fn() };
    renderForm(undefined, (ui) => <AuthStageContext value={stage}>{ui}</AuthStageContext>);

    await user.type(screen.getByLabelText("Email"), "not-an-address");
    await user.click(screen.getByRole("button", { name: /send reset link/i }));

    expect(await screen.findByText("Please enter a valid email address.")).toBeInTheDocument();
    expect(mocks.forgotPassword).not.toHaveBeenCalled();
    expect(stage.burst).not.toHaveBeenCalled();
  });

  it("bursts the stage when a valid request goes out", async () => {
    const user = userEvent.setup();
    const stage: AuthStageControls = { burst: vi.fn(), setDone: vi.fn() };
    renderForm("dana@example.com", (ui) => (
      <AuthStageContext value={stage}>{ui}</AuthStageContext>
    ));

    await user.click(screen.getByRole("button", { name: /send reset link/i }));

    await waitFor(() => expect(mocks.forgotPassword).toHaveBeenCalled());
    expect(stage.burst).toHaveBeenCalledTimes(1);
  });

  it("can go back to sign in", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.click(screen.getByRole("button", { name: /back to sign in/i }));

    expect(mocks.onBack).toHaveBeenCalled();
  });
});
