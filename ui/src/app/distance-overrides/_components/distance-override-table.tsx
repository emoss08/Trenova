import { DataTable } from "@/components/data-table/data-table";
import { searchParamsParser } from "@/hooks/use-data-table-state";
import { DistanceOverrideSchema } from "@/lib/schemas/distance-override-schema";
import { api } from "@/services/api";
import { Resource } from "@/types/audit-entry";
import type { ContextMenuAction } from "@/types/data-table";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./distance-override-columns";
import { CreateDistanceOverrideModal } from "./distance-override-create-modal";
import { EditDistanceOverrideModal } from "./distance-override-edit-modal";

export default function DistanceOverrideTable() {
  const columns = useMemo(() => getColumns(), []);
  const queryClient = useQueryClient();
  const [, setSearchParams] = useQueryStates(searchParamsParser);
  const { mutateAsync: deleteDistanceOverride } = useMutation({
    mutationFn: (id: DistanceOverrideSchema["id"]) =>
      api.distanceOverride.delete(id),
    onSuccess: () => {
      toast.success("Distance override deleted successfully");
      queryClient.invalidateQueries({
        queryKey: ["distance-override-list"],
      });
    },
    onError: (error) => {
      toast.error(`Failed to delete distance override: ${error.message}`);
    },
  });

  const contextMenuActions: ContextMenuAction<DistanceOverrideSchema>[] = [
    {
      id: "edit",
      label: "Edit Distance Override",
      onClick: (row) => {
        setSearchParams({
          modalType: "edit",
          entityId: row.original.id,
        });
      },
    },
    {
      id: "delete",
      label: "Delete Distance Override",
      variant: "destructive",
      onClick: (row) => {
        deleteDistanceOverride(row.original.id);
      },
    },
  ];
  return (
    <DataTable<DistanceOverrideSchema>
      resource={Resource.DistanceOverride}
      name="Distance Override"
      link="/distance-overrides/"
      extraSearchParams={{
        expandDetails: true,
      }}
      columns={columns}
      queryKey="distance-override-list"
      exportModelName="distance-override"
      TableModal={CreateDistanceOverrideModal}
      TableEditModal={EditDistanceOverrideModal}
      config={{
        enableFiltering: true,
        enableSorting: true,
        enableMultiSort: true,
        maxFilters: 5,
        maxSorts: 3,
        searchDebounce: 300,
        showFilterUI: true,
        showSortUI: true,
      }}
      contextMenuActions={contextMenuActions}
    />
  );
}
