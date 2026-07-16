import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FormControl, FormGroup } from "@/components/ui/form";
import { graphQLErrorMessage } from "@/lib/graphql";
import { addOrderCharge, updateOrderCharge, type OrderCharge } from "@/lib/graphql/order";
import { orderChargeFormSchema, type OrderChargeFormValues } from "@/types/order";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useEffect } from "react";
import { type Resolver, useForm } from "react-hook-form";
import { toast } from "sonner";
import { useOrderInvalidation } from "./use-order-invalidation";

type AddChargeDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orderId: string;
  currency: string;
  charge?: OrderCharge | null;
};

const emptyCharge: OrderChargeFormValues = {
  description: "",
  amount: null as unknown as number,
};

export function AddChargeDialog({
  open,
  onOpenChange,
  orderId,
  currency,
  charge,
}: AddChargeDialogProps) {
  const invalidateOrders = useOrderInvalidation();
  const isEditing = !!charge;

  const form = useForm<OrderChargeFormValues>({
    resolver: zodResolver(orderChargeFormSchema) as Resolver<OrderChargeFormValues>,
    defaultValues: emptyCharge,
    mode: "onChange",
  });

  useEffect(() => {
    if (!open) return;
    form.reset(
      charge ? { description: charge.description, amount: Number(charge.amount) } : emptyCharge,
    );
  }, [open, charge, form]);

  const { mutate, isPending } = useMutation({
    mutationFn: (values: OrderChargeFormValues) =>
      charge
        ? updateOrderCharge({
            orderId,
            chargeId: charge.id,
            description: values.description.trim(),
            amount: String(values.amount),
            version: charge.version,
          })
        : addOrderCharge(orderId, values.description.trim(), String(values.amount)),
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
      <DialogContent className="sm:max-w-100">
        <DialogHeader>
          <DialogTitle>{isEditing ? "Edit Charge" : "Add Charge"}</DialogTitle>
          <DialogDescription>
            Order-level charges not tied to a single leg roll into the order total and are billed
            exactly once on the first grouped invoice.
          </DialogDescription>
        </DialogHeader>
        <FormGroup cols={1} className="pb-4">
          <FormControl>
            <InputField
              control={form.control}
              name="description"
              label="Description"
              placeholder="e.g. Customs brokerage"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={form.control}
              name="amount"
              label="Amount"
              placeholder="0.00"
              decimalScale={2}
              thousandSeparator
              sideText={currency}
            />
          </FormControl>
        </FormGroup>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            disabled={!form.formState.isValid}
            isLoading={isPending}
            loadingText={isEditing ? "Saving..." : "Adding..."}
            onClick={() => void handleSubmit()}
          >
            {isEditing ? "Save Charge" : "Add Charge"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
