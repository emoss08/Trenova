import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import {
  customFieldDefinitionTableGraphQLConfig,
  type CustomFieldDefinitionRow,
} from "@/lib/graphql/custom-field-definition-table";
import { CustomFieldService } from "@/services/custom-field";
import type { RowAction, Row } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { PowerIcon, PowerOffIcon, TrashIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./custom-field-definition-columns";
import { CustomFieldDefinitionPanel } from "./custom-field-definition-panel";
import { DeleteDefinitionDialog } from "./delete-definition-dialog";

const customFieldService = new CustomFieldService();

export default function CustomFieldDefinitionTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedDefinition, setSelectedDefinition] = useState<CustomFieldDefinitionRow | null>(
    null,
  );

  const { mutateAsync: toggleActive } = useMutation({
    mutationFn: async ({
      id,
      isActive,
    }: {
      id: CustomFieldDefinitionRow["id"];
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
      toast.error(t("Failed to update custom field"), {
        description: error instanceof Error ? error.message : "An unexpected error occurred",
      });
    },
  });

  const handleDelete = useCallback((row: Row<CustomFieldDefinitionRow>) => {
    setSelectedDefinition(row.original);
    setDeleteDialogOpen(true);
  }, []);

  const handleToggleActive = useCallback(
    (row: Row<CustomFieldDefinitionRow>) =>
      toggleActive({
        id: row.original.id,
        isActive: !row.original.isActive,
      }).catch(() => undefined),
    [toggleActive],
  );

  const columns = useMemo(() => getColumns(t), [t]);

  const contextMenuActions = useMemo<RowAction<CustomFieldDefinitionRow>[]>(
    () => [
      {
        id: "deactivate",
        label: t("Deactivate"),
        icon: PowerOffIcon,
        onClick: handleToggleActive,
        hidden: (row) => !row.original.isActive,
      },
      {
        id: "activate",
        label: t("Activate"),
        icon: PowerIcon,
        onClick: handleToggleActive,
        hidden: (row) => row.original.isActive,
      },
      {
        id: "delete",
        label: t("Delete"),
        icon: TrashIcon,
        variant: "destructive",
        onClick: handleDelete,
      },
    ],
    [handleToggleActive, handleDelete, t],
  );

  return (
    <>
      <DataTable<CustomFieldDefinitionRow>
        name="Custom Field Definition"
        queryKey="custom-field-definition-list"
        graphql={customFieldDefinitionTableGraphQLConfig}
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
