import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  callOffPositioning,
  positionDriver,
  type FreightMove,
} from "@/lib/graphql/freight-outlook";
import { freightOutlook } from "@/lib/queries/freight-outlook";
import { useQueryClient } from "@tanstack/react-query";

/**
 * Sending a driver and calling one off. Both refresh the outlook, which counts a sent driver at
 * the target; realtime refreshes every other open outlook through the audit trail.
 */
export function useOutlookActions() {
  const queryClient = useQueryClient();
  const refresh = () => queryClient.invalidateQueries({ queryKey: freightOutlook._def });

  const send = useApiMutation({
    mutationFn: async (move: FreightMove) =>
      positionDriver({
        workerId: move.workerId ?? "",
        toCell: move.toCell,
        slotStart: move.slotStart,
      }),
    onSuccess: refresh,
    resourceName: "Positioning",
  });

  const callOff = useApiMutation({
    mutationFn: async (positioningId: string) => callOffPositioning(positioningId),
    onSuccess: refresh,
    resourceName: "Positioning",
  });

  return { send, callOff };
}
