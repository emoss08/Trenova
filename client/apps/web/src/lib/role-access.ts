import { impliedOperations } from "@/lib/implied-operations";
import type { ResourceDefinition } from "@/lib/role-api";
import type { AddPermission, DataScope, Operation } from "@trenova/shared/types/role";

export type AccessLevel = "none" | "view" | "full" | "custom";

export type ResourceAccess = {
  level: AccessLevel;
  /** Operations the grant reaches, held outright or implied by one that is. */
  reached: number;
  total: number;
};

const defaultScope: DataScope = "organization";

export function resourceAccess(
  resource: ResourceDefinition,
  granted: readonly string[],
): ResourceAccess {
  const total = resource.operations.length;
  if (granted.length === 0) {
    return { level: "none", reached: 0, total };
  }

  const held = new Set<string>([
    ...granted,
    ...impliedOperations(granted, resource.operations).keys(),
  ]);
  const reached = resource.operations.filter((op) => held.has(op.operation)).length;

  if (reached === total) {
    return { level: "full", reached, total };
  }
  if (reached === 1 && held.has("read")) {
    return { level: "view", reached, total };
  }
  return { level: "custom", reached, total };
}

/** The preset levels a resource offers: View only when it reads and does more than read. */
export function presetLevels(resource: ResourceDefinition): AccessLevel[] {
  const reads = resource.operations.some((op) => op.operation === "read");
  return reads && resource.operations.length > 1 ? ["none", "view", "full"] : ["none", "full"];
}

function operationsForLevel(
  resource: ResourceDefinition,
  level: AccessLevel,
): Operation[] | undefined {
  switch (level) {
    case "none":
      return [];
    case "view":
      return resource.operations.some((op) => op.operation === "read") ? ["read"] : undefined;
    case "full":
      return resource.operations.map((op) => op.operation as Operation);
    case "custom":
      return undefined;
  }
}

/**
 * Replaces the operations of each resource in `updates`, dropping a resource
 * left with none and adding one the role did not hold at the default scope.
 */
function replaceOperations(
  permissions: readonly AddPermission[],
  updates: ReadonlyMap<string, Operation[]>,
): AddPermission[] {
  const pending = new Map(updates);
  const next: AddPermission[] = [];

  for (const permission of permissions) {
    const operations = pending.get(permission.resource);
    if (operations === undefined) {
      next.push(permission);
      continue;
    }
    pending.delete(permission.resource);
    if (operations.length > 0) {
      next.push({ ...permission, operations });
    }
  }

  for (const [resource, operations] of pending) {
    if (operations.length > 0) {
      next.push({ resource, operations, dataScope: defaultScope });
    }
  }

  return next;
}

export function setAccessLevel(
  permissions: readonly AddPermission[],
  resources: readonly ResourceDefinition[],
  level: AccessLevel,
): AddPermission[] {
  const updates = new Map<string, Operation[]>();
  for (const resource of resources) {
    const operations = operationsForLevel(resource, level);
    if (operations !== undefined) {
      updates.set(resource.resource, operations);
    }
  }
  return replaceOperations(permissions, updates);
}

export function toggleOperation(
  permissions: readonly AddPermission[],
  resource: string,
  operation: Operation,
): AddPermission[] {
  const held = permissions.find((permission) => permission.resource === resource)?.operations ?? [];
  const operations = held.includes(operation)
    ? held.filter((candidate) => candidate !== operation)
    : [...held, operation];
  return replaceOperations(permissions, new Map([[resource, operations]]));
}

export function setDataScope(
  permissions: readonly AddPermission[],
  resource: string,
  dataScope: DataScope,
): AddPermission[] {
  return permissions.map((permission) =>
    permission.resource === resource ? { ...permission, dataScope } : permission,
  );
}
