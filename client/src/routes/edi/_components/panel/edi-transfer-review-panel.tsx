import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { usePermissionStore } from "@/stores/permission-store";
import type { DataTablePanelProps } from "@/types/data-table";
import type { EDIMappingProfileItem, EDITransfer, EDITransferStatus } from "@/types/edi";
import { Operation, Resource } from "@/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowRightIcon, CheckIcon, PackageIcon, RouteIcon, XIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { mappingKey } from "../edi-display-utils";
import { invalidateEDITransfers } from "./edi-panel-invalidation";
import { EDIReasonDialog } from "./edi-reason-dialog";
import { MappingReview } from "./edi-transfer-mapping-review";
import { EDITransferPanelContent } from "./edi-transfer-panel-content";
import {
  TenderFreightReview,
  TenderRouteReview,
  TransferOverview,
} from "./edi-transfer-tender-review";

const ACTIONABLE_TRANSFER_STATUSES = new Set<EDITransferStatus>([
  "Submitted",
  "MappingRequired",
  "PendingApproval",
]);

export function EDITransferReviewPanel({
  direction,
  open,
  onOpenChange,
  row: transfer,
}: DataTablePanelProps<EDITransfer> & {
  direction: "inbound" | "outbound";
}) {
  const queryClient = useQueryClient();
  const [rejectDialogOpen, setRejectDialogOpen] = useState(false);
  const [inlineMappings, setInlineMappings] = useState<Record<string, EDIMappingProfileItem>>({});
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );
  const isActionable = !!transfer && ACTIONABLE_TRANSFER_STATUSES.has(transfer.status);
  const { data: preview } = useQuery({
    ...queries.edi.mappingPreview(transfer?.id ?? ""),
    enabled: !!transfer && direction === "inbound" && isActionable,
  });

  const approveMutation = useApiMutation({
    mutationFn: () =>
      apiService.ediService.approveTransfer(transfer!.id, {
        mappings: Object.values(inlineMappings),
      }),
    onSuccess: async () => {
      toast.success("EDI transfer approval started.");
      await invalidateEDITransfers(queryClient, transfer?.id);
      onOpenChange(false);
    },
    onError: () => toast.error("Failed to approve transfer"),
  });

  const rejectMutation = useApiMutation({
    mutationFn: (reason: string) =>
      apiService.ediService.rejectTransfer(transfer!.id, { reason }),
    onSuccess: async () => {
      toast.success("EDI transfer rejected");
      setRejectDialogOpen(false);
      await invalidateEDITransfers(queryClient, transfer?.id);
      onOpenChange(false);
    },
    onError: () => toast.error("Failed to reject transfer"),
  });

  const cancelMutation = useApiMutation({
    mutationFn: () => apiService.ediService.cancelTransfer(transfer!.id),
    onSuccess: async () => {
      toast.success("EDI transfer canceled");
      await invalidateEDITransfers(queryClient, transfer?.id);
      onOpenChange(false);
    },
    onError: () => toast.error("Failed to cancel transfer"),
  });

  const unresolved = preview?.unresolved ?? [];
  const mappingRows = preview?.all ?? transfer?.mappingSnapshot ?? [];
  const approvalReady = unresolved.every(
    (row) => inlineMappings[mappingKey(row.entityType, row.sourceId)]?.targetId,
  );

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      size={direction === "inbound" ? "2xl" : "xl"}
      title={direction === "inbound" ? "Review Inbound Load Tender" : "Review Outbound Load Tender"}
      description={
        transfer?.tenderPayload.bol ? `BOL ${transfer.tenderPayload.bol}` : "Load tender"
      }
      footer={
        <>
          {transfer && canUpdate && direction === "inbound" && isActionable && (
            <div className="ml-auto flex gap-2">
              <Button variant="outline" onClick={() => setRejectDialogOpen(true)}>
                <XIcon data-icon="inline-start" />
                Reject
              </Button>
              <Button
                disabled={!approvalReady}
                isLoading={approveMutation.isPending}
                onClick={() => approveMutation.mutate(undefined)}
              >
                <CheckIcon data-icon="inline-start" />
                Approve
              </Button>
            </div>
          )}
          {transfer && canUpdate && direction === "outbound" && isActionable && (
            <Button
              className="ml-auto"
              variant="outline"
              isLoading={cancelMutation.isPending}
              onClick={() => cancelMutation.mutate(undefined)}
            >
              Cancel Transfer
            </Button>
          )}
        </>
      }
    >
      {transfer && (
        <EDITransferPanelContent transfer={transfer}>
          <TransferOverview transfer={transfer} mappingRows={mappingRows} />
          <Tabs defaultValue="tender" className="min-h-0 gap-3">
            <TabsList variant="underline" className="w-full border-b border-border">
              <TabsTrigger value="tender">
                <RouteIcon data-icon="inline-start" />
                Tender
              </TabsTrigger>
              <TabsTrigger value="freight">
                <PackageIcon data-icon="inline-start" />
                Freight
              </TabsTrigger>
              <TabsTrigger value="mappings">
                <ArrowRightIcon data-icon="inline-start" />
                Mappings
              </TabsTrigger>
            </TabsList>
            <TabsContent value="tender" className="mt-0 space-y-3">
              <TenderRouteReview transfer={transfer} mappingRows={mappingRows} />
            </TabsContent>
            <TabsContent value="freight" className="mt-0 space-y-3">
              <TenderFreightReview transfer={transfer} mappingRows={mappingRows} />
            </TabsContent>
            <TabsContent value="mappings" className="mt-0 space-y-3">
              <MappingReview
                canResolve={direction === "inbound" && isActionable && canUpdate}
                inlineMappings={inlineMappings}
                mappingRows={mappingRows}
                setInlineMappings={setInlineMappings}
                unresolved={unresolved}
              />
            </TabsContent>
          </Tabs>
        </EDITransferPanelContent>
      )}
      <EDIReasonDialog
        open={rejectDialogOpen}
        onOpenChange={setRejectDialogOpen}
        title="Reject Load Tender"
        description={
          transfer?.tenderPayload.bol
            ? `Reject the load tender for BOL ${transfer.tenderPayload.bol}.`
            : "Reject this load tender."
        }
        placeholder="Reason shared with the submitting partner"
        confirmLabel="Reject Transfer"
        isPending={rejectMutation.isPending}
        onConfirm={(reason) => rejectMutation.mutate(reason)}
      />
    </DataTablePanelContainer>
  );
}
