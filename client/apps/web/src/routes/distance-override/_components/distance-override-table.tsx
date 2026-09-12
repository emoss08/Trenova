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
  distanceOverrideTableGraphQLConfig,
  type DistanceOverrideRow,
} from "@/lib/graphql/distance-override-table";
import { DistanceOverrideService } from "@/services/distance-override";
import type { RowAction, Row } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2Icon, TrashIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./distance-override-columns";
import { DistanceOverridePanel } from "./distance-override-panel";

const distanceOverrideService = new DistanceOverrideService();

export default function DistanceOverrideTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedOverride, setSelectedOverride] = useState<DistanceOverrideRow | null>(null);

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await distanceOverrideService.delete(id);
    },
    onSuccess: () => {
      toast.success(t("Distance override deleted"));
      void queryClient.invalidateQueries({
        queryKey: ["distance-override-list"],
      });
      setDeleteDialogOpen(false);
      setSelectedOverride(null);
    },
    onError: (error) => {
      toast.error(t("Failed to delete distance override"), {
        description: error instanceof Error ? error.message : "An unexpected error occurred",
      });
    },
  });

  const handleDelete = useCallback((row: Row<DistanceOverrideRow>) => {
    setSelectedOverride(row.original);
    setDeleteDialogOpen(true);
  }, []);

  const columns = useMemo(() => getColumns(t), [t]);

  const contextMenuActions = useMemo<RowAction<DistanceOverrideRow>[]>(
    () => [
      {
        id: "delete",
        label: t("Delete"),
        icon: TrashIcon,
        variant: "destructive",
        onClick: handleDelete,
      },
    ],
    [handleDelete, t],
  );

  return (
    <>
      <DataTable<DistanceOverrideRow>
        name="Distance Override"
        queryKey="distance-override-list"
        graphql={distanceOverrideTableGraphQLConfig}
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
            <AlertDialogTitle>{t("Delete Distance Override")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "Are you sure you want to delete this distance override? This action cannot be undone.",
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => {
                if (selectedOverride?.id) {
                  deleteMutation.mutate(selectedOverride.id);
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
