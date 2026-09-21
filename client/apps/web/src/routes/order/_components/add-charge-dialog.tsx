"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { ChargeSplitEditor } from "@/components/billing/charge-split-editor";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { toChargeAllocationInput } from "@trenova/shared/lib/charge-split";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { addOrderCharge, updateOrderCharge, type OrderCharge } from "@/lib/graphql/order";
import { orderChargeFormSchema, type OrderChargeFormValues } from "@trenova/shared/types/order";
import type { ChargeAllocation } from "@trenova/shared/types/shipment";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useEffect } from "react";
import {
  type Control,
  type FieldValues,
  FormProvider,
  type Resolver,
  useForm,
  useWatch,
} from "react-hook-form";
import { toast } from "sonner";
import { useOrderInvalidation } from "./use-order-invalidation";

type AddChargeDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orderId: string;
  currency: string;
  charge?: OrderCharge | null;
  /** The order's customer, who pays any charge that is not split. */
  customer?: { id: string; name: string } | null;
};

const emptyCharge: OrderChargeFormValues = {
  description: "",
  amount: null as unknown as number,
  allocations: [],
};

/** GraphQL hands back decimal strings; the form works in numbers. */
function toFormAllocations(charge: OrderCharge): ChargeAllocation[] {
  return (charge.allocations ?? []).map(
    (row) =>
      ({
        id: row.id,
        billToCustomerId: row.billToCustomerId,
        method: row.method,
        percent: row.percent == null ? null : Number(row.percent),
        amount: row.amount == null ? null : Number(row.amount),
        sequence: row.sequence ?? 0,
        version: row.version ?? undefined,
        billToCustomer: row.billToCustomer
          ? {
              id: row.billToCustomer.id,
              name: row.billToCustomer.name,
              code: row.billToCustomer.code,
            }
          : null,
      }) as ChargeAllocation,
  );
}

export function AddChargeDialog({
  open,
  onOpenChange,
  orderId,
  currency,
  charge,
  customer,
}: AddChargeDialogProps) {
  const t = useT();

  const invalidateOrders = useOrderInvalidation();
  const isEditing = !!charge;

  const form = useForm<OrderChargeFormValues>({
    resolver: zodResolver(orderChargeFormSchema) as Resolver<OrderChargeFormValues>,
    defaultValues: emptyCharge,
    mode: "onChange",
  });
  const amount = useWatch({ control: form.control, name: "amount" });

  useEffect(() => {
    if (!open) return;
    form.reset(
      charge
        ? {
            description: charge.description,
            amount: Number(charge.amount),
            allocations: toFormAllocations(charge),
          }
        : emptyCharge,
    );
  }, [open, charge, form]);

  const { mutate, isPending } = useMutation({
    mutationFn: (values: OrderChargeFormValues) => {
      const allocations = values.allocations.map(toChargeAllocationInput);
      return charge
        ? updateOrderCharge({
            orderId,
            chargeId: charge.id,
            description: values.description.trim(),
            amount: String(values.amount),
            version: charge.version,
            allocations,
          })
        : addOrderCharge(
            orderId,
            values.description.trim(),
            String(values.amount),
            allocations.length > 0 ? allocations : undefined,
          );
    },
    onSuccess: () => {
      invalidateOrders();
      toast.success(isEditing ? "Charge updated" : "Charge added");
      onOpenChange(false);
    },
    onError: (error) =>
      toast.error(isEditing ? "Failed to update charge" : "Failed to add charge", {
        description: graphQLErrorMessage(
          error,
          isEditing ? "The charge could not be updated." : "The charge could not be added.",
        ),
      }),
  });

  const handleSubmit = form.handleSubmit((values) => mutate(values));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{isEditing ? t("Edit charge") : t("Add charge")}</DialogTitle>
          <DialogDescription>
            {t(
              "Order-level charges not tied to a single leg roll into the order total and are billed exactly once on the first grouped invoice.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <FormGroup cols={1}>
            <FormControl>
              <InputField
                control={form.control}
                name="description"
                label={t("Description")}
                placeholder={t("e.g. Customs brokerage")}
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={form.control}
                name="amount"
                label={t("Amount")}
                placeholder="0.00"
                decimalScale={2}
                thousandSeparator
                sideText={currency}
              />
            </FormControl>
            <FormControl>
              <ChargeSplitEditor
                control={form.control as unknown as Control<FieldValues>}
                name="allocations"
                chargeAmount={typeof amount === "number" && Number.isFinite(amount) ? amount : null}
                currencyCode={currency}
                defaultPayer={customer ? { id: customer.id, label: customer.name } : null}
              />
            </FormControl>
          </FormGroup>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            disabled={!form.formState.isValid}
            isLoading={isPending}
            loadingText={isEditing ? "Saving..." : "Adding..."}
            onClick={() => void handleSubmit()}
          >
            {isEditing ? t("Save charge") : t("Add charge")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
