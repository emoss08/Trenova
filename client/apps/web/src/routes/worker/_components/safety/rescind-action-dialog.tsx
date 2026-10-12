import { type TranslateFn, useT } from "@trenova/shared/i18n/use-t";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  rescindDisciplinaryAction,
  type WorkerDisciplinaryActionRow,
} from "@/lib/graphql/worker-safety";
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
  rescindActionFormSchema,
  type DisciplinaryLevel,
  type RescindActionFormValues,
} from "@trenova/shared/types/worker-safety";
import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useSafetyInvalidation } from "./use-safety-invalidation";

export type RescindActionDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  action?: WorkerDisciplinaryActionRow | null;
};

export function RescindActionDialog({
  open,
  onOpenChange,
  workerId,
  action,
}: RescindActionDialogProps) {
  const t = useT();

  const invalidate = useSafetyInvalidation(workerId);
  const form = useForm<RescindActionFormValues>({
    resolver: zodResolver(rescindActionFormSchema) as Resolver<RescindActionFormValues>,
    defaultValues: { reason: "" },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) reset({ reason: "" });
  }, [open, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    WorkerDisciplinaryActionRow,
    RescindActionFormValues,
    unknown,
    RescindActionFormValues
  >({
    form,
    resourceName: "Disciplinary action",
    mutationFn: (values) => {
      if (!action) throw new Error("Nothing to rescind");
      return rescindDisciplinaryAction({
        id: action.id,
        reason: values.reason,
        version: action.version,
      });
    },
    onSuccess: () => {
      toast.success(t("Action rescinded"), {
        description: t("It comes off the ladder but stays in the record with your reason."),
      });
      void invalidate();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {rescindTitle(action?.level as DisciplinaryLevel | undefined, t)}
          </DialogTitle>
          <DialogDescription>
            {t(
              "Use this when the action should not have been issued. It stops counting toward the next rung immediately; the row and your reason stay on the record.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            className="flex flex-col gap-4"
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup cols={1}>
              <FormControl cols="full">
                <TextareaField<RescindActionFormValues>
                  control={control}
                  name="reason"
                  label={t("Reason")}
                  placeholder={t("e.g. The delay was the shipper's, not the driver's")}
                  description={t(
                    "Saved on the rescinded row so an auditor can see why the action was withdrawn.",
                  )}
                  rules={{ required: true }}
                  maxLength={255}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending} loadingText={t("Rescinding...")}>
                {t("Rescind")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

function rescindTitle(level: DisciplinaryLevel | undefined, t: TranslateFn): string {
  switch (level) {
    case "Coaching":
      return t("Rescind this coaching");
    case "VerbalWarning":
      return t("Rescind this verbal warning");
    case "WrittenWarning":
      return t("Rescind this written warning");
    case "FinalWarning":
      return t("Rescind this final warning");
    case "Suspension":
      return t("Rescind this suspension");
    case "Termination":
      return t("Rescind this termination");
    default:
      return t("Rescind this action");
  }
}
