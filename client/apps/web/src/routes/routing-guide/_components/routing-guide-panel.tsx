import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  emptyRoutingGuidePayload,
  routingGuidePayloadSchema,
  type RoutingGuide,
  type RoutingGuidePayload,
  type RoutingGuidePayloadInput,
} from "@trenova/shared/types/routing-guide";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Trash2Icon } from "lucide-react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { RoutingGuideForm } from "./routing-guide-form";

const QUERY_KEY = "routing-guide-list";

function DeleteGuideAction({ row, onDeleted }: { row: RoutingGuide; onDeleted: () => void }) {
  const queryClient = useQueryClient();

  const deleteMutation = useMutation({
    mutationFn: (guideId: string) => apiService.routingGuideService.delete(guideId),
    onSuccess: () => {
      toast.success("Routing guide deleted");
      void queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });
      onDeleted();
    },
    onError: (error: unknown) => handleMutationError({ error, resourceName: "Routing Guide" }),
  });

  return (
    <AlertDialog>
      <AlertDialogTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            className="text-muted-foreground hover:text-destructive"
            aria-label="Delete routing guide"
            disabled={deleteMutation.isPending}
          >
            <Trash2Icon className="size-4" />
          </Button>
        }
      />
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete this routing guide?</AlertDialogTitle>
          <AlertDialogDescription>
            {`"${row.name}" will no longer match this lane. Tenders already running against it are
            unaffected, but new waterfalls cannot use it.`}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Keep guide</AlertDialogCancel>
          <AlertDialogAction
            onClick={() => {
              if (row.id) deleteMutation.mutate(row.id);
            }}
          >
            Delete guide
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

export function RoutingGuidePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<RoutingGuide>) {
  const form = useForm<RoutingGuidePayloadInput, unknown, RoutingGuidePayload>({
    resolver: zodResolver(routingGuidePayloadSchema),
    defaultValues: { ...emptyRoutingGuidePayload },
    mode: "onChange",
  });

  if (mode === "edit") {
    return (
      <FormEditPanel<RoutingGuidePayloadInput, RoutingGuide, RoutingGuidePayload, RoutingGuide>
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        queryKey={QUERY_KEY}
        title="Routing Guide"
        fieldKey="name"
        size="lg"
        formComponent={<RoutingGuideForm />}
        headerActions={
          row ? <DeleteGuideAction row={row} onDeleted={() => onOpenChange(false)} /> : undefined
        }
        mutationFn={(values, currentRow) => {
          if (!currentRow.id) {
            throw new Error("No Routing Guide ID selected");
          }
          return apiService.routingGuideService.update(currentRow.id, values);
        }}
      />
    );
  }

  return (
    <FormCreatePanel<RoutingGuidePayloadInput, RoutingGuide, RoutingGuidePayload, RoutingGuide>
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey={QUERY_KEY}
      title="Routing Guide"
      size="lg"
      description="Rank the carriers a lane should waterfall through, with the rate and offer window for each."
      formComponent={<RoutingGuideForm />}
      mutationFn={(values) => apiService.routingGuideService.create(values)}
    />
  );
}
