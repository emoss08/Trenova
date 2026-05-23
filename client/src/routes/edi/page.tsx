import { ComponentLoader } from "@/components/component-loader";
import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import type { EDIPageKind } from "./_components/edi-types";

const Table = lazy(() => import("./_components/edi-table"));

const pageTitles: Record<EDIPageKind, string> = {
  partners: "EDI Partners",
  "communication-profiles": "EDI Communication Profiles",
  "mapping-profiles": "EDI Mapping Profiles",
  designer: "Template Designer",
  inbound: "Inbound EDI Transfers",
  outbound: "Outbound EDI Transfers",
};

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

function EDIPage({ kind }: { kind: EDIPageKind }) {
  return (
    <PageLayout
      pageHeaderProps={{
        title: pageTitles[kind],
        description: "Internal load tender exchange, mapping, and lifecycle visibility",
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
