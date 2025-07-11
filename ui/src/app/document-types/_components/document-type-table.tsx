import { DataTable } from "@/components/data-table/data-table";
import { type DocumentTypeSchema } from "@/lib/schemas/document-type-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./document-type-columns";
import { DocumentTypeCreateModal } from "./document-type-create-modal";
import { EditDocumentTypeModal } from "./document-type-edit-modal";

export default function DocumentTypesDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<DocumentTypeSchema>
      resource={Resource.DocumentType}
      name="Document Type"
      link="/document-types/"
      queryKey="document-type-list"
      exportModelName="document-type"
      TableModal={DocumentTypeCreateModal}
      TableEditModal={EditDocumentTypeModal}
      columns={columns}
    />
  );
}
