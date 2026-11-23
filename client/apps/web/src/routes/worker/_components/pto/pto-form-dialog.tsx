import { useT } from "@trenova/shared/i18n/use-t";
import { WorkerAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { ptoTypeChoices } from "@/lib/choices";
import { fetchWorkerPtoAvailability } from "@/lib/graphql/pto-policy";
import { createWorkerPTO, updateWorkerPTO } from "@/lib/graphql/worker-mutations";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { PTOStatusBadge } from "@trenova/shared/components/status-badge";
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
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import {
  formatRange,
  getEndOfDay,
  getStartOfDay,
  inclusiveDays,
  toDateFromUnixSeconds,
} from "@trenova/shared/lib/date";
import {
  ptoFormSchema,
  type PTOFormValues,
  type PTOType,
  type WorkerPTO,
} from "@trenova/shared/types/worker";
import { useCallback, useEffect, useMemo, type ReactNode } from "react";
import { FormProvider, useForm, useWatch, type Control, type Resolver } from "react-hook-form";
import { ptoDecision, type PTODecisionSource } from "./pto-columns";
import { usePTOInvalidation } from "./use-pto-invalidation";

export type PTOFormDialogRecord = PTODecisionSource & {
  id?: string | null;
  version?: number | null;
  workerId?: string | null;
  type: PTOType;
  startDate: number;
  endDate: number;
  reason: string;
  worker?: { firstName: string; lastName: string } | null;
};

export type PTOFormDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  pto?: PTOFormDialogRecord | null;
  defaultWorkerId?: string;
  defaultRange?: { start: number; end: number } | null;
  lockWorker?: boolean;
  onSaved?: (pto: WorkerPTO) => void;
};

export function buildDefaults(
  pto: PTOFormDialogRecord | null | undefined,
  workerId?: string,
  range?: { start: number; end: number } | null,
): PTOFormValues {
  return {
    workerId: pto?.workerId ?? workerId ?? "",
    type: pto?.type ?? "Vacation",
    startDate: pto?.startDate ?? range?.start ?? 0,
    endDate: pto?.endDate ?? range?.end ?? 0,
    reason: pto?.reason ?? "",
  };
}

export function normalizePTODates(values: PTOFormValues): PTOFormValues {
  return {
    ...values,
    startDate: getStartOfDay(toDateFromUnixSeconds(values.startDate)),
    endDate: getEndOfDay(toDateFromUnixSeconds(values.endDate)),
  };
}

export type PTOAvailabilityInput = {
  workerId: string;
  ptoType: PTOFormValues["type"];
  startDate: number;
  endDate: number;
  excludePtoId?: string;
};

/**
 * What the policy says about a request before it is sent: the days it counts and what the
 * balance has left on the start date. Settled for a moment so typing a date does not ask on
 * every keystroke.
 */
export function usePTOAvailability({
  workerId,
  ptoType,
  startDate,
  endDate,
  excludePtoId,
}: PTOAvailabilityInput) {
  const hasDates = !!startDate && !!endDate && endDate >= startDate;
  const ready = hasDates && !!workerId;
  const debounced = useDebounce({ workerId, ptoType, startDate, endDate }, 400);
  return useQuery({
    queryKey: ["worker-pto-availability", debounced, excludePtoId ?? null],
    queryFn: ({ signal }) => {
      const normalized = normalizePTODates({
        workerId: debounced.workerId,
        type: debounced.ptoType,
        startDate: debounced.startDate,
        endDate: debounced.endDate,
        reason: "",
      });
      return fetchWorkerPtoAvailability(
        {
          workerId: debounced.workerId,
          ptoType: debounced.ptoType,
          startDate: normalized.startDate,
          endDate: normalized.endDate,
          excludePtoId: excludePtoId ?? undefined,
        },
        { signal },
      );
    },
    enabled: ready && !!debounced.workerId && !!debounced.startDate && !!debounced.endDate,
    staleTime: 30 * 1000,
  });
}

function DaysPreview(input: PTOAvailabilityInput) {
  const t = useT();
  const { ptoType, startDate, endDate } = input;
  const availability = usePTOAvailability(input);

  const hasDates = !!startDate && !!endDate && endDate >= startDate;
  if (!hasDates) {
    return null;
  }
  const days = inclusiveDays(startDate, endDate);
  const result = availability.data;

  return (
    <div className="flex flex-col gap-0.5">
      <p className="text-xs text-muted-foreground" data-testid="pto-days-preview">
        {t(
          "{0, plural, one {# day} other {# days}} · {1}",
          days,
          formatRange(startDate, endDate, { year: "always", join: "words" }),
        )}
      </p>
      {result?.tracked ? (
        <p
          className={result.allowed ? "text-xs text-muted-foreground" : "text-xs text-destructive"}
          data-testid="pto-availability-hint"
        >
          {result.allowed
            ? t(
                "{0} {1} days available on the start date{2}",
                Number(result.projectedAvailableDays).toFixed(2),
                ptoType.toLowerCase(),
                result.enforced ? "" : ` ${t("(not enforced)")}`,
              )
            : result.message}
        </p>
      ) : null}
    </div>
  );
}

