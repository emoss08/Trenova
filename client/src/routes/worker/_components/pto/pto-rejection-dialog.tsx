import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form } from "@/components/ui/form";
import type { ApiRequestError } from "@/lib/api";
import { apiService } from "@/services/api";
import type { TableSheetProps } from "@/types/data-table";
import {
  ptoRejectionRequestSchema,
  type PTORejectionRequest,
} from "@/types/worker";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { PTORejectionForm } from "./pto-rejection-form";

export function PTORejectionDialog({
  open,
  onOpenChange,
  ptoId,
}: TableSheetProps & {
  ptoId: string;
}) {
  const queryClient = useQueryClient();
  const form = useForm({
    resolver: zodResolver(ptoRejectionRequestSchema),
    defaultValues: {
      ptoId,
      reason: "",
    },
  });

  const {
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const { mutateAsync } = useMutation({
    mutationFn: async (values: PTORejectionRequest) => {
      await apiService.workerService.rejectPTO(values.ptoId, values.reason);
    },
    onSuccess: () => {
      toast.success("PTO rejected successfully", {
        description: `The PTO has been rejected`,
      });
      void queryClient.invalidateQueries({ queryKey: ["worker", "upcoming-pto"] });
      reset();
      onOpenChange(false);
    },
    onError: (error: ApiRequestError) => {
      if (error.isRateLimitError()) {
        toast.error("Rate limit exceeded", {
          description:
            "You have exceeded the rate limit. Please try again later.",
        });
      }
    },
  });

  const onSubmit = useCallback(
    async (values: PTORejectionRequest) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Reject PTO</DialogTitle>
          <DialogDescription>
            Reject the PTO and provide a reason for the rejection.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(e) => {
              e.preventDefault();
              e.stopPropagation();
              void handleSubmit(onSubmit)(e);
            }}
          >
            <PTORejectionForm />
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                Cancel
              </Button>
              <Button
                type="button"
                onClick={() => void handleSubmit(onSubmit)()}
                variant="destructive"
                isLoading={isSubmitting}
                loadingText="Rejecting PTO..."
              >
                Confirm Rejection
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
