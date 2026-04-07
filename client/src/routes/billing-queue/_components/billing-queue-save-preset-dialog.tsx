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
import { Input } from "@/components/ui/input";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import {
  billingQueueFilterPresetInputSchema,
  type BillingQueueFilterPresetInput,
} from "@/types/billing-queue";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  filters: Record<string, string | string[] | null>;
};

export function BillingQueueSavePresetDialog({ open, onOpenChange, filters }: Props) {
  const queryClient = useQueryClient();

  const cleanFilters = useMemo(() => {
    const result: Record<string, any> = {};
    for (const [key, value] of Object.entries(filters)) {
      if (Array.isArray(value) ? value.length > 0 : value != null && value !== "") {
        result[key] = value;
      }
    }
    return result;
  }, [filters]);

  const form = useForm<BillingQueueFilterPresetInput>({
    resolver: zodResolver(billingQueueFilterPresetInputSchema),
    defaultValues: {
      name: "",
      filters: cleanFilters,
      isDefault: false,
    },
  });

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { isSubmitting },
  } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: (values: BillingQueueFilterPresetInput) =>
      apiService.billingQueueService.createFilterPreset({
        ...values,
        filters: cleanFilters,
      }),
    resourceName: "BillingQueueFilterPreset",
    setFormError: setError,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["billing-queue-filter-presets"] });
      toast.success("Filter preset saved");
    },
  });

  const onSubmit = useCallback(
    async (values: BillingQueueFilterPresetInput) => {
      await mutateAsync(values);
      reset();
      onOpenChange(false);
    },
    [mutateAsync, reset, onOpenChange],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[400px]">
        <DialogHeader>
          <DialogTitle>Save Filter Preset</DialogTitle>
          <DialogDescription>
            Save the current filter combination as a reusable preset.
          </DialogDescription>
        </DialogHeader>
        <Form onSubmit={handleSubmit(onSubmit)}>
          <div className="py-4">
            <FormGroup>
              <FormControl>
                <Input
                  placeholder="Preset name"
                  {...register("name")}
                  className="h-8 text-sm"
                  autoFocus
                />
              </FormControl>
            </FormGroup>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" size="sm" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" size="sm" disabled={isSubmitting}>
              {isSubmitting ? "Saving..." : "Save"}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
