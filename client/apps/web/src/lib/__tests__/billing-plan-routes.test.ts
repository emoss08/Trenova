import { appNavigationModules } from "@/config/app-navigation";
import { isNavGroup, type NavGroup, type NavItem } from "@/config/navigation.types";
import { filterNavModules } from "@/hooks/use-filtered-navigation";
import { routes } from "@/router";
import { BASE_ONLY_UNAVAILABLE, seedPlan, unavailableFeatures } from "@/test/plan";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { User } from "@trenova/shared/types/user";
import type { LoaderFunctionArgs, RouteObject } from "react-router";
import { afterEach, describe, expect, it } from "vitest";

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
            name: "Carrier Co",
            brokerageEnabled: true,
            assetOperationsEnabled: true,
          },
        },
      ],
    } as unknown as User,
    isAuthenticated: true,
    checkAuth: async () => true,
    checkSession: async () => ({ status: "signed_in" }),
  } as unknown as Parameters<typeof useAuthStore.setState>[0]);
}

function joinPath(parent: string, child: string): string {
  if (child.startsWith("/")) return child;
  return `${parent.replace(/\/$/, "")}/${child}`;
}

function collectRoutes(
  entries: RouteObject[],
  parent = "",
  found = new Map<string, RouteObject>(),
): Map<string, RouteObject> {
  for (const entry of entries) {
    const path = entry.path === undefined ? parent : joinPath(parent, entry.path);
    if (entry.path !== undefined) found.set(path, entry);
    if (entry.children) collectRoutes(entry.children, path, found);
  }
  return found;
}

const routesByPath = collectRoutes(routes);

async function runLoader(path: string): Promise<{ value: unknown; thrown: unknown }> {
  const route = routesByPath.get(path);
  if (typeof route?.loader !== "function") throw new Error(`route ${path} has no loader`);
  try {
    const value = await route.loader({
      request: new Request(`http://localhost${path}`),
      params: {},
      context: undefined,
    } as unknown as LoaderFunctionArgs);
    return { value, thrown: undefined };
  } catch (thrown) {
    return { value: undefined, thrown };
  }
}

function itemIds(entries: (NavItem | NavGroup)[]): string[] {
  return entries.flatMap((entry) => (isNavGroup(entry) ? entry.items : [entry])).map((e) => e.id);
}

const invoiceReader = { [Resource.Invoice]: ALL_OPERATIONS };

afterEach(() => {
  usePermissionStore.getState().clearPermissions();
  useAuthStore.setState({ user: null, isAuthenticated: false });
});

describe("the factoring workspace on a plan without Accounting & Cash", () => {
  it("refuses the page with the plan refusal naming the add-on, not a blank page", async () => {
    signIn();
    seedPlan(invoiceReader, BASE_ONLY_UNAVAILABLE);

    const { thrown } = await runLoader("/billing/factoring");

    expect(thrown).toBeInstanceOf(Response);
    expect((thrown as Response).status).toBe(403);
    const body = (await (thrown as Response).json()) as Record<string, unknown>;
    expect(body.code).toBe("PLAN_RESTRICTED");
    expect(body.params).toEqual({ feature: "tms.factoring", addOn: "Accounting & Cash" });
  });

  it("opens the page when the plan includes factoring and the person reads invoices", async () => {
    signIn();
    seedPlan(invoiceReader, unavailableFeatures("ai.desk"));

    expect(await runLoader("/billing/factoring")).toEqual({ value: null, thrown: undefined });
  });
});

describe("billing and accounting navigation on a base-only plan", () => {
  function visible(permissions: Record<string, number>) {
    return filterNavModules(appNavigationModules, {
      hasPermission: (resource, operation) =>
        ((permissions[resource] ?? 0) & operation) === operation,
      capabilities: { brokerageEnabled: true, assetOperationsEnabled: true },
      unavailableFeatures: BASE_ONLY_UNAVAILABLE,
    });
  }

  it("leaves the factoring workspace out of billing although the person reads invoices", () => {
    const billing = visible(invoiceReader).find((module) => module.id === "billing");

    expect(billing).toBeDefined();
    expect(itemIds(billing!.navigation)).not.toContain("factoring");
    expect(itemIds(billing!.navigation)).toContain("invoices");
  });

  it("lists the invoice register in billing, right after invoices", () => {
    const billing = visible(invoiceReader).find((module) => module.id === "billing");
    const ids = itemIds(billing!.navigation);

    expect(ids.slice(ids.indexOf("invoices"), ids.indexOf("invoices") + 2)).toEqual([
      "invoices",
      "invoice-register",
    ]);
  });

  it("leaves the accounting menu out entirely when the plan leaves it nothing to show", () => {
    // The manifest drops every resource of an off-plan feature, so a person on a base plan holds
    // invoice access but nothing of Accounting & Cash.
    const modules = visible(invoiceReader);

    expect(modules.map((module) => module.id)).not.toContain("accounting");
  });

  it("keeps the accounting pages for a plan that includes the add-on", () => {
    const accounting = visible({
      ...invoiceReader,
      [Resource.AccountsReceivable]: Operation.Read,
      [Resource.CustomerPayment]: Operation.Read,
    }).find((module) => module.id === "accounting");

    expect(itemIds(accounting!.navigation)).toEqual(
      expect.arrayContaining([
        "accounting-dashboard",
        "ar-aging",
        "customer-ledger",
        "ar-open-items",
        "customer-payments",
      ]),
    );
    expect(itemIds(accounting!.navigation)).not.toContain("invoice-register");
  });

  it("serves the invoice register at its billing address", () => {
    expect(routesByPath.has("/billing/invoice-register")).toBe(true);
    expect(routesByPath.has("/accounting/ar/invoices")).toBe(false);
  });
});
