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

export function removeBundle(
  permissions: readonly AddPermission[],
  bundle: CapabilityBundle,
  bundles: readonly CapabilityBundle[],
): AddPermission[] {
  const kept = bundles.filter(
    (other) => other.id !== bundle.id && bundleState(other, permissions) === "full",
  );

  const remaining: AddPermission[] = [];
  const removing = new Map(bundle.grants.map((grant) => [grant.resource, grant.operations]));
  for (const permission of permissions) {
    const dropped = removing.get(permission.resource);
    if (!dropped) {
      remaining.push(permission);
      continue;
    }
    const operations = permission.operations.filter((operation) => !dropped.includes(operation));
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
