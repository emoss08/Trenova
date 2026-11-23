import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  issueDisciplinaryAction,
  type IssueActionResult,
  type WorkerSafetyEventRow,
} from "@/lib/graphql/worker-safety";
import { zodResolver } from "@hookform/resolvers/zod";
import { AlertTriangleIcon, Link01Icon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { disciplinaryLevelMeta } from "@trenova/shared/lib/safety";
import {
  DISCIPLINARY_LEVEL_LABELS,
  disciplinaryLevelSchema,
  issueActionFormSchema,
  type DisciplinaryLevel,
  type IssueActionFormValues,
} from "@trenova/shared/types/worker-safety";
import { useEffect, useRef } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { DISCIPLINE_STAIR } from "./safety-model";
import { safetyEventHeadline } from "./safety-event-row";
import { useSafetyInvalidation } from "./use-safety-invalidation";

function defaults(level: DisciplinaryLevel, safetyEventId: string | null): IssueActionFormValues {
  return {
    level,
    reason: "",
    details: null,
    occurredAt: null,
    expiresAt: null,
    suspensionDays: null,
    safetyEventId,
    recordEmploymentEvent: true,
  };
}

/** What the level field says about the chosen step against the stair. */
export function levelNote(
  level: DisciplinaryLevel,
  suggested: DisciplinaryLevel,
  highest: DisciplinaryLevel | null,
  t: TranslateFn,
): { text: string; warn: boolean } {
  const rank = DISCIPLINE_STAIR.indexOf(level);
  const next = DISCIPLINE_STAIR.indexOf(suggested);
  const top = highest ? DISCIPLINE_STAIR.indexOf(highest) : -1;
  if (rank === next) {
    return top < 0
      ? { text: t("The first step; there is no active action."), warn: false }
      : {
          text: t(
            "The next step after the active {0}.",
            DISCIPLINARY_LEVEL_LABELS[highest as DisciplinaryLevel].toLowerCase(),
          ),
          warn: false,
        };
  }
  if (rank <= top) {
    return {
      text: t(
        "At or below the active {0}, so it does not move them up the ladder.",
        DISCIPLINARY_LEVEL_LABELS[highest as DisciplinaryLevel].toLowerCase(),
      ),
      warn: false,
    };
  }
  return {
    text: t(
      "Skips past the suggested {0}. Say why in the details.",
      DISCIPLINARY_LEVEL_LABELS[suggested].toLowerCase(),
    ),
    warn: true,
  };
}

export type IssueActionPaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  workerName: string;
  suggestedLevel: DisciplinaryLevel;
  highestLevel: DisciplinaryLevel | null;
  /** The event the action is issued for, when it came from one. */
  event: WorkerSafetyEventRow | null;
  /** Tells the stair which step the form is on, so it previews it. */
  onLevelChange: (level: DisciplinaryLevel | null) => void;
  onSaved: () => void;
};

/**
 * Issues a disciplinary action beside the stair, which previews the chosen step. The level starts
 * on the server's suggested step; a suspension asks for its length, and a suspension or a
 * termination can be written to the employment history.
 */
