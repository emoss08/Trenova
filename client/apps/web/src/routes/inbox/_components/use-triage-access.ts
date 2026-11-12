import { usePermission } from "@/hooks/use-permission";
import { Operation, Resource } from "@trenova/shared/types/permission";

/**
 * What the reader may do in Triage, as the server will hold them to it:
 * working a message (handling, snoozing, linking, filing, replying) takes
 * inbound message update, giving messages to someone else takes assign, and
 * rules and settings take manage. A reader with none of them reads Triage
 * and is offered nothing the server would refuse.
 */
export function useTriageAccess() {
  const { allowed: canWork } = usePermission(Resource.InboundMessage, Operation.Update);
  const { allowed: canAssign } = usePermission(Resource.InboundMessage, Operation.Assign);
  const { allowed: canManage } = usePermission(Resource.InboundMessage, Operation.Manage);

  return { canWork, canAssign: canWork && canAssign, canManage };
}
