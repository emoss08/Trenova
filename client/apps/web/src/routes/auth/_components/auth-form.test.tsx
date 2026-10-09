import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AuthForm } from "./auth-form";

const mocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  login: vi.fn(),
  logout: vi.fn(),
  listProviders: vi.fn(),
  getSSOStartUrl: vi.fn(() => "/sso"),
  activateSessionRoles: vi.fn(),
  getUserOrganizations: vi.fn(),
  switchOrganization: vi.fn(),
  currentUser: vi.fn(),
  setUser: vi.fn(),
  fetchManifest: vi.fn(),
  clearPermissions: vi.fn(),
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
    logout: mocks.logout,
    listProviders: mocks.listProviders,
    getSSOStartUrl: mocks.getSSOStartUrl,
    activateSessionRoles: mocks.activateSessionRoles,
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
  useAuthStore: (
    selector?: (state: { setUser: typeof mocks.setUser; user: null; isLoading: false }) => unknown,
  ) => {
    const state = { setUser: mocks.setUser, user: null, isLoading: false } as const;
    return selector ? selector(state) : state;
  },
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

const ADMIN_ROLE = {
  id: "rol_admin",
  name: "Organization Administrator",
  description: "Full access to every resource",
  isSystem: true,
  permissionCount: 214,
};

function manifest(overrides: Record<string, unknown> = {}) {
  return {
    version: "1",
    userId: "usr_1",
    organizationId: "org_1",
    activeRoleIds: [],
    authorizedRoleIds: [],
    activeRoles: [],
    authorizedRoles: [],
    requiresRoleActivation: false,
    maxSensitivity: "internal",
    permissions: {},
    routeAccess: {},
    availableOrgs: [{ id: "org_1", name: "Alpha Logistics" }],
    checksum: "abc",
    expiresAt: 1782403304,
    ...overrides,
  };
}

function organization(id: string, name: string, isCurrent: boolean) {
  return {
    id,
    name,
    city: "Austin",
    state: "TX",
    logoUrl: null,
    isDefault: isCurrent,
    isCurrent,
  };
}

function renderAuthForm() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <AuthForm />
      </MemoryRouter>
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

describe("AuthForm", () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockClear());
    mocks.listProviders.mockResolvedValue([]);
    mocks.activateSessionRoles.mockResolvedValue({});
    mocks.logout.mockResolvedValue(undefined);
    mocks.fetchManifest.mockResolvedValue(manifest());
    mocks.login.mockResolvedValue({
      user: {
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
      },
      sessionId: "ses_01K5F3ABCDEFGHJKMNPQRSTVWX",
      expiresAt: 1782403304,
      csrfToken: "csrf",
      activeRoleIds: [],
      authorizedRoleIds: [],
      activeRoles: [],
      authorizedRoles: [],
      requiresRoleActivation: false,
    });
    mocks.getUserOrganizations.mockResolvedValue([
      organization("org_1", "Alpha Logistics", true),
      organization("org_2", "Bravo Freight", false),
    ]);
    mocks.currentUser.mockResolvedValue({ id: "usr_1", currentOrganizationId: "org_1" });
    mocks.switchOrganization.mockResolvedValue({ id: "usr_1", currentOrganizationId: "org_2" });
  });

  it("transitions from login to a dedicated organization selection step", async () => {
    renderAuthForm();

    expect(screen.getByRole("heading", { name: "Sign in" })).toBeInTheDocument();
    await submitCredentials();

    await waitFor(() => expect(screen.getByText("Select organization")).toBeInTheDocument());
    expect(screen.queryByRole("heading", { name: "Sign in" })).not.toBeInTheDocument();
    expect(screen.getByText("02 / 03")).toBeInTheDocument();
    expect(screen.getByText("Alpha Logistics")).toBeInTheDocument();
    expect(screen.getByText("Bravo Freight")).toBeInTheDocument();
  });

  it("skips the organization step and runs the role step when the manifest demands one", async () => {
    mocks.getUserOrganizations.mockResolvedValue([organization("org_1", "Alpha Logistics", true)]);
    mocks.fetchManifest.mockResolvedValue(
      manifest({
        requiresRoleActivation: true,
        authorizedRoleIds: [ADMIN_ROLE.id],
        authorizedRoles: [ADMIN_ROLE],
      }),
    );

    renderAuthForm();
    await submitCredentials();

    await waitFor(() => expect(screen.getByText("Select active roles")).toBeInTheDocument());
    // Two steps, not three — the organization step never ran.
    expect(screen.getByText("02 / 02")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Scope this session at Alpha Logistics. You can switch later without signing out.",
      ),
    ).toBeInTheDocument();
  });

  it("goes straight to the hand-off when the session already has active roles", async () => {
    mocks.getUserOrganizations.mockResolvedValue([organization("org_1", "Alpha Logistics", true)]);
    mocks.fetchManifest.mockResolvedValue(
      manifest({ activeRoleIds: [ADMIN_ROLE.id], activeRoles: [ADMIN_ROLE] }),
    );

    renderAuthForm();
    await submitCredentials();

    await waitFor(() => expect(screen.getByText("Opening Alpha Logistics")).toBeInTheDocument());
    expect(screen.getByRole("heading", { name: "Welcome back, Test." })).toBeInTheDocument();
    expect(screen.getByText("1 role · 214 permissions")).toBeInTheDocument();
    await waitFor(() => expect(mocks.navigate).toHaveBeenCalledWith("/", { replace: true }), {
      timeout: 4000,
    });
  });

  it("ends the session when the user steps back out of the organization choice", async () => {
    renderAuthForm();
    const user = await submitCredentials();

    await waitFor(() => expect(screen.getByText("Select organization")).toBeInTheDocument());
    await user.click(screen.getByRole("button", { name: /back/i }));

    await waitFor(() => expect(mocks.logout).toHaveBeenCalledTimes(1));
    expect(screen.getByRole("heading", { name: "Sign in" })).toBeInTheDocument();
  });
});