export function IssueActionPane({
  open,
  onOpenChange,
  workerId,
  workerName,
  suggestedLevel,
  highestLevel,
  event,
  onLevelChange,
  onSaved,
}: IssueActionPaneProps) {
  const t = useT();
  const rt = useRichT();
  const invalidate = useSafetyInvalidation(workerId);
  const form = useForm<IssueActionFormValues>({
    resolver: zodResolver(issueActionFormSchema) as Resolver<IssueActionFormValues>,
    defaultValues: defaults(suggestedLevel, event?.id ?? null),
  });
  const { control, handleSubmit, reset, setValue } = form;

  const loaded = useRef<string | null>(null);
  useEffect(() => {
    if (!open) {
      loaded.current = null;
      return;
    }
    const key = `${suggestedLevel}:${event?.id ?? ""}`;
    if (loaded.current === key) return;
    loaded.current = key;
    const values = defaults(suggestedLevel, event?.id ?? null);
    if (event) {
      const firstViolation = event.violations?.[0]?.description;
      values.reason = t(
        "{0}, {1}",
        firstViolation ?? safetyEventHeadline(event),
        formatUnixDateMedium(event.occurredAt),
      );
    }
    reset(values);
  }, [open, suggestedLevel, event, reset, t]);

  const level = useWatch({ control, name: "level" });
  const meta = disciplinaryLevelMeta(level);
  const isSuspension = level === "Suspension";
  const movesEmployment = isSuspension || meta.endsEmployment;

  useEffect(() => {
    onLevelChange(open ? level : null);
  }, [open, level, onLevelChange]);

  useEffect(() => {
    if (!isSuspension) setValue("suspensionDays", null);
  }, [isSuspension, setValue]);

  const options = disciplinaryLevelSchema.options.map((value) => ({
    value,
    label: t(DISCIPLINARY_LEVEL_LABELS[value]),
    description: value === suggestedLevel ? t("Suggested") : undefined,
  }));
  const note = levelNote(level, suggestedLevel, highestLevel, t);

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
    onSuccess: () => {
      void invalidate();
      onSaved();
    },
  });

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={t("Issue a disciplinary action")}
      description={t(
        "Moves {0} up the discipline ladder. They are asked to acknowledge it in Dash.",
        workerName,
      )}
      contentKey={event?.id ?? "stair"}
    >
      <WorkPaneDirtyTracker form={form} />
      <FormProvider {...form}>
        <Form
          onSubmit={(submitEvent) => {
            submitEvent.preventDefault();
            submitEvent.stopPropagation();
            void handleSubmit((values) => mutateAsync(values))(submitEvent);
          }}
        >
          <WorkPaneBody className="text-sm">
            {event ? (
              <p className="flex items-center gap-1.5 rounded-md border border-border-subtle bg-sunken px-2.5 py-1.5 text-xs text-muted-foreground">
                <Link01Icon className="size-3 shrink-0" />
                <span>
                  {rt(
                    "For the <b>{0}</b> on {1}",
                    { b: (chunk) => <b className="font-medium text-foreground">{chunk}</b> },
                    safetyEventHeadline(event).toLowerCase(),
                    formatUnixDateMedium(event.occurredAt),
                  )}
                </span>
              </p>
            ) : null}
            <FormGroup cols={1}>
              <FormControl cols="full">
                <SelectField<IssueActionFormValues>
                  control={control}
                  name="level"
                  label={t("Level")}
                  options={options}
                  placeholder={t("Pick a level")}
                  description={note.warn ? undefined : note.text}
                  required
                />
                {note.warn ? (
                  <p className="mt-1 text-xs text-warning-foreground">{note.text}</p>
                ) : null}
              </FormControl>
              <FormControl cols="full">
                <TextareaField<IssueActionFormValues>
                  control={control}
                  name="reason"
                  label={t("Reason")}
                  placeholder={t("What the worker is being disciplined for")}
                  description={t("Sent to the driver with the action.")}
                  required
                  maxLength={4000}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<IssueActionFormValues>
                  control={control}
                  name="details"
                  label={t("Details")}
                  placeholder={t("Context, prior conversations, what happens next")}
                  description={t("Kept on the action for the office; not sent to the driver.")}
                  maxLength={4000}
                />
              </FormControl>
              {isSuspension ? (
                <FormControl cols="full" className="motion-safe:animate-expand-in">
                  <NumberField<IssueActionFormValues>
                    control={control}
                    name="suspensionDays"
                    label={t("Suspension length")}
                    placeholder={t("e.g. 3")}
                    description={t("How many days the driver is off duty for this suspension.")}
                    sideText={t("days")}
                    min={1}
                    required
                  />
                </FormControl>
              ) : null}
              <FormControl cols="full">
                <AutoCompleteDateField<IssueActionFormValues>
                  control={control}
                  name="expiresAt"
                  label={t("Rolls off")}
                  placeholder={meta.endsEmployment ? t("Never") : t("One year from today")}
                  description={
                    meta.endsEmployment
                      ? t("A termination never rolls off the record.")
                      : t(
                          "Twelve months from today when left empty. After that date it no longer counts toward the next step.",
                        )
                  }
                />
              </FormControl>
              {movesEmployment ? (
                <FormControl cols="full" className="motion-safe:animate-expand-in">
                  <SwitchField<IssueActionFormValues>
                    control={control}
                    name="recordEmploymentEvent"
                    label={t("Record it on the employment history")}
                    description={
                      meta.endsEmployment
                        ? t("Writes a Terminated event to Employment.")
                        : t("Writes a Suspended event to Employment.")
                    }
                    position="left"
                    outlined
                  />
                </FormControl>
              ) : null}
              {meta.endsEmployment ? (
                <FormControl cols="full" className="motion-safe:animate-expand-in">
                  <Alert variant="destructive" size="sm">
                    <AlertTriangleIcon />
                    <AlertTitle>{t("This ends employment")}</AlertTitle>
                    <AlertDescription>
                      {t(
                        "Time off, the pay assignment and Dash access end with it, and the worker comes off the dispatch board.",
                      )}
                    </AlertDescription>
                  </Alert>
                </FormControl>
              ) : null}
            </FormGroup>
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button
              type="submit"
              size="sm"
              variant={meta.endsEmployment ? "destructive" : "default"}
              isLoading={isPending}
              loadingText={t("Issuing...")}
            >
              {issueLabel(level, t)}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}

function issueLabel(level: DisciplinaryLevel, t: TranslateFn): string {
  switch (level) {
    case "Coaching":
      return t("Issue coaching");
    case "VerbalWarning":
      return t("Issue verbal warning");
    case "WrittenWarning":
      return t("Issue written warning");
    case "FinalWarning":
      return t("Issue final warning");
    case "Suspension":
      return t("Issue suspension");
    default:
      return t("Issue termination");
  }
}
