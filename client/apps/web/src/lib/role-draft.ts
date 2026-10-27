/** A role as the editor holds it before it is saved. */
export type RoleDraft = {
  maxSensitivity: string;
  parentRoleIds: string[];
  permissions: { resource: string; operations: string[] }[];
};

/**
 * The role as the editor holds it, normalized so the same form state is
 * always the same request and the same cache entry.
 */
export function toRoleDraft(
  maxSensitivity: string,
  permissions: readonly { resource: string; operations: readonly string[] }[],
  parentRoleIds: readonly string[] = [],
): RoleDraft {
  return {
    maxSensitivity,
    parentRoleIds: [...parentRoleIds].sort(),
    permissions: permissions
      .map((permission) => ({
        resource: permission.resource,
        operations: [...permission.operations].sort(),
      }))
      .sort((a, b) => a.resource.localeCompare(b.resource)),
  };
}
