import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { ReferenceKind } from "@trenova/shared/types/generated/reference-views";
import { Operation } from "@trenova/shared/types/permission";

/**
 * Whether the person may read a kind of record whole, beyond its reference view. A linked
 * record's other fields, and a link to its own page, need this. False until the person's
 * permissions are known, and false for no kind.
 */
export function useCanReadKind(kind: ReferenceKind | null): boolean {
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const known = usePermissionStore((state) => state.manifest !== null);
  return kind !== null && known && hasPermission(kind, Operation.Read);
}
