import { FactoringVerdictBadge } from "@/components/factoring/factoring-badges";
import { FactoringChecksList } from "@/components/factoring/factoring-checks-list";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { SectionPanel } from "@/components/section-panel";
import { recordPath } from "@/config/record-links";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  INVOICE_FACTORING_OVERRIDE_LABELS,
  INVOICE_FACTORING_OVERRIDE_VALUES,
} from "@/lib/factoring";
import {
  setInvoiceFactoringOverride,
  type FactoringInvoiceState,
  type InvoiceFactoring,
} from "@/lib/graphql/factoring";
import { factoring as factoringQueries, invalidateFactoringQueries } from "@/lib/queries/factoring";
import { invalidateInvoiceQueries } from "@/lib/queries/invoice";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { LinkExternal01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import type { Invoice } from "@trenova/shared/types/invoice";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { Link } from "react-router";
import { toast } from "sonner";
import { z } from "zod";

const OVERRIDE_FORM_ID = "invoice-factoring-override-form";
const MAX_OVERRIDE_REASON = 500;

const overrideSchema = z
  .object({
    override: z.enum(["Default", "Exclude"]),
    reason: z.string().max(MAX_OVERRIDE_REASON, {
      error: () => translate("Keep the reason under 500 characters"),
    }),
  })
  .superRefine((values, ctx) => {
    if (values.override === "Exclude" && values.reason.trim() === "") {
      ctx.addIssue({
        code: "custom",
        path: ["reason"],
        message: translate("Say why this invoice stays out of factoring"),
      });
    }
  });

type OverrideValues = z.infer<typeof overrideSchema>;

/**
 * Whether an invoice is sold to a factor: the factor and agreement it was
 * assigned when it posted, whether it is ready, the checks it fails, and the
 * control that keeps it on the organization's own receivables. Shown only
 * for an invoice assigned to a factor or kept out of one, the only invoices
 * readiness is worked out for.
 */
export function InvoiceFactoringSection({ invoice }: { invoice: Invoice }) {
  if (!invoice.factoringCompanyId && invoice.factoringOverride !== "Exclude") {
    return null;
  }
  return <InvoiceFactoring invoice={invoice} />;
}

function InvoiceFactoring({ invoice }: { invoice: Invoice }) {
  const t = useT();
  const { data, isLoading } = useQuery(factoringQueries.invoice(invoice.id));

  if (isLoading) {
    return (
      <SectionPanel title={t("Factoring")}>
        <div className="space-y-2 p-3">
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-4 w-64" />
        </div>
      </SectionPanel>
    );
  }

  if (!data || (!data.factoringCompany && !data.factoringState)) {
    return null;
  }

  return <FactoringPanel factoring={data} />;
}

function FactoringPanel({ factoring }: { factoring: InvoiceFactoring }) {
  const t = useT();
  const state = factoring.factoringState;

  return (
    <SectionPanel
      title={t("Factoring")}
      count={state?.failedChecks}
      action={state ? <FactoringVerdictBadge verdict={state.verdict} /> : null}
      help={t(
        "An invoice assigned to a factor is remitted to it and carries its notice of assignment. It can be sold once every check passes.",
      )}
    >
      <div className="flex flex-col gap-3 p-3">
        <DescriptionList columns={2}>
          <DescriptionItem label={t("Factor")}>
            {factoring.factoringCompany ? (
              <Link
                to={recordPath("factoring_company", factoring.factoringCompany.id)}
                className="inline-flex items-center gap-1 hover:underline"
              >
                {factoring.factoringCompany.name}
                <LinkExternal01Icon className="size-2.5" />
              </Link>
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
          <DescriptionItem label={t("Agreement")}>
            {factoring.factoringAgreementId ? (
              <Link
                to={recordPath("factoring_agreement", factoring.factoringAgreementId)}
                className="inline-flex items-center gap-1 hover:underline"
              >
                {t("View agreement")}
                <LinkExternal01Icon className="size-2.5" />
              </Link>
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
          <DescriptionItem label={t("Ready since")}>
            {state?.readySince ? formatUnixDateTimeShort(state.readySince) : <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Checked")}>
            {state ? formatUnixDateTimeShort(state.evaluatedAt) : t("Not yet")}
          </DescriptionItem>
        </DescriptionList>
      </div>
      {state ? <FailedChecks state={state} invoiceNumber={factoring.number} /> : null}
      <OverrideForm factoring={factoring} />
    </SectionPanel>
  );
}

function FailedChecks({
  state,
  invoiceNumber,
}: {
  state: FactoringInvoiceState;
  invoiceNumber: string;
}) {
  if (state.verdict === "Excluded") {
    return null;
  }
  return (
    <div className="border-t">
      <FactoringChecksList state={state} invoiceNumber={invoiceNumber} showPassed={false} />
    </div>
  );
}

function OverrideForm({ factoring }: { factoring: InvoiceFactoring }) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canUpdate } = usePermission(Resource.Invoice, Operation.Update);
  const locked = factoring.status === "Voided" || factoring.billType === "CreditMemo";

  const form = useForm<OverrideValues>({
    resolver: zodResolver(overrideSchema) as Resolver<OverrideValues>,
    defaultValues: { override: factoring.factoringOverride, reason: "" },
  });
  const { reset } = form;

  useEffect(() => {
    reset({ override: factoring.factoringOverride, reason: "" });
  }, [factoring.id, factoring.factoringOverride, reset]);

  const override = useWatch({ control: form.control, name: "override" });

  const mutation = useApiMutation<FactoringInvoiceState, OverrideValues, unknown, OverrideValues>({
    form,
    resourceName: "invoice factoring",
    mutationFn: (values) =>
      setInvoiceFactoringOverride({
        invoiceId: factoring.id,
        override: values.override,
        reason: values.override === "Exclude" ? values.reason.trim() : null,
      }),
    onSuccess: (_state, values) => {
      invalidateFactoringQueries(queryClient);
      invalidateInvoiceQueries(queryClient);
      toast.success(
        values.override === "Exclude"
          ? t("{0} is kept out of factoring", factoring.number)
          : t("{0} follows the customer's factoring again", factoring.number),
      );
    },
  });

  if (!canUpdate || locked) {
    return (
      <div className="border-t p-3">
        <DescriptionList layout="inline">
          <DescriptionItem label={t("Factoring")}>
            {INVOICE_FACTORING_OVERRIDE_LABELS[factoring.factoringOverride]}
          </DescriptionItem>
          {factoring.factoringOverrideReason ? (
            <DescriptionItem label={t("Why it stays out")}>
              {factoring.factoringOverrideReason}
            </DescriptionItem>
          ) : null}
        </DescriptionList>
      </div>
    );
  }

  const options = INVOICE_FACTORING_OVERRIDE_VALUES.map((value) => ({
    value,
    label: INVOICE_FACTORING_OVERRIDE_LABELS[value],
  }));
  const changed = override !== factoring.factoringOverride;

  return (
    <div className="border-t p-3">
      <FormProvider {...form}>
        <Form
          id={OVERRIDE_FORM_ID}
          onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        >
          <FormGroup cols={1}>
            <FormControl>
              <SelectField
                control={form.control}
                name="override"
                label={t("Factoring")}
                description={t(
                  "Keep this invoice out of factoring to collect it yourself even though its customer is factored.",
                )}
                options={options}
              />
            </FormControl>
            {override === "Exclude" && changed ? (
              <FormControl>
                <TextareaField
                  control={form.control}
                  name="reason"
                  label={t("Reason")}
                  placeholder={t("Why this invoice stays on your own receivables")}
                  rows={2}
                  rules={{ required: true }}
                />
              </FormControl>
            ) : null}
          </FormGroup>
          {!changed && factoring.factoringOverrideReason ? (
            <DescriptionList layout="inline" className="mt-2">
              <DescriptionItem label={t("Why it stays out")}>
                {factoring.factoringOverrideReason}
              </DescriptionItem>
            </DescriptionList>
          ) : null}
        </Form>
      </FormProvider>
      {changed ? (
        <div className="mt-3 flex justify-end gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => reset({ override: factoring.factoringOverride, reason: "" })}
          >
            {t("Cancel")}
          </Button>
          <Button type="submit" size="sm" form={OVERRIDE_FORM_ID} isLoading={mutation.isPending}>
            {t("Save")}
          </Button>
        </div>
      ) : null}
    </div>
  );
}
