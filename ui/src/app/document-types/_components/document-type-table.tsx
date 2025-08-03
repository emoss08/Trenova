/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
