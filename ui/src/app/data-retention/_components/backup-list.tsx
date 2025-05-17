import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Icon } from "@/components/ui/icons";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { TerminalRestoreDialog } from "@/components/ui/terminal";
import { generateDateTimeStringFromUnixTimestamp } from "@/lib/date";
import { queries } from "@/lib/queries";
import { api } from "@/services/api";
import "@/styles/terminal.css";
import { DatabaseBackup } from "@/types/database-backup";
import {
  faDownload,
  faEllipsis,
  faExclamationTriangle,
  faRefresh,
  faSpinnerThird,
  faTrash,
} from "@fortawesome/pro-regular-svg-icons";
import {
  useMutation,
  useQueryClient,
  useSuspenseQuery,
} from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { toast } from "sonner";

function convertToHumanReadableSize(size: number) {
  const units = ["B", "KB", "MB", "GB", "TB"];
  let index = 0;
  let sizeInBytes = size;

  while (sizeInBytes >= 1024 && index < units.length - 1) {
    sizeInBytes /= 1024;
    index++;
  }

  return `${sizeInBytes.toFixed(2)} ${units[index]}`;
}

export default function BackupList() {
  const { data, isLoading, error } = useSuspenseQuery({
    ...queries.organization.getDatabaseBackups(),
  });

  const convertSize = useCallback(
    (size: number) => convertToHumanReadableSize(size),
    [],
  );

  if (isLoading) {
    return (
      <Card className="py-0">
        <CardHeader>
          <CardTitle>Database Backups</CardTitle>
          <CardDescription>
            Manage your database backups and restore them as needed.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex justify-center items-center py-10">
          <div className="flex flex-col items-center gap-2 text-muted-foreground">
            <Icon icon={faSpinnerThird} className="h-8 w-8 animate-spin" />
            <p>Loading backup list...</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Database Backups</CardTitle>
          <CardDescription>
            Manage your database backups and restore them as needed.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-6 border rounded-md border-dashed">
            <div className="flex flex-col items-center gap-2 text-muted-foreground max-w-md text-center">
              <Icon
                icon={faExclamationTriangle}
                className="h-8 w-8 text-destructive"
              />
              <h3 className="text-lg font-medium">Failed to load backups</h3>
              <p className="text-sm">{error.message}</p>
              <Button
                variant="outline"
                className="mt-2"
                onClick={() => window.location.reload()}
              >
                <Icon icon={faRefresh} className="mr-2 h-4 w-4" />
                Retry
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="py-0 pt-6">
      <CardHeader>
        <CardTitle>Database Backups</CardTitle>
        <CardDescription>
          Manage your database backups and restore them as needed.
        </CardDescription>
      </CardHeader>
      <CardContent className="p-0">
        <Table>
          <TableHeader className="bg-transparent">
            <TableRow className="hover:bg-transparent">
              <TableHead className="bg-transparent pl-6">Database</TableHead>
              <TableHead className="bg-transparent">Filename</TableHead>
              <TableHead className="bg-transparent">Size</TableHead>
              <TableHead className="bg-transparent">Created At</TableHead>
              <TableHead className="bg-transparent text-right pr-6">
                Actions
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.backups.map((backup) => (
              <TableRow className="hover:bg-transparent" key={backup.filename}>
                <TableCell className="pl-6">
                  <Badge variant="indigo">{backup.database}</Badge>
                </TableCell>
                <TableCell className="font-mono text-xs">
                  {backup.filename}
                </TableCell>
                <TableCell>{convertSize(backup.size)}</TableCell>
                <TableCell>
                  {generateDateTimeStringFromUnixTimestamp(backup.createdAt)}
                </TableCell>
                <TableCell className="text-right pr-6">
                  <BackupActions backup={backup} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

function BackupActions({ backup }: { backup: DatabaseBackup }) {
  const [restoreDialogOpen, setRestoreDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const queryClient = useQueryClient();

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: async () => {
      // API call to delete the backup
      return await api.databaseBackups.delete(backup.filename);
    },
    onSuccess: () => {
      toast.success("Backup deleted successfully");
      queryClient.invalidateQueries({
        queryKey: queries.organization.getDatabaseBackups._def,
      });
      setDeleteDialogOpen(false);
    },
    onError: (error) => {
      toast.error(
        `Failed to delete backup: ${error instanceof Error ? error.message : "Unknown error"}`,
      );
    },
  });

  // Restore mutation
  const restoreMutation = useMutation({
    mutationFn: async () => {
      // API call to restore the backup
      try {
        return await api.databaseBackups.restore(backup.filename);
      } catch (error) {
        // Ensure we capture the full error object
        console.error("Full restore error:", error);
        throw error; // Re-throw to be handled by the dialog
      }
    },
    onSuccess: () => {
      toast.success("Database restored successfully");
      setRestoreDialogOpen(false);
    },
    onError: (error) => {
      // We'll handle errors in the dialog component
      // But log it here too for debugging
      console.error("Restore mutation error handler:", error);
    },
  });

  // Download handler
  const handleDownload = () => {
    window.location.href = `/api/v1/database-backups/${backup.filename}`;
  };

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon">
            <Icon icon={faEllipsis} />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent side="left" align="end">
          <DropdownMenuItem
            className="flex items-center gap-2 cursor-pointer"
            onClick={() => setRestoreDialogOpen(true)}
            title="Restore"
            description="Restore this backup"
            startContent={<Icon icon={faRefresh} />}
          />

          <DropdownMenuItem
            className="flex items-center gap-2 cursor-pointer"
            onClick={handleDownload}
            title="Download"
            description="Save this backup locally"
            startContent={<Icon icon={faDownload} />}
          />

          <DropdownMenuItem
            className="flex items-center gap-2 cursor-pointer text-destructive"
            onClick={() => setDeleteDialogOpen(true)}
            title="Delete"
            description="Remove this backup"
            startContent={<Icon icon={faTrash} />}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <TerminalRestoreDialog
        backup={backup}
        open={restoreDialogOpen}
        onOpenChange={setRestoreDialogOpen}
        restoreMutation={restoreMutation}
      />

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              Delete Backup
            </AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this backup file? This action
              cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>

          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setDeleteDialogOpen(false)}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              type="submit"
              onClick={() => deleteMutation.mutate()}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending && (
                <Icon
                  icon={faSpinnerThird}
                  className="mr-2 h-4 w-4 animate-spin"
                />
              )}
              Delete Backup
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
