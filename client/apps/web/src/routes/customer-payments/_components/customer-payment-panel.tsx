import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { computeApplicationTotals, toMinor } from "@/lib/cash-application";
import { getTodayDate } from "@trenova/shared/lib/date";
import type { CustomerPaymentRow } from "@/lib/graphql/customer-payment";
import { postAndApplyCustomerPayment } from "@/lib/graphql/customer-payment";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  recordPaymentSchema,
  type RecordPaymentFormValues,
} from "@trenova/shared/types/customer-payment";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { parseAsString, useQueryStates } from "nuqs";
import { useCallback, useEffect, useMemo } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { PaymentDetail } from "./payment-detail";
import { RecordPaymentForm } from "./record-payment-form";

const prefillParamsParser = {
  customerId: parseAsString,
  invoiceIds: parseAsString,
};

function buildDefaults(customerId: string): RecordPaymentFormValues {
  return {
    customerId,
    paymentDate: getTodayDate(),
    accountingDate: getTodayDate(),
    amount: 0,
    paymentMethod: "ACH",
    referenceNumber: "",
    memo: "",
    applications: [],
  };
}

export function CustomerPaymentPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<CustomerPaymentRow>) {
  if (mode === "edit" && row) {
    const title = row.referenceNumber
      ? `Payment ${row.referenceNumber}`
      : `${formatCurrency(row.amountMinor / 100)} payment${
          row.customer ? ` from ${row.customer.name}` : ""
        }`;
    return (
      <DataTablePanelContainer open={open} onOpenChange={onOpenChange} title={title} size="xl">
        <PaymentDetail paymentId={row.id} onClose={() => onOpenChange(false)} />
      </DataTablePanelContainer>
    );
  }

  return <RecordPaymentPanel open={open} onOpenChange={onOpenChange} />;
}

function RecordPaymentPanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const [prefill, setPrefill] = useQueryStates(prefillParamsParser);
  const prefilledInvoiceIds = useMemo(
    () => (prefill.invoiceIds ? prefill.invoiceIds.split(",").filter(Boolean) : []),
    [prefill.invoiceIds],
  );

  const form = useForm<RecordPaymentFormValues>({
    resolver: zodResolver(recordPaymentSchema) as Resolver<RecordPaymentFormValues>,
    defaultValues: buildDefaults(prefill.customerId ?? ""),
  });
  const {
    setError,
    handleSubmit,
    reset,
    formState: { isSubmitting },
  } = form;

  useEffect(() => {
    if (open) {
      reset(buildDefaults(prefill.customerId ?? ""));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, reset]);

  const clearPrefill = useCallback(() => {
    void setPrefill({ customerId: null, invoiceIds: null });
  }, [setPrefill]);

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: RecordPaymentFormValues) => {
      const applications = values.applications
        .filter(
          (appRow) =>
            appRow.checked &&
            (toMinor(appRow.appliedAmount) > 0 || toMinor(appRow.shortPayAmount) > 0),
        )
        .map((appRow) => ({
          invoiceId: appRow.invoiceId,
          appliedAmountMinor: toMinor(appRow.appliedAmount),
          shortPayAmountMinor: toMinor(appRow.shortPayAmount),
        }));
      return postAndApplyCustomerPayment({
        customerId: values.customerId,
        paymentDate: values.paymentDate,
        accountingDate: values.accountingDate,
        amountMinor: toMinor(values.amount),
        paymentMethod: values.paymentMethod,
        referenceNumber: values.referenceNumber || undefined,
        memo: values.memo || undefined,
        applications,
      });
    },
    onSuccess: (created) => {
      toast.success("Payment posted", {
        description: `${formatCurrency(created.amountMinor / 100)} received — ${formatCurrency(
          created.appliedAmountMinor / 100,
        )} applied, ${formatCurrency(created.unappliedAmountMinor / 100)} unapplied.`,
      });
      void queryClient.invalidateQueries({ queryKey: ["customer-payment-list"] });
      void queryClient.invalidateQueries({ queryKey: queries.ar._def });
      clearPrefill();
      reset(buildDefaults(""));
      onOpenChange(false);
    },
    setFormError: setError,
    resourceName: "Customer Payment",
  });

  const onSubmit = async (values: RecordPaymentFormValues) => {
    const totals = computeApplicationTotals(values.applications, toMinor(values.amount));
    if (totals.isOverBudget) {
      toast.error("Over-applied", {
        description: "The applied total exceeds the payment amount.",
      });
      return;
    }
    if (totals.overAppliedRows.length > 0) {
      toast.error("Invalid application", {
        description: "One or more invoices would be over-applied.",
      });
      return;
    }
    await mutateAsync(values);
  };

  const handleClose = () => {
    onOpenChange(false);
    clearPrefill();
    reset(buildDefaults(""));
  };

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          clearPrefill();
        }
        onOpenChange(next);
      }}
      title="Record Payment"
      description="Post a customer payment and apply it across open invoices in one step."
      size="xl"
      footer={
        <>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" form="record-payment-form" isLoading={isSubmitting}>
            Post Payment
          </Button>
        </>
      }
    >
      <FormProvider {...form}>
        <Form id="record-payment-form" onSubmit={handleSubmit(onSubmit)}>
          <RecordPaymentForm prefilledInvoiceIds={prefilledInvoiceIds} />
        </Form>
      </FormProvider>
    </DataTablePanelContainer>
  );
}
