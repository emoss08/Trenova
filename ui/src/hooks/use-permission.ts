import { usePermissionContext } from "@/contexts/permission-context";
import type { ActionName, FieldAccess, Resource } from "@/types/permission";
import { useMemo } from "react";

/**
 * Hook for checking user permissions (v2 permission system)
 *
 * Provides optimized permission checking with sub-millisecond performance.
 * Uses the new policy-based permission system with bitfield operations.
 *
 * @example
 * const { can, canAny, canAll } = usePermissionV2();
 *
 * if (can('shipment', 'create')) {
 *   // Show create button
 * }
 *
 * if (canAny('shipment', ['update', 'delete'])) {
 *   // Show edit menu
 * }
 */
export function usePermissions() {
  const { can, canAny, canAll } = usePermissionContext();

  return useMemo(
    () => ({
      can,
      canAny,
      canAll,
    }),
    [can, canAny, canAll],
  );
}

/**
 * Hook for checking if user can access a specific resource/action
 *
 * @example
 * const canCreateShipment = useCanAccess('shipment', 'create');
 *
 * return (
 *   <button disabled={!canCreateShipment}>
 *     Create Shipment
 *   </button>
 * );
 */
export function useCanAccess(resource: Resource, action: ActionName): boolean {
  const { can } = usePermissionContext();

  return useMemo(() => can(resource, action), [can, resource, action]);
}

/**
 * Hook for checking field-level access
 *
 * @example
 * const { canAccess, canWrite, access } = useFieldAccess('shipment', 'price');
 *
 * if (!canAccess) {
 *   return null; // Hide field
 * }
 *
 * return (
 *   <input
 *     value={price}
 *     onChange={handleChange}
 *     disabled={!canWrite}
 *   />
 * );
 */
export function useFieldAccess(resource: Resource, field: string) {
  const { canAccessField, canWriteField, getFieldAccess } =
    usePermissionContext();

  return useMemo(() => {
    const canAccess = canAccessField(resource, field);
    const canWrite = canWriteField(resource, field);
    const access = getFieldAccess(resource, field);

    return {
      canAccess,
      canWrite,
      access,
      isReadOnly: access === "read_only",
      isHidden: access === "hidden" || !canAccess,
    };
  }, [canAccessField, canWriteField, getFieldAccess, resource, field]);
}

/**
 * Hook for checking multiple field access permissions at once
 *
 * @example
 * const fields = useFieldsAccess('shipment', ['price', 'quantity', 'notes']);
 *
 * return (
 *   <>
 *     {fields.price.canAccess && (
 *       <input disabled={!fields.price.canWrite} />
 *     )}
 *     {fields.quantity.canAccess && (
 *       <input disabled={!fields.quantity.canWrite} />
 *     )}
 *   </>
 * );
 */
export function useFieldsAccess(resource: Resource, fields: string[]) {
  const { canAccessField, canWriteField, getFieldAccess } =
    usePermissionContext();

  return useMemo(() => {
    const result: Record<
      string,
      {
        canAccess: boolean;
        canWrite: boolean;
        access: FieldAccess | null;
        isReadOnly: boolean;
        isHidden: boolean;
      }
    > = {};

    for (const field of fields) {
      const canAccess = canAccessField(resource, field);
      const canWrite = canWriteField(resource, field);
      const access = getFieldAccess(resource, field);

      result[field] = {
        canAccess,
        canWrite,
        access,
        isReadOnly: access === "read_only",
        isHidden: access === "hidden" || !canAccess,
      };
    }

    return result;
  }, [canAccessField, canWriteField, getFieldAccess, resource, fields]);
}

/**
 * Hook for organization management
 *
 * @example
 * const {
 *   currentOrg,
 *   availableOrgs,
 *   switchOrganization
 * } = useOrganization();
 *
 * return (
 *   <select
 *     value={currentOrg}
 *     onChange={(e) => switchOrganization(e.target.value)}
 *   >
 *     {availableOrgs.map(org => (
 *       <option key={org} value={org}>{org}</option>
 *     ))}
 *   </select>
 * );
 */
export function useOrganization() {
  const {
    getCurrentOrganization,
    getAvailableOrganizations,
    switchOrganization,
    isLoading,
  } = usePermissionContext();

  const currentOrg = getCurrentOrganization();
  const availableOrgs = getAvailableOrganizations();

  return useMemo(
    () => ({
      currentOrg,
      availableOrgs,
      switchOrganization,
      isLoading,
      canSwitch: availableOrgs.length > 1,
    }),
    [currentOrg, availableOrgs, switchOrganization, isLoading],
  );
}

/**
 * Hook for permission system utilities
 *
 * @example
 * const { refresh, invalidateCache, isExpired } = usePermissionUtils();
 *
 * if (isExpired) {
 *   return <button onClick={refresh}>Refresh Permissions</button>;
 * }
 */
export function usePermissionUtils() {
  const { refresh, invalidateCache, isExpired, isLoading, error, manifest } =
    usePermissionContext();

  return useMemo(
    () => ({
      refresh,
      invalidateCache,
      isExpired,
      isLoading,
      error,
      manifest,
    }),
    [refresh, invalidateCache, isExpired, isLoading, error, manifest],
  );
}
