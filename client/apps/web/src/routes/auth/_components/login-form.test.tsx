import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { LoginForm } from "./login-form";

const mocks = vi.hoisted(() => ({
  login: vi.fn(),
  listProviders: vi.fn(),
  getSSOStartUrl: vi.fn(() => "/sso"),
  setUser: vi.fn(),
  onAuthenticated: vi.fn(),
  onForgotPassword: vi.fn(),
}));

vi.mock("@trenova/shared/services/auth", () => ({
  authService: {
    login: mocks.login,
    listProviders: mocks.listProviders,
    getSSOStartUrl: mocks.getSSOStartUrl,
  },
}));

vi.mock("@trenova/shared/stores/auth-store", () => ({
  useAuthStore: (selector: (state: { setUser: typeof mocks.setUser }) => unknown) =>
    selector({ setUser: mocks.setUser }),
}));

function testUser() {
  return {
    id: "usr_1",
    version: 1,
    createdAt: 1,
    updatedAt: 1,
    businessUnitId: "bu_1",
    currentOrganizationId: "org_1",
    status: "Active",
    name: "Test User",
    username: "test",
    emailAddress: "test@example.com",
    profilePicUrl: "",
    thumbnailUrl: "",
    timezone: "America/New_York",
    timeFormat: "12-hour",
    isLocked: false,
    mustChangePassword: false,
  };
}

function loginResponse() {
  return {
    user: testUser(),
    sessionId: "ses_01K5F3ABCDEFGHJKMNPQRSTVWX",
    expiresAt: 1782403304,
    csrfToken: "csrf",
    activeRoleIds: [],
    authorizedRoleIds: [],
    activeRoles: [],
    authorizedRoles: [],
    requiresRoleActivation: false,
  };
}

function renderLoginForm(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>,
  );
}

async function submitCredentials() {
  const user = userEvent.setup();
  await user.type(screen.getByPlaceholderText("name@work-email.com"), "test@example.com");
  await user.type(screen.getByLabelText(/password/i), "password123");
  await user.click(screen.getByRole("button", { name: /sign in/i }));
  return user;
}

describe("LoginForm", () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockClear());
    mocks.login.mockResolvedValue(loginResponse());
    mocks.listProviders.mockResolvedValue([]);
    mocks.onAuthenticated.mockResolvedValue(undefined);
  });

  it("hands the authenticated session to the flow instead of routing itself", async () => {
    renderLoginForm(
      <LoginForm
        stepLabel="01 / 03"
        onAuthenticated={mocks.onAuthenticated}
        onForgotPassword={mocks.onForgotPassword}
      />,
    );
    await submitCredentials();

    await waitFor(() =>
      expect(mocks.onAuthenticated).toHaveBeenCalledWith(
        expect.objectContaining({ sessionId: "ses_01K5F3ABCDEFGHJKMNPQRSTVWX" }),
      ),
    );
    expect(mocks.setUser).toHaveBeenCalledWith(expect.objectContaining({ id: "usr_1" }));
  });

  it("renders the step label supplied by the flow", () => {
    renderLoginForm(
      <LoginForm
        stepLabel="01 / 02"
        onAuthenticated={mocks.onAuthenticated}
        onForgotPassword={mocks.onForgotPassword}
      />,
    );

    expect(screen.getByText("01 / 02")).toBeInTheDocument();
    expect(screen.getByText("Secure sign-in")).toBeInTheDocument();
  });

  it("swaps the credential form for the Dash hand-off on the driver tab", async () => {
    const user = userEvent.setup();
    renderLoginForm(
      <LoginForm
        stepLabel="01 / 03"
        onAuthenticated={mocks.onAuthenticated}
        onForgotPassword={mocks.onForgotPassword}
      />,
    );

    await user.click(screen.getByRole("tab", { name: "Driver" }));

    expect(screen.getByRole("link", { name: /continue to dash/i })).toHaveAttribute(
      "href",
      "/dash/login",
    );
    expect(screen.queryByPlaceholderText("name@work-email.com")).not.toBeInTheDocument();
  });

  it("hides the audience toggle on a tenant login page", () => {
    renderLoginForm(
      <LoginForm
        organizationSlug="alpha"
        tenantMetadata={{
          organizationId: "org_1",
          organizationName: "Alpha Logistics",
          organizationSlug: "alpha",
          enabledProviders: [],
          passwordEnabled: true,
          enforceSso: false,
        }}
        stepLabel="01 / 02"
        onAuthenticated={mocks.onAuthenticated}
        onForgotPassword={mocks.onForgotPassword}
      />,
    );

    expect(screen.queryByRole("tab", { name: "Driver" })).not.toBeInTheDocument();
    expect(screen.getByText("Alpha Logistics")).toBeInTheDocument();
  });
});
