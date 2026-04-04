import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import type { Document } from "@/types/document";
import { Loader2Icon, RotateCcwIcon } from "lucide-react";
import { formatFileSize } from "./document-upload-zone";

function formatShortDate(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

interface RestoreVersionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  versionToRestore: Document | null;
  currentVersion: Document | null;
  isRestoring: boolean;
  onConfirm: () => void;
}

export function RestoreVersionDialog({
  open,
  onOpenChange,
  versionToRestore,
  currentVersion,
  isRestoring,
  onConfirm,
}: RestoreVersionDialogProps) {
  if (!versionToRestore || !currentVersion) return null;

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogMedia>
            <RotateCcwIcon />
          </AlertDialogMedia>
          <AlertDialogTitle>Restore Version {versionToRestore.versionNumber}</AlertDialogTitle>
          <AlertDialogDescription>
            This will make version {versionToRestore.versionNumber} the current version, replacing
            version {currentVersion.versionNumber}.
          </AlertDialogDescription>
        </AlertDialogHeader>

        <div className="rounded-md border bg-muted/30 p-3">
          <table className="w-full text-xs">
            <thead>
              <tr className="text-muted-foreground">
                <th className="pb-2 text-left font-medium" />
                <th className="pb-2 text-left font-medium">Restoring</th>
                <th className="pb-2 text-left font-medium">Current</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              <tr>
                <td className="py-1.5 pr-3 text-muted-foreground">File</td>
                <td className="py-1.5 pr-3 font-medium">
                  {versionToRestore.originalName}
                </td>
                <td className="py-1.5 text-muted-foreground">
                  {currentVersion.originalName}
                </td>
              </tr>
              <tr>
                <td className="py-1.5 pr-3 text-muted-foreground">Size</td>
                <td className="py-1.5 pr-3">{formatFileSize(versionToRestore.fileSize)}</td>
                <td className="py-1.5 text-muted-foreground">
                  {formatFileSize(currentVersion.fileSize)}
                </td>
              </tr>
              <tr>
                <td className="py-1.5 pr-3 text-muted-foreground">Uploaded</td>
                <td className="py-1.5 pr-3">{formatShortDate(versionToRestore.createdAt)}</td>
                <td className="py-1.5 text-muted-foreground">
                  {formatShortDate(currentVersion.createdAt)}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm} disabled={isRestoring}>
            {isRestoring && <Loader2Icon className="mr-2 size-4 animate-spin" />}
            Restore
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
