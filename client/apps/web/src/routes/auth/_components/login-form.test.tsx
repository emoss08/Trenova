import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { LoginForm } from "./login-form";
import { AuthStageContext, type AuthStageControls } from "./stage/auth-stage-context";

const mocks = vi.hoisted(() => ({
  login: vi.fn(),
  verifyMFA: vi.fn(),
  listProviders: vi.fn(),
  getSSOStartUrl: vi.fn(() => "/sso"),
  setUser: vi.fn(),
  onAuthenticated: vi.fn(),
  onForgotPassword: vi.fn(),
}));

vi.mock("@trenova/shared/services/auth", () => ({
  authService: {
    login: mocks.login,
    verifyMFA: mocks.verifyMFA,
    listProviders: mocks.listProviders,
    getSSOStartUrl: mocks.getSSOStartUrl,
  },
}));

const publicConfig = vi.hoisted(() => ({ signupAvailable: false }));

vi.mock("@trenova/shared/hooks/use-public-config", () => ({
  usePublicConfig: () => ({
    config: {},
    isLoading: false,
    isCloud: publicConfig.signupAvailable,
    signupAvailable: publicConfig.signupAvailable,
  }),
}));

vi.mock("@/lib/edition", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/edition")>();
  return {
    ...actual,
    edition: {
      ...actual.edition,
      slots: {
        ...actual.edition.slots,
        LoginPrompt: ({ fallback }: { fallback: ReactNode }) => (
          <span data-testid="edition-login-prompt">{fallback}</span>
        ),
      },
    },
  };
});

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
  await user.type(screen.getByLabelText("Email"), "test@example.com");
  await user.type(screen.getByLabelText("Password"), "password123");
  await user.click(screen.getByRole("button", { name: /sign in/i }));
  return user;
}

