/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { API_URL } from "@/constants/env";
import { type DocumentUploadSchema } from "@/lib/schemas/document-schema";
import { cn, formatFileSize } from "@/lib/utils";
import type {
  FileUploadProps,
  UploadError,
  UploadFileParams,
  UploadingFile,
} from "@/types/file-uploader";
import { useMutation } from "@tanstack/react-query";
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { FileUploadCard } from "./file-upload-card";
import { FileUploadErrorCard } from "./file-upload-error-card";
import { FileUploadErrorDialog } from "./file-upload-error-dialog";
import { FileUploadDock } from "./file-upload-save-dock";
import { DocumentUploadSkeleton } from "./file-upload-skeleton";

export default function DocumentUpload({
  resourceType,
  resourceId,
  allowMultiple = false,
  documentTypes = [],
  maxFileSizeMB = 100,
  // * Only accept PDF, JPG, JPEG, PNG, excel, csv, and DOCX files
  acceptedFileTypes = "application/pdf,image/jpeg,image/jpg,image/png,application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/vnd.ms-excel,text/csv",
  onUploadComplete,
  onUploadError,
  onCancel,
}: FileUploadProps) {
  const [uploadingFiles, setUploadingFiles] = useState<UploadingFile[]>([]);
  const [uploadErrors, setUploadErrors] = useState<UploadError[]>([]);
  const [isHovering, setIsHovering] = useState(false);
  const [showErrorDialog, setShowErrorDialog] = useState(false);
  const [isDirty, setIsDirty] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const dropzoneRef = useRef<HTMLDivElement>(null);

  // * Track dirty state based on uploadingFiles
  useEffect(() => {
    const hasUnsavedFiles = uploadingFiles.some(
      (file) => file.status !== "success" && file.status !== "error",
    );

    // * Set dirty if there are any files that aren't in success or error state
    setIsDirty(hasUnsavedFiles && uploadingFiles.length > 0);
  }, [uploadingFiles]);

  // * Memoize default values to avoid unnecessary re-renders
  const defaultValues = useMemo(
    () => ({
      resourceType,
      resourceId,
      documentType: documentTypes.length > 0 ? documentTypes[0].value : "Other",
      description: "",
    }),
    [resourceType, resourceId, documentTypes],
  );

  const form = useForm<DocumentUploadSchema>({
    defaultValues,
  });

  const { getValues } = form;

  // * Create XHR and form data only once per file upload
  const createFormData = useCallback(
    (
      file: File,
      documentType: string,
      resourceId: string,
      resourceType: string,
    ) => {
      const formData = new FormData();
      formData.append("file", file);
      formData.append("resourceId", resourceId);
      formData.append("resourceType", resourceType);
      formData.append("documentType", documentType);
      return formData;
    },
    [],
  );

  // * Parse error messages from response
  const parseErrorMessage = useCallback((error: any): UploadError => {
    if (error instanceof Error) {
      // * Try to parse details from HTTP error messages
      const match = error.message.match(/HTTP Error: (\d+) (.+)/);
      if (match) {
        const statusCode = match[1];
        const errorMessage = match[2];

        try {
          // * Try to parse as JSON
          const errorDetails = JSON.parse(errorMessage);

          return {
            fileName: "Unknown",
            message: errorDetails.title || "Upload failed",
            details: errorDetails.detail || errorMessage,
            code: statusCode,
          };
        } catch {
          // * Not JSON, use as is
          return {
            fileName: "Unknown",
            message: `Server Error (${statusCode})`,
            details: errorMessage,
            code: statusCode,
          };
        }
      }

      // * Handle specific error messages
      if (error.message.includes("file extension not allowed")) {
        return {
          fileName: "Unknown",
          message: "File type not allowed",
          details: "The file extension is not supported on this server.",
        };
      }

      if (error.message.includes("file size exceeds")) {
        return {
          fileName: "Unknown",
          message: "File size exceeds maximum limit",
          details: "The file size exceeds the maximum limit of 100MB.",
        };
      }

      return {
        fileName: "Unknown",
        message: error.message,
        details: error.stack,
      };
    }

    return {
      fileName: "Unknown",
      message: "Unknown error occurred",
      details: JSON.stringify(error),
    };
  }, []);

  // * File upload mutation
  const { mutateAsync: uploadFileMutation, isPending: isUploading } =
    useMutation({
      mutationFn: async ({
        file,
        resourceId,
        resourceType,
        documentType,
        onProgress,
      }: UploadFileParams) => {
        // * Create form data once
        const formData = createFormData(
          file,
          documentType,
          resourceId,
          resourceType,
        );

        // * Create XMLHttpRequest to track progress
        const xhr = new XMLHttpRequest();

        // * Setup a promise to handle the async request
        return new Promise<any>((resolve, reject) => {
          // * Track upload progress
          xhr.upload.addEventListener("progress", (event) => {
            if (event.lengthComputable) {
              const progress = Math.round((event.loaded * 100) / event.total);
              onProgress(progress);
            }
          });

          // * Handle completion
          xhr.addEventListener("load", () => {
            if (xhr.status >= 200 && xhr.status < 300) {
              try {
                const response = JSON.parse(xhr.responseText);
                resolve(response);
              } catch {
                resolve(xhr.responseText);
              }
            } else {
              reject(
                new Error(`HTTP Error: ${xhr.status} ${xhr.responseText}`),
              );
            }
          });

          // * Handle errors
          xhr.addEventListener("error", () =>
            reject(new Error("Network Error")),
          );
          xhr.addEventListener("abort", () =>
            reject(new Error("Upload Aborted")),
          );

          // * Open and send the request
          xhr.open("POST", `${API_URL}/documents/upload/`);
          xhr.withCredentials = true;
          xhr.setRequestHeader("X-Request-ID", crypto.randomUUID());
          xhr.send(formData);
        });
      },
      onSuccess: (data) => {
        if (onUploadComplete) {
          onUploadComplete(data);
        }
      },
      onError: (error) => {
        if (onUploadError) {
          onUploadError(error);
        }
      },
    });

  // * Pre-calculate max file size in bytes for faster comparison
  const maxFileSizeBytes = useMemo(
    () => maxFileSizeMB * 1024 * 1024,
    [maxFileSizeMB],
  );

  const handleFileSelect = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const { files } = event.target;
      if (!files?.length) return;

      // * Process files in batch
      const newFiles: UploadingFile[] = [];
      const tooLargeFiles: string[] = [];
      const invalidFiles: string[] = [];

      for (let i = 0; i < files.length; i++) {
        const file = files[i];

        // * Check file size
        if (file.size > maxFileSizeBytes) {
          tooLargeFiles.push(file.name);
          continue;
        }

        // * Check file extension
        const fileExtension = file.name.split(".").pop();
        if (!fileExtension || !acceptedFileTypes.includes(fileExtension)) {
          invalidFiles.push(file.name);
          continue;
        }

        newFiles.push({
          file,
          progress: 0,
          documentType: getValues("documentType"),
          status: "uploading",
          fileSize: formatFileSize(file.size),
        });
      }

      // * Show error messages in batch
      if (tooLargeFiles.length) {
        // * Add to error collection instead of just toasting
        const newErrors = tooLargeFiles.map((fileName) => ({
          fileName,
          message: `File exceeds maximum size of ${maxFileSizeMB}MB`,
          details: `Please reduce the file size or use a different file.`,
        }));

        setUploadErrors((prev) => [...prev, ...newErrors]);

        // * Show error dialog if there are errors
        if (newErrors.length > 0) {
          setShowErrorDialog(true);
        }
      }

      if (invalidFiles.length) {
        // * Add to error collection instead of just toasting
        const newErrors = invalidFiles.map((fileName) => ({
          fileName,
          message: "File type not allowed",
          details: `Please use a different file.`,
        }));

        setUploadErrors((prev) => [...prev, ...newErrors]);

        // * Show error dialog if there are errors
        if (newErrors.length > 0) {
          setShowErrorDialog(true);
        }
      }

      // * If we're not allowing multiple files, replace existing files
      // * Otherwise append to existing files
      if (newFiles.length > 0) {
        setUploadingFiles((prev) =>
          allowMultiple ? [...prev, ...newFiles] : newFiles,
        );
      }

      // * Reset the file input so the same file can be selected again if needed
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    },
    [
      allowMultiple,
      getValues,
      maxFileSizeBytes,
      maxFileSizeMB,
      acceptedFileTypes,
    ],
  );

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

  const handleDrop = useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      e.stopPropagation();
      setIsHovering(false);

      const { files } = e.dataTransfer;
      if (!files?.length) return;

      // * Process files in batch
      const newFiles: UploadingFile[] = [];
      const tooLargeFiles: string[] = [];
      const invalidFiles: string[] = [];
      for (let i = 0; i < files.length; i++) {
        const file = files[i];

        // * Check file size
        if (file.size > maxFileSizeBytes) {
          tooLargeFiles.push(file.name);
          continue;
        }

        newFiles.push({
          file,
          progress: 0,
          documentType: getValues("documentType"),
          status: "uploading",
          fileSize: formatFileSize(file.size),
        });
      }

      // * Show error messages in batch
      if (tooLargeFiles.length) {
        // * Add to error collection instead of just toasting
        const newErrors = tooLargeFiles.map((fileName) => ({
          fileName,
          message: `File exceeds maximum size of ${maxFileSizeMB}MB`,
          details: `Please reduce the file size or use a different file.`,
        }));

        setUploadErrors((prev) => [...prev, ...newErrors]);

        // * Show error dialog if there are errors
        if (newErrors.length > 0) {
          setShowErrorDialog(true);
        }
      }

      if (invalidFiles.length) {
        // * Add to error collection instead of just toasting
        const newErrors = invalidFiles.map((fileName) => ({
          fileName,
          message: "File type not allowed",
          details: `Please use a different file.`,
        }));

        setUploadErrors((prev) => [...prev, ...newErrors]);

        // * Show error dialog if there are errors
        if (newErrors.length > 0) {
          setShowErrorDialog(true);
        }
      }

      // * If we're not allowing multiple files, replace existing files
      // * Otherwise append to existing files
      if (newFiles.length > 0) {
        setUploadingFiles((prev) =>
          allowMultiple ? [...prev, ...newFiles] : newFiles,
        );
      }
    },
    [allowMultiple, getValues, maxFileSizeBytes, maxFileSizeMB],
  );

  const removeFile = useCallback((index: number) => {
    setUploadingFiles((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const clearErrors = useCallback(() => {
    setUploadErrors([]);
    setShowErrorDialog(false);
  }, []);

  const triggerFileInput = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const handleCancelUpload = useCallback(() => {
    // * Mark all files as failed that are not already in success or error state
    setUploadingFiles((prev) =>
      prev.map((file) =>
        file.status !== "success" && file.status !== "error"
          ? { ...file, status: "error", error: "Upload cancelled" }
          : file,
      ),
    );

    // * Since we've marked all files as "handled", this will clear the dirty state
    setIsDirty(false);

    // * Call the onCancel callback if provided
    if (onCancel) {
      onCancel();
    }
  }, [onCancel]);

  const uploadFiles = useCallback(async () => {
    if (uploadingFiles.length === 0 || isUploading) return;

    try {
      // * Clear any previous errors
      setUploadErrors([]);
      setIsSubmitting(true);

      // * Get only files that haven't been processed yet
      const pendingFiles = uploadingFiles.filter(
        (file) => file.status !== "success" && file.status !== "error",
      );

      // * If no pending files, nothing to do
      if (pendingFiles.length === 0) {
        setIsSubmitting(false);
        return;
      }

      // * Upload each file in sequence to avoid overwhelming the server
      for (let i = 0; i < pendingFiles.length; i++) {
        const fileInfo = pendingFiles[i];
        // * Find the index in the original array
        const index = uploadingFiles.findIndex(
          (f) =>
            f.file.name === fileInfo.file.name && f.status === fileInfo.status,
        );

        if (index === -1) continue; // Skip if not found (shouldn't happen)

        // * Update file status to uploading
        setUploadingFiles((prev) =>
          prev.map((file, i) =>
            i === index ? { ...file, status: "uploading", progress: 0 } : file,
          ),
        );

        try {
          await uploadFileMutation({
            file: fileInfo.file,
            resourceId,
            resourceType,
            documentType: fileInfo.documentType,
            onProgress: (progress) => {
              setUploadingFiles((prev) =>
                prev.map((file, i) =>
                  i === index ? { ...file, progress } : file,
                ),
              );
            },
          });

          // * Update file status to success
          setUploadingFiles((prev) =>
            prev.map((file, i) =>
              i === index ? { ...file, status: "success" } : file,
            ),
          );
        } catch (error) {
          // * Parse error details
          const errorDetails = parseErrorMessage(error);
          errorDetails.fileName = fileInfo.file.name;

          // * Add to global error collection
          setUploadErrors((prev) => [...prev, errorDetails]);

          // * Update file status to error
          setUploadingFiles((prev) =>
            prev.map((file, i) =>
              i === index
                ? {
                    ...file,
                    status: "error",
                    error: errorDetails.message,
                  }
                : file,
            ),
          );
        }
      }

      // * Show success message
      const successCount = pendingFiles.filter(
        (f) =>
          f.status === "success" ||
          uploadingFiles.find((uf) => uf.file.name === f.file.name)?.status ===
            "success",
      ).length;

      if (successCount > 0) {
        toast.success(
          successCount > 1
            ? `Successfully uploaded ${successCount} documents`
            : "Document uploaded successfully",
        );
      }

      // * Show error dialog if there are errors
      if (uploadErrors.length > 0) {
        setShowErrorDialog(true);
      }

      // * Files either succeeded or failed, so they're no longer "dirty"
      setIsDirty(false);
    } catch (error) {
      console.error("error on upload", error);

      // * Add to error collection
      const errorDetails = parseErrorMessage(error);
      setUploadErrors((prev) => [...prev, errorDetails]);
      setShowErrorDialog(true);
    } finally {
      setIsSubmitting(false);
    }
  }, [
    isUploading,
    uploadingFiles,
    uploadFileMutation,
    resourceId,
    resourceType,
    parseErrorMessage,
    uploadErrors.length,
  ]);

  // * Memoize classNames for better performance
  const dropzoneClassName = useMemo(
    () =>
      cn(
        "border-2 border-dashed rounded-lg p-6 text-center cursor-pointer transition-colors",
        "bg-muted hover:bg-muted flex flex-col items-center justify-center",
        "min-h-[200px]",
      ),
    [],
  );

  // * Group errors by type
  const errorsByType = useMemo(() => {
    const grouped: Record<string, UploadError[]> = {};

    uploadErrors.forEach((error) => {
      const key = error.message;
      if (!grouped[key]) {
        grouped[key] = [];
      }
      grouped[key].push(error);
    });

    return grouped;
  }, [uploadErrors]);

  // * Count files by status
  const fileStats = useMemo(() => {
    const pendingCount = uploadingFiles.filter(
      (f) => f.status === "uploading",
    ).length;
    const successCount = uploadingFiles.filter(
      (f) => f.status === "success",
    ).length;
    const errorCount = uploadingFiles.filter(
      (f) => f.status === "error",
    ).length;

    return {
      total: uploadingFiles.length,
      pending: pendingCount,
      success: successCount,
      error: errorCount,
    };
  }, [uploadingFiles]);

  return (
    <>
      <div
        ref={dropzoneRef}
        className={dropzoneClassName}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        onClick={triggerFileInput}
      >
        <div className="flex items-center justify-center flex-col gap-y-3">
          <DocumentUploadSkeleton isHovering={isHovering} />
          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-1 text-lg">
              <p>Drag and drop files here, or</p>
              <p className="underline cursor-pointer text-semibold">Browse</p>
            </div>
            <p className="text-xs text-muted-foreground">
              {allowMultiple
                ? `Supports PDFs, images, and documents up to ${maxFileSizeMB}MB`
                : `Supports a single PDF, image, or document up to ${maxFileSizeMB}MB`}
            </p>
          </div>
        </div>
        <input
          type="file"
          ref={fileInputRef}
          className="hidden"
          accept={acceptedFileTypes}
          multiple={allowMultiple}
          onChange={handleFileSelect}
        />
      </div>

      {/* Error summary - only shown when errors exist but dialog is closed */}
      {uploadErrors.length > 0 && !showErrorDialog && (
        <FileUploadErrorCard
          errors={uploadErrors}
          onOpenChange={setShowErrorDialog}
        />
      )}

      {uploadingFiles.length > 0 && (
        <div className="mt-6 space-y-1">
          <p className="text-sm font-medium">Uploading files</p>
          <div className="flex flex-col overflow-y-auto max-h-[350px] bg-muted border border-dashed border-border p-1 rounded-md">
            <div className="flex flex-col gap-2">
              {uploadingFiles.map((fileInfo, index) => (
                <FileUploadCard
                  key={fileInfo.file.name}
                  fileInfo={fileInfo}
                  index={index}
                  removeFile={removeFile}
                />
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Error dialog */}
      {showErrorDialog && (
        <FileUploadErrorDialog
          errorsByType={errorsByType}
          clearErrors={clearErrors}
          open={showErrorDialog}
          onOpenChange={setShowErrorDialog}
        />
      )}

      <FileUploadDock
        fileStats={fileStats}
        isDirty={isDirty}
        isSubmitting={isSubmitting}
        onCancel={handleCancelUpload}
        onUpload={uploadFiles}
      />
    </>
  );
}
