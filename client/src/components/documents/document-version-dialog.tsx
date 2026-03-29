import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { Document } from "@/types/document";
import { HistoryIcon, RotateCcwIcon, UploadIcon } from "lucide-react";

interface DocumentVersionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  document: Document | null;
  versions: Document[];
  isLoading?: boolean;
  isRestoring?: boolean;
  onRestore: (document: Document) => void;
  onUploadNewVersion: (document: Document) => void;
}

function formatDate(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  return `${(bytes / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

export function DocumentVersionDialog({
  open,
  onOpenChange,
  document,
  versions,
  isLoading = false,
  isRestoring = false,
  onRestore,
  onUploadNewVersion,
}: DocumentVersionDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <HistoryIcon className="size-4" />
            Version History
          </DialogTitle>
          <DialogDescription>
            {document ? document.originalName : "Document versions"}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          {isLoading ? (
            <div className="rounded-lg border border-dashed p-6 text-sm text-muted-foreground">
              Loading versions...
            </div>
          ) : versions.length === 0 ? (
            <div className="rounded-lg border border-dashed p-6 text-sm text-muted-foreground">
              No versions found for this document.
            </div>
          ) : (
            versions.map((version) => (
              <div
                key={version.id}
                className="flex items-start justify-between gap-4 rounded-lg border p-4"
              >
                <div className="min-w-0 space-y-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-medium">{version.originalName}</span>
                    <Badge variant={version.isCurrentVersion ? "teal" : "secondary"}>
                      v{version.versionNumber}
                    </Badge>
                    {version.isCurrentVersion && <Badge variant="info">Current</Badge>}
                  </div>
                  <p className="text-sm text-muted-foreground">
                    {formatDate(version.createdAt)} • {formatFileSize(version.fileSize)}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {version.detectedKind || "Unclassified"} • Preview {version.previewStatus} • Text {version.contentStatus}
                  </p>
                </div>

                {!version.isCurrentVersion && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => onRestore(version)}
                    disabled={isRestoring}
                  >
                    <RotateCcwIcon className="mr-2 size-4" />
                    Restore
                  </Button>
                )}
              </div>
            ))
          )}
        </div>

        <DialogFooter>
          {document && (
            <Button
              variant="outline"
              onClick={() => onUploadNewVersion(document)}
            >
              <UploadIcon className="mr-2 size-4" />
              Upload New Version
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
