import { appAdminLinks } from "@/config/app-navigation";
import { useAccessibleAdminLinks } from "@/hooks/use-accessible-admin-links";
import { edition as hostEdition } from "@/lib/edition";
import { resolveEdition } from "@trenova/edition";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { publicConfigQueryOptions } from "@trenova/shared/hooks/use-public-config";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Resource, type PermissionManifest } from "@trenova/shared/types/permission";
import { SELF_HOSTED_PUBLIC_CONFIG, type PlatformMode } from "@trenova/shared/types/platform";
import type { User } from "@trenova/shared/types/user";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it } from "vitest";
import cloudEdition from "../index";
import { PLAN_USAGE_PATH } from "../lib/free-demo";
import { platformBilling } from "../lib/queries/platform-billing";
import { billingSummarySchema } from "../types/platform-billing";

const ALL_OPERATIONS = 0xffff;

function signIn() {
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
            name: "Demo Co",
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
      permissions: Object.fromEntries(
        Object.values(Resource).map((resource) => [resource, ALL_OPERATIONS]),
      ),
      routeAccess: {},
    } as unknown as PermissionManifest,
    lastFetched: Date.now(),
    isLoading: false,
  });
}

function renderAdminLinks(platformMode: PlatformMode) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  queryClient.setQueryData(publicConfigQueryOptions.queryKey, {
    ...SELF_HOSTED_PUBLIC_CONFIG,
    platformMode,
  });
  queryClient.setQueryData(
    platformBilling.summary().queryKey,
    billingSummarySchema.parse({ plan: { key: "free_demo" } }),
  );
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return renderHook(() => useAccessibleAdminLinks(), { wrapper });
}

describe("the Trenova Cloud edition", () => {
  afterEach(() => {
    usePermissionStore.getState().clearPermissions();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("is the edition the host compiled in", () => {
    expect(hostEdition.id).toBe("cloud");
  });

  it("mounts signup beside sign-in and the plan page in the admin area", () => {
    const { routes } = resolveEdition(cloudEdition);

    expect(routes.guest.map((route) => route.path)).toEqual(["/signup", "/signup/verify"]);
    expect(routes.admin.map((route) => route.path)).toEqual(["plan-usage"]);
    expect(routes.protected).toEqual([]);
    expect(routes.public).toEqual([]);
  });

  it("lists Plan & usage right after organization settings", () => {
    const hrefs = appAdminLinks.map((link) => link.href);

    expect(hrefs.indexOf(PLAN_USAGE_PATH)).toBe(hrefs.indexOf("/admin/organization-settings") + 1);
  });

  it("shows Plan & usage only on a Trenova Cloud install", () => {
    signIn();

    const selfHosted = renderAdminLinks("self_hosted").result.current.map((link) => link.href);
    const cloud = renderAdminLinks("cloud").result.current.map((link) => link.href);

    expect(selfHosted).not.toContain(PLAN_USAGE_PATH);
    expect(cloud).toContain(PLAN_USAGE_PATH);
    expect(selfHosted).toContain("/admin/organization-settings");
  });

  it("sends a visitor to the plan page from a route the plan withholds", () => {
    expect(hostEdition.plan.restrictedRedirect).toBe(PLAN_USAGE_PATH);
    expect(hostEdition.plan.usagePath).toBe(PLAN_USAGE_PATH);
  });

  it("ships a catalog for every locale the app translates into", async () => {
    for (const locale of ["es", "zh-TW", "zh-CN"] as const) {
      const loader = hostEdition.messages[locale];
      expect(loader).toBeDefined();
      await expect(loader?.()).resolves.toBeTypeOf("object");
    }
  });
});
