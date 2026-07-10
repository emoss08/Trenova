import { DataTable } from "@/components/data-table/data-table";
import { documentTypeTableGraphQLConfig } from "@/lib/graphql/document-type-table";
import type { DocumentType } from "@/types/document-type";
import { Resource } from "@/types/permission";
import { useMemo } from "react";
import { getColumns } from "./document-type-columns";
import { DocumentTypePanel } from "./document-type-panel";

export default function DocumentTypeTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<DocumentType>
      name="Document Type"
      queryKey="document-type-list"
      graphql={documentTypeTableGraphQLConfig}
      resource={Resource.DocumentType}
      columns={columns}
      TablePanel={DocumentTypePanel}
    />
  );
}
