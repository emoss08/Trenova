import { DocumentUploadZone, type RejectedFile } from "@/components/documents/document-upload-zone";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { TextShimmer } from "@/components/ui/text-shimmer";
import { m } from "motion/react";
import { AlertCircleIcon, LoaderCircleIcon } from "lucide-react";
import { useCallback } from "react";
import { toast } from "sonner";

type UploadPhaseProps = {
  currentUpload: {
    id: string;
    file: File;
    status: string;
    progress: number;
    error?: string;
  } | null;
  onFilesSelected: (files: File[]) => void;
  onRetry: (id: string) => void;
  onCancel: (id: string) => void;
  onRemove: (id: string) => void;
};

export function UploadPhase({
  currentUpload,
  onFilesSelected,
  onRetry,
  onCancel,
  onRemove,
}: UploadPhaseProps) {
  const handleRejected = useCallback((rejectedFiles: RejectedFile[]) => {
    for (const { file, reason } of rejectedFiles) {
      toast.error(
        reason === "size"
          ? `${file.name} exceeds 50 MB limit.`
          : `${file.name} is not a supported format.`,
      );
    }
  }, []);

  return (
    <div className="flex flex-1 flex-col items-center justify-center p-8">
      <m.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="w-full max-w-md space-y-6"
      >
        <div className="space-y-1 text-center">
          <h2 className="text-base font-medium">Upload rate confirmation</h2>
          <p className="text-sm text-muted-foreground">
            PDF or image. Shipment details are extracted automatically.
          </p>
        </div>

        <DocumentUploadZone
          onFilesSelected={onFilesSelected}
          onFilesRejected={handleRejected}
          disabled={!!currentUpload && currentUpload.status !== "error"}
          accept=".pdf,.jpg,.jpeg,.png,.webp"
        />

        {currentUpload && (
          <m.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            className="space-y-3 rounded-lg border bg-background p-3"
          >
            <div className="flex items-center justify-between gap-3">
              <div className="min-w-0">
                <div className="truncate text-sm">{currentUpload.file.name}</div>
                {currentUpload.status === "uploading" && (
                  <TextShimmer as="span" className="text-xs" duration={1.5}>
                    Uploading
                  </TextShimmer>
                )}
                {currentUpload.status === "success" && (
                  <span className="text-xs text-muted-foreground">Complete</span>
                )}
              </div>
              {currentUpload.status === "uploading" && (
                <LoaderCircleIcon className="size-3.5 shrink-0 animate-spin text-muted-foreground" />
              )}
            </div>

            <Progress
              value={currentUpload.progress}
              variant={currentUpload.status === "error" ? "error" : "default"}
              className="h-1"
            />

            {currentUpload.error && (
              <div className="flex items-start gap-2 text-xs text-destructive">
                <AlertCircleIcon className="mt-0.5 size-3 shrink-0" />
                <span>{currentUpload.error}</span>
              </div>
            )}

            {currentUpload.status === "error" && (
              <div className="flex gap-2">
                <Button variant="outline" size="sm" onClick={() => onRetry(currentUpload.id)}>
                  Retry
                </Button>
                <Button variant="ghost" size="sm" onClick={() => onRemove(currentUpload.id)}>
                  Remove
                </Button>
              </div>
            )}
            {currentUpload.status !== "success" && currentUpload.status !== "error" && (
              <Button variant="ghost" size="sm" onClick={() => onCancel(currentUpload.id)}>
                Cancel
              </Button>
            )}
          </m.div>
        )}
      </m.div>
    </div>
  );
}
