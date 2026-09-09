import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AppLayout } from "./app-layout";

const mocks = vi.hoisted(() => ({
  activateSessionRoles: vi.fn(),
  fetchManifest: vi.fn(),
  manifest: {
    version: "1.0",
    userId: "usr_1",
    organizationId: "org_1",
    activeRoleIds: [],
    authorizedRoleIds: ["rol_dispatch", "rol_billing"],
    activeRoles: [],
    authorizedRoles: [
      {
        id: "rol_dispatch",
        name: "Dispatcher",
        description: "Coordinates dispatch activity",
        isSystem: false,
      },
      {
        id: "rol_billing",
        name: "Billing Admin",
        description: "",
        isSystem: false,
      },
    ],
    requiresRoleActivation: true,
    maxSensitivity: "internal",
    permissions: {},
    routeAccess: {},
    availableOrgs: [{ id: "org_1", name: "Alpha Logistics" }],
    checksum: "abc123",
    expiresAt: 1782403304,
  },
}));

vi.mock("@/hooks/use-permission-polling", () => ({
  usePermissionPolling: vi.fn(),
}));

vi.mock("@/hooks/use-realtime-connection", () => ({
  useRealtimeConnection: vi.fn(),
}));

vi.mock("@/services/update", () => ({
  updateService: {
    getVersion: vi.fn().mockResolvedValue({ version: "4.12.0", environment: "development" }),
    getNetworkPulse: vi.fn().mockRejectedValue(new Error("disabled")),
  },
}));

vi.mock("@trenova/shared/stores/auth-store", () => ({
  // Metadata reads the whole store while the gate uses a selector, so both call shapes
  // have to work.
  useAuthStore: (selector?: (state: Record<string, unknown>) => unknown) => {
    const state = {
      user: { emailAddress: "test@example.com", memberships: [] },
      isLoading: false,
    };
    return selector ? selector(state) : state;
  },
}));

vi.mock("@trenova/shared/services/auth", () => ({
  authService: {
    activateSessionRoles: mocks.activateSessionRoles,
  },
}));

vi.mock("@trenova/shared/stores/permission-store", () => ({
  usePermissionStore: (
    selector: (state: {
      manifest: typeof mocks.manifest;
      fetchManifest: typeof mocks.fetchManifest;
    }) => unknown,
  ) => selector({ manifest: mocks.manifest, fetchManifest: mocks.fetchManifest }),
}));

function renderAppLayout() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <AppLayout />
    </QueryClientProvider>,
  );
}

describe("AppLayout role activation", () => {
  beforeEach(() => {
    mocks.activateSessionRoles.mockClear();
    mocks.fetchManifest.mockClear();
    mocks.activateSessionRoles.mockResolvedValue({
      activeRoleIds: ["rol_dispatch"],
      authorizedRoleIds: ["rol_dispatch", "rol_billing"],
      activeRoles: [mocks.manifest.authorizedRoles[0]],
      authorizedRoles: mocks.manifest.authorizedRoles,
      requiresRoleActivation: false,
    });
    mocks.fetchManifest.mockResolvedValue(undefined);
  });

  it("renders role names and submits selected role IDs", async () => {
    const user = userEvent.setup();

    renderAppLayout();

    expect(screen.getByText("Dispatcher")).toBeInTheDocument();
    expect(screen.getByText("Coordinates dispatch activity")).toBeInTheDocument();
    expect(screen.getByText("Billing Admin")).toBeInTheDocument();
    expect(screen.queryByText("Org admin")).not.toBeInTheDocument();

    await user.click(screen.getByText("Dispatcher"));
    await user.click(screen.getByRole("button", { name: /activate 1 role/i }));

    await waitFor(() => expect(mocks.activateSessionRoles).toHaveBeenCalledWith(["rol_dispatch"]));
    expect(mocks.fetchManifest).toHaveBeenCalledTimes(1);
  });

  it("shows the credential receipt without a session row", async () => {
    renderAppLayout();

    expect(screen.getByText("Identity")).toBeInTheDocument();
    expect(screen.getByText("test@example.com")).toBeInTheDocument();
    expect(screen.getByText("Workspace")).toBeInTheDocument();
    expect(screen.getByText("Alpha Logistics")).toBeInTheDocument();
    // The browser only ever sees a session id at login, so the gate omits that row
    // rather than leaving one that can never fill.
    expect(screen.queryByText("Session")).not.toBeInTheDocument();
  });
});
