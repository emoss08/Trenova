import { useT } from "@trenova/shared/i18n/use-t";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { formatFileSize } from "@/components/documents/document-upload-zone";
import {
  EDIInboundFileStatusBadge,
  EDIMessageAckStatusBadge,
} from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import type { EDIInboundFileRow } from "@/lib/graphql/edi-table";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type { EDIInboundFileStatus } from "@trenova/shared/types/edi";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { invalidateEDIInboundFiles } from "./edi-panel-invalidation";
import { DetailField, DetailSection, EDIPartnerRef, EDIRawContent } from "./edi-panel-primitives";

export const REPROCESSABLE_STATUSES = new Set<EDIInboundFileStatus>([
  "Quarantined",
  "PartiallyProcessed",
]);

export function InboundFilePanel({
  open,
  onOpenChange,
  row,
}: DataTablePanelProps<EDIInboundFileRow>) {
  const t = useT();

  const queryClient = useQueryClient();
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );
  const { data: file } = useQuery({
    ...queries.edi.inboundFile(row?.id ?? ""),
    enabled: open && !!row?.id,
  });
  const detail = file ?? row;

  const reprocessMutation = useApiMutation({
    mutationFn: (fileId: string) => apiService.ediService.reprocessInboundFile(fileId),
    onSuccess: async () => {
      toast.success(t("Inbound file reprocessed"));
      await invalidateEDIInboundFiles(queryClient, row?.id);
    },
    onError: () => toast.error(t("Failed to reprocess inbound file")),
  });

  if (!detail) return null;

  const canReprocess = canUpdate && REPROCESSABLE_STATUSES.has(detail.status);

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={detail.fileName}
      description={`Received via ${detail.method} · ${formatToUserTimezone(detail.receivedAt)}`}
      size="lg"
      footer={
        <div className="flex w-full items-center justify-end gap-2">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Close")}
          </Button>
          {canReprocess && (
            <Button
              type="button"
              isLoading={reprocessMutation.isPending}
              onClick={() => reprocessMutation.mutate(detail.id)}
            >
              {t("Reprocess File")}
            </Button>
          )}
        </div>
      }
    >
      <div className="flex min-h-0 flex-col gap-3">
        <DetailSection title={t("Processing")}>
          <DetailField label={t("Status")}>
            <EDIInboundFileStatusBadge status={detail.status} />
          </DetailField>
          <DetailField label={t("Transactions")}>{detail.transactionCount}</DetailField>
          <DetailField label={t("Processed At")}>
            {detail.processedAt ? formatToUserTimezone(detail.processedAt) : "—"}
          </DetailField>
          <DetailField label={t("ISA Control Number")}>
            <span className="font-mono text-xs">{detail.interchangeControlNumber || "—"}</span>
          </DetailField>
          {detail.failureReason && (
            <DetailField label={t("Processing Notes")} fullWidth>
              <span className="text-destructive text-xs">{detail.failureReason}</span>
            </DetailField>
          )}
        </DetailSection>
        <DetailSection title={t("Source")}>
          <DetailField label={t("Partner")}>
            <EDIPartnerRef partner={detail.partner} />
          </DetailField>
          <DetailField label={t("Method")}>
            <Badge variant="outline">{detail.method}</Badge>
          </DetailField>
          <DetailField label={t("Remote Path")} fullWidth>
            <span className="font-mono text-xs">{detail.remotePath}</span>
          </DetailField>
          <DetailField label={t("ISA Sender")}>
            <span className="font-mono text-xs">
              {detail.isaSenderQualifier || "—"}:{detail.isaSenderId || "—"}
            </span>
          </DetailField>
          <DetailField label={t("ISA Receiver")}>
            <span className="font-mono text-xs">
              {detail.isaReceiverQualifier || "—"}:{detail.isaReceiverId || "—"}
            </span>
          </DetailField>
          <DetailField label={t("Size")}>
            {detail.sizeBytes > 0 ? formatFileSize(detail.sizeBytes) : "—"}
          </DetailField>
          <DetailField label={t("Checksum")} fullWidth>
            <span className="font-mono text-xs break-all">{detail.checksum}</span>
          </DetailField>
        </DetailSection>
        {file?.messages && file.messages.length > 0 && (
          <DetailSection title={`Transactions (${file.messages.length})`} fullWidth>
            <div className="flex flex-col gap-2">
              {file.messages.map((message) => (
                <div
                  key={message.id}
                  className="bg-background flex items-center justify-between rounded-md border px-3 py-2"
                >
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary">{message.transactionSet}</Badge>
                    <span className="text-muted-foreground font-mono text-xs">
                      {t("ST {0}", message.transactionControlNumber || "—")}
                    </span>
                  </div>
                  <EDIMessageAckStatusBadge status={message.ackStatus ?? "NotExpected"} />
                </div>
              ))}
            </div>
          </DetailSection>
        )}
        {file?.rawContent && (
          <DetailSection title={t("Raw Content")} fullWidth>
            <EDIRawContent content={file.rawContent} />
          </DetailSection>
        )}
      </div>
    </DataTablePanelContainer>
  );
}
