import { usePermission } from "@/hooks/use-permission";
import type { IftaReturnPermissions } from "@/lib/ifta-return";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";

/**
 * What the reader may do to the quarter's return. `manage` is the separate
 * grant that pays for re-routing completed moves, which is why it is not one
 * of the status transitions.
 */
export type IftaReturnWorkspacePermissions = IftaReturnPermissions & { manage: boolean };

export function useIftaReturnPermissions(): IftaReturnWorkspacePermissions {
  const { allowed: create } = usePermission(Resource.IFTAReturn, Operation.Create);
  const { allowed: update } = usePermission(Resource.IFTAReturn, Operation.Update);
  const { allowed: approve } = usePermission(Resource.IFTAReturn, Operation.Approve);
  const { allowed: reopen } = usePermission(Resource.IFTAReturn, Operation.Reopen);
  const { allowed: submit } = usePermission(Resource.IFTAReturn, Operation.Submit);
  const { allowed: remove } = usePermission(Resource.IFTAReturn, Operation.Delete);
  const { allowed: exportReturn } = usePermission(Resource.IFTAReturn, Operation.Export);
  const { allowed: manage } = usePermission(Resource.IFTAReturn, Operation.Manage);

  return useMemo(
    () => ({
      create,
      update,
      approve,
      reopen,
      submit,
      delete: remove,
      export: exportReturn,
      manage,
    }),
    [create, update, approve, reopen, submit, remove, exportReturn, manage],
  );
}
