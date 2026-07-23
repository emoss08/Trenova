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
import { Separator } from "@/components/ui/separator";
import { formatToUserTimezone } from "@/lib/date";
import type { Document } from "@/types/document";
import { ChevronDownIcon, HistoryIcon, RotateCcwIcon, UploadIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { formatFileSize } from "./document-upload-zone";
import { RestoreVersionDialog } from "./restore-version-dialog";

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

function PreviewBadge({ status }: { status: string }) {
  const variant = status === "Ready" ? "teal" : status === "Pending" ? "warning" : "outline";
  return (
    <Badge variant={variant} className="h-5 px-1.5 py-0 text-[10px]">
      Preview {status}
    </Badge>
  );
}

function ContentBadge({ status }: { status: string }) {
  const variant =
    status === "Indexed" || status === "Extracted"
      ? "teal"
      : status === "Extracting" || status === "Pending"
        ? "warning"
        : "outline";
  return (
    <Badge variant={variant} className="h-5 px-1.5 py-0 text-[10px]">
      Text {status}
    </Badge>
  );
}

interface CompareField {
  label: string;
  selected: string;
  current: string;
}

function buildComparison(selected: Document, current: Document): CompareField[] {
  const fields: CompareField[] = [];

  if (selected.originalName !== current.originalName) {
    fields.push({
      label: "File Name",
      selected: selected.originalName,
      current: current.originalName,
    });
  }
  if (selected.fileSize !== current.fileSize) {
    fields.push({
      label: "File Size",
      selected: formatFileSize(selected.fileSize),
      current: formatFileSize(current.fileSize),
    });
  }
  if (selected.fileType !== current.fileType) {
    fields.push({
      label: "File Type",
      selected: selected.fileType,
      current: current.fileType,
    });
  }
  if (selected.detectedKind !== current.detectedKind) {
    fields.push({
      label: "Detected Kind",
      selected: selected.detectedKind || "Unclassified",
      current: current.detectedKind || "Unclassified",
    });
  }
  if (selected.previewStatus !== current.previewStatus) {
    fields.push({
      label: "Preview",
      selected: selected.previewStatus,
      current: current.previewStatus,
    });
  }
  if (selected.contentStatus !== current.contentStatus) {
    fields.push({
      label: "Text Extraction",
      selected: selected.contentStatus,
      current: current.contentStatus,
    });
  }

  return fields;
}

function VersionCompare({ selected, current }: { selected: Document; current: Document }) {
  const fields = buildComparison(selected, current);

  if (fields.length === 0) {
    return (
      <p className="text-xs text-muted-foreground italic">
        No metadata differences from the current version.
      </p>
    );
  }

  return (
    <table className="w-full text-xs">
      <thead>
        <tr className="text-muted-foreground">
          <th className="pb-1.5 text-left font-medium" />
          <th className="pb-1.5 text-left font-medium">This Version</th>
          <th className="pb-1.5 text-left font-medium">Current</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {fields.map((f) => (
          <tr key={f.label}>
            <td className="py-1 pr-3 text-muted-foreground">{f.label}</td>
            <td className="py-1 pr-3">{f.selected}</td>
            <td className="py-1 text-muted-foreground">{f.current}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function VersionStatusBadges({ version }: { version: Document }) {
  return (
    <div className="flex flex-wrap gap-1">
      {version.detectedKind && version.detectedKind !== "Other" && (
        <Badge variant="info" className="h-5 px-1.5 py-0 text-[10px]">
          {version.detectedKind}
        </Badge>
      )}
      <PreviewBadge status={version.previewStatus} />
      <ContentBadge status={version.contentStatus} />
    </div>
  );
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
  const [expandedVersionId, setExpandedVersionId] = useState<string | null>(null);
  const [restoreTarget, setRestoreTarget] = useState<Document | null>(null);

  const currentVersion = useMemo(
    () => versions.find((v) => v.isCurrentVersion) ?? null,
    [versions],
  );
  const previousVersions = useMemo(
    () =>
      versions.filter((v) => !v.isCurrentVersion).sort((a, b) => b.versionNumber - a.versionNumber),
    [versions],
  );

  const handleConfirmRestore = () => {
    if (restoreTarget) {
      onRestore(restoreTarget);
      setRestoreTarget(null);
    }
  };

  return (
    <>
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
              <>
                {currentVersion && (
                  <div className="rounded-lg border border-primary/20 bg-primary/5 p-4">
                    <div className="space-y-1.5">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-medium">{currentVersion.originalName}</span>
                        <Badge variant="teal">v{currentVersion.versionNumber}</Badge>
                        <Badge variant="info">Current</Badge>
                      </div>
                      <p className="text-sm text-muted-foreground">
                        {formatToUserTimezone(currentVersion.createdAt)} &middot;{" "}
                        {formatFileSize(currentVersion.fileSize)}
                      </p>
                      <VersionStatusBadges version={currentVersion} />
                    </div>
                  </div>
                )}

                {previousVersions.length > 0 && (
                  <>
                    <div className="flex items-center gap-3">
                      <Separator className="flex-1" />
                      <span className="text-xs font-medium text-muted-foreground">
                        Previous Versions
                      </span>
                      <Separator className="flex-1" />
                    </div>

                    {previousVersions.map((version) => {
                      const isExpanded = expandedVersionId === version.id;
                      return (
                        <div key={version.id} className="rounded-lg border">
                          <div className="flex items-start justify-between gap-4 p-4">
                            <button
                              type="button"
                              className="min-w-0 flex-1 cursor-pointer space-y-1.5 text-left"
                              onClick={() => setExpandedVersionId(isExpanded ? null : version.id)}
                            >
                              <div className="flex flex-wrap items-center gap-2">
                                <span className="font-medium">{version.originalName}</span>
                                <Badge variant="secondary">v{version.versionNumber}</Badge>
                                <ChevronDownIcon
                                  className={`size-3.5 text-muted-foreground transition-transform ${isExpanded ? "rotate-180" : ""}`}
                                />
                              </div>
                              <p className="text-sm text-muted-foreground">
                                {formatToUserTimezone(version.createdAt)} &middot;{" "}
                                {formatFileSize(version.fileSize)}
                              </p>
                              <VersionStatusBadges version={version} />
                            </button>

                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => setRestoreTarget(version)}
                              disabled={isRestoring}
                            >
                              <RotateCcwIcon className="mr-1.5 size-3.5" />
                              Restore
                            </Button>
                          </div>

                          {isExpanded && currentVersion && (
                            <div className="border-t bg-muted/30 px-4 py-3">
                              <VersionCompare selected={version} current={currentVersion} />
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </>
                )}
              </>
            )}
          </div>

          <DialogFooter>
            {document && (
              <Button variant="outline" onClick={() => onUploadNewVersion(document)}>
                <UploadIcon className="size-4" />
                Upload New Version
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <RestoreVersionDialog
        open={!!restoreTarget}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) setRestoreTarget(null);
        }}
        versionToRestore={restoreTarget}
        currentVersion={currentVersion}
        isRestoring={isRestoring}
        onConfirm={handleConfirmRestore}
      />
    </>
  );
}
