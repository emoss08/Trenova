import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { ptoTypeLabel } from "@/routes/worker/_components/pto/worker-pto-balances";
import {
  PTOFormFields,
  ReadOnlyPTO,
  usePTOAvailability,
  usePTOForm,
  type PTOFormDialogRecord,
  type PTOFormPreviewState,
} from "@/routes/worker/_components/pto/pto-form-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl } from "@trenova/shared/components/ui/form";
import { Form } from "@trenova/shared/components/ui/form";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatRange, inclusiveDays } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import { useState } from "react";
import { FormProvider } from "react-hook-form";

export type TimeOffPaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  /** The request to edit or read; null to record a new one. */
  pto: PTOFormDialogRecord | null;
  onSaved: (pto: WorkerPTO) => void;
};

/** A count of days as the reader's language writes it, to two places at most: "2", "0.5". */
function dayCount(value: number): string {
  return formatNumber(value, { maximumFractionDigits: 2 });
}

/** A balance in days to one place: "4.5". */
function balanceDays(value: number): string {
  return formatNumber(value, { minimumFractionDigits: 1, maximumFractionDigits: 1 });
}

/** The chosen days counted live, and what they leave of the balance they come from. */
function DayCount(state: PTOFormPreviewState) {
  const t = useT();
  const availability = usePTOAvailability(state);
  const result = availability.data;
  const counted = state.hasDateRange
    ? result
      ? Number(result.days)
      : inclusiveDays(state.startDate, state.endDate)
    : 0;
  const type = t(ptoTypeLabel(state.ptoType)).toLowerCase();
  const over = state.hasDateRange && result?.tracked === true && !result.allowed;
  const reversed = !!state.startDate && !!state.endDate && state.endDate < state.startDate;
  const available = result ? Number(result.projectedAvailableDays) : null;

  return (
    <FormControl cols="full">
      <div
        aria-live="polite"
        className="flex items-center gap-3 rounded-md border border-border-subtle bg-sunken px-3 py-2.5"
      >
        <span
          data-testid="pto-day-count"
          className={cn(
            "min-w-10 text-2xl font-semibold tabular-nums",
            (over || reversed) && "text-danger-foreground",
          )}
        >
          {dayCount(counted)}
        </span>
        <div className="min-w-0">
          <p className="font-medium">
            {state.hasDateRange
              ? t(
                  "{0, plural, one {day} other {days}}, {1}",
                  counted,
                  formatRange(state.startDate, state.endDate, { year: "always", join: "words" }),
                )
              : t("{0, plural, one {day} other {days}}", counted)}
          </p>
          <p
            className={cn(
              "text-xs tabular-nums",
              over || reversed ? "text-danger-foreground" : "text-muted-foreground",
            )}
          >
            {reversed
              ? t("The last day is before the first.")
              : !state.hasDateRange
                ? t("Pick the days to see what it leaves.")
                : !result
                  ? t("Counting against the policy…")
                  : !result.tracked
                    ? t("Their policy does not track {0} time off.", type)
                    : !result.allowed
                      ? (result.message ?? t("More than the balance allows."))
                      : available !== null
                        ? t(
                            "{0} of {1} {2} days would remain.",
                            balanceDays(available - counted),
                            balanceDays(available),
                            type,
                          )
                        : null}
          </p>
        </div>
      </div>
    </FormControl>
  );
}

/**
 * A time off request beside the requests table: recorded on the worker's behalf, edited while it
 * waits for a decision, and read once it is decided. The form is the roster dialog's own.
 */
export function TimeOffPane({
  open,
  onOpenChange,
  workerId,
  pto: opened,
  onSaved,
}: TimeOffPaneProps) {
  const t = useT();
  // The request as it was when the pane opened: the shared form resets whenever the record it is
  // given changes, and a refetch of the requests must never wipe what is being typed. Each opening
  // (and each other request) mounts the pane anew.
  const [pto] = useState(opened);
  const { form, submit, isSubmitting, isEdit, editable } = usePTOForm({
    open,
    pto,
    defaultWorkerId: workerId,
    onSaved,
    onDone: () => onOpenChange(false),
  });

  if (isEdit && !editable && pto) {
    return (
      <WorkPane
        open={open}
        onOpenChange={onOpenChange}
        title={t("Time off request")}
        contentKey={pto.id ?? undefined}
      >
        <WorkPaneBody>
          <ReadOnlyPTO pto={opened ?? pto} />
        </WorkPaneBody>
        <WorkPaneFooter>
          <WorkPaneCancel>{t("Close")}</WorkPaneCancel>
        </WorkPaneFooter>
      </WorkPane>
    );
  }

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={isEdit ? t("Edit time off request") : t("Request time off")}
      description={
        isEdit
          ? t("Change the days, type or reason while the request waits for a decision.")
          : t(
              "Recorded on their behalf. It waits for a decision like a request they send from Dash.",
            )
      }
      contentKey={pto?.id ?? "new"}
    >
      <WorkPaneDirtyTracker form={form} />
      <FormProvider {...form}>
        <Form
          onSubmit={(event) => {
            event.preventDefault();
            event.stopPropagation();
            void submit();
          }}
        >
          <WorkPaneBody>
            <PTOFormFields
              control={form.control}
              workerField="hidden"
              excludePtoId={pto?.id ?? undefined}
              renderPreview={(state) => <DayCount {...state} />}
            />
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={isSubmitting}>
              {isEdit ? t("Save changes") : t("Request time off")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}
