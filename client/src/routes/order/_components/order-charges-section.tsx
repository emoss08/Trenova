import { EmptyState } from "@/components/empty-state";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { FormSection } from "@/components/ui/form";
import { graphQLErrorMessage } from "@/lib/graphql";
import { fetchOrderDetail, removeOrderCharge, type OrderCharge } from "@/lib/graphql/order";
import { formatCurrency } from "@/lib/utils";
import type { Order } from "@/types/order";
import { useMutation, useQuery } from "@tanstack/react-query";
import { PencilIcon, PlusIcon, ReceiptTextIcon, Trash2Icon } from "lucide-react";
import { useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { AddChargeDialog } from "./add-charge-dialog";
import { useOrderInvalidation } from "./use-order-invalidation";

const ORDER_STATUSES_LOCKED_FOR_CHARGES = new Set(["Billed", "Closed", "Canceled"]);

export function OrderChargesSection() {
  const { control } = useFormContext<Order>();
  const orderId = useWatch({ control, name: "id" });
  const invalidateOrders = useOrderInvalidation();
  const [chargeDialogOpen, setChargeDialogOpen] = useState(false);
  const [chargeBeingEdited, setChargeBeingEdited] = useState<OrderCharge | null>(null);
  const [chargePendingRemoval, setChargePendingRemoval] = useState<OrderCharge | null>(null);

  const { data: order } = useQuery({
    queryKey: ["order-detail", orderId],
    queryFn: () => fetchOrderDetail(orderId!),
    enabled: !!orderId,
  });

  const { mutate: removeCharge, isPending: isRemoving } = useMutation({
    mutationFn: (chargeId: string) => removeOrderCharge(orderId!, chargeId),
    onSuccess: () => {
      invalidateOrders();
      toast.success("Charge removed");
    },
    onError: (error) =>
      toast.error("Failed to remove charge", {
        description: graphQLErrorMessage(error, "The charge could not be removed."),
      }),
    onSettled: () => setChargePendingRemoval(null),
  });

  if (!orderId) {
    return null;
  }

  const charges = order?.charges ?? [];
  const currency = order?.currencyCode ?? "USD";
  const chargesLocked = ORDER_STATUSES_LOCKED_FOR_CHARGES.has(order?.status ?? "");
  const chargesSubtotal = charges.reduce((sum, charge) => sum + Number(charge.amount), 0);

  const openAddCharge = () => {
    setChargeBeingEdited(null);
    setChargeDialogOpen(true);
  };

  const openEditCharge = (charge: OrderCharge) => {
    setChargeBeingEdited(charge);
    setChargeDialogOpen(true);
  };

  return (
    <FormSection
      title="Order Charges"
      titleCount={charges.length}
      description="Order-level charges not tied to a single leg (e.g. customs brokerage). These roll into the total and are billed exactly once on the first grouped invoice."
      className="border-t border-border pt-4"
      action={
        charges.length > 0 &&
        !chargesLocked && (
          <Button type="button" variant="outline" size="xxs" onClick={openAddCharge}>
            <PlusIcon className="size-3" />
            Add Charge
          </Button>
        )
      }
    >
      {charges.length > 0 ? (
        <div className="rounded-lg border">
          <div className="grid grid-cols-12 gap-2 border-b border-border px-4 py-2 text-2xs uppercase text-muted-foreground">
            <span className="col-span-7">Description</span>
            <span className="col-span-3 text-right">Amount</span>
            <span className="col-span-2" />
          </div>
          <div className="divide-y">
            {charges.map((charge) => {
              const invoiced = !!charge.invoiceId;
              const editable = !chargesLocked && !invoiced;

              return (
                <div
                  key={charge.id}
                  className="grid grid-cols-12 items-center gap-2 px-4 py-2 text-sm"
                >
                  <span className="col-span-7 flex items-center gap-2">
                    <span className="truncate">{charge.description}</span>
                    {invoiced && (
                      <Badge variant="outline" className="shrink-0">
                        Invoiced
                      </Badge>
                    )}
                  </span>
                  <span className="col-span-3 text-right tabular-nums">
                    {formatCurrency(Number(charge.amount), currency)}
                  </span>
                  <span className="col-span-2 flex justify-end gap-1">
                    {editable && (
                      <>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          onClick={() => openEditCharge(charge)}
                          aria-label="Edit charge"
                        >
                          <PencilIcon className="size-3.5" />
                        </Button>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          disabled={isRemoving}
                          onClick={() => setChargePendingRemoval(charge)}
                          aria-label="Remove charge"
                        >
                          <Trash2Icon className="size-3.5 text-destructive" />
                        </Button>
                      </>
                    )}
                  </span>
                </div>
              );
            })}
          </div>
          <div className="grid grid-cols-12 gap-2 border-t border-border px-4 py-2 text-sm font-medium">
            <span className="col-span-7">Charges subtotal</span>
            <span className="col-span-3 text-right tabular-nums">
              {formatCurrency(chargesSubtotal, currency)}
            </span>
            <span className="col-span-2" />
          </div>
        </div>
      ) : (
        <EmptyState
          className="border-bg-sidebar-border max-h-[160px] rounded-lg border p-4"
          title="No Order Charges"
          description="Add customs brokerage, order-wide fuel, or other order-level fees"
          icons={[ReceiptTextIcon]}
          action={
            chargesLocked
              ? undefined
              : {
                  label: "Add First Charge",
                  onClick: openAddCharge,
                  icon: PlusIcon,
                }
          }
        />
      )}

      <AddChargeDialog
        open={chargeDialogOpen}
        onOpenChange={setChargeDialogOpen}
        orderId={orderId}
        currency={currency}
        charge={chargeBeingEdited}
      />

      <AlertDialog
        open={!!chargePendingRemoval}
        onOpenChange={(nextOpen) => !nextOpen && setChargePendingRemoval(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove this charge?</AlertDialogTitle>
            <AlertDialogDescription>
              {chargePendingRemoval
                ? `"${chargePendingRemoval.description}" (${formatCurrency(
                    Number(chargePendingRemoval.amount),
                    currency,
                  )}) will be removed and the order total recalculated.`
                : ""}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep charge</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => chargePendingRemoval && removeCharge(chargePendingRemoval.id)}
            >
              Remove charge
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </FormSection>
  );
}
