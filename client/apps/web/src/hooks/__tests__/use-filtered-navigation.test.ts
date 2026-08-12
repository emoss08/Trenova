import { canAccessNavEntry, filterNavModules } from "@/hooks/use-filtered-navigation";
import type { NavAccessContext } from "@/hooks/use-filtered-navigation";
import type { NavGroup, NavItem, NavModule } from "@/config/navigation.types";
import {
  OrganizationCapability,
  type OrganizationCapabilities,
} from "@trenova/shared/types/organization-capability";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { describe, expect, it } from "vitest";

const brokerageOn: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: true,
};
const brokerageOff: OrganizationCapabilities = {
  brokerageEnabled: false,
  assetOperationsEnabled: true,
};

function context(
  capabilities: OrganizationCapabilities,
  granted: Record<string, number> = {},
): NavAccessContext {
  return {
    capabilities,
    hasPermission: (resource, operation) => ((granted[resource] ?? 0) & operation) === operation,
  };
}

const carriersItem: NavItem = {
  id: "carriers",
  label: "Carriers",
  path: "/dispatch/carriers",
  resource: Resource.Carrier,
  capability: OrganizationCapability.Brokerage,
};

const workersItem: NavItem = {
  id: "workers",
  label: "Workers",
  path: "/dispatch/workers",
  resource: Resource.Worker,
};

const unguardedItem: NavItem = {
  id: "home",
  label: "Home",
  path: "/",
};

const capabilityOnlyItem: NavItem = {
  id: "routing-guides",
  label: "Routing Guides",
  path: "/dispatch/routing-guides",
  capability: OrganizationCapability.Brokerage,
};

describe("canAccessNavEntry", () => {
  it("allows an entry whose capability is on and whose permission is granted", () => {
    expect(
      canAccessNavEntry(carriersItem, context(brokerageOn, { [Resource.Carrier]: Operation.Read })),
    ).toBe(true);
  });

  it("denies an entry whose capability is on but whose permission is missing", () => {
    expect(canAccessNavEntry(carriersItem, context(brokerageOn, {}))).toBe(false);
  });

  it("denies an entry whose capability is off even when the permission is granted", () => {
    expect(
      canAccessNavEntry(
        carriersItem,
        context(brokerageOff, { [Resource.Carrier]: Operation.Read }),
      ),
    ).toBe(false);
  });

  it("denies an entry whose capability is off and whose permission is missing", () => {
    expect(canAccessNavEntry(carriersItem, context(brokerageOff, {}))).toBe(false);
  });

  it("leaves entries without a capability field governed by permissions alone", () => {
    const granted = context(brokerageOff, { [Resource.Worker]: Operation.Read });
    expect(canAccessNavEntry(workersItem, granted)).toBe(true);
    expect(canAccessNavEntry(workersItem, context(brokerageOff, {}))).toBe(false);
  });

  it("always allows entries with neither a capability nor a resource", () => {
    expect(canAccessNavEntry(unguardedItem, context(brokerageOff, {}))).toBe(true);
  });

  it("gates a capability-only entry on the capability alone", () => {
    expect(canAccessNavEntry(capabilityOnlyItem, context(brokerageOn, {}))).toBe(true);
    expect(canAccessNavEntry(capabilityOnlyItem, context(brokerageOff, {}))).toBe(false);
  });

  it("checks a capability on a group and on a module the same way", () => {
    const group: NavGroup = {
      id: "carrier-config",
      label: "Carrier Configuration",
      items: [],
      capability: OrganizationCapability.Brokerage,
    };
    const module: NavModule = {
      id: "carrier-settlements",
      label: "Carrier Settlements",
      icon: () => null,
      basePath: "/carrier-settlements",
      navigation: [],
      capability: OrganizationCapability.Brokerage,
    };

    expect(canAccessNavEntry(group, context(brokerageOn, {}))).toBe(true);
    expect(canAccessNavEntry(group, context(brokerageOff, {}))).toBe(false);
    expect(canAccessNavEntry(module, context(brokerageOn, {}))).toBe(true);
    expect(canAccessNavEntry(module, context(brokerageOff, {}))).toBe(false);
  });
});

describe("filterNavModules", () => {
  const module: NavModule = {
    id: "dispatch",
    label: "Dispatch",
    icon: () => null,
    basePath: "/dispatch",
    navigation: [
      workersItem,
      carriersItem,
      {
        id: "carrier-config-group",
        label: "Carrier Configuration",
        items: [carriersItem],
      },
    ],
  };

  const granted = {
    [Resource.Worker]: Operation.Read,
    [Resource.Carrier]: Operation.Read,
  };

  it("keeps every entry when the capability is on", () => {
    const [filtered] = filterNavModules([module], context(brokerageOn, granted));

    expect(filtered.navigation.map((entry) => entry.id)).toEqual([
      "workers",
      "carriers",
      "carrier-config-group",
    ]);
  });

  it("drops the gated item and collapses the group left empty by it", () => {
    const [filtered] = filterNavModules([module], context(brokerageOff, granted));

    expect(filtered.navigation.map((entry) => entry.id)).toEqual(["workers"]);
  });

  it("drops a module whose navigation empties out", () => {
    const brokerageOnly: NavModule = {
      ...module,
      id: "carrier-settlements",
      navigation: [carriersItem],
    };

    expect(filterNavModules([brokerageOnly], context(brokerageOff, granted))).toEqual([]);
  });

  it("drops a module gated by capability before looking at its children", () => {
    const gatedModule: NavModule = {
      ...module,
      id: "carrier-settlements",
      capability: OrganizationCapability.Brokerage,
      navigation: [workersItem],
    };

    expect(filterNavModules([gatedModule], context(brokerageOff, granted))).toEqual([]);
    expect(filterNavModules([gatedModule], context(brokerageOn, granted))).toHaveLength(1);
  });

  it("keeps a chrome-less module even when every child is filtered away", () => {
    const chromeless: NavModule = {
      ...module,
      id: "home",
      hideSecondarySidebar: true,
      navigation: [carriersItem],
    };

    expect(filterNavModules([chromeless], context(brokerageOff, granted))).toHaveLength(1);
  });
});
