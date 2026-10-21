import { CheckboxField } from "@/components/fields/checkbox-field";
import { DateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { MoneyField } from "@/components/fields/money-field";
import { FactoringMovementPreview } from "@/components/factoring/factoring-movement-preview";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { FACTORING_SUBMISSION_ITEM_STATUS_LABELS } from "@/lib/factoring";
import {
  bookFactoringReserveMovement,
  type FactoringReserveAction,
  type FactoringReserveMovementPlan,
  type FactoringSubmission,
  type FactoringSubmissionDetail,
  type FactoringSubmissionItem,
} from "@/lib/graphql/factoring";
import { factoring as factoringQueries, invalidateFactoringQueries } from "@/lib/queries/factoring";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  keepPreviousData,
  useQuery,
  useQueryClient,
  type UseQueryResult,
} from "@tanstack/react-query";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
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
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinor } from "@trenova/shared/lib/benefits";
import { dateToUnixTimestamp, formatUnixDateMedium } from "@trenova/shared/lib/date";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { useMemo, useState } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import {
  reserveActionNeedsReason,
  reserveActionTakesCash,
  reserveMovementFormSchema,
  reserveMovementFormToInput,
  type ReserveMovementFormValues,
} from "./funding-schema";

const MOVEMENT_FORM_ID = "factoring-reserve-movement-form";
const PREVIEW_DEBOUNCE_MS = 400;

export type ReserveMovementTarget = {
  item: FactoringSubmissionItem;
  action: FactoringReserveAction;
};

/** What the factor would be owed or would pay by default, before a person says otherwise. */
export function defaultMovementCash(
  item: FactoringSubmissionItem,
  action: FactoringReserveAction,
): number {
  switch (action) {
    case "release":
      return item.reserveMinor ?? 0;
    case "chargeback":
      return (item.advanceMinor ?? 0) + (item.feeMinor ?? 0);
    case "reverse":
      return 0;
  }
}

/**
 * Books one movement on a funded invoice: the factor releasing its reserve,
 * a recourse factor charging the invoice back, or undoing the last
 * chargeback. The server plans the journal as the person edits, and nothing
 * is booked until they confirm it.
 */