describe("LoginForm", () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    publicConfig.signupAvailable = false;
    Object.values(mocks).forEach((mock) => mock.mockClear());
    mocks.login.mockResolvedValue(loginResponse());
    mocks.listProviders.mockResolvedValue([]);
    mocks.onAuthenticated.mockResolvedValue(undefined);
  });

  it("hands the line under the heading to the edition, with the host's line as fallback", () => {
    renderLoginForm(
      <LoginForm
        stepLabel="01 / 03"
        onAuthenticated={mocks.onAuthenticated}
        onForgotPassword={mocks.onForgotPassword}
      />,
    );

    expect(screen.getByTestId("edition-login-prompt")).toHaveTextContent(
      "Sign in with the account your organization set up for you.",
    );
  });

  it("offers no signup link where accounts are made by an administrator", () => {
    renderLoginForm(
      <LoginForm
        stepLabel="01 / 03"
        onAuthenticated={mocks.onAuthenticated}
        onForgotPassword={mocks.onForgotPassword}
      />,
    );

    expect(screen.queryByRole("link", { name: "Create an account" })).not.toBeInTheDocument();
    expect(screen.queryByText("Create an account")).not.toBeInTheDocument();
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

  it("asks for the authenticator code when the account has a second factor", async () => {
    mocks.login.mockResolvedValue({
      mfaRequired: true,
      mfaChallengeToken: "challenge-token",
      mfaMethods: ["totp", "recovery_code"],
      expiresAt: 1782403304,
    });
    mocks.verifyMFA.mockResolvedValue(loginResponse());
    renderLoginForm(
      <LoginForm
        stepLabel="01 / 03"
        onAuthenticated={mocks.onAuthenticated}
        onForgotPassword={mocks.onForgotPassword}
      />,
    );
    const user = await submitCredentials();

    const code = await screen.findByLabelText(/authentication code/i);
    expect(mocks.onAuthenticated).not.toHaveBeenCalled();
    expect(mocks.setUser).not.toHaveBeenCalled();

    await user.type(code, "123456");
    await user.click(screen.getByRole("button", { name: "Verify" }));

    await waitFor(() =>
      expect(mocks.verifyMFA).toHaveBeenCalledWith(
        expect.objectContaining({ challengeToken: "challenge-token", code: "123456" }),
        expect.anything(),
      ),
    );
    await waitFor(() =>
      expect(mocks.onAuthenticated).toHaveBeenCalledWith(
        expect.objectContaining({ sessionId: "ses_01K5F3ABCDEFGHJKMNPQRSTVWX" }),
      ),
    );
    expect(mocks.setUser).toHaveBeenCalledWith(expect.objectContaining({ id: "usr_1" }));
  });

  it("accepts a recovery code instead of an authenticator code", async () => {
    mocks.login.mockResolvedValue({
      mfaRequired: true,
      mfaChallengeToken: "challenge-token",
      expiresAt: 1782403304,
    });
    mocks.verifyMFA.mockResolvedValue(loginResponse());
    renderLoginForm(
      <LoginForm
        stepLabel="01 / 03"
        onAuthenticated={mocks.onAuthenticated}
        onForgotPassword={mocks.onForgotPassword}
      />,
    );
    const user = await submitCredentials();

    await user.click(await screen.findByRole("button", { name: "Use a recovery code" }));
    await user.type(screen.getByLabelText(/recovery code/i), "abcde-fghjk");
    await user.click(screen.getByRole("button", { name: "Verify" }));

    await waitFor(() =>
      expect(mocks.verifyMFA).toHaveBeenCalledWith(
        expect.objectContaining({ recoveryCode: "abcde-fghjk", code: "" }),
        expect.anything(),
      ),
    );
  });

  it("hands the second-factor step the flow's step label", async () => {
    mocks.login.mockResolvedValue({
      mfaRequired: true,
      mfaChallengeToken: "challenge-token",
      expiresAt: 1782403304,
    });
    renderLoginForm(
      <LoginForm
        stepLabel="01 / 02"
        onAuthenticated={mocks.onAuthenticated}
        onForgotPassword={mocks.onForgotPassword}
      />,
    );
    await submitCredentials();

    expect(await screen.findByText("01 / 02")).toBeInTheDocument();
  });

  it("is an office sign-in only, with no audience switch or Dash hand-off", () => {
    renderLoginForm(
      <LoginForm
        stepLabel="01 / 03"
        onAuthenticated={mocks.onAuthenticated}
        onForgotPassword={mocks.onForgotPassword}
      />,
    );

    expect(screen.queryByRole("tab")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /dash/i })).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Sign in" })).toBeInTheDocument();
  });

  it("refuses a malformed address before calling the server", async () => {
    const user = userEvent.setup();
    renderLoginForm(
      <LoginForm
        stepLabel="01 / 03"
        onAuthenticated={mocks.onAuthenticated}
        onForgotPassword={mocks.onForgotPassword}
      />,
    );

    await user.type(screen.getByLabelText("Email"), "not-an-address");
    await user.type(screen.getByLabelText("Password"), "password123");
    await user.click(screen.getByRole("button", { name: /sign in/i }));

    expect(
      await screen.findByText("Enter the work email you were invited with."),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toHaveAttribute("aria-invalid", "true");
    expect(mocks.login).not.toHaveBeenCalled();
  });

  it("reveals and hides the password from the eye button", async () => {
    const user = userEvent.setup();
    renderLoginForm(
      <LoginForm
        stepLabel="01 / 03"
        onAuthenticated={mocks.onAuthenticated}
        onForgotPassword={mocks.onForgotPassword}
      />,
    );

    const password = screen.getByLabelText("Password");
    expect(password).toHaveAttribute("type", "password");

    await user.click(screen.getByRole("button", { name: "Show password" }));
    expect(password).toHaveAttribute("type", "text");

    await user.click(screen.getByRole("button", { name: "Hide password" }));
    expect(password).toHaveAttribute("type", "password");
  });

  it("carries the typed address to recovery", async () => {
    const user = userEvent.setup();
    renderLoginForm(
      <LoginForm
        stepLabel="01 / 03"
        onAuthenticated={mocks.onAuthenticated}
        onForgotPassword={mocks.onForgotPassword}
      />,
    );

    await user.type(screen.getByLabelText("Email"), "dana@example.com");
    await user.click(screen.getByRole("button", { name: "Forgot?" }));

    expect(mocks.onForgotPassword).toHaveBeenCalledWith("dana@example.com");
  });

  it("bursts the stage on a valid submit and widens it once signed in", async () => {
    const stage: AuthStageControls = { burst: vi.fn(), setDone: vi.fn() };
    renderLoginForm(
      <AuthStageContext value={stage}>
        <LoginForm
          stepLabel="01 / 03"
          onAuthenticated={mocks.onAuthenticated}
          onForgotPassword={mocks.onForgotPassword}
        />
      </AuthStageContext>,
    );
    expect(stage.setDone).toHaveBeenLastCalledWith(false);

    await submitCredentials();

    await waitFor(() => expect(mocks.onAuthenticated).toHaveBeenCalled());
    expect(stage.burst).toHaveBeenCalledTimes(1);
    expect(stage.setDone).toHaveBeenLastCalledWith(true);
  });

  it("narrows the stage again when the flow cannot finish signing in", async () => {
    const stage: AuthStageControls = { burst: vi.fn(), setDone: vi.fn() };
    mocks.onAuthenticated.mockRejectedValue(new Error("organizations unavailable"));
    renderLoginForm(
      <AuthStageContext value={stage}>
        <LoginForm
          stepLabel="01 / 03"
          onAuthenticated={mocks.onAuthenticated}
          onForgotPassword={mocks.onForgotPassword}
        />
      </AuthStageContext>,
    );

    await submitCredentials();

    await waitFor(() => expect(stage.setDone).toHaveBeenLastCalledWith(false));
    expect(stage.setDone).toHaveBeenCalledWith(true);
  });

  it("shows a tenant's name, its SSO providers and the password form it allows", async () => {
    mocks.listProviders.mockResolvedValue([
      { id: "sso_1", name: "Contoso Entra", provider: "AzureAD" },
    ]);
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

    expect(screen.getByRole("heading", { name: "Alpha Logistics" })).toBeInTheDocument();
    expect(
      await screen.findByRole("link", { name: /continue with contoso entra/i }),
    ).toHaveAttribute("href", "/sso");
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
  });

  it("leaves out the password form on a tenant that signs in by SSO only", async () => {
    mocks.listProviders.mockResolvedValue([
      { id: "sso_1", name: "Contoso Okta", provider: "Okta" },
    ]);
    renderLoginForm(
      <LoginForm
        organizationSlug="alpha"
        tenantMetadata={{
          organizationId: "org_1",
          organizationName: "Alpha Logistics",
          organizationSlug: "alpha",
          enabledProviders: [],
          passwordEnabled: false,
          enforceSso: true,
        }}
        stepLabel="01 / 02"
        onAuthenticated={mocks.onAuthenticated}
        onForgotPassword={mocks.onForgotPassword}
      />,
    );

    expect(await screen.findByRole("link", { name: /continue with contoso okta/i })).toBeVisible();
    expect(screen.queryByLabelText("Email")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /sign in/i })).not.toBeInTheDocument();
  });
});
