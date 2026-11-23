import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useSingleFlight } from "@/hooks/use-single-flight";
import {
  amendWorkerEmploymentEvent,
  type WorkerEmploymentEventRow,
} from "@/lib/graphql/worker-employment";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  employmentEventAmendSchema,
  type EmploymentEventAmendValues,
} from "@trenova/shared/types/worker-employment";
import { useEffect, useRef, useState } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { employmentEventName } from "./employment-event-meta";
import { pairValues, ValueChips } from "./timeline-list";
import { useEmploymentInvalidation } from "./use-employment-invalidation";

function amendDefaults(event: WorkerEmploymentEventRow | null): EmploymentEventAmendValues {
  return {
    effectiveAt: event?.effectiveAt ?? 0,
    reason: event?.reason ?? null,
    notes: event?.notes ?? null,
    amendmentNote: "",
  };
}

export type AmendEventPaneProps = {
  /** The event being amended; the pane is open while it is set. */
  event: WorkerEmploymentEventRow | null;
  workerId: string;
  onClose: () => void;
};

/**
 * Amend an event's date, reason or notes. The original stays on the record beside the
 * amendment, and what the event already did to the worker is not replayed.
 */
export function AmendEventPane({ event: requested, workerId, onClose }: AmendEventPaneProps) {
  const t = useT();
  // The pane keeps drawing the last event while it folds away after closing.
  const [held, setHeld] = useState(requested);
  if (requested !== null && requested !== held) setHeld(requested);
  const event = requested ?? held;
  const invalidate = useEmploymentInvalidation(workerId);
  const form = useForm<EmploymentEventAmendValues>({
    resolver: zodResolver(employmentEventAmendSchema) as Resolver<EmploymentEventAmendValues>,
    defaultValues: amendDefaults(event),
  });
  const { control, handleSubmit, reset } = form;

  const loadedFor = useRef<string | null>(null);
  useEffect(() => {
    const key = requested ? `${requested.id}:${requested.version}` : null;
    if (key !== null && loadedFor.current !== key) reset(amendDefaults(requested));
    loadedFor.current = key;
  }, [requested, reset]);

  const save = useApiMutation<
    WorkerEmploymentEventRow,
    EmploymentEventAmendValues,
    unknown,
    EmploymentEventAmendValues
  >({
    form,
    resourceName: "Employment event",
    scope: { id: `worker-employment-amend:${event?.id ?? "none"}` },
    mutationFn: (values) => {
      if (!event) throw new Error("No event to amend");
      return amendWorkerEmploymentEvent({
        id: event.id,
        effectiveAt: values.effectiveAt,
        reason: values.reason,
        notes: values.notes,
        amendmentNote: values.amendmentNote,
        version: event.version,
      });
    },
    onSuccess: () => {
      void invalidate();
      onClose();
    },
  });
  const submit = useSingleFlight((values: EmploymentEventAmendValues) => save.mutateAsync(values));

  return (
    <WorkPane
      open={requested !== null}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={t("Amend event")}
      description={t("The original stays on the record beside the amendment.")}
      contentKey={event?.id}
    >
      <FormProvider {...form}>
        <WorkPaneDirtyTracker form={form} />
        <Form
          onSubmit={(formEvent) => {
            formEvent.preventDefault();
            formEvent.stopPropagation();
            void handleSubmit((submitted) => submit.run(submitted))(formEvent);
          }}
        >
          <WorkPaneBody>
            {event ? (
              <div className="flex flex-col gap-1 rounded-md border border-border-subtle bg-sunken px-2.5 py-2">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">{employmentEventName(event.kind)}</span>
                  <span className="ml-auto font-mono text-xs text-muted-foreground tabular-nums">
                    {formatUnixDateMedium(event.effectiveAt)}
                  </span>
                </div>
                <ValueChips pairs={pairValues(event)} />
              </div>
            ) : null}
            <FormGroup cols={1}>
              <FormControl>
                <AutoCompleteDateField<EmploymentEventAmendValues>
                  control={control}
                  name="effectiveAt"
                  label={t("Effective")}
                  required
                  placeholder={t("Pick a date")}
                  description={t(
                    "Moving the date re-orders the history; what the event already did is not replayed.",
                  )}
                />
              </FormControl>
              <FormControl>
                <InputField<EmploymentEventAmendValues>
                  control={control}
                  name="reason"
                  label={t("Reason")}
                  placeholder={t("Why the event happened")}
                  maxLength={255}
                />
              </FormControl>
              <FormControl>
                <TextareaField<EmploymentEventAmendValues>
                  control={control}
                  name="notes"
                  label={t("Notes")}
                  placeholder={t("Anything payroll or HR should know")}
                  maxLength={4000}
                />
              </FormControl>
              <FormControl>
                <TextareaField<EmploymentEventAmendValues>
                  control={control}
                  name="amendmentNote"
                  label={t("Why is this being amended?")}
                  placeholder={t("e.g. Effective date corrected to match the signed letter")}
                  required
                  maxLength={255}
                  description={t("Shown on the event with your name and today's date.")}
                />
              </FormControl>
            </FormGroup>
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={submit.pending} disabled={!event}>
              {t("Save amendment")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}
