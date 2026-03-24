import { DataTable } from "@/components/data-table/data-table";
import { CustomFieldService } from "@/services/custom-field";
import type { CustomFieldDefinition } from "@/types/custom-field";
import type { RowAction } from "@/types/data-table";
import { Resource } from "@/types/permission";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import { PowerIcon, PowerOffIcon, TrashIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./custom-field-definition-columns";
import { CustomFieldDefinitionPanel } from "./custom-field-definition-panel";
import { DeleteDefinitionDialog } from "./delete-definition-dialog";

const customFieldService = new CustomFieldService();

export default function CustomFieldDefinitionTable() {
  const queryClient = useQueryClient();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedDefinition, setSelectedDefinition] =
    useState<CustomFieldDefinition | null>(null);

  const toggleActiveMutation = useMutation({
    mutationFn: async ({
      id,
      isActive,
    }: {
      id: CustomFieldDefinition["id"];
      isActive: boolean;
    }) => {
      return customFieldService.patch(id, { isActive });
    },
    onSuccess: (_, variables) => {
      const action = variables.isActive ? "activated" : "deactivated";
      toast.success(`Custom field ${action}`, {
        description: `The custom field has been ${action} successfully.`,
      });
      void queryClient.invalidateQueries({
        queryKey: ["custom-field-definition-list"],
      });
    },
    onError: (error) => {
      toast.error("Failed to update custom field", {
        description:
          error instanceof Error
            ? error.message
            : "An unexpected error occurred",
      });
    },
  });

  const handleDelete = useCallback((row: Row<CustomFieldDefinition>) => {
    setSelectedDefinition(row.original);
    setDeleteDialogOpen(true);
  }, []);

  const handleToggleActive = useCallback(
    (row: Row<CustomFieldDefinition>) => {
      toggleActiveMutation.mutate({
        id: row.original.id,
        isActive: !row.original.isActive,
      });
    },
    // eslint-disable-next-line @tanstack/query/no-unstable-deps
    [toggleActiveMutation],
  );

  const togglePendingId = toggleActiveMutation.isPending
    ? (toggleActiveMutation.variables?.id ?? null)
    : null;

  const columns = useMemo(() => getColumns(), []);

  const contextMenuActions = useMemo<RowAction<CustomFieldDefinition>[]>(
    () => [
      {
        id: "deactivate",
        label: "Deactivate",
        icon: PowerOffIcon,
        onClick: handleToggleActive,
        hidden: (row) => !row.original.isActive,
        disabled: (row) => togglePendingId === row.original.id,
      },
      {
        id: "activate",
        label: "Activate",
        icon: PowerIcon,
        onClick: handleToggleActive,
        hidden: (row) => row.original.isActive,
        disabled: (row) => togglePendingId === row.original.id,
      },
      {
        id: "delete",
        label: "Delete",
        icon: TrashIcon,
        variant: "destructive",
        onClick: handleDelete,
      },
    ],
    [handleToggleActive, handleDelete, togglePendingId],
  );

  return (
    <>
      <DataTable<CustomFieldDefinition>
        name="Custom Field Definition"
        link="/custom-fields/definitions/"
        queryKey="custom-field-definition-list"
        exportModelName="custom-field-definition"
        resource={Resource.CustomFieldDefinition}
        columns={columns}
        contextMenuActions={contextMenuActions}
        TablePanel={CustomFieldDefinitionPanel}
      />
      <DeleteDefinitionDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        definition={selectedDefinition}
      />
    </>
  );
}
