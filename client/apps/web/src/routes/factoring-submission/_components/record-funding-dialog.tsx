import { DateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { MoneyField } from "@/components/fields/money-field";
import { SegmentedField } from "@/components/fields/segmented-field";
import { FactoringMovementPreview } from "@/components/factoring/factoring-movement-preview";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { FACTORING_SUBMISSION_STATUS_LABELS } from "@/lib/factoring";
import {
  recordFactoringFunding,
  type FactoringFundingPlan,
  type FactoringSubmission,
  type FactoringSubmissionDetail,
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
import { dateToUnixTimestamp } from "@trenova/shared/lib/date";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { useMemo } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import {
  fundingFormDefaults,
  fundingFormSchema,
  fundingFormToInput,
  undecidedItems,
  type FundingFormValues,
} from "./funding-schema";

const FUNDING_FORM_ID = "record-factoring-funding-form";
const PREVIEW_DEBOUNCE_MS = 400;

/**
 * Records what the factor did with a sent schedule, from its funding report
 * or remittance. Each undecided invoice is marked funded (at the advance and
 * fee the factor actually paid), rejected (with why), or left for later; the
 * journal each funded invoice books is planned by the server as the person
 * edits, and nothing is booked until they record it.
 */
export function RecordFundingDialog({
  submission,
  open,
  onOpenChange,
}: {
  submission: FactoringSubmissionDetail;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="xl">
        <DialogHeader>
          <DialogTitle>{t("Record funding")}</DialogTitle>
          <DialogDescription>
            {t(
              "Mark each invoice on schedule {0} as the factor's report shows it. Leave an invoice undecided until the factor decides it.",
              submission.number,
            )}
          </DialogDescription>
        </DialogHeader>
        {open ? (
          <FundingForm
            key={`${submission.id}-${submission.version}`}
            submission={submission}
            onDone={() => onOpenChange(false)}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function FundingForm({
  submission,
  onDone,
}: {
  submission: FactoringSubmissionDetail;
  onDone: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const items = useMemo(() => undecidedItems(submission), [submission]);
  const form = useForm<FundingFormValues>({
    resolver: zodResolver(fundingFormSchema) as Resolver<FundingFormValues>,
    defaultValues: fundingFormDefaults(submission, dateToUnixTimestamp(new Date())),
  });
  const values = useWatch({ control: form.control }) as FundingFormValues;
  const inputKey = useMemo(() => {
    if (!fundingFormSchema.safeParse(values).success) {
      return null;
    }
    const input = fundingFormToInput(submission, values);
    return input ? JSON.stringify(input) : null;
  }, [submission, values]);
  const debouncedKey = useDebounce(inputKey, PREVIEW_DEBOUNCE_MS);
  const planInput = useMemo(
    () =>
      debouncedKey
        ? (JSON.parse(debouncedKey) as Parameters<typeof factoringQueries.fundingPlan>[0])
        : null,
    [debouncedKey],
  );
  const plan = useQuery({
    ...factoringQueries.fundingPlan(
      planInput ?? { submissionId: "", version: 0, fundedOn: 0, items: [] },
    ),
    enabled: planInput !== null,
    placeholderData: keepPreviousData,
    retry: false,
  });

  const mutation = useApiMutation<
    FactoringSubmission,
    FundingFormValues,
    unknown,
    FundingFormValues
  >({
    form,
    resourceName: "factoring funding",
    mutationFn: (submitted) => {
      const input = fundingFormToInput(submission, submitted);
      if (!input) throw new Error("Mark at least one invoice funded or rejected");
      return recordFactoringFunding(input);
    },
    onSuccess: () => {
      invalidateFactoringQueries(queryClient);
      onDone();
    },
    onError: () => {
      invalidateFactoringQueries(queryClient);
    },
  });
  const itemsError =
    form.formState.errors.items?.root?.message ?? form.formState.errors.items?.message;
  const money = (minor: number) => formatMinor(minor, submission.currencyCode);
  const previewing = inputKey !== null && inputKey === debouncedKey;

  return (
    <>
      <div className="flex max-h-[65vh] min-w-0 flex-col gap-4 overflow-y-auto">
        <FormProvider {...form}>
          <Form
            id={FUNDING_FORM_ID}
            onSubmit={(event) => {
              event.stopPropagation();
              void form.handleSubmit((submitted) => mutation.mutate(submitted))(event);
            }}
          >
            <FormGroup cols={2}>
              <FormControl>
                <DateField
                  control={form.control}
                  name="fundedOn"
                  label={t("Funded on")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={form.control}
                  name="memo"
                  label={t("Memo")}
                  placeholder={t("The factor's remittance or report number")}
                />
              </FormControl>
            </FormGroup>
            <div className="flex flex-col gap-2">
              {items.map((item, index) => (
                <FundingItemRow
                  key={item.id}
                  index={index}
                  invoiceNumber={item.invoiceNumber}
                  customerName={item.customer?.name ?? ""}
                  face={money(item.faceMinor)}
                  expected={t(
                    "Expected {0} advance, {1} fee",
                    money(item.expectedAdvanceMinor),
                    money(item.expectedFeeMinor),
                  )}
                  form={form}
                />
              ))}
            </div>
            {itemsError ? (
              <p role="alert" className="text-danger text-xs">
                {itemsError}
              </p>
            ) : null}
          </Form>
        </FormProvider>
        <section className="flex flex-col gap-3">
          <h3 className="text-sm font-medium">{t("What it books")}</h3>
          {inputKey === null ? (
            <p className="text-muted-foreground text-sm">
              {t("Mark an invoice funded or rejected to see the journal it books.")}
            </p>
          ) : (
            <FundingPlanBody query={plan} currencyCode={submission.currencyCode} />
          )}
        </section>
      </div>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onDone}>
          {t("Cancel")}
        </Button>
        <Button
          type="submit"
          form={FUNDING_FORM_ID}
          isLoading={mutation.isPending}
          disabled={!previewing || !plan.isSuccess || plan.isPlaceholderData}
        >
          {t("Record funding")}
        </Button>
      </DialogFooter>
    </>
  );
}

function FundingItemRow({
  index,
  invoiceNumber,
  customerName,
  face,
  expected,
  form,
}: {
  index: number;
  invoiceNumber: string;
  customerName: string;
  face: string;
  expected: string;
  form: ReturnType<typeof useForm<FundingFormValues>>;
}) {
  const t = useT();
  const outcome = useWatch({ control: form.control, name: `items.${index}.outcome` });

  return (
    <div className="flex flex-col gap-2 rounded-md border p-3">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <div className="flex flex-col">
          <span className="font-mono text-sm font-medium">{invoiceNumber}</span>
          <span className="text-muted-foreground text-xs">{customerName}</span>
        </div>
        <div className="flex flex-col items-end">
          <span className="text-sm tabular-nums">{face}</span>
          <span className="text-muted-foreground text-xs">{expected}</span>
        </div>
      </div>
      <FormGroup cols={outcome === "Undecided" ? 1 : 3}>
        <FormControl>
          <SegmentedField
            control={form.control}
            name={`items.${index}.outcome`}
            label={t("Outcome")}
            options={[
              { value: "Funded", label: t("Funded") },
              { value: "Rejected", label: t("Rejected") },
              { value: "Undecided", label: t("Not decided") },
            ]}
          />
        </FormControl>
        {outcome === "Funded" ? (
          <>
            <FormControl>
              <MoneyField
                control={form.control}
                name={`items.${index}.advanceMinor`}
                label={t("Advance paid")}
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <MoneyField
                control={form.control}
                name={`items.${index}.feeMinor`}
                label={t("Fee kept")}
                rules={{ required: true }}
              />
            </FormControl>
          </>
        ) : null}
        {outcome === "Rejected" ? (
          <FormControl cols={2}>
            <InputField
              control={form.control}
              name={`items.${index}.reason`}
              label={t("Why the factor rejected it")}
              rules={{ required: true }}
            />
          </FormControl>
        ) : null}
      </FormGroup>
    </div>
  );
}

function FundingPlanBody({
  query,
  currencyCode,
}: {
  query: UseQueryResult<FactoringFundingPlan>;
  currencyCode: string;
}) {
  const t = useT();

  if (query.isLoading) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }

  if (query.error || !query.data) {
    const described = query.error ? describeError(query.error) : null;
    return (
      <Alert size="sm" variant="destructive">
        <AlertDescription>
          {described?.detail ?? t("The funding could not be planned. Try again in a moment.")}
        </AlertDescription>
      </Alert>
    );
  }

  const plan = query.data;
  const money = (minor: number) => formatMinor(minor, currencyCode);

  return (
    <div className="flex flex-col gap-4">
      <KpiStrip aria-label={t("Funding")}>
        <KpiStripItem
          label={t("Funded")}
          value={String(plan.fundedCount)}
          sub={t("{0, plural, one {# rejected} other {# rejected}}", plan.rejectCount)}
        />
        <KpiStripItem label={t("Advance")} value={money(plan.advanceMinor)} />
        <KpiStripItem label={t("Fee")} value={money(plan.feeMinor)} />
        <KpiStripItem label={t("Reserve held")} value={money(plan.reserveMinor)} />
      </KpiStrip>
      <p className="text-muted-foreground text-xs">
        {t("The schedule reads {0} afterwards.", FACTORING_SUBMISSION_STATUS_LABELS[plan.status])}
      </p>
      {plan.movements.map((movement) => (
        <FactoringMovementPreview
          key={movement.itemId}
          movement={movement}
          currencyCode={currencyCode}
          showInvoice
        />
      ))}
    </div>
  );
}
