import { useT } from "@trenova/shared/i18n/use-t";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { canAcknowledgeEvent, canResolveEvent } from "@/lib/carrier-intelligence";
import { acknowledgeCarrierIntelEvents } from "@/lib/graphql/carrier-intelligence";
import type { CarrierIntelEventRow } from "@/lib/graphql/carrier-monitoring-table";
import type { DockAction, RowAction } from "@trenova/shared/types/data-table";
import { CheckCheckIcon, CheckIcon, ExternalLinkIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { carrierPanelPath } from "@/lib/carrier-links";

export type UseEventInboxActionsParams = {
  canUpdate: boolean;
  onChanged: () => void | Promise<void>;
};

export function acknowledgeableEventIds(rows: readonly CarrierIntelEventRow[]): string[] {
  const ids: string[] = [];
  for (const row of rows) {
    if (canAcknowledgeEvent(row.status)) {
      ids.push(row.id);
    }
  }
  return ids;
}

export function useEventInboxActions({ canUpdate, onChanged }: UseEventInboxActionsParams) {
  const t = useT();
  const navigate = useNavigate();
  const [resolving, setResolving] = useState<CarrierIntelEventRow | null>(null);
  const [acknowledgingId, setAcknowledgingId] = useState<string | null>(null);

  const acknowledge = useCallback(
    async (rows: readonly CarrierIntelEventRow[]) => {
      const ids = acknowledgeableEventIds(rows);
      if (ids.length === 0) {
        toast.info(t("None of the selected events are open"), {
          description: t("Only open events can be acknowledged."),
        });
        return;
      }

      let acknowledged: number;
      try {
        acknowledged = await acknowledgeCarrierIntelEvents(ids);
      } catch (error) {
        handleMutationError({ error, resourceName: "Carrier intelligence event" });
        throw error;
      }

      const skipped = rows.length - ids.length;
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
      await onChanged();
    },
    [onChanged, t],
  );

  const dockActions = useMemo<DockAction<CarrierIntelEventRow>[]>(() => {
    if (!canUpdate) {
      return [];
    }
    return [
      {
        id: "acknowledge-events",
        label: t("Acknowledge"),
        loadingLabel: t("Acknowledging..."),
        icon: CheckIcon,
        onClick: acknowledge,
        clearSelectionOnSuccess: true,
      },
    ];
  }, [acknowledge, canUpdate, t]);

  const contextMenuActions = useMemo<RowAction<CarrierIntelEventRow>[]>(
    () => [
      {
        id: "acknowledge-event",
        label: t("Acknowledge"),
        icon: CheckIcon,
        hidden: (row) => !canUpdate || !canAcknowledgeEvent(row.original.status),
        isPending: (row) => acknowledgingId === row.original.id,
        onClick: async (row) => {
          setAcknowledgingId(row.original.id);
          try {
            await acknowledge([row.original]);
          } catch {
            return;
          } finally {
            setAcknowledgingId(null);
          }
        },
      },
      {
        id: "resolve-event",
        label: t("Resolve"),
        icon: CheckCheckIcon,
        hidden: (row) => !canUpdate || !canResolveEvent(row.original.status),
        onClick: (row) => setResolving(row.original),
      },
      {
        id: "open-carrier",
        label: t("Open carrier"),
        icon: ExternalLinkIcon,
        hidden: (row) => !row.original.carrierId,
        onClick: (row) => {
          if (row.original.carrierId) {
            void navigate(carrierPanelPath(row.original.carrierId, "intelligence"));
          }
        },
      },
    ],
    [acknowledge, acknowledgingId, canUpdate, navigate, t],
  );

  return {
    dockActions,
    contextMenuActions,
    resolving,
    setResolving,
  };
}
