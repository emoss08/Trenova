import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import type { EDIPageKind } from "./_components/edi-types";

const Table = lazy(() => import("./_components/edi-table"));

const pageTitles: Record<EDIPageKind, string> = {
  overview: "EDI Operations",
  partners: "EDI Partners",
  "communication-profiles": "EDI Communication Profiles",
  "mapping-profiles": "EDI Mapping Profiles",
  designer: "Template Designer",
  inbound: "Inbound EDI Transfers",
  outbound: "Outbound EDI Transfers",
  messages: "EDI Messages",
  "inbound-files": "EDI Inbound Files",
  "test-cases": "EDI Test Cases",
};

const pageDescriptions: Record<EDIPageKind, string> = {
  overview: "Live health view of deliveries, inbound processing, and acknowledgment state",
  partners: "Trading partners, connection requests, and partner defaults",
  "communication-profiles": "Transport endpoints, envelope identifiers, and credentials",
  "mapping-profiles": "Cross-organization entity mapping for tendered loads",
  designer: "Author, validate, and certify X12 document templates",
  inbound: "Load tenders received from partners awaiting review",
  outbound: "Load tenders submitted to partners and their lifecycle",
  messages: "Generated X12 documents with delivery and acknowledgment status",
  "inbound-files": "Files received from partner mailboxes and their processing state",
  "test-cases": "Certification scenarios that render payloads through partner templates",
};

export function EDIOverviewPage() {
  return <EDIPage kind="overview" />;
}

export function EDIPartnersPage() {
  return <EDIPage kind="partners" />;
}

export function EDICommunicationProfilesPage() {
  return <EDIPage kind="communication-profiles" />;
}

export function EDIMappingProfilesPage() {
  return <EDIPage kind="mapping-profiles" />;
}

export function EDIInboundTransfersPage() {
  return <EDIPage kind="inbound" />;
}

export function EDIOutboundTransfersPage() {
  return <EDIPage kind="outbound" />;
}

export function EDIDesignerPage() {
  return <EDIPage kind="designer" />;
}

export function EDIMessagesPage() {
  return <EDIPage kind="messages" />;
}

export function EDIInboundFilesPage() {
  return <EDIPage kind="inbound-files" />;
}

export function EDITestCasesPage() {
  return <EDIPage kind="test-cases" />;
}

function EDIPage({ kind }: { kind: EDIPageKind }) {
  return (
    <PageLayout
      pageHeaderProps={{
        title: pageTitles[kind],
        description: pageDescriptions[kind],
      }}
      className="p-0"
    >
      <DataTableLazyComponent
        fallback={
          kind === "designer" ? <ComponentLoader message="Loading Template Designer" /> : undefined
        }
      >
        <Table kind={kind} />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
