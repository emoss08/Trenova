import { cn } from "@/lib/utils";
import {
  forwardRef,
  useCallback,
  useImperativeHandle,
  useRef,
  useState,
} from "react";
import { Button } from "../ui/button";

export interface RejectedFile {
  file: File;
  reason: "size" | "type";
}

export interface DocumentUploadZoneHandle {
  openFilePicker: () => void;
}

interface DocumentUploadZoneProps {
  onFilesSelected: (files: File[]) => void;
  onFilesRejected?: (rejectedFiles: RejectedFile[]) => void;
  disabled?: boolean;
  accept?: string;
  maxSize?: number;
  className?: string;
}

const DEFAULT_ACCEPT =
  ".pdf,.jpg,.jpeg,.png,.webp,.gif,.doc,.docx,.xls,.xlsx,.txt,.csv";
const DEFAULT_MAX_SIZE = 50 * 1024 * 1024; // 50MB

// eslint-disable-next-line react-refresh/only-export-components
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / k ** i).toFixed(1))} ${sizes[i]}`;
}

export const DocumentUploadZone = forwardRef<
  DocumentUploadZoneHandle,
  DocumentUploadZoneProps
>(function DocumentUploadZone(
  {
    onFilesSelected,
    onFilesRejected,
    disabled = false,
    accept = DEFAULT_ACCEPT,
    maxSize = DEFAULT_MAX_SIZE,
    className,
  },
  ref,
) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [isDragging, setIsDragging] = useState(false);

  useImperativeHandle(ref, () => ({
    openFilePicker: () => {
      if (!disabled) {
        fileInputRef.current?.click();
      }
    },
  }));

  const handleFiles = useCallback(
    (fileList: FileList | null) => {
      if (!fileList || fileList.length === 0) return;

      const validFiles: File[] = [];
      const rejectedFiles: RejectedFile[] = [];

      Array.from(fileList).forEach((file) => {
        if (file.size > maxSize) {
          rejectedFiles.push({ file, reason: "size" });
        } else {
          validFiles.push(file);
        }
      });

      if (validFiles.length > 0) {
        onFilesSelected(validFiles);
      }

      if (rejectedFiles.length > 0 && onFilesRejected) {
        onFilesRejected(rejectedFiles);
      }
    },
    [onFilesSelected, onFilesRejected, maxSize],
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setIsDragging(false);
      if (disabled) return;
      handleFiles(e.dataTransfer.files);
    },
    [disabled, handleFiles],
  );

  const handleDragOver = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      if (!disabled) {
        setIsDragging(true);
      }
    },
    [disabled],
  );

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
  }, []);

  const handleClick = useCallback(() => {
    if (!disabled) {
      fileInputRef.current?.click();
    }
  }, [disabled]);

  return (
    <div
      onDrop={handleDrop}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      className={cn(
        "flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-6 transition-colors",
        "border-muted-foreground/25",
        isDragging && "border-primary bg-primary/5",
        disabled && "pointer-events-none opacity-50",
        className,
      )}
    >
      <div className="flex flex-col items-center gap-2">
        <Button
          type="button"
          variant="secondary"
          onClick={handleClick}
          disabled={disabled}
        >
          Select files
        </Button>
        <p className="text-sm text-muted-foreground">
          or drag and drop them here
        </p>
      </div>

      <ul className="mt-4 space-y-1 text-xs text-muted-foreground">
        <li>• Supported formats: PDF, images, Word, Excel, and text files</li>
        <li>• Maximum file size: 50 MB per file</li>
        <li>• You can upload multiple files at once</li>
      </ul>

      <input
        ref={fileInputRef}
        type="file"
        multiple
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
});
