import { PROTECTED_FIELD_GROUP_RESOURCES, type ProtectedFieldGroup } from "@/lib/protected-fields";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation } from "@trenova/shared/types/permission";

/**
 * Whether the viewer's role grants a group of protected fields. The server
 * withholds the group's fields from anyone without it, so a null in one of
 * them means "hidden from you" rather than "not on file".
 */
export function useProtectedFieldAccess(group: ProtectedFieldGroup): boolean {
  return usePermissionStore((state) =>
    state.hasPermission(PROTECTED_FIELD_GROUP_RESOURCES[group], Operation.Read),
  );
}
