import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import type { SCIMDirectory } from "@trenova/shared/types/iam";
import { Trash2Icon } from "lucide-react";
import { useState } from "react";

type DirectoryDetailHeaderProps = {
  directory?: SCIMDirectory;
  isDeleting: boolean;
  onEdit: (directory: SCIMDirectory) => void;
  onDelete: (directoryId: string) => void;
};

export function DirectoryDetailHeader({
  directory,
  isDeleting,
  onEdit,
  onDelete,
}: DirectoryDetailHeaderProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);

  return (
    <div className="rounded-lg border bg-background p-3">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="truncate text-base font-semibold tracking-tight">
              {directory?.tenantSlug || "Select a directory"}
            </h3>
            {directory && (
              <Badge variant={directory.enabled ? "active" : "inactive"}>
                {directory.enabled ? "Enabled" : "Disabled"}
              </Badge>
            )}
          </div>
          <p className="text-sm text-muted-foreground">
            Manage SCIM tokens, group-to-role mappings, and provisioning audit events.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={!directory}
            onClick={() => directory && onEdit(directory)}
          >
            Edit directory
          </Button>
          <Button
            variant="destructive"
            size="sm"
            disabled={!directory || isDeleting}
            onClick={() => setConfirmOpen(true)}
          >
            <Trash2Icon />
            Delete directory
          </Button>
        </div>
      </div>
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete SCIM Directory</AlertDialogTitle>
            <AlertDialogDescription>
              This removes the selected SCIM directory and its provisioning configuration. This
              action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="outline">Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={!directory || isDeleting}
              onClick={() => {
                if (!directory) return;
                onDelete(directory.id);
                setConfirmOpen(false);
              }}
            >
              Delete directory
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
