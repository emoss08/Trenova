import { useAccessibleAdminLinks } from "@/hooks/use-accessible-admin-links";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { publicConfigQueryOptions } from "@trenova/shared/hooks/use-public-config";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Resource, type PermissionManifest } from "@trenova/shared/types/permission";
import { SELF_HOSTED_PUBLIC_CONFIG } from "@trenova/shared/types/platform";
import type { User } from "@trenova/shared/types/user";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it } from "vitest";

const ALL_OPERATIONS = 0xffff;
const PLAN_PATH = "/admin/plan";

function signIn(permissions: Record<string, number>) {
  useAuthStore.setState({
    user: {
      currentOrganizationId: "org_01",
      memberships: [
        {
          userId: "usr_01",
          organizationId: "org_01",
          isDefault: true,
          organization: {
            id: "org_01",
            name: "Acme Freight",
            brokerageEnabled: true,
            assetOperationsEnabled: true,
          },
        },
      ],
    } as unknown as User,
    isAuthenticated: true,
  });
  usePermissionStore.setState({
    manifest: {
      version: "1.0",
      userId: "usr_01",
      organizationId: "org_01",
      activeRoleIds: [],
      authorizedRoleIds: [],
      activeRoles: [],
      authorizedRoles: [],
      requiresRoleActivation: false,
      maxSensitivity: "internal",
      permissions,
      routeAccess: {},
      availableOrgs: [],
      unavailableFeatures: [],
      checksum: "abc123",
      expiresAt: 1_782_403_304,
    } as unknown as PermissionManifest,
    lastFetched: Date.now(),
    isLoading: false,
  });
}

function everyPermission(): Record<string, number> {
  return Object.fromEntries(Object.values(Resource).map((resource) => [resource, ALL_OPERATIONS]));
}

function adminHrefs(managed: boolean): string[] {
  const queryClient = new QueryClient();
  queryClient.setQueryData(publicConfigQueryOptions.queryKey, {
    ...SELF_HOSTED_PUBLIC_CONFIG,
    managed,
  });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  const { result } = renderHook(() => useAccessibleAdminLinks(), { wrapper });
  return result.current.map((link) => link.href);
}

describe("the Your plan admin link", () => {
  afterEach(() => {
    usePermissionStore.getState().clearPermissions();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("is absent on a self-hosted install, even for an administrator", () => {
    signIn(everyPermission());

    expect(adminHrefs(false)).not.toContain(PLAN_PATH);
  });

  it("is listed on a managed instance for someone who may update the organization", () => {
    signIn(everyPermission());

    expect(adminHrefs(true)).toContain(PLAN_PATH);
  });

  it("is absent on a managed instance for someone who may only read the organization", () => {
    signIn({ ...everyPermission(), [Resource.Organization]: 1 });

    expect(adminHrefs(true)).not.toContain(PLAN_PATH);
  });
});
