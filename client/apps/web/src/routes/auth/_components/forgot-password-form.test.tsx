import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ForgotPasswordForm } from "./forgot-password-form";

const mocks = vi.hoisted(() => ({
  forgotPassword: vi.fn(),
  onBack: vi.fn(),
}));

vi.mock("@trenova/shared/services/auth", () => ({
  authService: { forgotPassword: mocks.forgotPassword },
}));

function renderForm(defaultEmail?: string) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <ForgotPasswordForm defaultEmail={defaultEmail} onBack={mocks.onBack} />
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

    expect(screen.getByPlaceholderText("name@work-email.com")).toHaveValue("dana@example.com");
  });

  it("requests a link for the address given", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.type(screen.getByPlaceholderText("name@work-email.com"), "dana@example.com");
    await user.click(screen.getByRole("button", { name: /send reset link/i }));

    await waitFor(() => expect(mocks.forgotPassword).toHaveBeenCalledWith("dana@example.com"));
  });

  // The server answers identically whether or not the address is registered. A
  // confirmation that said "sent" only for real accounts would hand back exactly the
  // distinction the server refuses to make.
  it("confirms non-committally, without claiming an account exists", async () => {
    const user = userEvent.setup();
    renderForm("nobody@example.com");

    await user.click(screen.getByRole("button", { name: /send reset link/i }));

    await waitFor(() => expect(screen.getByText("Check your inbox")).toBeInTheDocument());
    expect(screen.getByText(/if that address has an account/i)).toBeInTheDocument();
    expect(screen.queryByText(/we sent you/i)).not.toBeInTheDocument();
  });

  it("refuses a malformed address before calling the server", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.type(screen.getByPlaceholderText("name@work-email.com"), "not-an-address");
    await user.click(screen.getByRole("button", { name: /send reset link/i }));

    await waitFor(() =>
      expect(screen.getByText(/please enter a valid email address/i)).toBeInTheDocument(),
    );
    expect(mocks.forgotPassword).not.toHaveBeenCalled();
  });

  it("can go back to sign in", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.click(screen.getByRole("button", { name: /back/i }));

    expect(mocks.onBack).toHaveBeenCalled();
  });
});
