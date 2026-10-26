import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { EDIPageSkeleton } from "./_components/edi-skeleton";
import type { EDIPageKind } from "./_components/edi-types";
import { defineLabels } from "@trenova/shared/i18n/labels";

const Table = lazy(() => import("./_components/edi-table"));

const pageTitles: Record<EDIPageKind, string> = defineLabels({
  overview: "EDI Operations",
  partners: "EDI Partners",
  "communication-profiles": "EDI Communication Profiles",
  "mapping-profiles": "EDI Mapping Profiles",
  inbound: "Inbound EDI Transfers",
  outbound: "Outbound EDI Transfers",
  messages: "EDI Messages",
  "inbound-files": "EDI Inbound Files",
  flows: "EDI Flow Instances",
});

const pageDescriptions: Record<EDIPageKind, string> = defineLabels({
  overview: "Live health view of deliveries, inbound processing, and acknowledgment state",
  partners: "Trading partners, connection requests, and partner defaults",
  "communication-profiles":
    "Transport endpoints, credentials, and the partner's test or production system",
  "mapping-profiles": "Cross-organization entity mapping for tendered loads",
  inbound: "Load tenders received from partners awaiting review",
  outbound: "Load tenders submitted to partners and their lifecycle",
  messages: "Generated X12 documents with delivery and acknowledgment status",
  "inbound-files": "Files received from partner mailboxes and their processing state",
  flows:
    "Each load, tender and invoice a partner flow is following, and the ones held for a planner",
});

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

export function EDIMessagesPage() {
  return <EDIPage kind="messages" />;
}

export function EDIInboundFilesPage() {
  return <EDIPage kind="inbound-files" />;
}

export function EDIFlowInstancesPage() {
  return <EDIPage kind="flows" />;
}

function EDIPage({ kind }: { kind: EDIPageKind }) {
  const t = useT();
  const title = t(pageTitles[kind]);

  return (
    <PageLayout
      pageHeaderProps={{
        title,
        description: pageDescriptions[kind],
      }}
    >
      <LazyComponent fallback={<EDIPageSkeleton kind={kind} title={title} />}>
        <Table kind={kind} />
      </LazyComponent>
    </PageLayout>
  );
}
