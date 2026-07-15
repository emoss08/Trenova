import { EmptyState } from "@/components/empty-state";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { Button } from "@/components/ui/button";
import { FormSection } from "@/components/ui/form";
import { addOrderCharge, fetchOrderDetail, removeOrderCharge } from "@/lib/graphql/order";
import { formatCurrency } from "@/lib/utils";
import { orderChargeFormSchema, type Order, type OrderChargeFormValues } from "@/types/order";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PlusIcon, ReceiptTextIcon, Trash2Icon } from "lucide-react";
import { useCallback } from "react";
import { type Resolver, useForm, useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";

export function OrderChargesSection() {
  const { control } = useFormContext<Order>();
  const orderId = useWatch({ control, name: "id" });
  const queryClient = useQueryClient();

  const addForm = useForm<OrderChargeFormValues>({
    resolver: zodResolver(orderChargeFormSchema) as Resolver<OrderChargeFormValues>,
    defaultValues: { description: "", amount: null as unknown as number },
    mode: "onChange",
  });

  const { data: order } = useQuery({
    queryKey: ["order-detail", orderId],
    queryFn: () => fetchOrderDetail(orderId!),
    enabled: !!orderId,
  });

  const invalidateDetail = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["order-detail", orderId] });
  }, [queryClient, orderId]);

  const { mutate: addCharge, isPending: isAdding } = useMutation({
    mutationFn: (values: OrderChargeFormValues) =>
      addOrderCharge(orderId!, values.description.trim(), String(values.amount)),
    onSuccess: () => {
      addForm.reset({ description: "", amount: null as unknown as number });
      invalidateDetail();
      toast.success("Charge added");
    },
    onError: () => toast.error("Failed to add charge"),
  });

  const { mutate: removeCharge, isPending: isRemoving } = useMutation({
    mutationFn: (chargeId: string) => removeOrderCharge(orderId!, chargeId),
    onSuccess: () => {
      invalidateDetail();
      toast.success("Charge removed");
    },
    onError: () => toast.error("Failed to remove charge"),
  });

  const handleAdd = addForm.handleSubmit((values) => addCharge(values));

  if (!orderId) {
    return null;
  }

  const charges = order?.charges ?? [];
  const currency = order?.currencyCode ?? "USD";
  const canAdd = addForm.formState.isValid && !isAdding;

  return (
    <FormSection
      title="Order Charges"
      titleCount={charges.length}
      description="Order-level charges not tied to a single leg (e.g. customs brokerage). These roll into the total and the grouped invoice."
      className="border-t border-border pt-4"
    >
      {charges.length > 0 ? (
        <div className="rounded-lg border">
          <div className="grid grid-cols-12 gap-2 border-b border-border px-4 py-2 text-2xs uppercase text-muted-foreground">
            <span className="col-span-8">Description</span>
            <span className="col-span-3 text-right">Amount</span>
            <span className="col-span-1" />
          </div>
          <div className="divide-y">
            {charges.map((charge) => (
              <div key={charge.id} className="grid grid-cols-12 items-center gap-2 px-4 py-2 text-sm">
                <span className="col-span-8">{charge.description}</span>
                <span className="col-span-3 text-right tabular-nums">
                  {formatCurrency(Number(charge.amount), currency)}
                </span>
                <span className="col-span-1 flex justify-end">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    disabled={isRemoving}
                    onClick={() => removeCharge(charge.id)}
                    aria-label="Remove charge"
                  >
                    <Trash2Icon className="size-3.5 text-destructive" />
                  </Button>
                </span>
              </div>
            ))}
          </div>
        </div>
      ) : (
        <EmptyState
          className="border-bg-sidebar-border max-h-[160px] rounded-lg border p-4"
          title="No Order Charges"
          description="Add customs brokerage, order-wide fuel, or other order-level fees"
          icons={[ReceiptTextIcon]}
        />
      )}

      <div className="grid grid-cols-12 items-end gap-2">
        <div className="col-span-7">
          <InputField
            control={addForm.control}
            name="description"
            label="Description"
            placeholder="e.g. Customs brokerage"
          />
        </div>
        <div className="col-span-3">
          <NumberField
            control={addForm.control}
            name="amount"
            label="Amount"
            placeholder="0.00"
            decimalScale={2}
            thousandSeparator
            sideText={currency}
          />
        </div>
        <div className="col-span-2">
          <Button
            type="button"
            size="sm"
            className="w-full"
            disabled={!canAdd}
            onClick={() => void handleAdd()}
          >
            <PlusIcon className="mr-1.5 size-3.5" />
            Add
          </Button>
        </div>
      </div>
    </FormSection>
  );
}
