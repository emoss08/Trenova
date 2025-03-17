import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Progress } from "@/components/ui/progress";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { API_URL } from "@/constants/env";
import { cn } from "@/lib/utils";
import {
  faExclamationTriangle,
  faFile,
  faUpload,
  faXmark,
} from "@fortawesome/pro-regular-svg-icons";
import React, { useCallback, useRef, useState } from "react";
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
  showDocumentTypeSelection?: boolean;
  requireApproval?: boolean;
  className?: string;
}

export interface UploadingFile {
  file: File;
  progress: number;
  documentType: string;
  status: "uploading" | "success" | "error";
  error?: string;
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
  showDocumentTypeSelection = true,
  requireApproval = false,
  className,
}: FileUploadProps) {
  const [uploadingFiles, setUploadingFiles] = useState<UploadingFile[]>([]);
  const [isUploading, setIsUploading] = useState(false);
  const [defaultDocumentType, setDefaultDocumentType] = useState(
    documentTypes.length > 0 ? documentTypes[0].value : "Other",
  );
  const [description, setDescription] = useState("");
  const [isPublic, setIsPublic] = useState(false);
  const [needsApproval, setNeedsApproval] = useState(requireApproval);

  const fileInputRef = useRef<HTMLInputElement>(null);

  // Default document types if none provided
  const effectiveDocumentTypes =
    documentTypes.length > 0
      ? documentTypes
      : [
          { value: "License", label: "License" },
          { value: "Registration", label: "Registration" },
          { value: "Insurance", label: "Insurance" },
          { value: "Invoice", label: "Invoice" },
          { value: "ProofOfDelivery", label: "Proof of Delivery" },
          { value: "BillOfLading", label: "Bill of Lading" },
          { value: "Other", label: "Other" },
        ];

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
          documentType: defaultDocumentType,
          status: "uploading",
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
    [allowMultiple, defaultDocumentType, maxFileSizeMB],
  );

  const handleDragOver = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      e.stopPropagation();

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
          documentType: defaultDocumentType,
          status: "uploading",
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
    [allowMultiple, defaultDocumentType, maxFileSizeMB],
  );

  const removeFile = useCallback((index: number) => {
    setUploadingFiles((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const updateFileDocumentType = useCallback(
    (index: number, documentType: string) => {
      setUploadingFiles((prev) =>
        prev.map((file, i) => (i === index ? { ...file, documentType } : file)),
      );
    },
    [],
  );

  const uploadFile = async (fileInfo: UploadingFile, index: number) => {
    try {
      const formData = new FormData();
      formData.append("file", fileInfo.file);
      formData.append("resourceId", resourceId);
      formData.append("resourceType", resourceType);
      formData.append("documentType", fileInfo.documentType);
      formData.append("description", description);
      formData.append("isPublic", isPublic.toString());
      formData.append("requireApproval", needsApproval.toString());

      // Create XMLHttpRequest to track progress
      const xhr = new XMLHttpRequest();

      // Setup a promise to handle the async request
      const uploadPromise = new Promise<any>((resolve, reject) => {
        // Track upload progress
        xhr.upload.addEventListener("progress", (event) => {
          if (event.lengthComputable) {
            const progress = Math.round((event.loaded * 100) / event.total);
            setUploadingFiles((prev) =>
              prev.map((file, i) =>
                i === index ? { ...file, progress } : file,
              ),
            );
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
      });

      // Open and send the request
      xhr.open("POST", `${API_URL}/documents/upload/`);
      xhr.withCredentials = true;

      // Set any authentication headers if needed
      // xhr.setRequestHeader("Authorization", "Bearer your-token");
      xhr.setRequestHeader("X-Request-ID", crypto.randomUUID());

      xhr.send(formData);

      // Wait for the upload to complete
      const response = await uploadPromise;

      // Update file status
      setUploadingFiles((prev) =>
        prev.map((file, i) =>
          i === index ? { ...file, status: "success" } : file,
        ),
      );

      if (onUploadComplete) {
        onUploadComplete(response);
      }

      return response;
    } catch (error) {
      setUploadingFiles((prev) =>
        prev.map((file, i) =>
          i === index
            ? {
                ...file,
                status: "error",
                error: error instanceof Error ? error.message : "Upload failed",
              }
            : file,
        ),
      );

      if (onUploadError) {
        onUploadError(error);
      }

      throw error;
    }
  };

  // Then modify your uploadFiles function:
  const uploadFiles = async () => {
    if (uploadingFiles.length === 0 || isUploading) return;

    setIsUploading(true);

    try {
      // Upload each file in sequence to avoid overwhelming the server
      for (let i = 0; i < uploadingFiles.length; i++) {
        await uploadFile(uploadingFiles[i], i);
      }

      // Show success message
      toast.success(
        uploadingFiles.length > 1
          ? `Successfully uploaded ${uploadingFiles.length} documents`
          : "Document uploaded successfully",
      );
    } catch {
      toast.error("One or more files failed to upload");
    } finally {
      setIsUploading(false);
    }
  };

  const triggerFileInput = () => {
    if (fileInputRef.current) {
      fileInputRef.current.click();
    }
  };

  return (
    <Card className={cn("w-full", className)}>
      <CardHeader>
        <CardTitle className="text-xl font-semibold">
          <Icon icon={faUpload} className="mr-2" />
          Document Upload
        </CardTitle>
      </CardHeader>
      <CardContent>
        {/* File Drop Zone */}
        <div
          className={cn(
            "border-2 border-dashed rounded-lg p-6 text-center cursor-pointer transition-colors",
            "hover:bg-muted/50 flex flex-col items-center justify-center",
            "min-h-[200px]",
          )}
          onDragOver={handleDragOver}
          onDrop={handleDrop}
          onClick={triggerFileInput}
        >
          <Icon
            icon={faUpload}
            className="text-4xl text-muted-foreground mb-4"
          />
          <p className="text-lg font-medium">
            Drop files here or click to browse
          </p>
          <p className="text-sm text-muted-foreground mt-2">
            {allowMultiple
              ? "You can upload multiple files"
              : "You can upload one file at a time"}
          </p>
          <p className="text-xs text-muted-foreground mt-1">
            Maximum file size: {maxFileSizeMB}MB
          </p>
          <input
            type="file"
            ref={fileInputRef}
            className="hidden"
            accept={acceptedFileTypes}
            multiple={allowMultiple}
            onChange={handleFileSelect}
          />
        </div>

        {/* Document Settings */}
        <div className="mt-6 space-y-4">
          {showDocumentTypeSelection && (
            <div className="space-y-2">
              <Label htmlFor="documentType">Default Document Type</Label>
              <Select
                value={defaultDocumentType}
                onValueChange={setDefaultDocumentType}
              >
                <SelectTrigger id="documentType">
                  <SelectValue placeholder="Select document type" />
                </SelectTrigger>
                <SelectContent>
                  {effectiveDocumentTypes.map((type) => (
                    <SelectItem key={type.value} value={type.value}>
                      {type.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          <div className="space-y-2">
            <Label htmlFor="description">Description (Optional)</Label>
            <Input
              id="description"
              placeholder="Enter document description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>

          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <Switch
                id="isPublic"
                checked={isPublic}
                onCheckedChange={setIsPublic}
              />
              <Label htmlFor="isPublic">Publicly accessible</Label>
            </div>

            <div className="flex items-center space-x-2">
              <Switch
                id="needsApproval"
                checked={needsApproval}
                onCheckedChange={setNeedsApproval}
              />
              <Label htmlFor="needsApproval">Requires approval</Label>
            </div>
          </div>
        </div>

        {/* File List */}
        {uploadingFiles.length > 0 && (
          <div className="mt-6 space-y-4">
            <h3 className="font-medium">
              Selected Files ({uploadingFiles.length})
            </h3>
            <div className="space-y-3">
              {uploadingFiles.map((fileInfo, index) => (
                <div
                  key={`${fileInfo.file.name}-${index}`}
                  className="p-3 border rounded-md"
                >
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center space-x-2 overflow-hidden">
                      <Icon icon={faFile} className="text-blue-500" />
                      <span
                        className="font-medium truncate"
                        title={fileInfo.file.name}
                      >
                        {fileInfo.file.name}
                      </span>
                      <Badge
                        variant={
                          fileInfo.status === "success"
                            ? "active"
                            : fileInfo.status === "error"
                              ? "inactive"
                              : "secondary"
                        }
                      >
                        {fileInfo.status === "uploading" &&
                          `${fileInfo.progress}%`}
                        {fileInfo.status === "success" && "Complete"}
                        {fileInfo.status === "error" && "Failed"}
                      </Badge>
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
                              disabled={
                                isUploading && fileInfo.status === "uploading"
                              }
                            >
                              <Icon icon={faXmark} className="h-4 w-4" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Remove file</TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </div>
                  </div>

                  {fileInfo.status === "uploading" && (
                    <Progress value={fileInfo.progress} className="h-2" />
                  )}

                  {fileInfo.status === "error" && fileInfo.error && (
                    <Alert variant="destructive" className="mt-2 py-2">
                      <Icon
                        icon={faExclamationTriangle}
                        className="h-4 w-4 mr-2"
                      />
                      <AlertDescription>{fileInfo.error}</AlertDescription>
                    </Alert>
                  )}

                  {showDocumentTypeSelection && (
                    <div className="mt-2">
                      <Select
                        value={fileInfo.documentType}
                        onValueChange={(value) =>
                          updateFileDocumentType(index, value)
                        }
                        disabled={isUploading}
                      >
                        <SelectTrigger id={`docType-${index}`} className="h-8">
                          <SelectValue placeholder="Document type" />
                        </SelectTrigger>
                        <SelectContent>
                          {effectiveDocumentTypes.map((type) => (
                            <SelectItem key={type.value} value={type.value}>
                              {type.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}
      </CardContent>

      <CardFooter className="flex justify-end">
        <Button
          onClick={uploadFiles}
          disabled={uploadingFiles.length === 0 || isUploading}
          className="w-full sm:w-auto"
        >
          {isUploading ? (
            <>Uploading...</>
          ) : (
            <>
              <Icon icon={faUpload} className="mr-2" />
              Upload{" "}
              {uploadingFiles.length > 0 ? `(${uploadingFiles.length})` : ""}
            </>
          )}
        </Button>
      </CardFooter>
    </Card>
  );
}
