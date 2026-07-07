import { DataTable } from "@/components/data-table/data-table";
import { ediTableGraphQLConfigs } from "@/lib/graphql/edi-table";
import { apiService } from "@/services/api";
import { usePermissionStore } from "@/stores/permission-store";
import type { DataTablePanelProps, DockAction } from "@/types/data-table";
import type {
  EDICommunicationProfile,
  EDIInboundFile,
  EDIMappingProfile,
  EDIMessage,
  EDIPartner,
  EDITestCaseRow,
  EDITransfer,
} from "@/types/edi";
import { Operation, Resource } from "@/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon, CircleXIcon, RefreshCwIcon, RotateCcwIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { notifyEDIBulkOutcome } from "./edi-bulk-actions";
import { getCommunicationProfileColumns } from "./edi-communication-profile-columns";
import { DesignerWorkspace } from "./edi-designer-workspace";
import { getInboundFileColumns } from "./edi-inbound-file-columns";
import { getMappingProfileColumns } from "./edi-mapping-profile-columns";
import { getMessageColumns } from "./edi-message-columns";
import { EDIOverview } from "./overview/edi-overview";
import { getPartnerColumns } from "./edi-partner-columns";
import { getTestCaseColumns } from "./edi-test-case-columns";
import { getTransferColumns } from "./edi-transfer-columns";
import type { EDIPageKind } from "./edi-types";
import { CommunicationProfilePanel } from "./panel/edi-communication-profile-panel";
import { InboundFilePanel, REPROCESSABLE_STATUSES } from "./panel/edi-inbound-file-panel";
import { invalidateEDIInboundFiles, invalidateEDIMessages, invalidateEDITransfers } from "./panel/edi-panel-invalidation";
import { EDIReasonDialog } from "./panel/edi-reason-dialog";
import { MappingProfileTablePanel } from "./panel/edi-mapping-profile-panel";
import { MessagePanel, RETRYABLE_DELIVERY_STATUSES } from "./panel/edi-message-panel";
import { PartnerPanel } from "./panel/edi-partner-panel";
import { TestCasePanel } from "./panel/edi-test-case-panel";
import {
  ACTIONABLE_TRANSFER_STATUSES,
  EDITransferReviewPanel,
} from "./panel/edi-transfer-review-panel";
import { PendingConnectionsPanel } from "./panel/pending-connections-panel";

export default function EdiTable({ kind }: { kind: EDIPageKind }) {
  return (
    <>
      {kind === "overview" && <EDIOverview />}
      {kind === "partners" && <PartnersWorkspace />}
      {kind === "communication-profiles" && <CommunicationProfilesWorkspace />}
      {kind === "mapping-profiles" && <MappingProfilesWorkspace />}
      {kind === "designer" && <DesignerWorkspace />}
      {(kind === "inbound" || kind === "outbound") && <TransfersWorkspace direction={kind} />}
      {kind === "messages" && <MessagesWorkspace />}
      {kind === "inbound-files" && <InboundFilesWorkspace />}
      {kind === "test-cases" && <TestCasesWorkspace />}
    </>
  );
}

function PartnersWorkspace() {
  const columns = useMemo(() => getPartnerColumns(), []);

  return (
    <div className="flex flex-col gap-4 px-3">
      <PendingConnectionsPanel />
      <DataTable<EDIPartner>
        name="EDI Connection"
        queryKey="edi-partner-list"
        resource={Resource.EDI}
        columns={columns}
        TablePanel={PartnerPanel}
        graphql={ediTableGraphQLConfigs.partners}
      />
    </div>
  );
}

function MappingProfilesWorkspace() {
  const columns = useMemo(() => getMappingProfileColumns(), []);

  return (
    <Outer>
      <DataTable<EDIMappingProfile>
        name="EDI Mapping Profile"
        queryKey="edi-mapping-profile-list"
        resource={Resource.EDI}
        columns={columns}
        TablePanel={MappingProfileTablePanel}
        graphql={ediTableGraphQLConfigs.mappingProfiles}
        enableReadOnlyPanel
      />
    </Outer>
  );
}

