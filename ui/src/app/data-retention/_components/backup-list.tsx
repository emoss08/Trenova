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
import { API_URL } from "@/constants/env";
import { generateDateTimeStringFromUnixTimestamp } from "@/lib/date";
import { queries } from "@/lib/queries";
import {
  deleteDatabaseBackup,
  restoreDatabaseBackup,
} from "@/services/organization";
import { DatabaseBackup } from "@/types/database-backup";
import {
  faDatabase,
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
      <Card>
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

  if (!data?.backups || data.backups.length === 0) {
    return <BackupEmptyState />;
  }

  return (
    <Card>
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
      await deleteDatabaseBackup(backup.filename);
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
      await restoreDatabaseBackup(backup.filename);
    },
    onSuccess: () => {
      toast.success("Database restored successfully");
      setRestoreDialogOpen(false);
    },
    onError: (error) => {
      toast.error(
        `Failed to restore database: ${error instanceof Error ? error.message : "Unknown error"}`,
      );
    },
  });

  // Download handler
  const handleDownload = () => {
    window.location.href = `${API_URL}/database-backups/${backup.filename}`;
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

      <AlertDialog open={restoreDialogOpen} onOpenChange={setRestoreDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              Restore Database
            </AlertDialogTitle>
            <AlertDialogDescription>
              You are about to restore the database from this backup. This
              action will overwrite your current database with the data from
              this backup file.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setRestoreDialogOpen(false)}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={() => restoreMutation.mutate()}
              disabled={restoreMutation.isPending}
            >
              {restoreMutation.isPending && (
                <Icon
                  icon={faSpinnerThird}
                  className="mr-2 h-4 w-4 animate-spin"
                />
              )}
              Restore Database
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

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

function BackupEmptyState() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Database Backups</CardTitle>
        <CardDescription>
          Manage your database backups and restore them as needed.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-center py-10 border rounded-md border-dashed">
          <div className="flex flex-col items-center gap-2 text-foreground max-w-md text-center">
            <Icon icon={faDatabase} className="size-12" />
            <h3 className="text-lg font-medium">No backups found</h3>
            <p className="text-sm text-muted-foreground">
              No database backups have been created yet. Configure automatic
              backups or create your first backup manually.
            </p>
            <Button className="mt-2">
              <Icon icon={faDownload} className="mr-1 mb-0.5 size-4" />
              Create First Backup
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
