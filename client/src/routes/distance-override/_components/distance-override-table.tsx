import { DataTable } from "@/components/data-table/data-table";
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
import { DistanceOverrideService } from "@/services/distance-override";
import type { RowAction } from "@/types/data-table";
import type { DistanceOverride } from "@/types/distance-override";
import { Resource } from "@/types/permission";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import { Loader2Icon, TrashIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./distance-override-columns";
import { DistanceOverridePanel } from "./distance-override-panel";

const distanceOverrideService = new DistanceOverrideService();

export default function DistanceOverrideTable() {
  const queryClient = useQueryClient();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedOverride, setSelectedOverride] =
    useState<DistanceOverride | null>(null);

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await distanceOverrideService.delete(id);
    },
    onSuccess: () => {
      toast.success("Distance override deleted");
      void queryClient.invalidateQueries({
        queryKey: ["distance-override-list"],
      });
      setDeleteDialogOpen(false);
      setSelectedOverride(null);
    },
    onError: (error) => {
      toast.error("Failed to delete distance override", {
        description:
          error instanceof Error
            ? error.message
            : "An unexpected error occurred",
      });
    },
  });

  const handleDelete = useCallback((row: Row<DistanceOverride>) => {
    setSelectedOverride(row.original);
    setDeleteDialogOpen(true);
  }, []);

  const columns = useMemo(() => getColumns(), []);

  const contextMenuActions = useMemo<RowAction<DistanceOverride>[]>(
    () => [
      {
        id: "delete",
        label: "Delete",
        icon: TrashIcon,
        variant: "destructive",
        onClick: handleDelete,
      },
    ],
    [handleDelete],
  );

  return (
    <>
      <DataTable<DistanceOverride>
        name="Distance Override"
        link="/distance-overrides/"
        queryKey="distance-override-list"
        exportModelName="distance-override"
        resource={Resource.DistanceOverride}
        columns={columns}
        contextMenuActions={contextMenuActions}
        TablePanel={DistanceOverridePanel}
      />
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <TrashIcon />
            </AlertDialogMedia>
            <AlertDialogTitle>Delete Distance Override</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this distance override? This action
              cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => {
                if (selectedOverride?.id) {
                  deleteMutation.mutate(selectedOverride.id);
                }
              }}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending && (
                <Loader2Icon className="mr-2 size-4 animate-spin" />
              )}
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
