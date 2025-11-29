import { DataTable } from "@/components/data-table/data-table";
import type { DocumentTemplateSchema } from "@/lib/schemas/document-template-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./document-template-columns";
import { DocumentTemplateCreateDialog } from "./document-template-create-dialog";
import { DocumentTemplateEditDialog } from "./document-template-edit-dialog";

export default function DocumentTemplateTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<DocumentTemplateSchema>
      resource={Resource.DocumentTemplate}
      name="Document Template"
      link="/document-templates/"
      queryKey="document-template-list"
      exportModelName="document-template"
      columns={columns}
      extraSearchParams={{ includeType: "true" }}
      TableModal={DocumentTemplateCreateDialog}
      TableEditModal={DocumentTemplateEditDialog}
    />
  );
}
