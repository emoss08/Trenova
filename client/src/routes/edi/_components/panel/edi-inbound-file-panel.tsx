import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { formatFileSize } from "@/components/documents/document-upload-zone";
import {
  EDIInboundFileStatusBadge,
  EDIMessageAckStatusBadge,
} from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { formatToUserTimezone } from "@/lib/date";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { usePermissionStore } from "@/stores/permission-store";
import type { DataTablePanelProps } from "@/types/data-table";
import type { EDIInboundFile, EDIInboundFileStatus } from "@/types/edi";
import { Operation, Resource } from "@/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { invalidateEDIInboundFiles } from "./edi-panel-invalidation";
import {
  DetailField,
  DetailSection,
  EDIPartnerRef,
  EDIRawContent,
} from "./edi-panel-primitives";

const REPROCESSABLE_STATUSES = new Set<EDIInboundFileStatus>([
  "Quarantined",
  "PartiallyProcessed",
]);

export function InboundFilePanel({
  open,
  onOpenChange,
  row,
}: DataTablePanelProps<EDIInboundFile>) {
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
      toast.success("Inbound file reprocessed");
      await invalidateEDIInboundFiles(queryClient, row?.id);
    },
    onError: () => toast.error("Failed to reprocess inbound file"),
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
            Close
          </Button>
          {canReprocess && (
            <Button
              type="button"
              isLoading={reprocessMutation.isPending}
              onClick={() => reprocessMutation.mutate(detail.id)}
            >
              Reprocess File
            </Button>
          )}
        </div>
      }
    >
      <div className="flex min-h-0 flex-col gap-3">
        <DetailSection title="Processing">
          <DetailField label="Status">
            <EDIInboundFileStatusBadge status={detail.status} />
          </DetailField>
          <DetailField label="Transactions">{detail.transactionCount}</DetailField>
          <DetailField label="Processed At">
            {detail.processedAt ? formatToUserTimezone(detail.processedAt) : "—"}
          </DetailField>
          <DetailField label="ISA Control Number">
            <span className="font-mono text-xs">{detail.interchangeControlNumber || "—"}</span>
          </DetailField>
          {detail.failureReason && (
            <DetailField label="Processing Notes" fullWidth>
              <span className="text-xs text-destructive">{detail.failureReason}</span>
            </DetailField>
          )}
        </DetailSection>
        <DetailSection title="Source">
          <DetailField label="Partner">
            <EDIPartnerRef partner={detail.partner} />
          </DetailField>
          <DetailField label="Method">
            <Badge variant="outline">{detail.method}</Badge>
          </DetailField>
          <DetailField label="Remote Path" fullWidth>
            <span className="font-mono text-xs">{detail.remotePath}</span>
          </DetailField>
          <DetailField label="ISA Sender">
            <span className="font-mono text-xs">
              {detail.isaSenderQualifier || "—"}:{detail.isaSenderId || "—"}
            </span>
          </DetailField>
          <DetailField label="ISA Receiver">
            <span className="font-mono text-xs">
              {detail.isaReceiverQualifier || "—"}:{detail.isaReceiverId || "—"}
            </span>
          </DetailField>
          <DetailField label="Size">
            {detail.sizeBytes > 0 ? formatFileSize(detail.sizeBytes) : "—"}
          </DetailField>
          <DetailField label="Checksum" fullWidth>
            <span className="font-mono text-xs break-all">{detail.checksum}</span>
          </DetailField>
        </DetailSection>
        {detail.messages && detail.messages.length > 0 && (
          <DetailSection title={`Transactions (${detail.messages.length})`} fullWidth>
            <div className="flex flex-col gap-2">
              {detail.messages.map((message) => (
                <div
                  key={message.id}
                  className="flex items-center justify-between rounded-md border bg-background px-3 py-2"
                >
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary">{message.transactionSet}</Badge>
                    <span className="font-mono text-xs text-muted-foreground">
                      ST {message.transactionControlNumber || "—"}
                    </span>
                  </div>
                  <EDIMessageAckStatusBadge status={message.ackStatus ?? "NotExpected"} />
                </div>
              ))}
            </div>
          </DetailSection>
        )}
        {detail.rawContent && (
          <DetailSection title="Raw Content" fullWidth>
            <EDIRawContent content={detail.rawContent} />
          </DetailSection>
        )}
      </div>
    </DataTablePanelContainer>
  );
}
