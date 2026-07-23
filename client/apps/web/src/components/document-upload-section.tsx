import { apiService } from "@/services/api";
import type { Document } from "@/types/document";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  DownloadIcon,
  FileIcon,
  Trash2Icon,
  UploadCloudIcon,
} from "lucide-react";
import { useCallback, useRef, useState } from "react";
import { toast } from "sonner";
import { Button } from "./ui/button";
import { FormSection } from "./ui/form";

interface DocumentUploadSectionProps {
  resourceId: string;
  resourceType: string;
  disabled?: boolean;
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 Bytes";
  const k = 1024;
  const sizes = ["Bytes", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / k ** i).toFixed(2))} ${sizes[i]}`;
}

function formatDate(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleDateString();
}

function DocumentRow({
  document,
  onDownload,
  onDelete,
  isDeleting,
}: {
  document: Document;
  onDownload: (doc: Document) => void;
  onDelete: (doc: Document) => void;
  isDeleting: boolean;
}) {
  return (
    <div className="flex items-center justify-between rounded-md border p-3">
      <div className="flex items-center gap-3">
        <FileIcon className="size-5 text-muted-foreground" />
        <div>
          <p className="text-sm font-medium">{document.originalName}</p>
          <p className="text-xs text-muted-foreground">
            {formatFileSize(document.fileSize)} &bull;{" "}
            {formatDate(document.createdAt)}
          </p>
        </div>
      </div>
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onDownload(document)}
          aria-label="Download document"
        >
          <DownloadIcon className="size-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onDelete(document)}
          disabled={isDeleting}
          aria-label="Delete document"
        >
          <Trash2Icon className="size-4 text-destructive" />
        </Button>
      </div>
    </div>
  );
}

export function DocumentUploadSection({
  resourceId,
  resourceType,
  disabled = false,
}: DocumentUploadSectionProps) {
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const queryKey = ["documents", resourceType, resourceId];

  const { data: documents = [], isLoading } = useQuery({
    queryKey,
    queryFn: () =>
      apiService.documentService.getByResource(resourceType, resourceId, undefined, { includeDocumentType: "true" }),
    enabled: !!resourceId,
  });

  const uploadMutation = useMutation({
    mutationFn: (file: File) =>
      apiService.documentService.upload({
        file,
        resourceId,
        resourceType,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey });
      toast.success("Document uploaded successfully");
    },
    onError: (error) => {
      toast.error(`Upload failed: ${error.message}`);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (documentId: string) =>
      apiService.documentService.delete(documentId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey });
      toast.success("Document deleted successfully");
      setDeletingId(null);
    },
    onError: (error) => {
      toast.error(`Delete failed: ${error.message}`);
      setDeletingId(null);
    },
  });

  const { mutate: uploadDocument } = uploadMutation;
  const { mutate: deleteDocument } = deleteMutation;

  const handleFiles = useCallback(
    (files: FileList | null) => {
      if (!files || files.length === 0) return;
      Array.from(files).forEach((file) => {
        uploadDocument(file);
      });
    },
    [uploadDocument],
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setIsDragging(false);
      if (disabled) return;
      handleFiles(e.dataTransfer.files);
    },
    [disabled, handleFiles],
  );

  const handleDragOver = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      if (!disabled) {
        setIsDragging(true);
      }
    },
    [disabled],
  );

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  }, []);

  const handleDownload = useCallback(async (doc: Document) => {
    try {
      const url = await apiService.documentService.getDownloadUrl(doc.id);
      window.open(url, "_blank");
    } catch {
      toast.error("Failed to get download URL");
    }
  }, []);

  const handleDelete = useCallback(
    (doc: Document) => {
      setDeletingId(doc.id);
      deleteDocument(doc.id);
    },
    [deleteDocument],
  );

  const handleClick = useCallback(() => {
    if (!disabled) {
      fileInputRef.current?.click();
    }
  }, [disabled]);

  if (!resourceId) {
    return (
      <FormSection title="Documents" className="border-t pt-2">
        <p className="text-sm text-muted-foreground">
          Save the record to upload documents.
        </p>
      </FormSection>
    );
  }

  return (
    <FormSection title="Documents" className="border-t pt-2">
      <div className="space-y-4">
        <div
          onClick={handleClick}
          onDrop={handleDrop}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onKeyDown={(e) => {
            if (e.key === "Enter" || e.key === " ") {
              handleClick();
            }
          }}
          role="button"
          tabIndex={disabled ? -1 : 0}
          aria-label="Upload documents"
          aria-disabled={disabled}
          className={`flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-muted-foreground/25 p-6 transition-colors hover:border-muted-foreground/50 ${
            isDragging ? "border-primary bg-primary/5" : ""
          } ${disabled ? "cursor-not-allowed opacity-50" : ""}`}
        >
          <UploadCloudIcon className="mb-2 size-10 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">
            {uploadMutation.isPending
              ? "Uploading..."
              : "Drop files here or click to upload"}
          </p>
          <p className="text-xs text-muted-foreground/70">
            PDF, Images, Documents up to 50MB
          </p>
          <input
            ref={fileInputRef}
            type="file"
            multiple
            className="hidden"
            onChange={(e) => handleFiles(e.target.files)}
            disabled={disabled || uploadMutation.isPending}
            accept=".pdf,.jpg,.jpeg,.png,.webp,.doc,.docx"
          />
        </div>

        {isLoading ? (
          <p className="text-sm text-muted-foreground">Loading documents...</p>
        ) : documents.length > 0 ? (
          <div className="space-y-2">
            {documents.map((doc) => (
              <DocumentRow
                key={doc.id}
                document={doc}
                onDownload={handleDownload}
                onDelete={handleDelete}
                isDeleting={deletingId === doc.id}
              />
            ))}
          </div>
        ) : null}
      </div>
    </FormSection>
  );
}