export function ReserveMovementDialog({
  submission,
  target,
  onOpenChange,
}: {
  submission: FactoringSubmissionDetail;
  target: ReserveMovementTarget | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const copy = target ? movementCopy(target, t) : null;

  return (
    <Dialog open={target !== null} onOpenChange={onOpenChange}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{copy?.title}</DialogTitle>
          <DialogDescription>{copy?.description}</DialogDescription>
        </DialogHeader>
        {target ? (
          <MovementForm
            key={`${target.item.id}-${target.action}-${submission.version}`}
            submission={submission}
            target={target}
            onDone={() => onOpenChange(false)}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function movementCopy(target: ReserveMovementTarget, t: ReturnType<typeof useT>) {
  const invoice = target.item.invoiceNumber;
  switch (target.action) {
    case "release":
      return {
        title: t("Release reserve"),
        description: t(
          "Book the factor's release of invoice {0}'s reserve. Whatever the factor kept beyond what it paid is booked as its fee, and the invoice's factoring is then settled.",
          invoice,
        ),
        submit: t("Release reserve"),
        cashToggle: t("The factor paid a different amount"),
        cashLabel: t("Amount the factor paid"),
        reasonLabel: t("Memo"),
      };
    case "chargeback":
      return {
        title: t("Charge back"),
        description: t(
          "Book the factor's chargeback of invoice {0}. The organization repays the factor, the factor gives up the reserve, and the invoice is the organization's to collect again.",
          invoice,
        ),
        submit: t("Record chargeback"),
        cashToggle: t("A different amount was repaid"),
        cashLabel: t("Amount repaid to the factor"),
        reasonLabel: t("Why the factor charged it back"),
      };
    case "reverse":
      return {
        title: t("Reverse chargeback"),
        description: t(
          "Undo invoice {0}'s last chargeback with the opposite entry, when the factor withdrew it or it was recorded by mistake. The invoice goes back to funded with the factor.",
          invoice,
        ),
        submit: t("Reverse chargeback"),
        cashToggle: "",
        cashLabel: "",
        reasonLabel: t("Why the chargeback is reversed"),
      };
  }
}

function MovementForm({
  submission,
  target,
  onDone,
}: {
  submission: FactoringSubmissionDetail;
  target: ReserveMovementTarget;
  onDone: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const copy = movementCopy(target, t);
  const [todayUnix] = useState(() => dateToUnixTimestamp(new Date()));
  const form = useForm<ReserveMovementFormValues>({
    resolver: zodResolver(reserveMovementFormSchema) as Resolver<ReserveMovementFormValues>,
    defaultValues: {
      action: target.action,
      occurredOn: todayUnix,
      overrideCash: false,
      cashMinor: defaultMovementCash(target.item, target.action),
      reason: "",
    },
  });
  const values = useWatch({ control: form.control }) as ReserveMovementFormValues;
  const inputKey = useMemo(() => {
    if (!reserveMovementFormSchema.safeParse(values).success) {
      return null;
    }
    const input = reserveMovementFormToInput(submission, target.item.id, values);
    return input ? JSON.stringify(input) : null;
  }, [submission, target.item.id, values]);
  const debouncedKey = useDebounce(inputKey, PREVIEW_DEBOUNCE_MS);
  const planInput = useMemo(
    () =>
      debouncedKey
        ? (JSON.parse(debouncedKey) as Parameters<typeof factoringQueries.movementPlan>[1])
        : null,
    [debouncedKey],
  );
  const plan = useQuery({
    ...factoringQueries.movementPlan(
      target.action,
      planInput ?? {
        submissionId: submission.id,
        itemId: target.item.id,
        version: submission.version,
        occurredOn: 0,
      },
    ),
    enabled: planInput !== null,
    placeholderData: keepPreviousData,
    retry: false,
  });

  const mutation = useApiMutation<
    FactoringSubmission,
    ReserveMovementFormValues,
    unknown,
    ReserveMovementFormValues
  >({
    form,
    resourceName: "factoring reserve",
    mutationFn: (submitted) => {
      const input = reserveMovementFormToInput(submission, target.item.id, submitted);
      if (!input) throw new Error("Enter the day it happened");
      return bookFactoringReserveMovement(target.action, input);
    },
    onSuccess: () => {
      invalidateFactoringQueries(queryClient);
      onDone();
    },
    onError: () => {
      invalidateFactoringQueries(queryClient);
    },
  });
  const overrideCash = useWatch({ control: form.control, name: "overrideCash" });
  const takesCash = reserveActionTakesCash(target.action);
  const needsReason = reserveActionNeedsReason(target.action);
  const previewing = inputKey !== null && inputKey === debouncedKey;
  const money = (minor: number) => formatMinor(minor, submission.currencyCode);

  return (
    <>
      <div className="flex max-h-[65vh] min-w-0 flex-col gap-4 overflow-y-auto">
        <FormProvider {...form}>
          <Form
            id={MOVEMENT_FORM_ID}
            onSubmit={(event) => {
              event.stopPropagation();
              void form.handleSubmit((submitted) => mutation.mutate(submitted))(event);
            }}
          >
            <FormGroup cols={2}>
              <FormControl>
                <DateField
                  control={form.control}
                  name="occurredOn"
                  label={t("Date")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={form.control}
                  name="reason"
                  label={copy.reasonLabel}
                  rules={{ required: needsReason }}
                />
              </FormControl>
              {takesCash ? (
                <FormControl cols="full">
                  <CheckboxField
                    control={form.control}
                    name="overrideCash"
                    label={copy.cashToggle}
                    description={t(
                      "Otherwise {0}.",
                      money(defaultMovementCash(target.item, target.action)),
                    )}
                  />
                </FormControl>
              ) : null}
              {takesCash && overrideCash ? (
                <FormControl>
                  <MoneyField
                    control={form.control}
                    name="cashMinor"
                    label={copy.cashLabel}
                    rules={{ required: true }}
                  />
                </FormControl>
              ) : null}
            </FormGroup>
          </Form>
        </FormProvider>
        <section className="flex flex-col gap-3">
          <h3 className="text-sm font-medium">{t("What it books")}</h3>
          {inputKey === null ? (
            <p className="text-muted-foreground text-sm">
              {t("Fill in the form to see the journal it books.")}
            </p>
          ) : (
            <MovementPlanBody query={plan} currencyCode={submission.currencyCode} />
          )}
        </section>
      </div>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onDone}>
          {t("Cancel")}
        </Button>
        <Button
          type="submit"
          form={MOVEMENT_FORM_ID}
          variant={target.action === "chargeback" ? "destructive" : "default"}
          isLoading={mutation.isPending}
          disabled={!previewing || !plan.isSuccess || plan.isPlaceholderData}
        >
          {copy.submit}
        </Button>
      </DialogFooter>
    </>
  );
}

function MovementPlanBody({
  query,
  currencyCode,
}: {
  query: UseQueryResult<FactoringReserveMovementPlan>;
  currencyCode: string;
}) {
  const t = useT();

  if (query.isLoading) {
    return <Skeleton className="h-32 w-full" />;
  }

  if (query.error || !query.data) {
    const described = query.error ? describeError(query.error) : null;
    return (
      <Alert size="sm" variant="destructive">
        <AlertDescription>
          {described?.detail ?? t("The entry could not be planned. Try again in a moment.")}
        </AlertDescription>
      </Alert>
    );
  }

  const plan = query.data;

  return (
    <div className="flex flex-col gap-3">
      {plan.reverses ? (
        <p className="text-muted-foreground text-xs">
          {t(
            "Reverses the chargeback of {0}, {1} repaid to the factor.",
            formatUnixDateMedium(plan.reverses.occurredOn),
            formatMinor(-plan.reverses.cashMinor, currencyCode),
          )}
        </p>
      ) : null}
      {plan.movement ? (
        <FactoringMovementPreview movement={plan.movement} currencyCode={currencyCode} />
      ) : (
        <p className="text-muted-foreground text-sm">
          {t("Nothing is booked: the factor held no reserve and paid nothing.")}
        </p>
      )}
      <p className="text-muted-foreground text-xs">
        {t(
          "The invoice reads {0} on the schedule afterwards.",
          FACTORING_SUBMISSION_ITEM_STATUS_LABELS[plan.itemStatus],
        )}
      </p>
    </div>
  );
}
