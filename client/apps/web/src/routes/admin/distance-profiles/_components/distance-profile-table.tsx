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
import { distanceProfileTableGraphQLConfig } from "@/lib/graphql/distance-profile-table";
import { DistanceProfileService } from "@/services/distance-profile";
import type { RowAction } from "@/types/data-table";
import type { DistanceProfile } from "@/types/distance-profile";
import { Resource } from "@/types/permission";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import { CheckCircleIcon, Loader2Icon, TrashIcon } from "lucide-react";
import { useRef, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./distance-profile-columns";
import { DistanceProfilePanel } from "./distance-profile-panel";

const distanceProfileService = new DistanceProfileService();
const columns = getColumns();

export default function DistanceProfileTable() {
  const queryClient = useQueryClient();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const selectedProfileRef = useRef<DistanceProfile | null>(null);

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await distanceProfileService.delete(id);
    },
    onSuccess: () => {
      toast.success("Distance profile deleted");
      void queryClient.invalidateQueries({ queryKey: ["distance-profile-list"] });
      setDeleteDialogOpen(false);
      selectedProfileRef.current = null;
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
      void queryClient.invalidateQueries({ queryKey: ["distance-profile-list"] });
    },
    onError: (error) => {
      toast.error("Failed to set default profile", {
        description: error instanceof Error ? error.message : "An unexpected error occurred",
      });
    },
  });

  const handleDelete = (row: Row<DistanceProfile>) => {
    selectedProfileRef.current = row.original;
    setDeleteDialogOpen(true);
  };

  const handleSetDefault = (row: Row<DistanceProfile>) => {
    if (row.original.id) {
      setDefaultMutation.mutate(row.original.id);
    }
  };

  const contextMenuActions: RowAction<DistanceProfile>[] = [
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
  ];

  return (
    <>
      <DataTable<DistanceProfile>
        name="Distance Profile"
        queryKey="distance-profile-list"
        graphql={distanceProfileTableGraphQLConfig}
        resource={Resource.DistanceProfile}
        columns={columns}
        contextMenuActions={contextMenuActions}
        TablePanel={DistanceProfilePanel}
      />
      <AlertDialog
        open={deleteDialogOpen}
        onOpenChange={(open) => {
          setDeleteDialogOpen(open);
          if (!open) {
            selectedProfileRef.current = null;
          }
        }}
      >
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
                if (selectedProfileRef.current?.id) {
                  deleteMutation.mutate(selectedProfileRef.current.id);
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
