import { useT } from "@trenova/shared/i18n/use-t";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { closeWorkerSafetyEvent, type WorkerSafetyEventRow } from "@/lib/graphql/worker-safety";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import {
  closeSafetyEventFormSchema,
  type CloseSafetyEventFormValues,
} from "@trenova/shared/types/worker-safety";
import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useSafetyInvalidation } from "./use-safety-invalidation";

export type CloseEventDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  event?: WorkerSafetyEventRow | null;
};

export function CloseEventDialog({ open, onOpenChange, workerId, event }: CloseEventDialogProps) {
  const t = useT();

  const invalidate = useSafetyInvalidation(workerId);
  const form = useForm<CloseSafetyEventFormValues>({
    resolver: zodResolver(closeSafetyEventFormSchema) as Resolver<CloseSafetyEventFormValues>,
    defaultValues: { resolution: "" },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) reset({ resolution: event?.resolution ?? "" });
  }, [open, event, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    WorkerSafetyEventRow,
    CloseSafetyEventFormValues,
    unknown,
    CloseSafetyEventFormValues
  >({
    form,
    resourceName: "Safety event",
    mutationFn: (values) => {
      if (!event) throw new Error("Nothing to close");
      return closeWorkerSafetyEvent({
        id: event.id,
        resolution: values.resolution,
        version: event.version,
      });
    },
    onSuccess: () => {
      toast.success(t("Safety event closed"), {
        description: t("The resolution stays on the record and in the audit log."),
      });
      void invalidate();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Close this event")}</DialogTitle>
          <DialogDescription>
            {t(
              "Say what was done about it — coaching, a repair, a dismissed citation. Points already recorded stay on the scorecard until they roll off.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup className="pb-2" cols={1}>
              <FormControl cols="full">
                <TextareaField<CloseSafetyEventFormValues>
                  control={control}
                  name="resolution"
                  label={t("Resolution")}
                  placeholder={t("e.g. Coached on backing procedure; dock damage repaired")}
                  description={t(
                    "Kept on the event and in the audit log; a closed event cannot be deleted without reopening it first.",
                  )}
                  rules={{ required: true }}
                  maxLength={4000}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending} loadingText={t("Closing...")}>
                {t("Close event")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
