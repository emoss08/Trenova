import { useT } from "@trenova/shared/i18n/use-t";
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
} from "@trenova/shared/components/ui/alert-dialog";
import {
  distanceProfileTableGraphQLConfig,
  type DistanceProfileRow,
} from "@/lib/graphql/distance-profile-table";
import { DistanceProfileService } from "@/services/distance-profile";
import type { RowAction, Row } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CheckCircleIcon, Loader2Icon, TrashIcon } from "lucide-react";
import { useRef, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./distance-profile-columns";
import { DistanceProfilePanel } from "./distance-profile-panel";

const distanceProfileService = new DistanceProfileService();
const columns = getColumns();

export default function DistanceProfileTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const selectedProfileRef = useRef<DistanceProfileRow | null>(null);

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await distanceProfileService.delete(id);
    },
    onSuccess: () => {
      toast.success(t("Distance profile deleted"));
      void queryClient.invalidateQueries({ queryKey: ["distance-profile-list"] });
      setDeleteDialogOpen(false);
      selectedProfileRef.current = null;
    },
    onError: (error) => {
      toast.error(t("Failed to delete distance profile"), {
        description: error instanceof Error ? error.message : "An unexpected error occurred",
      });
    },
  });

  const setDefaultMutation = useMutation({
    mutationFn: (id: string) => distanceProfileService.setDefault(id),
    onSuccess: () => {
      toast.success(t("Default distance profile updated"));
      void queryClient.invalidateQueries({ queryKey: ["distance-profile-list"] });
    },
    onError: (error) => {
      toast.error(t("Failed to set default profile"), {
        description: error instanceof Error ? error.message : "An unexpected error occurred",
      });
    },
  });

  const handleDelete = (row: Row<DistanceProfileRow>) => {
    selectedProfileRef.current = row.original;
    setDeleteDialogOpen(true);
  };

  const handleSetDefault = (row: Row<DistanceProfileRow>) =>
    row.original.id
      ? setDefaultMutation.mutateAsync(row.original.id).catch(() => undefined)
      : undefined;

  const contextMenuActions: RowAction<DistanceProfileRow>[] = [
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
      <DataTable<DistanceProfileRow>
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
            <AlertDialogTitle>{t("Delete Distance Profile")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("Are you sure you want to delete this distance profile? Default profiles cannot be deleted.")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
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
              {t("Delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
