import type { QuickActionCommand } from "@/config/navigation.types";
import type { OrganizationCapabilities } from "@trenova/shared/types/organization-capability";
import { Operation } from "@trenova/shared/types/permission";
import { describe, expect, it } from "vitest";
import {
  buildAppCommands,
  buildQuickActionCommands,
  type AppCommandContext,
} from "../palette-commands";
import { t } from "./palette-test-fixtures";

const capabilities = {} as OrganizationCapabilities;

const QUICK_ACTIONS: QuickActionCommand[] = [
  {
    id: "create-shipment",
    label: "New shipment",
    description: "Add a new shipment",
    path: "/shipment-management/shipments",
    resource: "shipment",
    requiredOperation: Operation.Create,
    query: { panelType: "create" },
  },
  {
    id: "open-insights",
    label: "Operational insights",
    description: "Findings",
    path: "/insights",
    resource: "insight",
    requiredOperation: Operation.Read,
  },
  {
    id: "open-assistant",
    label: "Open assistant",
    description: "Ask",
    path: "/assistant",
    action: "open-assistant",
    resource: "assistant",
    requiredOperation: Operation.Read,
  },
];

function appContext(overrides: Partial<AppCommandContext> = {}): AppCommandContext {
  return {
    theme: "system",
    unreadNotifications: 0,
    canUseAssistant: true,
    currentHref: "/billing/invoices?item=inv_1",
    mac: true,
    ...overrides,
  };
}

describe("buildQuickActionCommands", () => {
  it("files each quick action where a person would look for it", () => {
    const commands = buildQuickActionCommands(
      QUICK_ACTIONS,
      () => undefined,
      { hasPermission: () => true, capabilities },
      t,
    );

    expect(commands.map((entry) => [entry.id, entry.group])).toEqual([
      ["quick:create-shipment", "create"],
      ["quick:open-insights", "navigation"],
      ["quick:open-assistant", "assistant"],
    ]);
    expect(commands[0]?.intent).toEqual({
      type: "navigate",
      href: "/shipment-management/shipments?panelType=create",
    });
    expect(commands[2]?.intent).toEqual({ type: "assistant", mode: "open" });
  });

  it("drops a quick action the person lacks the permission for", () => {
    const commands = buildQuickActionCommands(
      QUICK_ACTIONS,
      () => undefined,
      {
        hasPermission: (resource, operation) =>
          !(resource === "shipment" && operation === Operation.Create),
        capabilities,
      },
      t,
    );

    expect(commands.map((entry) => entry.id)).not.toContain("quick:create-shipment");
  });
});

describe("buildAppCommands", () => {
  const idsOf = (context: AppCommandContext) =>
    buildAppCommands(context, t).map((entry) => entry.id);

  it("checks the theme in use and only that one", () => {
    const commands = buildAppCommands(appContext({ theme: "dark" }), t);
    const checked = commands.filter((entry) => entry.checked).map((entry) => entry.id);

    expect(checked).toEqual(["app:theme-dark"]);
  });

  it("offers to mark notifications read only when some are unread", () => {
    expect(idsOf(appContext({ unreadNotifications: 0 }))).not.toContain(
      "app:notifications-read-all",
    );
    expect(idsOf(appContext({ unreadNotifications: 3 }))).toContain("app:notifications-read-all");
  });

  it("leaves the assistant out for someone who cannot use it", () => {
    expect(idsOf(appContext({ canUseAssistant: false }))).not.toContain("app:assistant-new");
  });

  it("copies the page as it is, query string included", () => {
    const copy = buildAppCommands(appContext(), t).find(
      (entry) => entry.id === "app:copy-page-link",
    );

    expect(copy?.intent).toEqual({ type: "copy-link", href: "/billing/invoices?item=inv_1" });
  });

  it("marks signing out as the one destructive command", () => {
    const destructive = buildAppCommands(appContext(), t).filter((entry) => entry.destructive);

    expect(destructive.map((entry) => entry.id)).toEqual(["app:sign-out"]);
  });
});
