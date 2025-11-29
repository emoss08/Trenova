import { DataTableLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const DocumentTemplateTable = lazy(
  () => import("./_components/document-template-table"),
);

export function DocumentTemplates() {
  return (
    <>
      <MetaTags title="Document Templates" description="Document Templates" />
      <div className="flex flex-col gap-y-3">
        <Header />
        <DataTableLazyComponent>
          <DocumentTemplateTable />
        </DataTableLazyComponent>
      </div>
    </>
  );
}

function Header() {
  return (
    <div className="flex items-start justify-between">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">
          Document Templates
        </h1>
        <p className="text-muted-foreground">
          Manage and configure document templates for your organization
        </p>
      </div>
    </div>
  );
}
