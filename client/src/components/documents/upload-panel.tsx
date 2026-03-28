import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import type { UploadState } from "@/types/upload";
import {
  AlertCircleIcon,
  CheckCircleIcon,
  ChevronDownIcon,
  ChevronUpIcon,
  Loader2Icon,
  PlusIcon,
  RotateCcwIcon,
  ServerCrashIcon,
  WifiOffIcon,
  XIcon,
} from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import { useCallback, useMemo, useRef, useState } from "react";
import { DocumentFileTypeIcon } from "./document-file-type-icon";
import { type RejectedFile } from "./document-upload-zone";

type UploadFilter = "all" | "completed" | "failed";

interface UploadPanelProps {
  isOpen: boolean;
  onClose: () => void;
  uploads: UploadState[];
  onFilesSelected: (files: File[]) => void;
  onFilesRejected?: (rejectedFiles: RejectedFile[]) => void;
  onCancel?: (id: string) => void;
  onRetry?: (id: string) => void;
  onRemove?: (id: string) => void;
  onClearCompleted?: () => void;
  disabled?: boolean;
  title?: string;
  accept?: string;
  maxFileSize?: number;
  multiple?: boolean;
  supportedFormatsLabel?: string;
  maxFileSizeLabel?: string;
}

const MAX_RETRIES = 3;

const MAX_FILE_SIZE = 50 * 1024 * 1024;
const ACCEPTED_TYPES =
  ".pdf,.jpg,.jpeg,.png,.webp,.gif,.doc,.docx,.xls,.xlsx,.txt,.csv";

