import { handleMutationError } from "@/hooks/use-api-mutation";
import { canAcknowledgeEvent } from "@/lib/carrier-intelligence";
import {
  acknowledgeCarrierIntelEvents,
  CARRIER_INTEL_EVENTS_KEY,
  CARRIER_INTELLIGENCE_KEY,
  type CarrierIntelEvent,
} from "@/lib/graphql/carrier-intelligence";
import { CARRIER_INTEL_EVENT_LIST_KEY } from "@/lib/graphql/carrier-monitoring-table";
import { queries } from "@/lib/queries";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback, useState } from "react";
import { toast } from "sonner";

export function acknowledgeableEventIds(
  events: readonly Pick<CarrierIntelEvent, "id" | "status">[],
): string[] {
  const ids: string[] = [];
  for (const event of events) {
    if (canAcknowledgeEvent(event.status)) {
      ids.push(event.id);
    }
  }
  return ids;
}

export function useInvalidateInbox() {
  const queryClient = useQueryClient();
  return useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [CARRIER_INTEL_EVENT_LIST_KEY] }),
      queryClient.invalidateQueries({ queryKey: [CARRIER_INTEL_EVENTS_KEY] }),
      queryClient.invalidateQueries({ queryKey: [CARRIER_INTELLIGENCE_KEY] }),
      queryClient.invalidateQueries({
        queryKey: queries.carrierIntelSettings.monitoringStatus().queryKey,
      }),
    ]);
  }, [queryClient]);
}

export function useAcknowledgeEvents() {
  const t = useT();
  const invalidate = useInvalidateInbox();
  const [pending, setPending] = useState(false);

  const acknowledge = useCallback(
    async (events: readonly Pick<CarrierIntelEvent, "id" | "status">[]): Promise<boolean> => {
      const ids = acknowledgeableEventIds(events);
      if (ids.length === 0) {
        toast.info(t("Nothing to acknowledge"), {
          description: t("Only open events can be acknowledged."),
        });
        return false;
      }

      setPending(true);
      try {
        const acknowledged = await acknowledgeCarrierIntelEvents(ids);
        const skipped = events.length - ids.length;
        toast.success(
          t("{0, plural, one {# event acknowledged} other {# events acknowledged}}", acknowledged),
          skipped > 0
            ? {
                description: t(
                  "{0, plural, one {# selected event was} other {# selected events were}} not open and left unchanged.",
                  skipped,
                ),
              }
            : undefined,
        );
        await invalidate();
        return true;
      } catch (error) {
        handleMutationError({ error, resourceName: "Carrier intelligence event" });
        return false;
      } finally {
        setPending(false);
      }
    },
    [invalidate, t],
  );

  return { acknowledge, pending };
}
