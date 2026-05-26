import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AuthForm } from "./auth-form";

const mocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  login: vi.fn(),
  getSSOStartUrl: vi.fn(() => "/sso"),
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

vi.mock("@/services/auth", () => ({
  authService: {
    login: mocks.login,
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

vi.mock("@/stores/auth-store", () => ({
  useAuthStore: (
    selector?: (state: { setUser: typeof mocks.setUser; user: null; isLoading: false }) => unknown,
  ) => {
    const state = { setUser: mocks.setUser, user: null, isLoading: false } as const;
    return selector ? selector(state) : state;
  },
}));

vi.mock("@/stores/permission-store", () => ({
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
  await user.type(screen.getByPlaceholderText("*****"), "password123");
  await user.click(screen.getByRole("button", { name: /sign in/i }));
}

describe("AuthForm", () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockClear());
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
        isPlatformAdmin: false,
      },
      sessionId: "ses_1",
      expiresAt: 1782403304,
      csrfToken: "csrf",
      activeRoleIds: [],
      authorizedRoleIds: [],
      activeRoles: [],
      authorizedRoles: [],
      requiresRoleActivation: false,
    });
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
      {
        id: "org_2",
        name: "Bravo Freight",
        city: "Denver",
        state: "CO",
        logoUrl: null,
        isDefault: false,
        isCurrent: false,
      },
    ]);
  });

  it("transitions from login to a dedicated organization selection step", async () => {
    renderAuthForm();

    expect(screen.getByText("Welcome back!")).toBeInTheDocument();
    await submitCredentials();

    await waitFor(() => expect(screen.queryByText("Welcome back!")).not.toBeInTheDocument());
    expect(screen.getByText("Select organization")).toBeInTheDocument();
    expect(screen.getByText("Alpha Logistics")).toBeInTheDocument();
    expect(screen.getByText("Bravo Freight")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Continue" })).toBeInTheDocument();
  });
});
