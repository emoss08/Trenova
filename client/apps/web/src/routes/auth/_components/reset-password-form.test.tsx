import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ResetPasswordForm } from "./reset-password-form";

const mocks = vi.hoisted(() => ({
  resetPassword: vi.fn(),
  onDone: vi.fn(),
  onRequestNewLink: vi.fn(),
}));

vi.mock("@trenova/shared/services/auth", () => ({
  authService: { resetPassword: mocks.resetPassword },
}));

function renderForm(token = "raw-token") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <ResetPasswordForm
        token={token}
        onDone={mocks.onDone}
        onRequestNewLink={mocks.onRequestNewLink}
      />
    </QueryClientProvider>,
  );
}

async function fillPasswords(newPassword: string, confirmPassword = newPassword) {
  const user = userEvent.setup();
  await user.type(screen.getByLabelText(/^new password/i), newPassword);
  await user.type(screen.getByLabelText(/confirm new password/i), confirmPassword);
  return user;
}

describe("ResetPasswordForm", () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockClear());
    mocks.resetPassword.mockResolvedValue({ message: "ok" });
  });

  it("redeems the token from the link with the chosen password", async () => {
    renderForm();
    const user = await fillPasswords("a-good-password");
    await user.click(screen.getByRole("button", { name: /set new password/i }));

    await waitFor(() =>
      expect(mocks.resetPassword).toHaveBeenCalledWith("raw-token", "a-good-password"),
    );
    expect(mocks.onDone).toHaveBeenCalled();
  });

  it("refuses a password shorter than the server's minimum without burning the link", async () => {
    renderForm();
    const user = await fillPasswords("short");
    await user.click(screen.getByRole("button", { name: /set new password/i }));

    await waitFor(() => expect(screen.getByText(/at least 8 characters/i)).toBeInTheDocument());
    expect(mocks.resetPassword).not.toHaveBeenCalled();
  });

  it("refuses a mismatched confirmation", async () => {
    renderForm();
    const user = await fillPasswords("a-good-password", "a-different-password");
    await user.click(screen.getByRole("button", { name: /set new password/i }));

    await waitFor(() => expect(screen.getByText(/passwords do not match/i)).toBeInTheDocument());
    expect(mocks.resetPassword).not.toHaveBeenCalled();
  });

  // Mail clients wrap long links, so a truncated one is the common failure. It has to
  // say so rather than render a form that cannot possibly succeed.
  it("explains a link that arrived without a token", () => {
    renderForm("");

    expect(screen.getByText("This link is incomplete")).toBeInTheDocument();
    expect(screen.queryByLabelText(/^new password/i)).not.toBeInTheDocument();
  });
});

describe("ResetPasswordForm with a token", () => {
  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockClear());
    mocks.resetPassword.mockResolvedValue({ message: "ok" });
    renderForm();
  });

  afterEach(() => {
    cleanup();
  });

  it("shows both password fields", () => {
    expect(screen.getByLabelText(/^new password/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/confirm new password/i)).toBeInTheDocument();
  });
});
