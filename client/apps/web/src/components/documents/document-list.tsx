import { cn } from "@trenova/shared/lib/utils";
import type { Document } from "@trenova/shared/types/document";
import { FileIcon } from "lucide-react";
import { DocumentCard } from "./document-card";
import { DocumentGridCard } from "./document-grid-card";
import type { ViewMode } from "./document-toolbar";

interface DocumentListProps {
  documents: Document[];
  viewMode?: ViewMode;
  onPreview?: (document: Document) => void;
  onDownload?: (document: Document) => void;
  onDelete?: (document: Document) => void;
  onInspect?: (document: Document) => void;
  onVersions?: (document: Document) => void;
  deletingId?: string | null;
  isLoading?: boolean;
  className?: string;
  selectedIds?: Set<string>;
  onSelectDocument?: (documentId: string) => void;
  documentTypeMap?: Map<string, string>;
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      <div className="bg-muted mb-4 rounded-full p-4">
        <FileIcon className="text-muted-foreground size-8" />
      </div>
      <h3 className="text-sm font-medium">No documents</h3>
      <p className="text-muted-foreground mt-1 text-sm">
        Upload documents using the drop zone above.
      </p>
    </div>
  );
}

function LoadingStateList() {
  return (
    <div className="space-y-2">
      {[1, 2, 3].map((i) => (
        <div
          key={i}
          className="bg-card flex items-center gap-3 rounded-lg border p-3"
        >
          <div className="bg-muted size-12 animate-pulse rounded-md" />
          <div className="flex-1 space-y-2">
            <div className="bg-muted h-4 w-3/4 animate-pulse rounded" />
            <div className="bg-muted h-3 w-1/2 animate-pulse rounded" />
          </div>
        </div>
      ))}
    </div>
  );
}

function LoadingStateGrid() {
  return (
    <div className="grid grid-cols-2 gap-4">
      {[1, 2, 3, 4].map((i) => (
        <div
          key={i}
          className="bg-card flex flex-col overflow-hidden rounded-lg border"
        >
          <div className="bg-muted/30 flex aspect-square items-center justify-center p-4">
            <div className="bg-muted size-16 animate-pulse rounded-lg" />
          </div>
          <div className="flex flex-col gap-1.5 border-t px-3 py-2.5">
            <div className="bg-muted h-4 w-3/4 animate-pulse rounded" />
            <div className="bg-muted h-3 w-1/2 animate-pulse rounded" />
          </div>
        </div>
      ))}
    </div>
  );
}

export function DocumentList({
  documents,
  viewMode = "grid",
  onPreview,
  onDownload,
  onDelete,
  onInspect,
  onVersions,
  deletingId,
  isLoading = false,
  className,
  selectedIds,
  onSelectDocument,
  documentTypeMap,
}: DocumentListProps) {
  if (isLoading) {
    return viewMode === "grid" ? <LoadingStateGrid /> : <LoadingStateList />;
  }

  if (documents.length === 0) {
    return <EmptyState />;
  }

  if (viewMode === "grid") {
    return (
      <div className={cn("grid grid-cols-2 gap-4", className)}>
        {documents.map((document) => (
          <DocumentGridCard
            key={document.id}
            document={document}
            onPreview={onPreview}
            onDownload={onDownload}
            onDelete={onDelete}
            onInspect={onInspect}
            onVersions={onVersions}
            isDeleting={deletingId === document.id}
            isSelected={selectedIds?.has(document.id)}
            onSelect={onSelectDocument}
            documentTypeName={
              document.documentTypeId
                ? documentTypeMap?.get(document.documentTypeId)
                : undefined
            }
          />
        ))}
      </div>
    );
  }

  return (
    <div className={cn("space-y-2", className)}>
      {documents.map((document) => (
        <DocumentCard
          key={document.id}
          document={document}
          onPreview={onPreview}
          onDownload={onDownload}
          onDelete={onDelete}
          onInspect={onInspect}
          onVersions={onVersions}
          isDeleting={deletingId === document.id}
          isSelected={selectedIds?.has(document.id)}
          onSelect={onSelectDocument}
          documentTypeName={
            document.documentTypeId
              ? documentTypeMap?.get(document.documentTypeId)
              : undefined
          }
        />
      ))}
    </div>
  );
}
