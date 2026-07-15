import { EmptyState } from "@/components/empty-state";
import { Button } from "@/components/ui/button";
import { FormSection } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { addOrderCharge, fetchOrderDetail, removeOrderCharge } from "@/lib/graphql/order";
import { formatCurrency } from "@/lib/utils";
import type { Order } from "@/types/order";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PlusIcon, ReceiptTextIcon, Trash2Icon } from "lucide-react";
import { useCallback, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";

export function OrderChargesSection() {
  const { control } = useFormContext<Order>();
  const orderId = useWatch({ control, name: "id" });
  const queryClient = useQueryClient();
  const [description, setDescription] = useState("");
  const [amount, setAmount] = useState("");

  const { data: order } = useQuery({
    queryKey: ["order-detail", orderId],
    queryFn: () => fetchOrderDetail(orderId!),
    enabled: !!orderId,
  });

  const invalidateDetail = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["order-detail", orderId] });
  }, [queryClient, orderId]);

  const { mutate: addCharge, isPending: isAdding } = useMutation({
    mutationFn: () => addOrderCharge(orderId!, description.trim(), amount),
    onSuccess: () => {
      setDescription("");
      setAmount("");
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

  if (!orderId) {
    return null;
  }

  const charges = order?.charges ?? [];
  const currency = order?.currencyCode ?? "USD";
  const canAdd = description.trim().length > 0 && amount.trim().length > 0 && !isAdding;

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
        <div className="col-span-7 flex flex-col gap-1">
          <label className="text-2xs text-muted-foreground">Description</label>
          <Input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="e.g. Customs brokerage"
          />
        </div>
        <div className="col-span-3 flex flex-col gap-1">
          <label className="text-2xs text-muted-foreground">Amount</label>
          <Input
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            placeholder="0.00"
            inputMode="decimal"
          />
        </div>
        <div className="col-span-2">
          <Button type="button" size="sm" className="w-full" disabled={!canAdd} onClick={() => addCharge()}>
            <PlusIcon className="mr-1.5 size-3.5" />
            Add
          </Button>
        </div>
      </div>
    </FormSection>
  );
}
