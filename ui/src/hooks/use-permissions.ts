import { useHasPermission } from "@/stores/user-store";
import type { Resource } from "@/types/audit-entry";
import type { Action } from "@/types/roles-permissions";

/**
 * Hook to check user permissions.
 * Provides a 'can' function to verify if the current user has a specific permission.
 *
 * @example
 * const { can } = usePermissions();
 * if (can('shipment', 'create')) {
 *   // User can create shipments
 * }
 */
export function usePermissions() {
  const hasPermission = useHasPermission();

  /**
   * Checks if the current user has the specified permission.
   * @param resource The resource type (e.g., 'shipment', 'user').
   * @param action The action to perform (e.g., 'create', 'read', 'update').
   * @returns True if the user has the permission, false otherwise.
   */
  const can = (resource: Resource, action: Action): boolean => {
    return hasPermission(resource, action);
  };

  return { can };
}