export function ReadOnlyPTO({ pto }: { pto: PTOFormDialogRecord }) {
  const t = useT();

  const decision = ptoDecision(pto);

  return (
    <div className="flex flex-col gap-3 pb-2 text-sm">
      <div className="flex items-center gap-2">
        <PTOStatusBadge status={pto.status} />
        <span className="text-xs text-muted-foreground">
          {t("Only requested time off can be edited.")}
        </span>
      </div>
      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1">
        <dt className="text-muted-foreground">{t("Worker")}</dt>
        <dd>
          {pto.worker?.firstName} {pto.worker?.lastName}
        </dd>
        <dt className="text-muted-foreground">{t("Type")}</dt>
        <dd>{pto.type}</dd>
        <dt className="text-muted-foreground">{t("Dates")}</dt>
        <dd className="tabular-nums">
          {t(
            "{0} ({1} days)",
            formatRange(pto.startDate, pto.endDate, { year: "always", join: "words" }),
            inclusiveDays(pto.startDate, pto.endDate),
          )}
        </dd>
        <dt className="text-muted-foreground">{t("Reason")}</dt>
        <dd>{pto.reason}</dd>
        {decision ? (
          <>
            <dt className="text-muted-foreground">{decision.verb}</dt>
            <dd>
              {decision.actor}
              {decision.note ? (
                <span className="block text-xs text-muted-foreground">{decision.note}</span>
              ) : null}
            </dd>
          </>
        ) : null}
      </dl>
    </div>
  );
}

export type UsePTOFormOptions = {
  open: boolean;
  pto?: PTOFormDialogRecord | null;
  defaultWorkerId?: string;
  defaultRange?: { start: number; end: number } | null;
  onSaved?: (pto: WorkerPTO) => void;
  /** Closes the host once the request is saved. */
  onDone: () => void;
};

/**
 * The time off form without its frame: the form, its save and whether the record can still be
 * edited. The roster's dialog and the worker page's work pane both host it.
 */
