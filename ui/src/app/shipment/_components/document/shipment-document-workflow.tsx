import { AddDocumentCard } from "@/components/document-workflow/document-workflow-add-card";
import { CategoryCard } from "@/components/document-workflow/document-workflow-category-card";
import { DocumentCard } from "@/components/document-workflow/document-workflow-preview";
import {
  CategoryListSkeleton,
  DocumentListSkeleton,
  NoDocumentRequirements,
} from "@/components/document-workflow/document-workflow-skeleton";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Icon } from "@/components/ui/icons";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { http } from "@/lib/http-client";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { Resource } from "@/types/audit-entry";
import { TableSheetProps } from "@/types/data-table";
import { Document } from "@/types/document";
import { APIError } from "@/types/errors";
import { Shipment } from "@/types/shipment";
import { faArrowRight } from "@fortawesome/pro-solid-svg-icons";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";

type ShipmentDocumentWorkflowProps = {
  shipmentId: Shipment["id"];
  customerId: string;
} & TableSheetProps;

export function ShipmentDocumentWorkflow({
  shipmentId,
  customerId,
  ...props
}: ShipmentDocumentWorkflowProps) {
  const [activeCategory, setActiveCategory] = useState<string | null>(null);
  const queryClient = useQueryClient();
  const [isUploading, setIsUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [offset, setOffset] = useState(0);
  const [limit] = useState(20);
  const [allDocuments, setAllDocuments] = useState<Document[]>([]);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [totalCount, setTotalCount] = useState(0);

  const { data: docRequirements, isLoading: isLoadingRequirements } =
    useSuspenseQuery({
      ...queries.customer.getDocumentRequirements(customerId),
    });

  const { data: documents, isLoading: isLoadingDocuments } = useSuspenseQuery({
    ...queries.document.documentsByResourceID(
      Resource.Shipment,
      shipmentId,
      limit,
      offset,
    ),
  });

  useEffect(() => {
    if (!documents) return;

    if (offset === 0) {
      setAllDocuments(documents.results);
    } else {
      const existingIds = new Set(allDocuments.map((doc) => doc.id));
      const newDocs = documents.results.filter(
        (doc) => !existingIds.has(doc.id),
      );
      setAllDocuments((prev) => [...prev, ...newDocs]);
    }

    setTotalCount(documents.count);
  }, [documents, offset, allDocuments]);

  const documentCategories = useMemo(() => {
    if (!docRequirements || docRequirements.length === 0) return [];

    return docRequirements
      .map((req) => {
        const docsOfType =
          allDocuments.filter((doc) => doc.documentTypeId === req.docId) || [];

        const complete = docsOfType.length > 0;

        return {
          id: req.docId,
          name: req.name,
          description:
            req.description || `Document required for shipment processing`,
          color: req.color || "#6E56CF",
          requirements: [req],
          complete,
          documentsCount: docsOfType.length,
        };
      })
      .sort((a, b) => {
        if (a.complete !== b.complete) {
          return a.complete ? 1 : -1;
        }
        return a.name.localeCompare(b.name);
      });
  }, [docRequirements, allDocuments]);

  useEffect(() => {
    if (documentCategories.length > 0 && !activeCategory) {
      setActiveCategory(documentCategories[0].id);
    }
  }, [documentCategories, activeCategory]);

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

  const activeCategoryData = activeCategory
    ? documentCategories.find((cat) => cat.id === activeCategory)
    : null;

  const billingReadiness = useMemo(() => {
    const requiredCategories = documentCategories;
    const completedRequired = documentCategories.filter((cat) => cat.complete);

    return {
      total: requiredCategories.length,
      completed: completedRequired.length,
      ready:
        requiredCategories.length > 0 &&
        requiredCategories.length === completedRequired.length,
    };
  }, [documentCategories]);

  const loadMoreDocuments = useCallback(async () => {
    if (isLoadingMore || allDocuments.length >= totalCount) return;

    setIsLoadingMore(true);
    const nextOffset = offset + limit;
    setOffset(nextOffset);

    try {
      await queryClient.prefetchQuery({
        ...queries.document.documentsByResourceID(
          Resource.Shipment,
          shipmentId,
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
  ]);

  const hasMoreDocuments = useMemo(() => {
    return allDocuments.length < totalCount;
  }, [allDocuments.length, totalCount]);

  const handleFileUpload = useCallback(
    async (files: FileList) => {
      if (!files.length || !activeCategory) {
        console.error("Missing files or active category");
        return;
      }

      setIsUploading(true);

      try {
        // Create an array of promises for each file upload
        const uploadPromises = Array.from(files).map(async (file) => {
          const formData = new FormData();
          formData.append("file", file);
          formData.append("resourceId", shipmentId);
          formData.append("resourceType", "shipment");
          formData.append("documentTypeId", activeCategory);

          const response = await http.post("/documents/upload/", formData);

          if (response.status !== 200) {
            throw new Error(`HTTP Error: ${response.status}`);
          }

          return response;
        });

        // Wait for all uploads to complete
        await Promise.all(uploadPromises);

        // Show success message based on number of files
        if (files.length === 1) {
          toast.success("Document uploaded successfully");
        } else {
          toast.success(`${files.length} Documents uploaded successfully`);
        }

        // Refresh the document list
        setOffset(0);
        await queryClient.invalidateQueries({
          queryKey: queries.document.documentsByResourceID._def,
        });
      } catch (error: unknown) {
        const err = error as APIError;
        if (err.data) {
          err.data.invalidParams.forEach((param) => {
            toast.error(param.reason);
          });
        } else {
          toast.error(err.message || "Failed to upload document");
        }
      } finally {
        setIsUploading(false);

        if (fileInputRef.current) {
          fileInputRef.current.value = "";
        }
      }
    },
    [shipmentId, activeCategory, queryClient],
  );

  const handleFileInputChange = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const { files } = event.target;
      if (files?.length) {
        handleFileUpload(files);
      }
    },
    [handleFileUpload],
  );

  const triggerFileUpload = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  return (
    <Dialog {...props}>
      <DialogContent className="sm:max-w-screen-xl p-0 overflow-hidden">
        <VisuallyHidden>
          <DialogHeader>
            <DialogTitle>Document Workflow</DialogTitle>
            <DialogDescription>
              Upload and manage shipment documents
            </DialogDescription>
          </DialogHeader>
        </VisuallyHidden>
        <DialogBody className="p-0">
          <div className="flex h-[calc(100vh-100px)] max-h-[800px]">
            <div className="w-1/4 bg-muted/20 border-r border-border">
              <div className="p-4 border-b border-border">
                <h2 className="text-lg font-semibold">Document Requirements</h2>
                <p className="text-sm text-muted-foreground">
                  Complete all document requirements to process the shipment
                </p>
                {billingReadiness.total > 0 && (
                  <div className="mt-3 p-2 rounded-md bg-background border border-border">
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-sm font-medium">Billing Ready</span>
                      {billingReadiness.ready ? (
                        <Button
                          title="Transfer to Billing"
                          aria-label="Transfer to Billing"
                          variant="outline"
                          size="xs"
                        >
                          Transfer to Billing
                          <Icon icon={faArrowRight} className="size-4" />
                        </Button>
                      ) : (
                        <Badge withDot={false} variant="outline">
                          {billingReadiness.completed}/{billingReadiness.total}
                        </Badge>
                      )}
                    </div>
                    <div className="w-full h-1 bg-muted rounded-full overflow-hidden">
                      <div
                        className={cn(
                          "h-full rounded-full",
                          billingReadiness.ready
                            ? "bg-green-600"
                            : "bg-primary",
                        )}
                        style={{
                          width: `${
                            billingReadiness.total > 0
                              ? (billingReadiness.completed /
                                  billingReadiness.total) *
                                100
                              : 0
                          }%`,
                        }}
                      />
                    </div>
                  </div>
                )}
              </div>
              <ScrollArea className="h-[calc(100%-140px)]">
                <div className="p-2">
                  {isLoadingRequirements ? (
                    <CategoryListSkeleton />
                  ) : documentCategories.length > 0 ? (
                    documentCategories.map((category) => (
                      <CategoryCard
                        key={category.id}
                        category={category}
                        isActive={category.id === activeCategory}
                        onClick={() => setActiveCategory(category.id)}
                      />
                    ))
                  ) : (
                    <NoDocumentRequirements customerId={customerId} />
                  )}
                </div>
              </ScrollArea>
            </div>
            <div className="w-3/4 flex flex-col">
              <div className="p-4 border-b border-border">
                <div className="flex justify-between items-center">
                  <div>
                    <h2 className="text-lg font-semibold">
                      {activeCategoryData?.name || "Document Management"}
                    </h2>
                    <p className="text-sm text-muted-foreground">
                      {activeCategoryData?.description ||
                        "Upload and manage shipment documents"}
                    </p>
                  </div>
                </div>
              </div>

              <div className="flex-1 p-4 overflow-auto">
                {isLoadingDocuments && offset === 0 ? (
                  <DocumentListSkeleton />
                ) : filteredDocuments.length > 0 ? (
                  <div className="space-y-4">
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                      <AddDocumentCard
                        onUpload={triggerFileUpload}
                        isUploading={isUploading}
                        handleFileUpload={handleFileUpload}
                      />
                      {filteredDocuments.map((doc) => (
                        <DocumentCard key={doc.id} document={doc} />
                      ))}
                    </div>
                    {isLoadingMore && (
                      <div className="flex justify-center py-4">
                        <Skeleton className="h-8 w-8 rounded-full animate-spin" />
                      </div>
                    )}
                    {hasMoreDocuments && activeCategory && !isLoadingMore && (
                      <div className="flex justify-center py-4">
                        <Button
                          variant="outline"
                          onClick={loadMoreDocuments}
                          className="gap-2"
                        >
                          Load More Documents
                        </Button>
                      </div>
                    )}
                  </div>
                ) : (
                  <div className="h-full flex items-center justify-center">
                    <div className="w-full max-w-sm">
                      <AddDocumentCard
                        onUpload={triggerFileUpload}
                        isUploading={isUploading}
                        handleFileUpload={handleFileUpload}
                      />
                    </div>
                  </div>
                )}
              </div>
              <input
                type="file"
                multiple
                ref={fileInputRef}
                className="hidden"
                accept="application/pdf,image/jpeg,image/jpg,image/png,application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/vnd.ms-excel,text/csv"
                onChange={handleFileInputChange}
              />
            </div>
          </div>
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}
