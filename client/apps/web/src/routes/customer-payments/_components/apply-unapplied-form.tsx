import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { Button } from "@/components/ui/button";
import { Form } from "@/components/ui/form";
import { FormControl, FormGroup } from "@/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  allocateBudget,
  computeApplicationTotals,
  openItemToApplicationRow,
  toMinor,
} from "@/lib/cash-application";
import { getTodayDate } from "@/lib/date";
import type { CustomerPaymentDetail } from "@/lib/graphql/customer-payment";
import { applyUnappliedCustomerPayment } from "@/lib/graphql/customer-payment";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@/lib/utils";
import {
  applyUnappliedSchema,
  type ApplyUnappliedFormValues,
} from "@/types/customer-payment";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeftIcon } from "lucide-react";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { CashApplicationEditor } from "./cash-application-editor";

export function ApplyUnappliedForm({
  payment,
  onBack,
  onDone,
}: {
  payment: CustomerPaymentDetail;
  onBack: () => void;
  onDone: () => void;
}) {
  const queryClient = useQueryClient();
  const budgetMinor = payment.unappliedAmountMinor;

  const form = useForm<ApplyUnappliedFormValues>({
    resolver: zodResolver(applyUnappliedSchema) as Resolver<ApplyUnappliedFormValues>,
    defaultValues: {
      accountingDate: getTodayDate(),
      applications: [],
    },
  });
  const {
    setError,
    setValue,
    getValues,
    handleSubmit,
    formState: { isSubmitting },
  } = form;

  const { data: openItems, isLoading: itemsLoading } = useQuery(
    queries.ar.openItems({ customerId: payment.customerId }),
  );

  useEffect(() => {
    if (!openItems) return;
    setValue(
      "applications",
      openItems.map((item) => openItemToApplicationRow(item, false)),
    );
  }, [openItems, setValue]);

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: ApplyUnappliedFormValues) => {
      const applications = values.applications
        .filter(
          (row) => row.checked && (toMinor(row.appliedAmount) > 0 || toMinor(row.shortPayAmount) > 0),
        )
        .map((row) => ({
          invoiceId: row.invoiceId,
          appliedAmountMinor: toMinor(row.appliedAmount),
          shortPayAmountMinor: toMinor(row.shortPayAmount),
        }));
      return applyUnappliedCustomerPayment({
        paymentId: payment.id,
        accountingDate: values.accountingDate,
        applications,
      });
    },
    onSuccess: () => {
      toast.success("Unapplied cash applied", {
        description: "Invoice balances and the GL were updated.",
      });
      void queryClient.invalidateQueries({ queryKey: ["customer-payment-list"] });
      void queryClient.invalidateQueries({
        queryKey: queries.customerPayment.detail(payment.id).queryKey,
      });
      void queryClient.invalidateQueries({ queryKey: queries.ar._def });
      onDone();
    },
    setFormError: setError,
    resourceName: "Customer Payment",
  });

  const onSubmit = async (values: ApplyUnappliedFormValues) => {
    const totals = computeApplicationTotals(values.applications, budgetMinor);
    if (totals.appliedMinor <= 0) {
      toast.error("Nothing to apply", {
        description: "Check at least one invoice and enter an applied amount.",
      });
      return;
    }
    if (totals.isOverBudget || totals.overAppliedRows.length > 0) {
      toast.error("Invalid application", {
        description: totals.isOverBudget
          ? "The applied total exceeds the unapplied cash on this payment."
          : "One or more invoices would be over-applied.",
      });
      return;
    }
    await mutateAsync(values);
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <Button type="button" variant="ghost" size="sm" onClick={onBack} className="h-7 text-xs">
          <ArrowLeftIcon className="size-3.5" />
          Back to payment
        </Button>
        <p className="text-xs text-muted-foreground">
          Unapplied cash available:{" "}
          <span className="font-semibold text-foreground tabular-nums">
            {formatCurrency(budgetMinor / 100)}
          </span>
        </p>
      </div>

      <FormProvider {...form}>
        <Form id="apply-unapplied-form" onSubmit={handleSubmit(onSubmit)}>
          <div className="flex flex-col gap-4">
            <FormGroup cols={2}>
              <FormControl>
                <AutoCompleteDateField
                  control={form.control}
                  name="accountingDate"
                  label="Accounting Date"
                  rules={{ required: "Accounting date is required" }}
                  placeholder="Select date"
                  description="The GL date for this application. It must fall within an open fiscal period."
                />
              </FormControl>
            </FormGroup>

            <CashApplicationEditor
              budgetMinor={budgetMinor}
              budgetLabel="Unapplied"
              onAutoApply={() =>
                setValue("applications", allocateBudget(getValues("applications"), budgetMinor), {
                  shouldDirty: true,
                })
              }
              isLoadingItems={itemsLoading}
              emptyMessage="This customer has no open invoices to apply against."
            />

            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={onBack} disabled={isSubmitting}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isSubmitting}>
                Apply Cash
              </Button>
            </div>
          </div>
        </Form>
      </FormProvider>
    </div>
  );
}