export function usePTOForm({
  open,
  pto,
  defaultWorkerId,
  defaultRange,
  onSaved,
  onDone,
}: UsePTOFormOptions) {
  const invalidate = usePTOInvalidation();
  const isEdit = !!pto?.id;
  const editable = !isEdit || pto?.status === "Requested";

  const form = useForm<PTOFormValues>({
    resolver: zodResolver(ptoFormSchema) as Resolver<PTOFormValues>,
    defaultValues: buildDefaults(pto, defaultWorkerId, defaultRange),
  });
  const { reset } = form;

  useEffect(() => {
    if (open) {
      reset(buildDefaults(pto, defaultWorkerId, defaultRange));
    }
  }, [open, pto, defaultWorkerId, defaultRange, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    WorkerPTO,
    PTOFormValues,
    unknown,
    PTOFormValues
  >({
    form,
    resourceName: "PTO",
    scope: { id: `worker-pto-form:${pto?.id ?? defaultWorkerId ?? "new"}` },
    mutationFn: async (values) => {
      const normalized = normalizePTODates(values);
      if (isEdit && pto?.id) {
        return updateWorkerPTO({
          id: pto.id,
          version: pto.version ?? 0,
          type: normalized.type,
          startDate: normalized.startDate,
          endDate: normalized.endDate,
          reason: normalized.reason,
        });
      }
      return createWorkerPTO({
        workerId: normalized.workerId,
        type: normalized.type,
        startDate: normalized.startDate,
        endDate: normalized.endDate,
        reason: normalized.reason,
      });
    },
    onSuccess: (saved) => {
      void invalidate();
      onSaved?.(saved);
      onDone();
    },
  });

  const submit = useCallback(
    () =>
      form.handleSubmit(async (values) => {
        await mutateAsync(values);
      })(),
    [form, mutateAsync],
  );

  return { form, submit, isSubmitting: isPending, isEdit, editable };
}

export type PTOFormPreviewState = PTOAvailabilityInput & { hasDateRange: boolean };

export type PTOFormFieldsProps = {
  control: Control<PTOFormValues>;
  /** Whether the worker can be picked, is shown but fixed, or is left out (the worker's own page). */
  workerField: "editable" | "locked" | "hidden";
  excludePtoId?: string;
  /** Draws what the chosen days come to; the dialog's line under the dates by default. */
  renderPreview?: (state: PTOFormPreviewState) => ReactNode;
};

function defaultPreview(state: PTOFormPreviewState): ReactNode {
  if (!state.hasDateRange) return null;
  return (
    <FormControl cols="full">
      <DaysPreview {...state} />
    </FormControl>
  );
}

/** The time off form's fields: worker, type, the two days, what they come to and the reason. */
export function PTOFormFields({
  control,
  workerField,
  excludePtoId,
  renderPreview = defaultPreview,
}: PTOFormFieldsProps) {
  const t = useT();
  const [workerId, ptoType, startDate, endDate] = useWatch({
    control,
    name: ["workerId", "type", "startDate", "endDate"],
  });
  const hasDateRange = !!startDate && !!endDate && endDate >= startDate;

  return (
    <FormGroup cols={2}>
      {workerField === "hidden" ? null : (
        <FormControl cols="full">
          <WorkerAutocompleteField<PTOFormValues>
            control={control}
            name="workerId"
            label={t("Worker")}
            required
            placeholder={t("Select worker")}
            description={t("The worker taking time off.")}
            disabled={workerField === "locked"}
          />
        </FormControl>
      )}
      <FormControl cols="full">
        <SelectField<PTOFormValues>
          control={control}
          name="type"
          label={t("Type")}
          required
          options={ptoTypeChoices}
          placeholder={t("Select type")}
          description={t("Decides which balance the days are taken from.")}
        />
      </FormControl>
      <FormControl>
        <AutoCompleteDateField<PTOFormValues>
          control={control}
          name="startDate"
          label={t("First day")}
          required
          placeholder={t("First day off")}
          description={t("The first day off; weekends count only if the policy says so.")}
        />
      </FormControl>
      <FormControl>
        <AutoCompleteDateField<PTOFormValues>
          control={control}
          name="endDate"
          label={t("Last day")}
          required
          placeholder={t("Last day off")}
          description={t("The last day off, counted inclusively.")}
        />
      </FormControl>
      {renderPreview({
        workerId,
        ptoType,
        startDate,
        endDate,
        excludePtoId,
        hasDateRange,
      })}
      <FormControl cols="full">
        <TextareaField<PTOFormValues>
          control={control}
          name="reason"
          label={t("Reason")}
          required
          placeholder={t("e.g. Family wedding")}
          maxLength={255}
          description={t("Seen by whoever reviews the request.")}
        />
      </FormControl>
    </FormGroup>
  );
}

export function PTOFormDialog({
  open,
  onOpenChange,
  pto,
  defaultWorkerId,
  defaultRange,
  lockWorker = false,
  onSaved,
}: PTOFormDialogProps) {
  const t = useT();
  const { form, submit, isSubmitting, isEdit, editable } = usePTOForm({
    open,
    pto,
    defaultWorkerId,
    defaultRange,
    onSaved,
    onDone: () => onOpenChange(false),
  });

  const title = useMemo(() => {
    if (!isEdit) return t("Request PTO");
    return editable ? t("Edit PTO Request") : t("PTO Request");
  }, [editable, isEdit, t]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? editable
                ? t(
                    "Adjust the dates, type, or reason. The worker is notified once a decision is made.",
                  )
                : t("This request has already been decided.")
              : t("Request time off on behalf of a worker. It will appear in the approval queue.")}
          </DialogDescription>
        </DialogHeader>
        {isEdit && !editable && pto ? (
          <>
            <ReadOnlyPTO pto={pto} />
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Close")}
              </Button>
            </DialogFooter>
          </>
        ) : (
          <FormProvider {...form}>
            <Form
              className="flex flex-col gap-4"
              onSubmit={(e) => {
                e.preventDefault();
                e.stopPropagation();
                void submit();
              }}
            >
              <PTOFormFields
                control={form.control}
                workerField={isEdit || lockWorker ? "locked" : "editable"}
                excludePtoId={pto?.id ?? undefined}
              />
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                  {t("Cancel")}
                </Button>
                <Button
                  type="button"
                  onClick={() => void submit()}
                  isLoading={isSubmitting}
                  loadingText={isEdit ? t("Saving...") : t("Requesting...")}
                >
                  {isEdit ? t("Save changes") : t("Request PTO")}
                </Button>
              </DialogFooter>
            </Form>
          </FormProvider>
        )}
      </DialogContent>
    </Dialog>
  );
}
