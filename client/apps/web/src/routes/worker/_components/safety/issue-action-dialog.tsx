import { useT } from "@trenova/shared/i18n/use-t";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { issueDisciplinaryAction, type IssueActionResult } from "@/lib/graphql/worker-safety";
import { zodResolver } from "@hookform/resolvers/zod";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
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
import { disciplinaryLevelMeta } from "@trenova/shared/lib/safety";
import {
  DISCIPLINARY_LEVEL_LABELS,
  disciplinaryLevelSchema,
  issueActionFormSchema,
  type DisciplinaryLevel,
  type IssueActionFormValues,
} from "@trenova/shared/types/worker-safety";
import { TriangleAlertIcon } from "lucide-react";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useSafetyInvalidation } from "./use-safety-invalidation";

const LEVEL_OPTIONS = disciplinaryLevelSchema.options.map((value) => ({
  value,
  label: DISCIPLINARY_LEVEL_LABELS[value],
}));

export type IssueActionDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  /** The rung the ladder says comes next. */
  suggestedLevel: DisciplinaryLevel;
  /** Pre-links the action to the event it came from. */
  safetyEventId?: string | null;
};

export function IssueActionDialog({
  open,
  onOpenChange,
  workerId,
  suggestedLevel,
  safetyEventId,
}: IssueActionDialogProps) {
  const t = useT();

  const invalidate = useSafetyInvalidation(workerId);
  const form = useForm<IssueActionFormValues>({
    resolver: zodResolver(issueActionFormSchema) as Resolver<IssueActionFormValues>,
    defaultValues: {
      level: suggestedLevel,
      reason: "",
      details: null,
      occurredAt: null,
      expiresAt: null,
      suspensionDays: null,
      safetyEventId: safetyEventId ?? null,
      recordEmploymentEvent: true,
    },
  });
  const { control, handleSubmit, reset, setValue } = form;

  useEffect(() => {
    if (!open) return;
    reset({
      level: suggestedLevel,
      reason: "",
      details: null,
      occurredAt: null,
      expiresAt: null,
      suspensionDays: null,
      safetyEventId: safetyEventId ?? null,
      recordEmploymentEvent: true,
    });
  }, [open, suggestedLevel, safetyEventId, reset]);

  const level = useWatch({ control, name: "level" });
  const meta = disciplinaryLevelMeta(level);
  const isSuspension = level === "Suspension";
  const movesEmployment = isSuspension || meta.endsEmployment;

  useEffect(() => {
    if (!isSuspension) setValue("suspensionDays", null);
  }, [isSuspension, setValue]);

  const { mutateAsync, isPending } = useApiMutation<
    IssueActionResult,
    IssueActionFormValues,
    unknown,
    IssueActionFormValues
  >({
    form,
    resourceName: "Disciplinary action",
    mutationFn: (values) =>
      issueDisciplinaryAction({
        workerId,
        level: values.level,
        reason: values.reason,
        details: values.details ?? undefined,
        occurredAt: values.occurredAt ?? undefined,
        expiresAt: values.expiresAt ?? undefined,
        suspensionDays: values.suspensionDays ?? undefined,
        safetyEventId: values.safetyEventId ?? undefined,
        recordEmploymentEvent: movesEmployment && values.recordEmploymentEvent,
      }),
    onSuccess: (result) => {
      toast.success(`${meta.label} issued`, {
        description: result.employmentEvent
          ? "The timeline was updated and the driver has been notified."
          : "The driver has been notified and can acknowledge it in Dash.",
      });
      void invalidate();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{t("Issue a disciplinary action")}</DialogTitle>
          <DialogDescription>
            {t(
              "The ladder suggests the next rung from what is still active. Actions roll off after a year unless you set another date; terminations never do.",
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
            <FormGroup className="pb-2" cols={2}>
              <FormControl cols="full">
                <SelectField<IssueActionFormValues>
                  control={control}
                  name="level"
                  label={t("Level")}
                  options={LEVEL_OPTIONS}
                  placeholder={t("Pick a level")}
                  description={t(
                    "Preset to the next rung from what is still active; choose another if the conduct warrants it.",
                  )}
                  rules={{ required: true }}
                />
              </FormControl>
              {meta.endsEmployment ? (
                <FormControl cols="full">
                  <Alert variant="destructive" className="py-2">
                    <TriangleAlertIcon className="size-4" />
                    <AlertTitle>{t("This ends employment")}</AlertTitle>
                    <AlertDescription>
                      {t(
                        "A termination closes PTO and pay assignments, cancels upcoming time off, and takes the worker off the dispatch board.",
                      )}
                    </AlertDescription>
                  </Alert>
                </FormControl>
              ) : null}
              <FormControl cols="full">
                <TextareaField<IssueActionFormValues>
                  control={control}
                  name="reason"
                  label={t("Reason")}
                  placeholder={t("What the worker is being disciplined for")}
                  description={t(
                    "Sent to the driver in the notification and copied onto the timeline event for a suspension or termination.",
                  )}
                  rules={{ required: true }}
                  maxLength={4000}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<IssueActionFormValues>
                  control={control}
                  name="details"
                  label={t("Details")}
                  placeholder={t("Context, prior conversations, what happens next")}
                  description={t(
                    "Internal context kept on the action; it is not sent to the driver.",
                  )}
                  maxLength={4000}
                />
              </FormControl>
              {isSuspension ? (
                <FormControl>
                  <NumberField<IssueActionFormValues>
                    control={control}
                    name="suspensionDays"
                    placeholder={t("e.g. 3")}
                    label={t("Suspension length")}
                    description={t("How many days the driver is off duty for this suspension.")}
                    sideText="days"
                    min={1}
                    rules={{ required: true }}
                  />
                </FormControl>
              ) : null}
              <FormControl cols="full">
                <AutoCompleteDateField<IssueActionFormValues>
                  control={control}
                  name="expiresAt"
                  label={t("Rolls off")}
                  placeholder={meta.endsEmployment ? "Never" : "One year from today"}
                  description={
                    meta.endsEmployment
                      ? "A termination never rolls off the record."
                      : "When it stops counting toward the next rung; leave blank for one year from today."
                  }
                />
              </FormControl>
              {movesEmployment ? (
                <FormControl cols="full">
                  <SwitchField<IssueActionFormValues>
                    control={control}
                    name="recordEmploymentEvent"
                    label={t("Record it on the timeline")}
                    description={
                      meta.endsEmployment
                        ? "Records a Terminated event so PTO, pay and dispatch follow."
                        : "Records a Suspended event so the worker comes off the dispatch board."
                    }
                    position="left"
                    outlined
                  />
                </FormControl>
              ) : null}
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button
                type="submit"
                variant={meta.endsEmployment ? "destructive" : "default"}
                isLoading={isPending}
                loadingText={t("Issuing...")}
              >
                {t("Issue {0}", meta.label.toLowerCase())}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
