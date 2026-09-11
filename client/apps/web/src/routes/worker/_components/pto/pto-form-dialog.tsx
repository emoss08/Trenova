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
import { useCallback, useEffect, useMemo } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
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

function DaysPreview({
  workerId,
  ptoType,
  startDate,
  endDate,
  excludePtoId,
}: {
  workerId: string;
  ptoType: PTOFormValues["type"];
  startDate: number;
  endDate: number;
  excludePtoId?: string;
}) {
  const t = useT();

  const hasDates = !!startDate && !!endDate && endDate >= startDate;
  const ready = hasDates && !!workerId;
  const debounced = useDebounce({ workerId, ptoType, startDate, endDate }, 400);
  const availability = useQuery({
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

  if (!hasDates) {
    return null;
  }
  const days = inclusiveDays(startDate, endDate);
  const result = availability.data;

  return (
    <div className="flex flex-col gap-0.5">
      <p className="text-muted-foreground text-xs" data-testid="pto-days-preview">
        {t("{0} day{1} · {2}", days, days === 1 ? "" : "s", formatRange(startDate, endDate))}
      </p>
      {result?.tracked ? (
        <p
          className={result.allowed ? "text-muted-foreground text-xs" : "text-destructive text-xs"}
          data-testid="pto-availability-hint"
        >
          {result.allowed
            ? `${Number(result.projectedAvailableDays).toFixed(2)} ${ptoType.toLowerCase()} days available on the start date${result.enforced ? "" : " (not enforced)"}`
            : result.message}
        </p>
      ) : null}
    </div>
  );
}

function ReadOnlyPTO({ pto }: { pto: PTOFormDialogRecord }) {
  const t = useT();

  const decision = ptoDecision(pto);

  return (
    <div className="flex flex-col gap-3 pb-2 text-sm">
      <div className="flex items-center gap-2">
        <PTOStatusBadge status={pto.status} />
        <span className="text-muted-foreground text-xs">
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
          {t("{0} ({1} days)", formatRange(pto.startDate, pto.endDate), inclusiveDays(pto.startDate, pto.endDate))}
        </dd>
        <dt className="text-muted-foreground">{t("Reason")}</dt>
        <dd>{pto.reason}</dd>
        {decision ? (
          <>
            <dt className="text-muted-foreground">{decision.verb}</dt>
            <dd>
              {decision.actor}
              {decision.note ? (
                <span className="text-muted-foreground block text-xs">{decision.note}</span>
              ) : null}
            </dd>
          </>
        ) : null}
      </dl>
    </div>
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

  const invalidate = usePTOInvalidation();
  const isEdit = !!pto?.id;
  const editable = !isEdit || pto?.status === "Requested";

  const form = useForm<PTOFormValues>({
    resolver: zodResolver(ptoFormSchema) as Resolver<PTOFormValues>,
    defaultValues: buildDefaults(pto, defaultWorkerId, defaultRange),
  });
  const {
    control,
    handleSubmit,
    reset,
    formState: { isSubmitting },
  } = form;

  useEffect(() => {
    if (open) {
      reset(buildDefaults(pto, defaultWorkerId, defaultRange));
    }
  }, [open, pto, defaultWorkerId, defaultRange, reset]);

  const [workerId, ptoType, startDate, endDate] = useWatch({
    control,
    name: ["workerId", "type", "startDate", "endDate"],
  });
  const hasDateRange = !!startDate && !!endDate && endDate >= startDate;

  const { mutateAsync } = useApiMutation<WorkerPTO, PTOFormValues, unknown, PTOFormValues>({
    form,
    resourceName: "PTO",
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
      toast.success(isEdit ? "PTO updated" : "PTO requested", {
        description: isEdit
          ? "The request has been updated."
          : "The request is waiting for approval.",
      });
      void invalidate();
      onSaved?.(saved);
      onOpenChange(false);
    },
  });

  const onSubmit = useCallback(
    async (values: PTOFormValues) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  const title = useMemo(() => {
    if (!isEdit) return "Request PTO";
    return editable ? "Edit PTO Request" : "PTO Request";
  }, [editable, isEdit]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? editable
                ? "Adjust the dates, type, or reason. The worker is notified once a decision is made."
                : "This request has already been decided."
              : "Request time off on behalf of a worker. It will appear in the approval queue."}
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
              onSubmit={(e) => {
                e.preventDefault();
                e.stopPropagation();
                void handleSubmit(onSubmit)(e);
              }}
            >
              <FormGroup className="pb-2" cols={2}>
                <FormControl cols="full">
                  <WorkerAutocompleteField<PTOFormValues>
                    control={control}
                    name="workerId"
                    label={t("Worker")}
                    rules={{ required: true }}
                    placeholder={t("Select worker")}
                    description={t("The worker taking time off.")}
                    disabled={isEdit || lockWorker}
                  />
                </FormControl>
                <FormControl cols="full">
                  <SelectField<PTOFormValues>
                    control={control}
                    name="type"
                    label={t("Type")}
                    rules={{ required: true }}
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
                    rules={{ required: true }}
                    placeholder={t("First day off")}
                    description={t("The first day off; weekends count only if the policy says so.")}
                  />
                </FormControl>
                <FormControl>
                  <AutoCompleteDateField<PTOFormValues>
                    control={control}
                    name="endDate"
                    label={t("Last day")}
                    rules={{ required: true }}
                    placeholder={t("Last day off")}
                    description={t("The last day off, counted inclusively.")}
                  />
                </FormControl>
                {hasDateRange ? (
                  <FormControl cols="full">
                    <DaysPreview
                      workerId={workerId}
                      ptoType={ptoType}
                      startDate={startDate}
                      endDate={endDate}
                      excludePtoId={pto?.id ?? undefined}
                    />
                  </FormControl>
                ) : null}
                <FormControl cols="full">
                  <TextareaField<PTOFormValues>
                    control={control}
                    name="reason"
                    label={t("Reason")}
                    rules={{ required: true }}
                    placeholder={t("e.g. Family wedding")}
                    maxLength={255}
                    description={t("Seen by whoever reviews the request.")}
                  />
                </FormControl>
              </FormGroup>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                  {t("Cancel")}
                </Button>
                <Button
                  type="button"
                  onClick={() => void handleSubmit(onSubmit)()}
                  isLoading={isSubmitting}
                  loadingText={isEdit ? "Saving..." : "Requesting..."}
                >
                  {isEdit ? "Save Changes" : "Request PTO"}
                </Button>
              </DialogFooter>
            </Form>
          </FormProvider>
        )}
      </DialogContent>
    </Dialog>
  );
}
