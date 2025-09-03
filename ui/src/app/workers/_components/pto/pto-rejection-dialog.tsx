import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form } from "@/components/ui/form";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import {
  PTORejectionRequestSchema,
  ptoRejectionRequestSchema,
} from "@/lib/schemas/worker-schema";
import { api } from "@/services/api";
import { TableSheetProps } from "@/types/data-table";
import { APIError } from "@/types/errors";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { PTORejectionForm } from "./pto-rejection-form";

import { Button, FormSaveButton } from "@/components/ui/button";
export function PTORejectionDialog({
  open,
  onOpenChange,
  ptoId,
}: TableSheetProps & {
  ptoId: string;
}) {
  const form = useForm({
    resolver: zodResolver(ptoRejectionRequestSchema),
    defaultValues: {
      ptoId,
      reason: "",
    },
  });

  const {
    formState: { isSubmitting, isSubmitSuccessful },
    handleSubmit,
    reset,
  } = form;

  const { mutateAsync } = useMutation({
    mutationFn: async (values: PTORejectionRequestSchema) => {
      await api.worker.rejectPTO(values.ptoId, values.reason);
    },
    onSuccess: () => {
      toast.success("PTO rejected successfully", {
        description: `The PTO has been rejected`,
      });
      broadcastQueryInvalidation({
        queryKey: ["worker", "upcoming-pto"],
        options: {
          correlationId: `reject-pto-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      onOpenChange(false);
    },
    onError: (error: APIError) => {
      if (error.isRateLimitError()) {
        toast.error("Rate limit exceeded", {
          description:
            "You have exceeded the rate limit. Please try again later.",
        });
      }
    },
  });

  const onSubmit = useCallback(
    async (values: PTORejectionRequestSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  useEffect(() => {
    if (isSubmitSuccessful) {
      reset();
    }
  }, [isSubmitSuccessful, reset]);

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
            className="space-y-0 p-0"
            onSubmit={(e) => {
              e.preventDefault();
              e.stopPropagation();
              handleSubmit(onSubmit)(e);
            }}
          >
            <DialogBody>
              <PTORejectionForm />
            </DialogBody>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                Cancel
              </Button>
              <FormSaveButton
                type="button"
                onClick={() => handleSubmit(onSubmit)()}
                isSubmitting={isSubmitting}
                title="PTO rejection"
                text="Confirm Rejection"
                variant="destructive"
              />
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
