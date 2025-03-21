import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { API_URL } from "@/constants/env";
import { DocumentUploadSchema } from "@/lib/schemas/document-schema";
import { cn, formatFileSize, getFileIcon } from "@/lib/utils";
import {
  faExclamationTriangle,
  faFile,
  faImage,
  faTrash,
} from "@fortawesome/pro-regular-svg-icons";
import { useMutation } from "@tanstack/react-query";
import React, { useCallback, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { Icon } from "./icons";

export interface DocumentType {
  value: string;
  label: string;
}

export interface FileUploadProps {
  // Required props
  resourceType: string;
  resourceId: string;

  // Optional props
  allowMultiple?: boolean;
  documentTypes?: DocumentType[];
  maxFileSizeMB?: number;
  acceptedFileTypes?: string;
  onUploadComplete?: (response: any) => void;
  onUploadError?: (error: any) => void;
  onCancel?: () => void;
  showDocumentTypeSelection?: boolean;
  requireApproval?: boolean;
}

export interface UploadingFile {
  file: File;
  progress: number;
  documentType: string;
  status: "uploading" | "success" | "error";
  error?: string;
  fileSize?: string; // Formatted file size for display
}

export interface UploadFileParams {
  file: File;
  resourceId: string;
  resourceType: string;
  documentType: string;
  description: string;
  onProgress: (progress: number) => void;
}

export function DocumentUpload({
  resourceType,
  resourceId,
  allowMultiple = false,
  documentTypes = [],
  maxFileSizeMB = 100,
  acceptedFileTypes = "*",
  onUploadComplete,
  onUploadError,
  onCancel,
}: FileUploadProps) {
  const [uploadingFiles, setUploadingFiles] = useState<UploadingFile[]>([]);
  const [isHovering, setIsHovering] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const dropzoneRef = useRef<HTMLDivElement>(null);

  const form = useForm<DocumentUploadSchema>({
    defaultValues: {
      resourceType,
      resourceId,
      documentType: documentTypes.length > 0 ? documentTypes[0].value : "Other",
      description: "",
    },
  });

  const { getValues, watch } = form;
  const description = watch("description");

  // File upload mutation
  const uploadFileMutation = useMutation({
    mutationFn: async ({
      file,
      resourceId,
      resourceType,
      documentType,
      description,
      onProgress,
    }: UploadFileParams) => {
      const formData = new FormData();
      formData.append("file", file);
      formData.append("resourceId", resourceId);
      formData.append("resourceType", resourceType);
      formData.append("documentType", documentType);
      formData.append("description", description);

      // Create XMLHttpRequest to track progress
      const xhr = new XMLHttpRequest();

      // Setup a promise to handle the async request
      return new Promise<any>((resolve, reject) => {
        // Track upload progress
        xhr.upload.addEventListener("progress", (event) => {
          if (event.lengthComputable) {
            const progress = Math.round((event.loaded * 100) / event.total);
            onProgress(progress);
          }
        });

        // Handle completion
        xhr.addEventListener("load", () => {
          if (xhr.status >= 200 && xhr.status < 300) {
            try {
              const response = JSON.parse(xhr.responseText);
              resolve(response);
            } catch {
              resolve(xhr.responseText);
            }
          } else {
            reject(new Error(`HTTP Error: ${xhr.status}`));
          }
        });

        // Handle errors
        xhr.addEventListener("error", () => reject(new Error("Network Error")));
        xhr.addEventListener("abort", () =>
          reject(new Error("Upload Aborted")),
        );

        // Open and send the request
        xhr.open("POST", `${API_URL}/documents/upload/`);
        xhr.withCredentials = true;

        // Set any authentication headers if needed
        // xhr.setRequestHeader("Authorization", "Bearer your-token");
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

  const handleFileSelect = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const { files } = event.target;

      if (!files || files.length === 0) return;

      const newFiles: UploadingFile[] = [];

      for (let i = 0; i < files.length; i++) {
        const file = files[i];

        // Check file size
        if (file.size > maxFileSizeMB * 1024 * 1024) {
          toast.error(
            `${file.name} exceeds the maximum size of ${maxFileSizeMB}MB`,
          );
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

      // If we're not allowing multiple files, replace existing files
      // Otherwise append to existing files
      if (!allowMultiple) {
        setUploadingFiles(newFiles);
      } else {
        setUploadingFiles((prev) => [...prev, ...newFiles]);
      }

      // Reset the file input so the same file can be selected again if needed
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    },
    [allowMultiple, getValues, maxFileSizeMB],
  );

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

      if (!files || files.length === 0) return;

      const newFiles: UploadingFile[] = [];

      for (let i = 0; i < files.length; i++) {
        const file = files[i];

        // Check file size
        if (file.size > maxFileSizeMB * 1024 * 1024) {
          toast.error(
            `${file.name} exceeds the maximum size of ${maxFileSizeMB}MB`,
          );
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

      // If we're not allowing multiple files, replace existing files
      // Otherwise append to existing files
      if (!allowMultiple) {
        setUploadingFiles(newFiles);
      } else {
        setUploadingFiles((prev) => [...prev, ...newFiles]);
      }
    },
    [allowMultiple, getValues, maxFileSizeMB],
  );

  const removeFile = useCallback((index: number) => {
    setUploadingFiles((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const uploadFiles = async () => {
    if (uploadingFiles.length === 0 || uploadFileMutation.isPending) return;

    try {
      // Upload each file in sequence to avoid overwhelming the server
      for (let i = 0; i < uploadingFiles.length; i++) {
        const fileInfo = uploadingFiles[i];
        const index = i;

        // Update file status to uploading
        setUploadingFiles((prev) =>
          prev.map((file, i) =>
            i === index ? { ...file, status: "uploading", progress: 0 } : file,
          ),
        );

        try {
          await uploadFileMutation.mutateAsync({
            file: fileInfo.file,
            resourceId,
            resourceType,
            documentType: fileInfo.documentType,
            description: description || "",
            onProgress: (progress) => {
              setUploadingFiles((prev) =>
                prev.map((file, i) =>
                  i === index ? { ...file, progress } : file,
                ),
              );
            },
          });

          // Update file status to success
          setUploadingFiles((prev) =>
            prev.map((file, i) =>
              i === index ? { ...file, status: "success" } : file,
            ),
          );
        } catch (error) {
          // Update file status to error
          setUploadingFiles((prev) =>
            prev.map((file, i) =>
              i === index
                ? {
                    ...file,
                    status: "error",
                    error:
                      error instanceof Error ? error.message : "Upload failed",
                  }
                : file,
            ),
          );
        }
      }

      // Show success message
      const successCount = uploadingFiles.filter(
        (f) => f.status === "success",
      ).length;
      if (successCount > 0) {
        toast.success(
          successCount > 1
            ? `Successfully uploaded ${successCount} documents`
            : "Document uploaded successfully",
        );
      }
    } catch (error) {
      console.error("error on upload", error);
      toast.error("One or more files failed to upload");
    }
  };

  const triggerFileInput = () => {
    if (fileInputRef.current) {
      fileInputRef.current.click();
    }
  };

  return (
    <>
      <div
        ref={dropzoneRef}
        className={cn(
          "border-2 border-dashed rounded-lg p-6 text-center cursor-pointer transition-colors",
          "bg-muted/50 hover:bg-muted/70 flex flex-col items-center justify-center",
          "min-h-[200px]",
        )}
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
              <button
                className="underline cursor-pointer text-semibold"
                onClick={triggerFileInput}
              >
                Browse
              </button>
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
      {uploadingFiles.length > 0 && (
        <div className="mt-6 space-y-2">
          <p className="text-sm font-medium">Uploading files</p>
          <div className="space-y-3">
            {uploadingFiles.map((fileInfo, index) => (
              <div
                key={`${fileInfo.file.name}-${index}`}
                className="p-2 border rounded-md"
              >
                <div className="flex items-center justify-between mb-1">
                  <div className="flex items-center space-x-2 overflow-hidden">
                    <div className="relative flex size-8 shrink-0 overflow-hidden rounded-sm">
                      <div className="bg-muted flex size-full items-center justify-center rounded-sm">
                        <Icon
                          icon={getFileIcon(fileInfo.file.type)}
                          className="size-4"
                        />
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <span
                        className="text-sm font-medium truncate"
                        title={fileInfo.file.name}
                      >
                        {fileInfo.file.name}
                      </span>
                      <span className="text-xs text-muted-foreground">
                        {fileInfo.fileSize ||
                          formatFileSize(fileInfo.file.size)}
                      </span>
                    </div>
                  </div>

                  <div className="flex items-center space-x-2">
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={(e) => {
                              e.stopPropagation();
                              removeFile(index);
                            }}
                          >
                            <Icon icon={faTrash} className="size-4" />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>Remove file</TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <Progress
                    value={fileInfo.progress}
                    className="h-2"
                    indicatorClassName={cn({
                      "bg-green-500": fileInfo.status === "success",
                      "bg-red-500": fileInfo.status === "error",
                    })}
                  />
                  <span className="text-xs text-muted-foreground">{`${fileInfo.progress}%`}</span>
                </div>

                {fileInfo.status === "error" && fileInfo.error && (
                  <Alert variant="destructive" className="mt-2 py-2">
                    <Icon
                      icon={faExclamationTriangle}
                      className="size-4 mr-2"
                    />
                    <AlertDescription>{fileInfo.error}</AlertDescription>
                  </Alert>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
      <div className="flex justify-end mt-4 gap-2">
        <Button variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button
          onClick={uploadFiles}
          disabled={uploadingFiles.length === 0 || uploadFileMutation.isPending}
          isLoading={uploadFileMutation.isPending}
        >
          Continue
        </Button>
      </div>
    </>
  );
}

interface DocumentUploadSkeletonProps {
  isHovering: boolean;
}

function DocumentUploadSkeleton({ isHovering }: DocumentUploadSkeletonProps) {
  return (
    <div className="flex items-center justify-center relative h-20 w-24 bg-background dark:bg-background/50 rounded-sm size-full">
      {/* Left document */}
      <div
        className={cn(
          "absolute z-10 bg-foreground/20 rounded-md h-7 w-24 p-1 transform duration-400",
          isHovering
            ? "translate-x-[-24px] translate-y-[-8px] rotate-[-5deg]"
            : "translate-x-[-12px] translate-y-[-14px] rotate-0",
        )}
      >
        <div className="flex items-center gap-x-1">
          <div className="flex items-center justify-center bg-blue-500 rounded-sm size-5 p-1">
            <Icon icon={faFile} className="size-4 text-white" />
          </div>
          <div className="flex flex-col gap-0.5 size-full">
            <div className="w-full h-1.5 bg-muted-foreground rounded-md" />
            <div className="w-10 h-1.5 bg-muted-foreground/50 rounded-md" />
          </div>
        </div>
      </div>

      {/* Right document */}
      <div
        className={cn(
          "absolute z-10 bg-foreground/20 rounded-md h-7 w-24 p-1 transform duration-400",
          isHovering
            ? "translate-x-[24px] translate-y-[8px] rotate-[5deg]"
            : "translate-x-[12px] translate-y-[18px] rotate-0",
        )}
      >
        <div className="flex items-center gap-x-1">
          <div className="flex items-center justify-center bg-pink-500 rounded-sm size-5 p-1">
            <Icon icon={faImage} className="size-4 text-white" />
          </div>
          <div className="flex flex-col gap-0.5 size-full">
            <div className="w-full h-1.5 bg-muted-foreground rounded-md" />
            <div className="w-10 h-1.5 bg-muted-foreground/50 rounded-md" />
          </div>
        </div>
      </div>
    </div>
  );
}
