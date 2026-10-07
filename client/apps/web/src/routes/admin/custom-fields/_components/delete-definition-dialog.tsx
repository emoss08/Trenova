import { useT } from "@trenova/shared/i18n/use-t";
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
import { ApiRequestError } from "@trenova/shared/lib/api";
import type { CustomFieldDefinitionRow } from "@/lib/graphql/custom-field-definition-table";
import { CustomFieldService } from "@/services/custom-field";
import type { DefinitionUsageStats } from "@/types/custom-field";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AlertTriangleIcon, SpinnerIcon, Trash01Icon } from "@trenova/shared/components/icons";
import { useState } from "react";
import { toast } from "sonner";
import { useRichT } from "@trenova/shared/i18n/rich";

type DeleteDefinitionDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  definition: CustomFieldDefinitionRow | null;
};

const customFieldService = new CustomFieldService();

export function DeleteDefinitionDialog({
  open,
  onOpenChange,
  definition,
}: DeleteDefinitionDialogProps) {
  const t = useT();
  const rt = useRichT();

  const queryClient = useQueryClient();
  const [usageStatsState, setUsageStatsState] = useState<{
    definitionId: string;
    stats: DefinitionUsageStats;
  } | null>(null);
  const usageStats =
    definition?.id && usageStatsState?.definitionId === definition.id
      ? usageStatsState.stats
      : null;

  const deleteMutation = useMutation({
    mutationFn: async () => {
      if (!definition?.id) throw new Error("No definition to delete");
      await customFieldService.delete(definition.id);
    },
    onSuccess: () => {
      toast.success(t("Custom field deleted"), {
        description: t('"{0}" has been deleted successfully.', definition?.label ?? ""),
      });
      void queryClient.invalidateQueries({
        queryKey: ["custom-field-definition-list"],
      });
      handleClose();
    },
    onError: (error) => {
      if (error instanceof ApiRequestError && error.isConflictError()) {
        const stats = error.getUsageStats() as DefinitionUsageStats;
        if (definition?.id) {
          setUsageStatsState({
            definitionId: definition.id,
            stats,
          });
        }
      } else {
        toast.error(t("Failed to delete custom field"), {
          description: error instanceof Error ? error.message : t("An unexpected error occurred"),
        });
      }
    },
  });

  const handleClose = () => {
    setUsageStatsState(null);
    onOpenChange(false);
  };

  const handleDelete = () => {
    deleteMutation.mutate();
  };

  if (!definition) return null;

  const hasExistingValues = usageStats && usageStats.totalValueCount > 0;

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogMedia
            className={hasExistingValues ? "bg-danger-subtle text-destructive" : ""}
          >
            {hasExistingValues ? <AlertTriangleIcon /> : <Trash01Icon />}
          </AlertDialogMedia>
          <AlertDialogTitle>
            {hasExistingValues ? t("Cannot delete custom field") : t("Delete custom field")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {hasExistingValues ? (
              <span className="space-y-2">
                <span className="block">
                  {rt(
                    "This custom field has <b>{0, plural, one {# value} other {# values}}</b> across <b>{1, plural, one {# resource} other {# resources}}</b>.",
                    { b: (c) => <strong>{c}</strong> },
                    usageStats.totalValueCount,
                    usageStats.resourceCount,
                  )}
                </span>
                <span className="block font-medium">
                  {t(
                    "To remove this field, deactivate it instead. This will hide the field from forms while preserving existing data.",
                  )}
                </span>
              </span>
            ) : (
              <span>
                {rt(
                  'Are you sure you want to delete the custom field "<b>{0}</b>"? This action cannot be undone.',
                  { b: (c) => <strong>{c}</strong> },
                  t(definition.label),
                )}
              </span>
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={handleClose}>
            {hasExistingValues ? t("Close") : t("Cancel")}
          </AlertDialogCancel>
          {!hasExistingValues && (
            <AlertDialogAction
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending && <SpinnerIcon className="mr-2 size-4 animate-spin" />}
              {t("Delete")}
            </AlertDialogAction>
          )}
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
