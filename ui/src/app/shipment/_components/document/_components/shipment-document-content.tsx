import { AddDocumentCard } from "@/components/document-workflow/document-workflow-add-card";
import { DocumentCard } from "@/components/document-workflow/document-workflow-preview";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { Resource } from "@/types/audit-entry";
import type { Document, DocumentCategory } from "@/types/document";
import { useQueryClient } from "@tanstack/react-query";
import React, { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";

export default function ShipmentDocumentContent({
  isLoading,
  activeCategory,
  allDocuments,
  documentCategories,
  totalCount,
  paginationState,
  setOffset,
  shipmentId,
  fileInputRef,
  isUploading,
  handleFileUpload,
}: {
  isLoading: boolean;
  activeCategory: string | null;
  allDocuments: Document[];
  documentCategories: DocumentCategory[];
  totalCount: number;
  paginationState: {
    offset: number;
    limit: number;
  };
  setOffset: (offset: number) => void;
  fileInputRef: React.RefObject<HTMLInputElement | null>;
  isUploading: boolean;
  handleFileUpload: (files: FileList) => Promise<void>;
  shipmentId?: string;
}) {
  const queryClient = useQueryClient();
  const [isLoadingMore, setIsLoadingMore] = useState<boolean>(false);
  const { offset, limit } = paginationState;

  const filteredDocuments = useMemo(() => {
    if (!allDocuments || !activeCategory) return [];

    const activeCat = documentCategories.find(
      (cat) => cat.id === activeCategory,
    );
    if (!activeCat) return [];

    return allDocuments.filter((doc) => {
      return doc.documentTypeId === activeCategory || doc.id === activeCategory;
    });
  }, [allDocuments, activeCategory, documentCategories]);

  const loadMoreDocuments = useCallback(async () => {
    if (isLoadingMore || allDocuments.length >= totalCount) return;

    setIsLoadingMore(true);
    const nextOffset = offset + limit;
    setOffset(nextOffset);

    try {
      await queryClient.prefetchQuery({
        ...queries.document.documentsByResourceID(
          Resource.Shipment,
          shipmentId ?? "",
          limit,
          nextOffset,
        ),
      });
    } catch (error) {
      console.error("Error loading more documents:", error);
      toast.error("Failed to load more documents");
    } finally {
      setIsLoadingMore(false);
    }
  }, [
    queryClient,
    shipmentId,
    offset,
    limit,
    isLoadingMore,
    allDocuments.length,
    totalCount,
    setOffset,
  ]);

  const hasMoreDocuments = useMemo(() => {
    return allDocuments.length < totalCount;
  }, [allDocuments.length, totalCount]);

  const triggerFileUpload = useCallback(() => {
    fileInputRef.current?.click();
  }, [fileInputRef]);

  return (
    <DocumentContntInner>
      {isLoading && offset === 0 ? (
        <DocumentListSkeleton />
      ) : filteredDocuments.length > 0 ? (
        <DocumentListInner>
          <DocumentList>
            <AddDocumentCard
              onUpload={triggerFileUpload}
              isUploading={isUploading}
              handleFileUpload={handleFileUpload}
            />
            {filteredDocuments.map((doc) => (
              <DocumentCard key={doc.id} document={doc} />
            ))}
          </DocumentList>
          {isLoadingMore && <DocumentListSkeleton />}
          {hasMoreDocuments && activeCategory && !isLoadingMore && (
            <DocumentListFooter>
              <Button
                variant="outline"
                onClick={loadMoreDocuments}
                className="gap-2"
              >
                Load More Documents
              </Button>
            </DocumentListFooter>
          )}
        </DocumentListInner>
      ) : (
        <AddDocumentCardInner>
          <AddDocumentCard
            onUpload={triggerFileUpload}
            isUploading={isUploading}
            handleFileUpload={handleFileUpload}
          />
        </AddDocumentCardInner>
      )}
    </DocumentContntInner>
  );
}

function DocumentList({ children }: { children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {children}
    </div>
  );
}

function DocumentListInner({ children }: { children: React.ReactNode }) {
  return <div className="space-y-4">{children}</div>;
}

function DocumentContntInner({ children }: { children: React.ReactNode }) {
  return <div className="flex-1 p-4 overflow-auto">{children}</div>;
}

function AddDocumentCardInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="h-full flex items-center justify-center w-full">
      {children}
    </div>
  );
}

function DocumentListFooter({ children }: { children: React.ReactNode }) {
  return <div className="flex justify-center py-4">{children}</div>;
}

function DocumentListSkeleton() {
  return (
    <div className="flex justify-center py-4">
      <Skeleton className="size-8 rounded-full animate-spin" />
    </div>
  );
}
