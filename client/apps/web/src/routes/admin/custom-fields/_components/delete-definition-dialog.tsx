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
import { ApiRequestError } from "@/lib/api";
import { CustomFieldService } from "@/services/custom-field";
import type {
  CustomFieldDefinition,
  DefinitionUsageStats,
} from "@/types/custom-field";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AlertTriangleIcon, Loader2Icon, TrashIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

type DeleteDefinitionDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  definition: CustomFieldDefinition | null;
};

const customFieldService = new CustomFieldService();

export function DeleteDefinitionDialog({
  open,
  onOpenChange,
  definition,
}: DeleteDefinitionDialogProps) {
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
      toast.success("Custom field deleted", {
        description: `"${definition?.label}" has been deleted successfully.`,
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
        toast.error("Failed to delete custom field", {
          description:
            error instanceof Error
              ? error.message
              : "An unexpected error occurred",
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
            className={
              hasExistingValues ? "bg-destructive/10 text-destructive" : ""
            }
          >
            {hasExistingValues ? <AlertTriangleIcon /> : <TrashIcon />}
          </AlertDialogMedia>
          <AlertDialogTitle>
            {hasExistingValues
              ? "Cannot Delete Custom Field"
              : "Delete Custom Field"}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {hasExistingValues ? (
              <span className="space-y-2">
                <span className="block">
                  This custom field has{" "}
                  <strong>{usageStats.totalValueCount} values</strong> across{" "}
                  <strong>{usageStats.resourceCount} resources</strong>.
                </span>
                <span className="block font-medium">
                  To remove this field, deactivate it instead. This will hide
                  the field from forms while preserving existing data.
                </span>
              </span>
            ) : (
              <span>
                Are you sure you want to delete the custom field &quot;
                <strong>{definition.label}</strong>&quot;? This action cannot be
                undone.
              </span>
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={handleClose}>
            {hasExistingValues ? "Close" : "Cancel"}
          </AlertDialogCancel>
          {!hasExistingValues && (
            <AlertDialogAction
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending && (
                <Loader2Icon className="mr-2 size-4 animate-spin" />
              )}
              Delete
            </AlertDialogAction>
          )}
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
