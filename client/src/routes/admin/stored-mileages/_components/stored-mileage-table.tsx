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
import { StoredMileageService } from "@/services/stored-mileage";
import type { RowAction } from "@/types/data-table";
import { Resource } from "@/types/permission";
import type { StoredMileage } from "@/types/stored-mileage";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import { Loader2Icon, TrashIcon } from "lucide-react";
import { useRef, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./stored-mileage-columns";

const storedMileageService = new StoredMileageService();
const columns = getColumns();

export default function StoredMileageTable() {
  const queryClient = useQueryClient();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const selectedMileageRef = useRef<StoredMileage | null>(null);

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await storedMileageService.delete(id);
    },
    onSuccess: () => {
      toast.success("Stored mileage deactivated");
      void queryClient.invalidateQueries({ queryKey: ["stored-mileage-list"] });
      setDeleteDialogOpen(false);
      selectedMileageRef.current = null;
    },
    onError: (error) => {
      toast.error("Failed to deactivate stored mileage", {
        description: error instanceof Error ? error.message : "An unexpected error occurred",
      });
    },
  });

  const handleDelete = (row: Row<StoredMileage>) => {
    selectedMileageRef.current = row.original;
    setDeleteDialogOpen(true);
  };

  const contextMenuActions: RowAction<StoredMileage>[] = [
    {
      id: "deactivate",
      label: "Deactivate",
      icon: TrashIcon,
      variant: "destructive",
      disabled: (row) => row.original.status !== "Active",
      onClick: handleDelete,
    },
  ];

  return (
    <>
      <DataTable<StoredMileage>
        name="Stored Mileage"
        link="/stored-mileages/"
        queryKey="stored-mileage-list"
        exportModelName="stored-mileage"
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
            <AlertDialogTitle>Deactivate Stored Mileage</AlertDialogTitle>
            <AlertDialogDescription>
              This keeps the record for audit/history but removes it from future mileage lookups.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
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
              Deactivate
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
