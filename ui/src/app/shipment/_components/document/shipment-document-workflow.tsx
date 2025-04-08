import { DocumentUploadSkeleton } from "@/components/file-uploader/file-upload-skeleton";
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
import { API_URL } from "@/constants/env";
import { queries } from "@/lib/queries";
import { cn, truncateText } from "@/lib/utils";
import { Resource } from "@/types/audit-entry";
import { CustomerDocumentRequirement } from "@/types/customer";
import { TableSheetProps } from "@/types/data-table";
import { Document } from "@/types/document";
import { Shipment } from "@/types/shipment";
import {
  faArrowRight,
  faCheck,
  faDownload,
  faEye,
} from "@fortawesome/pro-solid-svg-icons";
import { Viewer, Worker } from "@react-pdf-viewer/core";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { motion } from "framer-motion";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
type DocumentCategory = {
  id: string;
  name: string;
  description: string;
  color: string;
  requirements: CustomerDocumentRequirement[];
  complete: boolean;
  documentsCount: number;
};

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
        const formData = new FormData();
        formData.append("file", files[0]);
        formData.append("resourceId", shipmentId);
        formData.append("resourceType", "shipment");
        formData.append("documentTypeId", activeCategory);

        const response = await fetch(`${API_URL}/documents/upload/`, {
          method: "POST",
          body: formData,
          credentials: "include",
          headers: {
            "X-Request-ID": crypto.randomUUID(),
          },
        });

        if (!response.ok) {
          const errorText = await response.text();
          throw new Error(`HTTP Error: ${response.status} ${errorText}`);
        }

        toast.success("Document uploaded successfully");

        setOffset(0);
        await queryClient.invalidateQueries({
          queryKey: queries.document.documentsByResourceID._def,
        });
      } catch (error) {
        console.error("Upload error:", error);
        toast.error("Failed to upload document. Please try again.");
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
                    <div className="p-4 text-center text-muted-foreground">
                      <p>No document requirements found</p>
                    </div>
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
                      />
                    </div>
                  </div>
                )}
              </div>
              <input
                type="file"
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

function CategoryCard({
  category,
  isActive,
  onClick,
}: {
  category: DocumentCategory;
  isActive: boolean;
  onClick: () => void;
}) {
  const categoryStyle = useMemo(() => {
    const bgColor = isActive ? "bg-accent/50" : "bg-background";

    return `border-border ${bgColor}`;
  }, [isActive]);

  return (
    <div
      className={cn(
        "p-3 rounded-md mb-2 cursor-pointer transition-all",
        "border hover:bg-accent/30",
        categoryStyle,
        !category.complete && "bg-muted/20",
      )}
      onClick={onClick}
      style={{
        borderLeftWidth: "4px",
        borderLeftColor: category.color,
      }}
    >
      <div className="flex items-center justify-between mb-1">
        <h3 className="font-medium truncate">{category.name}</h3>
        {category.complete ? (
          <Badge
            withDot={false}
            variant="active"
            className="flex items-center gap-1 text-xs"
          >
            <Icon icon={faCheck} className="size-3" />
            Complete
          </Badge>
        ) : (
          <Badge withDot={false} variant="inactive" className="text-xs">
            Not Complete
          </Badge>
        )}
      </div>

      <div className="text-xs text-muted-foreground flex justify-between">
        <span className="truncate">
          {truncateText(category.description, 25)}
        </span>
        <span>
          {category.documentsCount} Document
          {category.documentsCount > 1 ? "s" : ""}
        </span>
      </div>
    </div>
  );
}