function CommunicationProfilesWorkspace() {
  const columns = useMemo(() => getCommunicationProfileColumns(), []);

  return (
    <Outer>
      <DataTable<EDICommunicationProfile>
        name="EDI Communication Profile"
        queryKey="edi-communication-profile-list"
        resource={Resource.EDI}
        columns={columns}
        TablePanel={CommunicationProfilePanel}
        graphql={ediTableGraphQLConfigs.communicationProfiles}
      />
    </Outer>
  );
}

function TransfersWorkspace({ direction }: { direction: "inbound" | "outbound" }) {
  const columns = useMemo(() => getTransferColumns(direction), [direction]);
  const queryClient = useQueryClient();
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );
  const [rejectRows, setRejectRows] = useState<EDITransfer[]>([]);
  const [rejectOpen, setRejectOpen] = useState(false);
  const [rejectPending, setRejectPending] = useState(false);
  const TransferPanel = useCallback(
    (props: DataTablePanelProps<EDITransfer>) => (
      <EDITransferReviewPanel {...props} direction={direction} />
    ),
    [direction],
  );

  const handleBulkApprove = useCallback(
    async (rows: EDITransfer[]) => {
      const eligible = rows.filter((row) => ACTIONABLE_TRANSFER_STATUSES.has(row.status));
      if (eligible.length === 0) {
        toast.info("None of the selected transfers are awaiting review");
        return;
      }
      const result = await apiService.ediService.bulkApproveTransfers(
        eligible.map((row) => row.id),
      );
      await invalidateEDITransfers(queryClient);
      notifyEDIBulkOutcome(result, {
        entity: "transfer",
        verbPast: "Queued approval for",
        skipped: rows.length - eligible.length,
      });
    },
    [queryClient],
  );

  const handleBulkRejectRequest = useCallback((rows: EDITransfer[]) => {
    const eligible = rows.filter((row) => ACTIONABLE_TRANSFER_STATUSES.has(row.status));
    if (eligible.length === 0) {
      toast.info("None of the selected transfers are awaiting review");
      return;
    }
    setRejectRows(eligible);
    setRejectOpen(true);
  }, []);

  const handleBulkRejectConfirm = useCallback(
    async (reason: string) => {
      setRejectPending(true);
      try {
        const result = await apiService.ediService.bulkRejectTransfers(
          rejectRows.map((row) => row.id),
          reason,
        );
        await invalidateEDITransfers(queryClient);
        notifyEDIBulkOutcome(result, {
          entity: "transfer",
          verbPast: "Rejected",
        });
        setRejectOpen(false);
        setRejectRows([]);
      } finally {
        setRejectPending(false);
      }
    },
    [queryClient, rejectRows],
  );

  const dockActions = useMemo<DockAction<EDITransfer>[]>(() => {
    if (!canUpdate || direction !== "inbound") return [];
    return [
      {
        id: "bulk-approve-transfers",
        label: "Approve",
        loadingLabel: "Approving...",
        icon: CircleCheckIcon,
        onClick: handleBulkApprove,
        clearSelectionOnSuccess: true,
      },
      {
        id: "bulk-reject-transfers",
        label: "Reject",
        loadingLabel: "Rejecting...",
        icon: CircleXIcon,
        variant: "destructive",
        onClick: handleBulkRejectRequest,
      },
    ];
  }, [canUpdate, direction, handleBulkApprove, handleBulkRejectRequest]);

  return (
    <Outer>
      <DataTable<EDITransfer>
        name="EDI Transfer"
        queryKey={
          direction === "inbound" ? "edi-inbound-transfer-list" : "edi-outbound-transfer-list"
        }
        resource={Resource.EDI}
        columns={columns}
        TablePanel={TransferPanel}
        graphql={
          direction === "inbound"
            ? ediTableGraphQLConfigs.inboundTransfers
            : ediTableGraphQLConfigs.outboundTransfers
        }
        enableCreateAction={false}
        enableReadOnlyPanel
        enableRowSelection={dockActions.length > 0}
        dockActions={dockActions}
      />
      <EDIReasonDialog
        open={rejectOpen}
        onOpenChange={setRejectOpen}
        title={`Reject ${rejectRows.length} Load Tender(s)`}
        description="The rejection reason is sent back to the trading partner on the outbound 990 response."
        placeholder="Explain why these tenders are being rejected"
        confirmLabel="Reject Tenders"
        isPending={rejectPending}
        onConfirm={handleBulkRejectConfirm}
      />
    </Outer>
  );
}

