import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { LoginForm } from "./login-form";

const mocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  login: vi.fn(),
  listProviders: vi.fn(),
  getSSOStartUrl: vi.fn(() => "/sso"),
  getUserOrganizations: vi.fn(),
  switchOrganization: vi.fn(),
  currentUser: vi.fn(),
  setUser: vi.fn(),
  fetchManifest: vi.fn(),
  clearPermissions: vi.fn(),
  onOrganizationSelectionRequired: vi.fn(),
}));

vi.mock("react-router", async (importActual) => {
  const actual = await importActual<typeof import("react-router")>();
  return {
    ...actual,
    useNavigate: () => mocks.navigate,
  };
});

vi.mock("@trenova/shared/services/auth", () => ({
  authService: {
    login: mocks.login,
    listProviders: mocks.listProviders,
    getSSOStartUrl: mocks.getSSOStartUrl,
  },
}));

vi.mock("@/services/api", () => ({
  apiService: {
    userService: {
      getUserOrganizations: mocks.getUserOrganizations,
      switchOrganization: mocks.switchOrganization,
      currentUser: mocks.currentUser,
    },
  },
}));

vi.mock("@trenova/shared/stores/auth-store", () => ({
  useAuthStore: (selector: (state: { setUser: typeof mocks.setUser }) => unknown) =>
    selector({ setUser: mocks.setUser }),
}));

vi.mock("@trenova/shared/stores/permission-store", () => ({
  usePermissionStore: (
    selector: (state: {
      fetchManifest: typeof mocks.fetchManifest;
      clearPermissions: typeof mocks.clearPermissions;
    }) => unknown,
  ) =>
    selector({
      fetchManifest: mocks.fetchManifest,
      clearPermissions: mocks.clearPermissions,
    }),
}));

function testUser(currentOrganizationId = "org_1") {
  return {
    id: "usr_1",
    version: 1,
    createdAt: 1,
    updatedAt: 1,
    businessUnitId: "bu_1",
    currentOrganizationId,
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
    sessionId: "ses_1",
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
  await user.type(screen.getByPlaceholderText("*****"), "password123");
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
    mocks.fetchManifest.mockResolvedValue(undefined);
    mocks.getUserOrganizations.mockResolvedValue([
      {
        id: "org_1",
        name: "Alpha Logistics",
        city: "Austin",
        state: "TX",
        logoUrl: null,
        isDefault: true,
        isCurrent: true,
      },
    ]);
    mocks.switchOrganization.mockResolvedValue(testUser("org_2"));
    mocks.currentUser.mockResolvedValue(testUser("org_1"));
  });

  it("requests organization selection before navigation for non-slug multi-org login", async () => {
    const organizations = [
      {
        id: "org_1",
        name: "Alpha Logistics",
        city: "Austin",
        state: "TX",
        logoUrl: null,
        isDefault: true,
        isCurrent: true,
      },
      {
        id: "org_2",
        name: "Bravo Freight",
        city: "Denver",
        state: "CO",
        logoUrl: null,
        isDefault: false,
        isCurrent: false,
      },
    ];
    mocks.getUserOrganizations.mockResolvedValue([...organizations]);

    renderLoginForm(
      <LoginForm onOrganizationSelectionRequired={mocks.onOrganizationSelectionRequired} />,
    );
    await submitCredentials();

    await waitFor(() =>
      expect(mocks.onOrganizationSelectionRequired).toHaveBeenCalledWith(organizations),
    );
    expect(mocks.clearPermissions).toHaveBeenCalled();
    expect(mocks.navigate).not.toHaveBeenCalled();
    expect(mocks.fetchManifest).not.toHaveBeenCalled();
  });

  it("skips organization selection for slug login", async () => {
    renderLoginForm(<LoginForm organizationSlug="alpha" />);
    await submitCredentials();

    await waitFor(() => expect(mocks.fetchManifest).toHaveBeenCalledTimes(1));
    expect(mocks.getUserOrganizations).not.toHaveBeenCalled();
    expect(mocks.navigate).toHaveBeenCalledWith("/", { replace: true });
  });
});
