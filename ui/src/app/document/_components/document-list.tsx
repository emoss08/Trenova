/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { queries } from "@/lib/queries";
import { Resource } from "@/types/audit-entry";
import { useSuspenseQuery } from "@tanstack/react-query";
import { DocumentPreview } from "./document-preview";

export function DocumentList({
  resourceType,
  resourceId,
}: {
  resourceType: Resource;
  resourceId: string;
}) {
  const { data: documents, isLoading } = useSuspenseQuery({
    ...queries.document.documentsByResourceID(resourceType, resourceId),
  });

  if (isLoading) {
    return <p>loading...</p>;
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-6 gap-4">
      {documents.results.map((document) => (
        <DocumentPreview
          key={document.id}
          doc={document}
          handleDocumentClick={() => {}}
          handleDocumentDoubleClick={() => {}}
        />
      ))}
    </div>
  );
}
