import { DataTable } from "@/components/data-table/data-table";
import type { DataTablePanelProps } from "@/types/data-table";
import type { EDICommunicationProfile, EDIPartner, EDITransfer } from "@/types/edi";
import { Resource } from "@/types/permission";
import { useCallback, useMemo } from "react";
import { getCommunicationProfileColumns } from "./edi-communication-profile-columns";
import { DesignerWorkspace } from "./edi-designer-workspace";
import { getPartnerColumns } from "./edi-partner-columns";
import { getTransferColumns } from "./edi-transfer-columns";
import type { EDIPageKind } from "./edi-types";
import { CommunicationProfilePanel } from "./panel/edi-communication-profile-panel";
import { EDITransferReviewPanel } from "./panel/edi-transfer-review-panel";
import { MappingProfilesWorkspace } from "./panel/edi-mapping-profile-panel";
import { PartnerPanel } from "./panel/edi-partner-panel";
import { PendingConnectionsPanel } from "./panel/pending-connections-panel";

export default function EdiTable({ kind }: { kind: EDIPageKind }) {
  return (
    <>
      {kind === "partners" && <PartnersWorkspace />}
      {kind === "communication-profiles" && <CommunicationProfilesWorkspace />}
      {kind === "mapping-profiles" && <MappingProfilesWorkspace />}
      {kind === "designer" && <DesignerWorkspace />}
      {(kind === "inbound" || kind === "outbound") && <TransfersWorkspace direction={kind} />}
    </>
  );
}

function PartnersWorkspace() {
  const columns = useMemo(() => getPartnerColumns(), []);

  return (
    <div className="flex flex-col gap-4">
      <PendingConnectionsPanel />
      <DataTable<EDIPartner>
        name="EDI Connection"
        link="/edi/partners/"
        queryKey="edi-partner-list"
        exportModelName="edi-partner"
        resource={Resource.EDI}
        columns={columns}
        TablePanel={PartnerPanel}
        preferDetailRowForEdit
      />
    </div>
  );
}

function CommunicationProfilesWorkspace() {
  const columns = useMemo(() => getCommunicationProfileColumns(), []);

  return (
    <DataTable<EDICommunicationProfile>
      name="EDI Communication Profile"
      link="/edi/communication-profiles/"
      queryKey="edi-communication-profile-list"
      exportModelName="edi-communication-profile"
      resource={Resource.EDI}
      columns={columns}
      TablePanel={CommunicationProfilePanel}
      preferDetailRowForEdit
    />
  );
}

function TransfersWorkspace({ direction }: { direction: "inbound" | "outbound" }) {
  const columns = useMemo(() => getTransferColumns(direction), [direction]);
  const TransferPanel = useCallback(
    (props: DataTablePanelProps<EDITransfer>) => (
      <EDITransferReviewPanel {...props} direction={direction} />
    ),
    [direction],
  );

  return (
    <DataTable<EDITransfer>
      name="EDI Transfer"
      link={direction === "inbound" ? "/edi/transfers/inbound/" : "/edi/transfers/outbound/"}
      detailLink="/edi/transfers/"
      queryKey={
        direction === "inbound" ? "edi-inbound-transfer-list" : "edi-outbound-transfer-list"
      }
      exportModelName={`edi-${direction}-transfer`}
      resource={Resource.EDI}
      columns={columns}
      TablePanel={TransferPanel}
      preferDetailRowForEdit
      enableCreateAction={false}
      enableReadOnlyPanel
    />
  );
}