function DocumentCard({ document }: { document: Document }) {
  const [isHovering, setIsHovering] = useState(false);

  console.log(document);

  return (
    <div
      className="border border-border rounded-md overflow-hidden bg-card relative h-[200px] cursor-pointer"
      onMouseEnter={() => setIsHovering(true)}
      onMouseLeave={() => setIsHovering(false)}
    >
      {/* Document preview area */}
      <div className="h-full w-full flex items-center justify-center p-2">
        <div className="bg-muted/40 p-3 rounded-md">
          <DocumentPreview document={document} />
        </div>
      </div>

      {/* Footer that slides up on hover */}
      <motion.div
        className="absolute bottom-0 left-0 right-0 bg-card border-t border-border p-3"
        initial={{ y: "100%" }}
        animate={{ y: isHovering ? 0 : "100%" }}
        transition={{ duration: 0.2 }}
      >
        <div className="space-y-2">
          <div className="flex justify-between items-center">
            <span className="text-xs text-muted-foreground">Uploaded</span>
            <span className="text-xs font-medium">
              {new Date(document.createdAt).toLocaleDateString()}
            </span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-xs text-muted-foreground">Type</span>
            <span className="text-xs font-medium">
              {document.documentTypeId || "Document"}
            </span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-xs text-muted-foreground">Size</span>
            <span className="text-xs font-medium">
              {document.fileSize
                ? `${Math.round(document.fileSize / 1024)} KB`
                : "Unknown"}
            </span>
          </div>
          <div className="flex gap-2 pt-1">
            <Button size="sm" variant="outline" className="w-full text-xs">
              <Icon icon={faDownload} className="mr-1 size-3" />
              Download
            </Button>
            <Button size="sm" variant="outline" className="w-full text-xs">
              <Icon icon={faEye} className="mr-1 size-3" />
              View
            </Button>
          </div>
        </div>
      </motion.div>
    </div>
  );
}

function DocumentPreview({ document }: { document: Document }) {
  if (document.fileType.includes("pdf")) {
    return (
      <Worker workerUrl="https://unpkg.com/pdfjs-dist@3.11.174/build/pdf.worker.min.js">
        <Viewer fileUrl={document.presignedUrl || ""} />
      </Worker>
    );
  }

  if (document.fileType.includes("image")) {
    return <img src={document.presignedUrl || ""} alt={document.fileName} />;
  }

  return <div>Document Type Not Supported</div>;
}

function CategoryListSkeleton() {
  return (
    <div className="space-y-2">
      {[1, 2, 3, 4].map((i) => (
        <div key={i} className="p-3 border border-border rounded-md">
          <div className="flex justify-between mb-1">
            <Skeleton className="h-5 w-24" />
            <Skeleton className="h-5 w-16" />
          </div>
          <Skeleton className="h-4 w-full mt-2" />
          <Skeleton className="h-1 w-full mt-2" />
        </div>
      ))}
    </div>
  );
}

function DocumentListSkeleton() {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {[1, 2, 3, 4, 5, 6].map((i) => (
        <div
          key={i}
          className="border border-border rounded-md overflow-hidden"
        >
          <div className="p-3 border-b border-border">
            <Skeleton className="h-5 w-full" />
          </div>
          <div className="p-3">
            <div className="space-y-2">
              <div className="flex justify-between">
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-4 w-24" />
              </div>
              <div className="flex justify-between">
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-4 w-16" />
              </div>
              <div className="flex justify-between">
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-4 w-24" />
              </div>
            </div>
          </div>
          <div className="p-3 bg-muted/10 flex justify-between">
            <Skeleton className="h-8 w-20" />
            <Skeleton className="h-8 w-20" />
          </div>
        </div>
      ))}
    </div>
  );
}

function AddDocumentCard({
  onUpload,
  isUploading,
}: {
  onUpload: () => void;
  isUploading: boolean;
}) {
  const [isHovering, setIsHovering] = useState(false);

  // * Memoize event handlers
  const handleDragOver = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsHovering(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsHovering(false);
  }, []);

  const handleMouseEnter = useCallback(() => {
    setIsHovering(true);
  }, []);

  const handleMouseLeave = useCallback(() => {
    setIsHovering(false);
  }, []);

  const handleDrop = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsHovering(false);
  }, []);

  return (
    <div
      className="flex justify-center items-center border border-dashed border-border rounded-md overflow-hidden hover:shadow-md transition-shadow hover:bg-accent/5 cursor-pointer"
      onClick={onUpload}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      <div className="flex items-center justify-center flex-col gap-y-3 p-8">
        <DocumentUploadSkeleton isHovering={isHovering} />
        <div className="flex flex-col gap-y-1 justify-center text-center items-center">
          <div className="flex items-center gap-1 text-sm">
            <p>Drag and drop files here, or</p>
            <p className="underline cursor-pointer text-semibold">Browse</p>
          </div>
          <p className="text-2xs text-muted-foreground">
            Supports PDF, images and documents up to 100MB
          </p>
        </div>
        {isUploading && (
          <div className="mt-2 w-full">
            <div className="w-full h-1 bg-muted rounded-full overflow-hidden">
              <div
                className="h-full bg-primary rounded-full animate-pulse"
                style={{ width: "90%" }}
              />
            </div>
            <p className="text-xs text-muted-foreground text-center mt-1">
              Uploading...
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
