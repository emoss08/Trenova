import { LazyComponent } from "@/components/error-boundary";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { http } from "@/lib/http-client";
import { queries } from "@/lib/queries";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { Resource } from "@/types/audit-entry";
import { TableSheetProps } from "@/types/data-table";
import { Document } from "@/types/document";
import { APIError } from "@/types/errors";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { lazy, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { ShipmentDocumentSidebar } from "./shipment-document-sidebar";
import { ShipmentDocumentWorkflowHeader } from "./shipment-document-workflow-header";

const ShipmentDocumentContent = lazy(
  () => import("./shipment-document-content"),
);

type ShipmentDocumentWorkflowProps = {
  shipmentId: ShipmentSchema["id"];
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
  const [totalCount, setTotalCount] = useState(0);

  const { data: docRequirements, isLoading: isLoadingRequirements } =
    useSuspenseQuery({
      ...queries.customer.getDocumentRequirements(customerId),
    });

  const { data: documents, isLoading: isLoadingDocuments } = useSuspenseQuery({
    ...queries.document.documentsByResourceID(
      Resource.Shipment,
      shipmentId ?? "",
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

  const activeCategoryData = activeCategory
    ? documentCategories.find((cat) => cat.id === activeCategory)
    : null;

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
          formData.append("resourceId", shipmentId ?? "");
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
            <ShipmentDocumentSidebar
              documentCategories={documentCategories}
              isLoadingRequirements={isLoadingRequirements}
              activeCategory={activeCategory}
              setActiveCategory={setActiveCategory}
              customerId={customerId}
            />
            <div className="w-3/4 flex flex-col">
              <ShipmentDocumentWorkflowHeader
                activeCategoryData={activeCategoryData}
              />
              <LazyComponent>
                <ShipmentDocumentContent
                  isLoading={isLoadingDocuments}
                  activeCategory={activeCategory}
                  allDocuments={allDocuments}
                  documentCategories={documentCategories}
                  totalCount={totalCount}
                  paginationState={{ offset, limit }}
                  shipmentId={shipmentId}
                  setOffset={setOffset}
                  fileInputRef={fileInputRef}
                  isUploading={isUploading}
                  handleFileUpload={handleFileUpload}
                />
              </LazyComponent>
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
