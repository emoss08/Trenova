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
import { DistanceProfileService } from "@/services/distance-profile";
import type { RowAction } from "@/types/data-table";
import type { DistanceProfile } from "@/types/distance-profile";
import { Resource } from "@/types/permission";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import { CheckCircleIcon, Loader2Icon, TrashIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./distance-profile-columns";
import { DistanceProfilePanel } from "./distance-profile-panel";

const distanceProfileService = new DistanceProfileService();

export default function DistanceProfileTable() {
  const queryClient = useQueryClient();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedProfile, setSelectedProfile] = useState<DistanceProfile | null>(null);

  const invalidateList = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["distance-profile-list"] });
  }, [queryClient]);

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await distanceProfileService.delete(id);
    },
    onSuccess: () => {
      toast.success("Distance profile deleted");
      invalidateList();
      setDeleteDialogOpen(false);
      setSelectedProfile(null);
    },
    onError: (error) => {
      toast.error("Failed to delete distance profile", {
        description: error instanceof Error ? error.message : "An unexpected error occurred",
      });
    },
  });

  const setDefaultMutation = useMutation({
    mutationFn: (id: string) => distanceProfileService.setDefault(id),
    onSuccess: () => {
      toast.success("Default distance profile updated");
      invalidateList();
    },
    onError: (error) => {
      toast.error("Failed to set default profile", {
        description: error instanceof Error ? error.message : "An unexpected error occurred",
      });
    },
  });
  const { mutate: setDefault } = setDefaultMutation;

  const handleDelete = useCallback((row: Row<DistanceProfile>) => {
    setSelectedProfile(row.original);
    setDeleteDialogOpen(true);
  }, []);

  const handleSetDefault = useCallback(
    (row: Row<DistanceProfile>) => {
      if (row.original.id) {
        setDefault(row.original.id);
      }
    },
    [setDefault],
  );

  const columns = useMemo(() => getColumns(), []);

  const contextMenuActions = useMemo<RowAction<DistanceProfile>[]>(
    () => [
      {
        id: "set-default",
        label: "Set Default",
        icon: CheckCircleIcon,
        disabled: (row) => row.original.isDefault || row.original.status !== "Active",
        onClick: handleSetDefault,
      },
      {
        id: "delete",
        label: "Delete",
        icon: TrashIcon,
        variant: "destructive",
        disabled: (row) => row.original.isDefault,
        onClick: handleDelete,
      },
    ],
    [handleDelete, handleSetDefault],
  );

  return (
    <>
      <DataTable<DistanceProfile>
        name="Distance Profile"
        link="/distance-profiles/"
        queryKey="distance-profile-list"
        exportModelName="distance-profile"
        resource={Resource.DistanceProfile}
        columns={columns}
        contextMenuActions={contextMenuActions}
        TablePanel={DistanceProfilePanel}
      />
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <TrashIcon />
            </AlertDialogMedia>
            <AlertDialogTitle>Delete Distance Profile</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this distance profile? Default profiles cannot be
              deleted.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => {
                if (selectedProfile?.id) {
                  deleteMutation.mutate(selectedProfile.id);
                }
              }}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending && <Loader2Icon className="mr-2 size-4 animate-spin" />}
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