function MessagesWorkspace() {
  const columns = useMemo(() => getMessageColumns(), []);
  const queryClient = useQueryClient();
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );

  const handleBulkRetry = useCallback(
    async (rows: EDIMessage[]) => {
      const eligible = rows.filter(
        (row) =>
          row.direction === "Outbound" && RETRYABLE_DELIVERY_STATUSES.has(row.deliveryStatus),
      );
      if (eligible.length === 0) {
        toast.info("None of the selected messages are retryable");
        return;
      }
      const result = await apiService.ediService.bulkRetryMessageDelivery(
        eligible.map((row) => row.id),
      );
      await invalidateEDIMessages(queryClient);
      notifyEDIBulkOutcome(result, {
        entity: "message",
        verbPast: "Queued delivery retry for",
        skipped: rows.length - eligible.length,
      });
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<EDIMessage>[]>(() => {
    if (!canUpdate) return [];
    return [
      {
        id: "bulk-retry-delivery",
        label: "Retry Delivery",
        loadingLabel: "Queueing retries...",
        icon: RotateCcwIcon,
        onClick: handleBulkRetry,
        clearSelectionOnSuccess: true,
      },
    ];
  }, [canUpdate, handleBulkRetry]);

  return (
    <Outer>
      <DataTable<EDIMessage>
        name="EDI Message"
        queryKey="edi-message-list"
        resource={Resource.EDI}
        columns={columns}
        TablePanel={MessagePanel}
        graphql={ediTableGraphQLConfigs.messages}
        enableCreateAction={false}
        enableReadOnlyPanel
        enableRowSelection={dockActions.length > 0}
        dockActions={dockActions}
      />
    </Outer>
  );
}

function InboundFilesWorkspace() {
  const columns = useMemo(() => getInboundFileColumns(), []);
  const queryClient = useQueryClient();
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );

  const handleBulkReprocess = useCallback(
    async (rows: EDIInboundFile[]) => {
      const eligible = rows.filter((row) => REPROCESSABLE_STATUSES.has(row.status));
      if (eligible.length === 0) {
        toast.info("None of the selected files can be reprocessed");
        return;
      }
      const result = await apiService.ediService.bulkReprocessInboundFiles(
        eligible.map((row) => row.id),
      );
      await invalidateEDIInboundFiles(queryClient);
      notifyEDIBulkOutcome(result, {
        entity: "file",
        verbPast: "Reprocessed",
        skipped: rows.length - eligible.length,
      });
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<EDIInboundFile>[]>(() => {
    if (!canUpdate) return [];
    return [
      {
        id: "bulk-reprocess-files",
        label: "Reprocess",
        loadingLabel: "Reprocessing...",
        icon: RefreshCwIcon,
        onClick: handleBulkReprocess,
        clearSelectionOnSuccess: true,
      },
    ];
  }, [canUpdate, handleBulkReprocess]);

  return (
    <Outer>
      <DataTable<EDIInboundFile>
        name="EDI Inbound File"
        queryKey="edi-inbound-file-list"
        resource={Resource.EDI}
        columns={columns}
        TablePanel={InboundFilePanel}
        graphql={ediTableGraphQLConfigs.inboundFiles}
        enableCreateAction={false}
        enableReadOnlyPanel
        enableRowSelection={dockActions.length > 0}
        dockActions={dockActions}
      />
    </Outer>
  );
}

function TestCasesWorkspace() {
  const columns = useMemo(() => getTestCaseColumns(), []);

  return (
    <Outer>
      <DataTable<EDITestCaseRow>
        name="EDI Test Case"
        queryKey="edi-test-case-list"
        resource={Resource.EDI}
        columns={columns}
        TablePanel={TestCasePanel}
        graphql={ediTableGraphQLConfigs.testCases}
      />
    </Outer>
  );
}

function Outer({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col px-3">{children}</div>;
}
