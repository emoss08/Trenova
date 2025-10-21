import { DataTableLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const DocumentTypesDataTable = lazy(
  () => import("./_components/document-type-table"),
);

export function DocumentTypes() {
  return (
    <>
      <MetaTags title="Document Types" description="Document Types" />
      <DataTableLazyComponent>
        <DocumentTypesDataTable />
      </DataTableLazyComponent>
    </>
  );
}
