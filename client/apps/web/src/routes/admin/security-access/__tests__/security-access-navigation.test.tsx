import { adminLinks } from "@/config/navigation.config";
import { useAccessibleAdminLinks } from "@/hooks/use-accessible-admin-links";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { publicConfigQueryOptions } from "@trenova/shared/hooks/use-public-config";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { SELF_HOSTED_PUBLIC_CONFIG } from "@trenova/shared/types/platform";
import type { User } from "@trenova/shared/types/user";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { securityAccess } from "../_components/security-access/use-security-access";

const PATH = "/admin/security-access";

function adminHrefs(permissions: Record<string, number>): string[] {
  useAuthStore.setState({
    user: { currentOrganizationId: "org_01", memberships: [] } as unknown as User,
    isAuthenticated: true,
  });
  seedPermissions(permissions);
  const queryClient = new QueryClient();
  queryClient.setQueryData(publicConfigQueryOptions.queryKey, SELF_HOSTED_PUBLIC_CONFIG);
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  const { result } = renderHook(() => useAccessibleAdminLinks(), { wrapper });
  return result.current.map((link) => link.href);
}

/** Every resource a tab of the page is shown on, read off the page's own access rules. */
function resourcesTheTabsRead(): string[] {
  const read = new Set<string>();
  securityAccess((resource, operation) => {
    if (operation === Operation.Read) read.add(resource);
    return false;
  });
  read.delete(Resource.User);
  read.delete(Resource.APIKey);
  return [...read].sort();
}

afterEach(() => {
  clearPermissions();
  useAuthStore.setState({ user: null, isAuthenticated: false });
});

describe("the Security & access admin link", () => {
  it("sits next to organization settings", () => {
    const hrefs = adminLinks.map((link) => link.href);
    expect(hrefs.indexOf(PATH)).toBe(hrefs.indexOf("/admin/organization-settings") + 1);
  });

  // The link is for whoever the page shows something to: one tab is enough.
  it("is listed for someone who may read only one part of the page", () => {
    expect(adminHrefs({ [Resource.AuthEvent]: Operation.Read })).toContain(PATH);
    expect(adminHrefs({ [Resource.SCIMDirectory]: Operation.Read })).toContain(PATH);
  });

  it("is absent for someone who may read none of it", () => {
    expect(adminHrefs({ [Resource.Shipment]: Operation.Read })).not.toContain(PATH);
  });

  // The page links to users and API keys from its briefing; those pages are not its tabs.
  it("is gated on exactly the resources the page's tabs read", () => {
    const link = adminLinks.find((entry) => entry.href === PATH);

    expect([...(link?.anyResources ?? [])].sort()).toEqual(resourcesTheTabsRead());
  });
});
