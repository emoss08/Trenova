import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AuthForm, formatSessionId } from "./auth-form";

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
  getVersion: vi.fn(),
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

vi.mock("@/services/update", () => ({
  updateService: { getVersion: mocks.getVersion },
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
  await user.type(screen.getByPlaceholderText("name@work-email.com"), "test@example.com");
  await user.type(screen.getByLabelText(/password/i), "password123");
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
    mocks.getVersion.mockResolvedValue({ version: "4.12.0", environment: "development" });
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

    expect(screen.getByText("Welcome back")).toBeInTheDocument();
    expect(screen.getByText("01 / 03")).toBeInTheDocument();
    await submitCredentials();

    await waitFor(() => expect(screen.getByText("Select organization")).toBeInTheDocument());
    expect(screen.queryByText("Welcome back")).not.toBeInTheDocument();
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

    await waitFor(() => expect(screen.getByText("Entering Alpha Logistics")).toBeInTheDocument());
    expect(screen.getByText("1 role · 214 permissions")).toBeInTheDocument();
    await waitFor(() => expect(mocks.navigate).toHaveBeenCalledWith("/", { replace: true }), {
      timeout: 4000,
    });
  });

  it("fills the credential receipt with a trimmed session id once the credential is issued", async () => {
    mocks.getUserOrganizations.mockResolvedValue([organization("org_1", "Alpha Logistics", true)]);
    mocks.fetchManifest.mockResolvedValue(
      manifest({ activeRoleIds: [ADMIN_ROLE.id], activeRoles: [ADMIN_ROLE] }),
    );

    renderAuthForm();
    await submitCredentials();

    await waitFor(() => expect(screen.getByText("Issued")).toBeInTheDocument());
    expect(screen.getByText("ses_01K5F3AB")).toBeInTheDocument();
    expect(screen.queryByText("ses_01K5F3ABCDEFGHJKMNPQRSTVWX")).not.toBeInTheDocument();
    expect(screen.getByText("Authorized")).toBeInTheDocument();
  });

  it("ends the session when the user steps back out of the organization choice", async () => {
    renderAuthForm();
    const user = await submitCredentials();

    await waitFor(() => expect(screen.getByText("Select organization")).toBeInTheDocument());
    await user.click(screen.getByRole("button", { name: /back/i }));

    await waitFor(() => expect(mocks.logout).toHaveBeenCalledTimes(1));
    expect(screen.getByText("Welcome back")).toBeInTheDocument();
  });
});

describe("formatSessionId", () => {
  it("keeps the prefix and the leading identifier characters", () => {
    expect(formatSessionId("ses_01K5F3ABCDEFGHJKMNPQRSTVWX")).toBe("ses_01K5F3AB");
  });

  it("trims an unprefixed id to the same body length", () => {
    expect(formatSessionId("01K5F3ABCDEFGHJKMNPQRSTVWX")).toBe("01K5F3AB");
  });

  it("passes through an absent id", () => {
    expect(formatSessionId(undefined)).toBeUndefined();
  });
});
