import { useT } from "@trenova/shared/i18n/use-t";
import { translateMessage } from "@trenova/shared/i18n/message";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { FlowInstanceStatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import type { EDIFlowInstanceRow } from "@/lib/graphql/edi-table";
import { lading } from "@/lib/queries/lading";
import { apiService } from "@/services/api";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { DetailField, DetailSection, EDIEmptyState, EDIPartnerRef } from "./edi-panel-primitives";

const RELEASABLE = new Set(["Held", "Escalated"]);

/** One flow instance: where its load stands, what it waits for, and every event it handled. */
export function FlowInstancePanel({
  open,
  onOpenChange,
  row,
}: DataTablePanelProps<EDIFlowInstanceRow>) {
  const t = useT();
  const queryClient = useQueryClient();
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );
  const { data: detail } = useQuery({
    ...lading.flowInstance(row?.id ?? ""),
    enabled: open && !!row?.id,
  });

  const release = useApiMutation({
    mutationFn: (instanceId: string) => apiService.ladingService.releaseFlowInstance(instanceId),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["edi-flow-instance-list"] }),
        queryClient.invalidateQueries({ queryKey: lading.flowInstance._def }),
      ]);
    },
  });

  if (!row) return null;

  const status = detail?.instance.status ?? row.status;
  const summary = detail?.summary;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row.key}
      description={t("Flow {0}", row.flow)}
      size="lg"
      footer={
        <>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Close")}
          </Button>
          {canUpdate && RELEASABLE.has(status) && (
            <Button
              type="button"
              isLoading={release.isPending}
              onClick={() => release.mutate(row.id)}
            >
              {t("Release")}
            </Button>
          )}
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <DetailSection title={t("Instance")}>
          <DetailField label={t("Status")}>
            <FlowInstanceStatusBadge status={status} />
          </DetailField>
          <DetailField label={t("Partner")}>
            <EDIPartnerRef partner={row.partner} />
          </DetailField>
          <DetailField label={t("Load")}>
            {summary
              ? summary.load.exists
                ? summary.load.state
                : t("No load yet")
              : "—"}
          </DetailField>
          <DetailField label={t("Next deadline")}>
            {row.wakeAt ? formatToUserTimezone(row.wakeAt) : "—"}
          </DetailField>
          {row.reason && (
            <DetailField label={t("Why it waits")} fullWidth>
              {row.reason}
            </DetailField>
          )}
          {summary?.declined && summary.declineReason && (
            <DetailField label={t("Declined because")} fullWidth>
              {summary.declineReason}
            </DetailField>
          )}
        </DetailSection>

        <DetailSection title={t("Awaiting acknowledgment")} fullWidth>
          {summary && summary.pending.length > 0 ? (
            <ul className="flex flex-col gap-1 text-sm">
              {summary.pending.map((pending) => (
                <li key={pending.id} className="flex items-center justify-between gap-2">
                  <span className="font-mono">
                    {pending.set} {pending.control}
                  </span>
                  <span className="text-muted-foreground">
                    {t(
                      "Due {0}, {1, plural, =0 {not resent} one {resent once} other {resent # times}}",
                      formatToUserTimezone(pending.due),
                      pending.resends,
                    )}
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <EDIEmptyState message={t("Nothing sent is waiting for an acknowledgment")} />
          )}
        </DetailSection>

        <DetailSection title={t("Events")} fullWidth>
          {detail && detail.events.length > 0 ? (
            <ol className="flex flex-col gap-2 text-sm">
              {detail.events.map((event) => (
                <li key={event.id} className="flex flex-col gap-1 rounded-md border p-2">
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-medium">{event.kind}</span>
                    <span className="text-muted-foreground">
                      {formatToUserTimezone(event.createdAt)}
                    </span>
                  </div>
                  {event.actions.length > 0 && (
                    <div className="flex flex-wrap gap-1">
                      {event.actions.map((action, index) => (
                        <Badge
                          // oxlint-disable-next-line react/no-array-index-key -- an event's actions have no id and never reorder
                          key={index}
                          variant="neutral"
                          appearance="outline"
                        >
                          {action.set ? `${action.kind} ${action.set}` : action.kind}
                        </Badge>
                      ))}
                    </div>
                  )}
                  {event.error && (
                    <span className="text-destructive">{translateMessage(event.error, t)}</span>
                  )}
                </li>
              ))}
            </ol>
          ) : (
            <EDIEmptyState message={t("No events yet")} />
          )}
        </DetailSection>
      </div>
    </DataTablePanelContainer>
  );
}
