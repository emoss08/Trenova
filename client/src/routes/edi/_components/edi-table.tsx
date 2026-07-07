import { DataTable } from "@/components/data-table/data-table";
import { ediTableGraphQLConfigs } from "@/lib/graphql/edi-table";
import type { DataTablePanelProps } from "@/types/data-table";
import type {
  EDICommunicationProfile,
  EDIInboundFile,
  EDIMappingProfile,
  EDIMessage,
  EDIPartner,
  EDITestCaseRow,
  EDITransfer,
} from "@/types/edi";
import { Resource } from "@/types/permission";
import { useCallback, useMemo } from "react";
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
import { InboundFilePanel } from "./panel/edi-inbound-file-panel";
import { MappingProfileTablePanel } from "./panel/edi-mapping-profile-panel";
import { MessagePanel } from "./panel/edi-message-panel";
import { PartnerPanel } from "./panel/edi-partner-panel";
import { TestCasePanel } from "./panel/edi-test-case-panel";
import { EDITransferReviewPanel } from "./panel/edi-transfer-review-panel";
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
  const TransferPanel = useCallback(
    (props: DataTablePanelProps<EDITransfer>) => (
      <EDITransferReviewPanel {...props} direction={direction} />
    ),
    [direction],
  );

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
      />
    </Outer>
  );
}

function MessagesWorkspace() {
  const columns = useMemo(() => getMessageColumns(), []);

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
      />
    </Outer>
  );
}

function InboundFilesWorkspace() {
  const columns = useMemo(() => getInboundFileColumns(), []);

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