function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / k ** i).toFixed(1))} ${sizes[i]}`;
}

function isAcceptedFileType(file: File, accept: string): boolean {
  if (!accept.trim()) {
    return true;
  }

  const acceptedTypes = accept
    .split(",")
    .map((entry) => entry.trim().toLowerCase())
    .filter(Boolean);

  if (acceptedTypes.length === 0) {
    return true;
  }

  const lowerName = file.name.toLowerCase();
  const lowerType = file.type.toLowerCase();

  return acceptedTypes.some((entry) => {
    if (entry.startsWith(".")) {
      return lowerName.endsWith(entry);
    }

    if (entry.endsWith("/*")) {
      const baseType = entry.slice(0, -1);
      return lowerType.startsWith(baseType);
    }

    return lowerType === entry;
  });
}

function getErrorIcon(errorType?: string) {
  switch (errorType) {
    case "network":
      return <WifiOffIcon className="size-4 text-orange-400" />;
    case "server":
      return <ServerCrashIcon className="size-4 text-red-400" />;
    default:
      return <AlertCircleIcon className="size-4 text-red-400" />;
  }
}

function getErrorMessage(
  error?: string,
  errorType?: string,
  retryCount?: number,
) {
  const baseMessage = error || "Failed";
  const retryText =
    retryCount && retryCount > 0
      ? ` (attempt ${retryCount}/${MAX_RETRIES})`
      : "";

  if (errorType === "network") {
    return `Network error${retryText}`;
  }
  if (errorType === "validation") {
    return `${baseMessage}${retryText}`;
  }
  if (errorType === "server") {
    return `Server error${retryText}`;
  }
  return `${baseMessage}${retryText}`;
}

function UploadItem({
  upload,
  onCancel,
  onRetry,
  onRemove,
}: {
  upload: UploadState;
  onCancel?: (id: string) => void;
  onRetry?: (id: string) => void;
  onRemove?: (id: string) => void;
}) {
  const { id, file, progress, status, error, errorType, retryCount } = upload;
  const canRetry = (retryCount ?? 0) < MAX_RETRIES;

  return (
    <div className="flex items-center gap-3 rounded-md bg-transparent p-1">
      <DocumentFileTypeIcon
        fileType={file.type || "application/octet-stream"}
        fileName={file.name}
        size="sm"
      />

      <div className="min-w-0 flex-1">
        <p
          className="truncate text-sm font-medium text-foreground"
          title={file.name}
        >
          {file.name}
        </p>
        <div className="flex items-center gap-2">
          {status === "uploading" && (
            <>
              <div className="h-1 flex-1 overflow-hidden rounded-full bg-blue-500/20">
                <div
                  className="h-full bg-blue-500 transition-all duration-300"
                  style={{ width: `${progress}%` }}
                />
              </div>
              <span className="text-xs text-muted-foreground">{progress}%</span>
            </>
          )}
          {status === "processing" && (
            <span className="text-xs text-muted-foreground">
              Compressing...
            </span>
          )}
          {status === "uploaded" && (
            <span className="text-xs text-muted-foreground">Uploaded, waiting...</span>
          )}
          {status === "verifying" && (
            <span className="text-xs text-muted-foreground">Verifying...</span>
          )}
          {status === "paused" && (
            <span className="text-xs text-muted-foreground">Paused</span>
          )}
          {status === "retrying" && (
            <span className="text-xs text-muted-foreground">Retrying...</span>
          )}
          {status === "completing" && (
            <span className="text-xs text-muted-foreground">Finalizing...</span>
          )}
          {status === "quarantined" && (
            <span className="text-xs text-red-400">Quarantined</span>
          )}
          {status === "pending" && (
            <span className="text-xs text-muted-foreground">
              {retryCount && retryCount > 0
                ? `Retrying (${retryCount})...`
                : "Waiting..."}
            </span>
          )}
          {status === "success" && (
            <span className="text-xs text-muted-foreground">
              {formatFileSize(file.size)}
            </span>
          )}
          {status === "error" && (
            <span className="truncate text-xs text-red-400">
              {getErrorMessage(error, errorType, retryCount)}
            </span>
          )}
        </div>
      </div>

      <div className="flex shrink-0 items-center gap-1">
        {status === "uploading" && (
          <Loader2Icon className="size-4 animate-spin text-blue-400" />
        )}
        {status === "processing" && (
          <Loader2Icon className="size-4 animate-spin text-blue-400" />
        )}
        {status === "uploaded" && (
          <Loader2Icon className="size-4 animate-spin text-blue-400" />
        )}
        {status === "verifying" && (
          <Loader2Icon className="size-4 animate-spin text-blue-400" />
        )}
        {status === "paused" && (
          <AlertCircleIcon className="size-4 text-amber-400" />
        )}
        {status === "retrying" && (
          <Loader2Icon className="size-4 animate-spin text-blue-400" />
        )}
        {status === "completing" && (
          <Loader2Icon className="size-4 animate-spin text-blue-400" />
        )}
        {status === "success" && (
          <CheckCircleIcon className="size-4 text-green-400" />
        )}
        {(status === "error" || status === "paused" || status === "quarantined") && (
          <>
            {status === "error" ? getErrorIcon(errorType) : null}
            {onRetry && canRetry && (
              <Button
                variant="ghost"
                size="icon-xs"
                onClick={() => onRetry(id)}
                className="text-muted-foreground hover:bg-muted hover:text-foreground"
                title="Retry upload"
              >
                <RotateCcwIcon className="size-3.5" />
              </Button>
            )}
            {onRemove && (
              <Button
                variant="ghost"
                size="icon-xs"
                onClick={() => onRemove(id)}
                className="text-muted-foreground hover:bg-muted hover:text-foreground"
                title="Remove"
              >
                <XIcon className="size-3.5" />
              </Button>
            )}
          </>
        )}
        {(status === "pending" ||
          status === "processing" ||
          status === "uploading" ||
          status === "uploaded" ||
          status === "verifying" ||
          status === "retrying" ||
          status === "completing") &&
          onCancel && (
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={() => onCancel(id)}
              className="text-muted-foreground hover:bg-muted hover:text-foreground"
            >
              <XIcon className="size-3.5" />
            </Button>
          )}
      </div>
    </div>
  );
}

function FullDropzone({
  onFilesSelected,
  onFilesRejected,
  disabled,
  accept,
  maxFileSize,
  multiple,
  supportedFormatsLabel,
  maxFileSizeLabel,
}: {
  onFilesSelected: (files: File[]) => void;
  onFilesRejected?: (rejectedFiles: RejectedFile[]) => void;
  disabled?: boolean;
  accept: string;
  maxFileSize: number;
  multiple: boolean;
  supportedFormatsLabel: string;
  maxFileSizeLabel: string;
}) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [isDragging, setIsDragging] = useState(false);

  const handleFiles = useCallback(
    (fileList: FileList | null) => {
      if (!fileList || fileList.length === 0) return;

      const validFiles: File[] = [];
      const rejectedFiles: RejectedFile[] = [];

      Array.from(fileList).forEach((file) => {
        if (!isAcceptedFileType(file, accept)) {
          rejectedFiles.push({ file, reason: "type" });
          return;
        }

        if (file.size > maxFileSize) {
          rejectedFiles.push({ file, reason: "size" });
        } else {
          validFiles.push(file);
        }
      });

      const uploadableFiles = multiple ? validFiles : validFiles.slice(0, 1);

      if (uploadableFiles.length > 0) {
        onFilesSelected(uploadableFiles);
      }

      if (rejectedFiles.length > 0 && onFilesRejected) {
        onFilesRejected(rejectedFiles);
      }
    },
    [accept, maxFileSize, multiple, onFilesSelected, onFilesRejected],
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

  return (
    <div
      onDrop={handleDrop}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      className={cn(
        "flex flex-col items-center justify-center rounded-md p-8 transition-colors",
        "border-muted-foreground/25",
        isDragging && "border-primary bg-primary/5",
        disabled && "pointer-events-none opacity-50",
      )}
    >
      <Button
        type="button"
        variant="secondary"
        onClick={() => fileInputRef.current?.click()}
        disabled={disabled}
      >
        Select files
      </Button>
      <p className="mt-2 text-sm text-muted-foreground">
        or drag and drop them here
      </p>
      <p className="mt-3 text-xs text-muted-foreground">
        {supportedFormatsLabel}, up to {maxFileSizeLabel}
      </p>
      <input
        ref={fileInputRef}
        type="file"
        multiple={multiple}
        className="hidden"
        onChange={(e) => {
          handleFiles(e.target.files);
          e.target.value = "";
        }}
        disabled={disabled}
        accept={accept}
      />
    </div>
  );
}

export function UploadPanel({
  isOpen,
  onClose,
  uploads,
  onFilesSelected,
  onFilesRejected,
  onCancel,
  onRetry,
  onRemove,
  onClearCompleted,
  disabled,
  title = "Upload Documents",
  accept = ACCEPTED_TYPES,
  maxFileSize = MAX_FILE_SIZE,
  multiple = true,
  supportedFormatsLabel = "PDF, images, and documents",
  maxFileSizeLabel = "50 MB",
}: UploadPanelProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [isCollapsed, setIsCollapsed] = useState(false);
  const [filter, setFilter] = useState<UploadFilter>("all");
  const [isDraggingOver, setIsDraggingOver] = useState(false);

  const counts = useMemo(() => {
    return {
      all: uploads.length,
      completed: uploads.filter((u) => u.status === "success").length,
      failed: uploads.filter((u) => u.status === "error").length,
    };
  }, [uploads]);

  const filteredUploads = useMemo(() => {
    switch (filter) {
      case "completed":
        return uploads.filter((u) => u.status === "success");
      case "failed":
        return uploads.filter((u) => u.status === "error");
      default:
        return uploads;
    }
  }, [uploads, filter]);

  const activeCount = uploads.filter(
    (u) =>
      u.status === "uploading" ||
      u.status === "pending" ||
      u.status === "processing",
  ).length;

  const hasUploads = uploads.length > 0;
  const isEffectivelyCollapsed = hasUploads ? isCollapsed : false;

  const handleFiles = useCallback(
    (fileList: FileList | null) => {
      if (!fileList || fileList.length === 0) return;

      const validFiles: File[] = [];
      const rejectedFiles: RejectedFile[] = [];

      Array.from(fileList).forEach((file) => {
        if (!isAcceptedFileType(file, accept)) {
          rejectedFiles.push({ file, reason: "type" });
          return;
        }

        if (file.size > maxFileSize) {
          rejectedFiles.push({ file, reason: "size" });
        } else {
          validFiles.push(file);
        }
      });

      const uploadableFiles = multiple ? validFiles : validFiles.slice(0, 1);

      if (uploadableFiles.length > 0) {
        onFilesSelected(uploadableFiles);
      }

      if (rejectedFiles.length > 0 && onFilesRejected) {
        onFilesRejected(rejectedFiles);
      }
    },
    [accept, maxFileSize, multiple, onFilesSelected, onFilesRejected],
  );

  const handleDragOver = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      if (!disabled && hasUploads && e.dataTransfer.types.includes("Files")) {
        setIsDraggingOver(true);
      }
    },
    [disabled, hasUploads],
  );

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX;
    const y = e.clientY;
    if (x < rect.left || x > rect.right || y < rect.top || y > rect.bottom) {
      setIsDraggingOver(false);
    }
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setIsDraggingOver(false);
      if (disabled) return;
      handleFiles(e.dataTransfer.files);
    },
    [disabled, handleFiles],
  );

  const tabs: { value: UploadFilter; label: string; count: number }[] = [
    { value: "all", label: "All", count: counts.all },
    { value: "completed", label: "Completed", count: counts.completed },
    { value: "failed", label: "Failed", count: counts.failed },
  ];

  return (
    <AnimatePresence>
      {isOpen && (
        <m.div
          initial={{ opacity: 0, y: 20, scale: 0.95 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          exit={{ opacity: 0, y: 20, scale: 0.95 }}
          transition={{
            type: "spring",
            stiffness: 400,
            damping: 30,
            bounce: 0,
          }}
          className="dark fixed right-3 bottom-3 z-50 w-[400px] overflow-hidden rounded-lg border bg-popover shadow-2xl"
          onDragOver={hasUploads ? handleDragOver : undefined}
          onDragLeave={hasUploads ? handleDragLeave : undefined}
          onDrop={hasUploads ? handleDrop : undefined}
        >
          {isDraggingOver && (
            <div className="absolute inset-0 z-10 flex items-center justify-center rounded-lg border-2 border-dashed border-primary bg-primary/10">
              <p className="text-sm font-medium text-primary">
                Drop files to add
              </p>
            </div>
          )}

          <div className="flex items-center justify-between border-b border-border px-3 py-2">
            <h3 className="text-sm font-medium text-foreground">{title}</h3>
            <div className="flex items-center gap-1">
              {counts.completed > 0 && onClearCompleted && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={onClearCompleted}
                  className="h-7 px-2 text-xs text-muted-foreground hover:bg-muted hover:text-foreground"
                >
                  Clear
                </Button>
              )}
              {hasUploads && (
                <Button
                  variant="ghost"
                  size="icon-xs"
                  onClick={() => setIsCollapsed(!isCollapsed)}
                  className="text-muted-foreground hover:bg-muted hover:text-foreground"
                >
                  {isCollapsed ? (
                    <ChevronUpIcon className="size-4" />
                  ) : (
                    <ChevronDownIcon className="size-4" />
                  )}
                </Button>
              )}
              <Button
                variant="ghost"
                size="icon-xs"
                onClick={onClose}
                className="text-muted-foreground hover:bg-muted hover:text-foreground"
              >
                <XIcon className="size-4" />
              </Button>
            </div>
          </div>

          <AnimatePresence initial={false}>
            {!isEffectivelyCollapsed && (
              <m.div
                initial={{ height: 0, opacity: 0 }}
                animate={{ height: "auto", opacity: 1 }}
                exit={{ height: 0, opacity: 0 }}
                transition={{
                  type: "spring",
                  stiffness: 300,
                  damping: 30,
                  bounce: 0,
                }}
                style={{ overflow: "hidden" }}
              >
                {hasUploads ? (
                  <>
                    <div className="flex gap-1 px-3 pt-2">
                      {tabs.map((tab) => (
                        <button
                          key={tab.value}
                          onClick={() => setFilter(tab.value)}
                          className={cn(
                            "rounded px-2 py-1 text-xs font-medium transition-colors",
                            filter === tab.value
                              ? "bg-primary text-primary-foreground"
                              : "text-muted-foreground hover:bg-muted hover:text-foreground",
                          )}
                        >
                          {tab.label}
                          {tab.count > 0 && (
                            <span className="ml-1">({tab.count})</span>
                          )}
                        </button>
                      ))}
                    </div>
                    <ScrollArea className="mt-2 h-56 px-3">
                      <div className="space-y-1.5">
                        {filteredUploads.length === 0 ? (
                          <p className="py-4 text-center text-xs text-muted-foreground">
                            No uploads in this category
                          </p>
                        ) : (
                          filteredUploads.map((upload) => (
                            <UploadItem
                              key={upload.id}
                              upload={upload}
                              onCancel={onCancel}
                              onRetry={onRetry}
                              onRemove={onRemove}
                            />
                          ))
                        )}
                      </div>
                    </ScrollArea>
                  </>
                ) : (
                  <FullDropzone
                    onFilesSelected={onFilesSelected}
                    onFilesRejected={onFilesRejected}
                    disabled={disabled}
                    accept={accept}
                    maxFileSize={maxFileSize}
                    multiple={multiple}
                    supportedFormatsLabel={supportedFormatsLabel}
                    maxFileSizeLabel={maxFileSizeLabel}
                  />
                )}
              </m.div>
            )}
          </AnimatePresence>
          {hasUploads && (
            <div className="flex items-center justify-between border-t border-border bg-muted/50 px-3 py-2">
              <span className="text-xs text-muted-foreground">
                {activeCount > 0
                  ? `Uploading ${activeCount} file${activeCount > 1 ? "s" : ""}...`
                  : `${counts.completed} completed`}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => fileInputRef.current?.click()}
                disabled={disabled}
                className="text-foreground"
              >
                <PlusIcon className="size-3.5" />
                Add
              </Button>
              <input
                ref={fileInputRef}
                type="file"
                multiple={multiple}
                className="hidden"
                onChange={(e) => {
                  handleFiles(e.target.files);
                  e.target.value = "";
                }}
                disabled={disabled}
                accept={accept}
              />
            </div>
          )}
        </m.div>
      )}
    </AnimatePresence>
  );
}
