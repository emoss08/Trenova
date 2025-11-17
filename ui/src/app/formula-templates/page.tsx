import { DataTableLazyComponent } from "@/components/error-boundary";
import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const FormulaTemplateTable = lazy(
  () => import("./_components/formula-template-table"),
);

export function FormulaTemplates() {
  return (
    <>
      <MetaTags
        title="Formula Templates"
        description="Manage formula templates for calculating shipment rates"
      />
      <FormSaveProvider>
        <DataTableLazyComponent>
          <FormulaTemplateTable />
        </DataTableLazyComponent>
      </FormSaveProvider>
    </>
  );
}
