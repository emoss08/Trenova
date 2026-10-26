import type { BundleGrant, CapabilityBundle } from "@/lib/role-api";
import type { AddPermission, DataScope, Operation } from "@trenova/shared/types/role";

export type BundleState = "full" | "partial" | "none";

const scopeRank: Record<DataScope, number> = {
  own: 0,
  team: 1,
  organization: 2,
  business_unit: 3,
  all: 4,
};

function byResource(permissions: readonly AddPermission[]): Map<string, AddPermission> {
  return new Map(permissions.map((permission) => [permission.resource, permission]));
}

export function bundleState(
  bundle: CapabilityBundle,
  permissions: readonly AddPermission[],
): BundleState {
  const held = byResource(permissions);
  let total = 0;
  let present = 0;
  for (const grant of bundle.grants) {
    const operations = held.get(grant.resource)?.operations ?? [];
    for (const operation of grant.operations) {
      total += 1;
      if (operations.includes(operation)) {
        present += 1;
      }
    }
  }
  if (present === 0) {
    return "none";
  }
  return present === total ? "full" : "partial";
}

export function grantsToPermissions(grants: readonly BundleGrant[]): AddPermission[] {
  return mergeGrants([], grants);
}

export function applyBundle(
  permissions: readonly AddPermission[],
  bundle: CapabilityBundle,
): AddPermission[] {
  return mergeGrants(permissions, bundle.grants);
}

/**
 * Takes away every operation the removed bundles grant, all at once, then
 * gives back what each other fully held bundle grants. Removing them together
 * matters: one at a time, a bundle inside another would come back with it.
 */
export function removeBundles(
  permissions: readonly AddPermission[],
  removed: readonly CapabilityBundle[],
  bundles: readonly CapabilityBundle[],
): AddPermission[] {
  const removedIds = new Set(removed.map((bundle) => bundle.id));
  const kept = bundles.filter(
    (other) => !removedIds.has(other.id) && bundleState(other, permissions) === "full",
  );

  const dropping = new Map<string, Set<Operation>>();
  for (const bundle of removed) {
    for (const grant of bundle.grants) {
      const operations = dropping.get(grant.resource) ?? new Set<Operation>();
      for (const operation of grant.operations) {
        operations.add(operation);
      }
      dropping.set(grant.resource, operations);
    }
  }

  const remaining: AddPermission[] = [];
  for (const permission of permissions) {
    const dropped = dropping.get(permission.resource);
    if (!dropped) {
      remaining.push(permission);
      continue;
    }
    const operations = permission.operations.filter((operation) => !dropped.has(operation));
    if (operations.length > 0) {
      remaining.push({ ...permission, operations });
    }
  }

  return kept.reduce<AddPermission[]>((acc, other) => mergeGrants(acc, other.grants), remaining);
}

export function bundlesIncluding(
  resource: string,
  bundles: readonly CapabilityBundle[],
  permissions: readonly AddPermission[],
): CapabilityBundle[] {
  return bundles.filter(
    (bundle) =>
      bundle.grants.some((grant) => grant.resource === resource) &&
      bundleState(bundle, permissions) === "full",
  );
}

function mergeGrants(
  permissions: readonly AddPermission[],
  grants: readonly BundleGrant[],
): AddPermission[] {
  const merged = new Map<string, AddPermission>(
    permissions.map((permission) => [permission.resource, { ...permission }]),
  );
  const order = permissions.map((permission) => permission.resource);

  for (const grant of grants) {
    const existing = merged.get(grant.resource);
    if (!existing) {
      merged.set(grant.resource, {
        resource: grant.resource,
        operations: [...grant.operations],
        dataScope: grant.dataScope,
      });
      order.push(grant.resource);
      continue;
    }
    const operations = new Set<Operation>(existing.operations);
    for (const operation of grant.operations) {
      operations.add(operation);
    }
    existing.operations = [...operations];
    if (scopeRank[grant.dataScope] > scopeRank[existing.dataScope]) {
      existing.dataScope = grant.dataScope;
    }
  }

  return order.map((resource) => merged.get(resource)!);
}
