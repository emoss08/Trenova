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
import { Power01Icon, PowerOffIcon, Trash01Icon } from "@trenova/shared/components/icons";
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
      if (variables.isActive) {
        toast.success(t("Custom field activated"), {
          description: t("The custom field has been activated successfully."),
        });
      } else {
        toast.success(t("Custom field deactivated"), {
          description: t("The custom field has been deactivated successfully."),
        });
      }
      void queryClient.invalidateQueries({
        queryKey: ["custom-field-definition-list"],
      });
    },
    onError: (error) => {
      toast.error(t("Failed to update custom field"), {
        description: error instanceof Error ? error.message : t("An unexpected error occurred"),
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
        icon: Power01Icon,
        onClick: handleToggleActive,
        hidden: (row) => row.original.isActive,
      },
      {
        id: "delete",
        label: t("Delete"),
        icon: Trash01Icon,
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
        emptyTitle={t("No custom field definitions yet")}
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
