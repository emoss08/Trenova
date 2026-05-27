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
    availableOrgs: [],
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

vi.mock("@/services/auth", () => ({
  authService: {
    activateSessionRoles: mocks.activateSessionRoles,
  },
}));

vi.mock("@/stores/permission-store", () => ({
  usePermissionStore: (
    selector: (state: {
      manifest: typeof mocks.manifest;
      fetchManifest: typeof mocks.fetchManifest;
    }) => unknown,
  ) => selector({ manifest: mocks.manifest, fetchManifest: mocks.fetchManifest }),
}));

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

    render(<AppLayout />);

    expect(screen.getByText("Dispatcher")).toBeInTheDocument();
    expect(screen.getByText("Coordinates dispatch activity")).toBeInTheDocument();
    expect(screen.getByText("Billing Admin")).toBeInTheDocument();
    expect(screen.queryByText("Org admin")).not.toBeInTheDocument();

    await user.click(screen.getByText("Dispatcher"));
    await user.click(screen.getByRole("button", { name: /activate roles/i }));

    await waitFor(() => expect(mocks.activateSessionRoles).toHaveBeenCalledWith(["rol_dispatch"]));
    expect(mocks.fetchManifest).toHaveBeenCalledTimes(1);
  });
});
