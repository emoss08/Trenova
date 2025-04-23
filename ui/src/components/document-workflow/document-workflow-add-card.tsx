import { cn } from "@/lib/utils";
import { useCallback, useEffect, useState } from "react";
import { DocumentUploadSkeleton } from "../file-uploader/file-upload-skeleton";

export function AddDocumentCard({
  onUpload,
  isUploading,
  handleFileUpload,
}: {
  onUpload: () => void;
  isUploading: boolean;
  handleFileUpload: (file: FileList) => void;
}) {
  const [isHovering, setIsHovering] = useState(false);
  const [isDragging, setIsDragging] = useState(false);

  // Set up global event listeners to prevent default browser behavior
  useEffect(() => {
    const handleGlobalDragOver = (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
    };

    const handleGlobalDrop = (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
    };

    // Add event listeners to the document
    document.addEventListener("dragover", handleGlobalDragOver);
    document.addEventListener("drop", handleGlobalDrop);

    // Clean up
    return () => {
      document.removeEventListener("dragover", handleGlobalDragOver);
      document.removeEventListener("drop", handleGlobalDrop);
    };
  }, []);

  // Track if we're in the middle of a drag operation
  useEffect(() => {
    const handleDragEnter = () => {
      setIsDragging(true);
    };

    const handleDragEnd = () => {
      setIsDragging(false);
      setIsHovering(false);
    };

    document.addEventListener("dragenter", handleDragEnter);
    document.addEventListener("dragend", handleDragEnd);
    document.addEventListener("drop", handleDragEnd);

    return () => {
      document.removeEventListener("dragenter", handleDragEnter);
      document.removeEventListener("dragend", handleDragEnd);
      document.removeEventListener("drop", handleDragEnd);
    };
  }, []);

  // * Memoize event handlers
  const handleDragOver = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsHovering(true);
  }, []);

  const handleDragEnter = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsHovering(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();

    // Check if we're leaving the component entirely
    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX;
    const y = e.clientY;

    // If the pointer is outside the bounds of our component
    if (x < rect.left || x >= rect.right || y < rect.top || y >= rect.bottom) {
      setIsHovering(false);
    }
  }, []);

  const handleMouseEnter = useCallback(() => {
    setIsHovering(true);
  }, []);

  const handleMouseLeave = useCallback(() => {
    if (!isDragging) {
      setIsHovering(false);
    }
  }, [isDragging]);

  const handleDrop = useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      e.stopPropagation();
      setIsHovering(false);
      setIsDragging(false);

      // Process the dropped files
      if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
        handleFileUpload(e.dataTransfer.files);
      }
    },
    [handleFileUpload],
  );

  return (
    <div
      className={cn(
        "flex justify-center items-center border border-dashed rounded-md overflow-hidden transition-all cursor-pointer",
        isHovering ? "bg-muted" : "border-border hover:bg-muted",
      )}
      onClick={onUpload}
      onDragOver={handleDragOver}
      onDragEnter={handleDragEnter}
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
