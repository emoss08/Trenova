import { FormulaTemplateAutocompleteField } from "@/components/autocomplete-fields";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { apiService } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

const rerateSchema = z.object({
  formulaTemplateId: z.string().min(1, "Required"),
});

type RerateFormValues = z.infer<typeof rerateSchema>;

export function BillingQueueRerateDialog({
  open,
  onOpenChange,
  itemId,
  currentTemplateId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  itemId: string;
  currentTemplateId?: string;
}) {
  const queryClient = useQueryClient();

  const form = useForm<RerateFormValues>({
    resolver: zodResolver(rerateSchema),
    defaultValues: {
      formulaTemplateId: currentTemplateId ?? "",
    },
  });

  const { control, handleSubmit, reset } = form;

  const { mutateAsync, isPending } = useMutation({
    mutationFn: (values: RerateFormValues) =>
      apiService.billingQueueService.updateCharges(itemId, {
        formulaTemplateId: values.formulaTemplateId,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
      void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
      toast.success("Shipment re-rated with new formula template");
    },
    onError: () => {
      toast.error("Failed to re-rate shipment");
    },
  });

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  const onSubmit = useCallback(
    async (values: RerateFormValues) => {
      await mutateAsync(values);
      handleClose();
    },
    [mutateAsync, handleClose],
  );

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent className="sm:max-w-100">
        <DialogHeader>
          <DialogTitle>Change Formula Template</DialogTitle>
          <DialogDescription>
            Select a different formula template to re-rate the freight charge.
          </DialogDescription>
        </DialogHeader>
        <Form onSubmit={handleSubmit(onSubmit)}>
          <FormGroup cols={1} className="pb-4">
            <FormControl>
              <FormulaTemplateAutocompleteField
                control={control}
                name="formulaTemplateId"
                label="Formula Template"
                rules={{ required: true }}
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <Button type="submit" isLoading={isPending} loadingText="Re-rating...">
              Re-rate
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
