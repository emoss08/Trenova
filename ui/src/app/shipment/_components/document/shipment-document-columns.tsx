/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import { FileTypeCard } from "@/components/file-uploader/file-type-card";
import { UserAvatar } from "@/components/nav-user";
import { DocumentStatusBadge } from "@/components/status-badge";
import { Separator } from "@/components/ui/separator";
import { generateDateOnlyString, toDate } from "@/lib/date";
import { formatFileSize } from "@/lib/utils";
import { type Document } from "@/types/document";
import { type ColumnDef } from "@tanstack/react-table";

function DocumentTableCell({
  doc,
  onClick,
}: {
  doc: Document;
  onClick: () => void;
}) {
  return (
    <div
      onClick={onClick}
      className="group flex items-center gap-2 px-1 py-1.5 text-left text-sm cursor-pointer"
    >
      <FileTypeCard status="success" fileType={doc.fileType} />
      <div className="grid w-full flex-1 text-left leading-tight">
        <span className="group-hover:underline text-sm font-semibold truncate max-w-[250px]">
          {doc.fileName}
        </span>
        <div className="flex items-center gap-2">
          <span className="text-xs">{formatFileSize(doc.fileSize)}</span>
          <Separator className="h-6 w-px bg-border" orientation="vertical" />
          <span className="text-xs">{doc.fileType}</span>
        </div>
      </div>
    </div>
  );
}

export function getColumns({
  handleDocumentClick,
}: {
  handleDocumentClick: (doc: Document) => void;
}): ColumnDef<Document>[] {
  return [
    {
      id: "status",
      accessorKey: "status",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Status" />
      ),
      cell: ({ row }) => {
        const { status } = row.original;
        return <DocumentStatusBadge status={status} />;
      },
    },
    {
      id: "document",
      header: "Document",
      cell: ({ row }) => {
        return (
          <DocumentTableCell
            doc={row.original}
            onClick={() => handleDocumentClick(row.original)}
          />
        );
      },
    },
    {
      id: "uploadedBy",
      header: "Uploaded By",
      cell: ({ row }) => {
        const { uploadedBy } = row.original;
        if (!uploadedBy) return <p>-</p>;
        return <UserAvatar user={uploadedBy} />;
      },
    },
    {
      id: "createdAt",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Created At" />
      ),
      cell: ({ row }) => {
        const { createdAt } = row.original;
        const date = toDate(createdAt as number);
        if (!date) return <p>-</p>;

        return <p>{generateDateOnlyString(date)}</p>;
      },
    },
  ];
}
