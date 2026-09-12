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
  storedMileageTableGraphQLConfig,
  type StoredMileageRow,
} from "@/lib/graphql/stored-mileage-table";
import { StoredMileageService } from "@/services/stored-mileage";
import type { RowAction, Row } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2Icon, TrashIcon } from "lucide-react";
import { useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./stored-mileage-columns";

const storedMileageService = new StoredMileageService();

export default function StoredMileageTable() {
  const t = useT();
  const columns = useMemo(() => getColumns(t), [t]);

  const queryClient = useQueryClient();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const selectedMileageRef = useRef<StoredMileageRow | null>(null);

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await storedMileageService.delete(id);
    },
    onSuccess: () => {
      toast.success(t("Stored mileage deactivated"));
      void queryClient.invalidateQueries({ queryKey: ["stored-mileage-list"] });
      setDeleteDialogOpen(false);
      selectedMileageRef.current = null;
    },
    onError: (error) => {
      toast.error(t("Failed to deactivate stored mileage"), {
        description: error instanceof Error ? error.message : "An unexpected error occurred",
      });
    },
  });

  const handleDelete = (row: Row<StoredMileageRow>) => {
    selectedMileageRef.current = row.original;
    setDeleteDialogOpen(true);
  };

  const contextMenuActions: RowAction<StoredMileageRow>[] = [
    {
      id: "deactivate",
      label: t("Deactivate"),
      icon: TrashIcon,
      variant: "destructive",
      disabled: (row) => row.original.status !== "Active",
      onClick: handleDelete,
    },
  ];

  return (
    <>
      <DataTable<StoredMileageRow>
        name="Stored Mileage"
        queryKey="stored-mileage-list"
        graphql={storedMileageTableGraphQLConfig}
        resource={Resource.StoredMileage}
        columns={columns}
        contextMenuActions={contextMenuActions}
      />
      <AlertDialog
        open={deleteDialogOpen}
        onOpenChange={(open) => {
          setDeleteDialogOpen(open);
          if (!open) {
            selectedMileageRef.current = null;
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <TrashIcon />
            </AlertDialogMedia>
            <AlertDialogTitle>{t("Deactivate Stored Mileage")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "This keeps the record for audit/history but removes it from future mileage lookups.",
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => {
                if (selectedMileageRef.current?.id) {
                  deleteMutation.mutate(selectedMileageRef.current.id);
                }
              }}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending && <Loader2Icon className="mr-2 size-4 animate-spin" />}
              {t("Deactivate")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
