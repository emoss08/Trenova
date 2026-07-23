import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import {
  EDIMessageAckStatusBadge,
  EDIMessageDeliveryStatusBadge,
} from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type { EDIMessage, EDIMessageDeliveryStatus } from "@trenova/shared/types/edi";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { invalidateEDIMessages } from "./edi-panel-invalidation";
import {
  DetailField,
  DetailSection,
  EDIPartnerRef,
  EDIRawContent,
} from "./edi-panel-primitives";

export const RETRYABLE_DELIVERY_STATUSES = new Set<EDIMessageDeliveryStatus>([
  "Queued",
  "Failed",
  "DeadLettered",
]);

export function MessagePanel({ open, onOpenChange, row }: DataTablePanelProps<EDIMessage>) {
  const queryClient = useQueryClient();
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );
  const { data: message } = useQuery({
    ...queries.edi.message(row?.id ?? ""),
    enabled: open && !!row?.id,
  });
  const detail = message ?? row;

  const retryMutation = useApiMutation({
    mutationFn: (messageId: string) => apiService.ediService.retryMessageDelivery(messageId),
    onSuccess: async () => {
      toast.success("Delivery retry queued");
      await invalidateEDIMessages(queryClient, row?.id);
    },
    onError: () => toast.error("Failed to queue delivery retry"),
  });

  const replayMutation = useApiMutation({
    mutationFn: (messageId: string) => apiService.ediService.replayMessageDelivery(messageId),
    onSuccess: async () => {
      toast.success("Replay queued — the document will be re-delivered to the partner");
      await invalidateEDIMessages(queryClient, row?.id);
    },
    onError: () => toast.error("Failed to queue the replay"),
  });

  if (!detail) return null;

  const canRetry =
    canUpdate &&
    detail.direction === "Outbound" &&
    !!detail.deliveryStatus &&
    RETRYABLE_DELIVERY_STATUSES.has(detail.deliveryStatus);
  const canReplay =
    canUpdate &&
    detail.direction === "Outbound" &&
    detail.deliveryStatus === "Sent" &&
    !detail.rawPurgedAt;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={`EDI ${detail.transactionSet} Message`}
      description={`${detail.direction} · control number ${detail.interchangeControlNumber || "—"}`}
      size="lg"
      footer={
        <div className="flex w-full items-center justify-end gap-2">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
          {canReplay && (
            <Button
              type="button"
              variant="outline"
              isLoading={replayMutation.isPending}
              onClick={() => replayMutation.mutate(detail.id)}
              title="Queue this already-delivered document for another delivery to the partner"
            >
              Replay Delivery
            </Button>
          )}
          {canRetry && (
            <Button
              type="button"
              isLoading={retryMutation.isPending}
              onClick={() => retryMutation.mutate(detail.id)}
            >
              Retry Delivery
            </Button>
          )}
        </div>
      }
    >
      <div className="flex min-h-0 flex-col gap-3">
        <DetailSection title="Overview">
          <DetailField label="Transaction Set">
            <Badge variant="secondary">{detail.transactionSet}</Badge>
          </DetailField>
          <DetailField label="Direction">{detail.direction}</DetailField>
          <DetailField label="Partner">
            <EDIPartnerRef partner={detail.partner} />
          </DetailField>
          <DetailField label="X12 Version">{detail.x12Version}</DetailField>
          <DetailField label="ISA Control Number">
            <span className="font-mono text-xs">{detail.interchangeControlNumber || "—"}</span>
          </DetailField>
          <DetailField label="GS / ST Control Numbers">
            <span className="font-mono text-xs">
              {detail.groupControlNumber || "—"} / {detail.transactionControlNumber || "—"}
            </span>
          </DetailField>
          <DetailField label="Segments">{detail.segmentCount}</DetailField>
          <DetailField label="Generated">{formatToUserTimezone(detail.generatedAt)}</DetailField>
        </DetailSection>
        {detail.direction === "Outbound" && (
          <DetailSection title="Delivery">
            <DetailField label="Status">
              {detail.deliveryStatus ? (
                <EDIMessageDeliveryStatusBadge status={detail.deliveryStatus} />
              ) : (
                "Not queued"
              )}
            </DetailField>
            <DetailField label="Attempts">{detail.deliveryAttempts}</DetailField>
            <DetailField label="Remote Path">
              {detail.deliveryRemotePath ? (
                <span className="font-mono text-xs">{detail.deliveryRemotePath}</span>
              ) : (
                "—"
              )}
            </DetailField>
            <DetailField label="Sent At">
              {detail.deliverySentAt ? formatToUserTimezone(detail.deliverySentAt) : "—"}
            </DetailField>
            {detail.deliveryLastError && (
              <DetailField label="Last Error" fullWidth>
                <span className="text-xs text-destructive">{detail.deliveryLastError}</span>
              </DetailField>
            )}
          </DetailSection>
        )}
        <DetailSection title="Acknowledgment">
          <DetailField label="Status">
            <EDIMessageAckStatusBadge status={detail.ackStatus ?? "NotExpected"} />
          </DetailField>
          <DetailField label="Received At">
            {detail.ackReceivedAt ? formatToUserTimezone(detail.ackReceivedAt) : "—"}
          </DetailField>
          {detail.ackLastError && (
            <DetailField label="Details" fullWidth>
              <span className="text-xs text-destructive">{detail.ackLastError}</span>
            </DetailField>
          )}
        </DetailSection>
        {detail.rawX12 && (
          <DetailSection title="Raw X12" fullWidth>
            <EDIRawContent content={detail.rawX12} />
          </DetailSection>
        )}
      </div>
    </DataTablePanelContainer>
  );
}
