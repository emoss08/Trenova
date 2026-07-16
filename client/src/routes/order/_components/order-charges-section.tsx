import { EmptyState } from "@/components/empty-state";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
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
import {
  addOrderCharge,
  fetchOrderDetail,
  removeOrderCharge,
  updateOrderCharge,
  type OrderCharge,
} from "@/lib/graphql/order";
import { formatCurrency } from "@/lib/utils";
import { orderChargeFormSchema, type Order, type OrderChargeFormValues } from "@/types/order";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery } from "@tanstack/react-query";
import { CheckIcon, PencilIcon, PlusIcon, ReceiptTextIcon, Trash2Icon, XIcon } from "lucide-react";
import { useState } from "react";
import { type Resolver, useForm, useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { useOrderInvalidation } from "./use-order-invalidation";

const ORDER_STATUSES_LOCKED_FOR_CHARGES = new Set(["Billed", "Closed", "Canceled"]);

export function OrderChargesSection() {
  const { control } = useFormContext<Order>();
  const orderId = useWatch({ control, name: "id" });
  const invalidateOrders = useOrderInvalidation();
  const [editingChargeId, setEditingChargeId] = useState<string | null>(null);
  const [chargePendingRemoval, setChargePendingRemoval] = useState<OrderCharge | null>(null);

  const addForm = useForm<OrderChargeFormValues>({
    resolver: zodResolver(orderChargeFormSchema) as Resolver<OrderChargeFormValues>,
    defaultValues: { description: "", amount: null as unknown as number },
    mode: "onChange",
  });

  const editForm = useForm<OrderChargeFormValues>({
    resolver: zodResolver(orderChargeFormSchema) as Resolver<OrderChargeFormValues>,
    defaultValues: { description: "", amount: null as unknown as number },
    mode: "onChange",
  });

  const { data: order } = useQuery({
    queryKey: ["order-detail", orderId],
    queryFn: () => fetchOrderDetail(orderId!),
    enabled: !!orderId,
  });

  const { mutate: addCharge, isPending: isAdding } = useMutation({
    mutationFn: (values: OrderChargeFormValues) =>
      addOrderCharge(orderId!, values.description.trim(), String(values.amount)),
    onSuccess: () => {
      addForm.reset({ description: "", amount: null as unknown as number });
      invalidateOrders();
      toast.success("Charge added");
    },
    onError: (error) =>
      toast.error("Failed to add charge", {
        description: graphQLErrorMessage(error, "The charge could not be added."),
      }),
  });

  const { mutate: saveCharge, isPending: isSaving } = useMutation({
    mutationFn: ({ charge, values }: { charge: OrderCharge; values: OrderChargeFormValues }) =>
      updateOrderCharge({
        orderId: orderId!,
        chargeId: charge.id,
        description: values.description.trim(),
        amount: String(values.amount),
        version: charge.version,
      }),
    onSuccess: () => {
      setEditingChargeId(null);
      invalidateOrders();
      toast.success("Charge updated");
    },
    onError: (error) =>
      toast.error("Failed to update charge", {
        description: graphQLErrorMessage(error, "The charge could not be updated."),
      }),
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

  const handleAdd = addForm.handleSubmit((values) => addCharge(values));

  if (!orderId) {
    return null;
  }

  const charges = order?.charges ?? [];
  const currency = order?.currencyCode ?? "USD";
  const chargesLocked = ORDER_STATUSES_LOCKED_FOR_CHARGES.has(order?.status ?? "");
  const chargesSubtotal = charges.reduce((sum, charge) => sum + Number(charge.amount), 0);
  const canAdd = addForm.formState.isValid && !isAdding;

  const startEditing = (charge: OrderCharge) => {
    editForm.reset({ description: charge.description, amount: Number(charge.amount) });
    setEditingChargeId(charge.id);
  };

  const handleSave = (charge: OrderCharge) =>
    editForm.handleSubmit((values) => saveCharge({ charge, values }))();

  return (
    <FormSection
      title="Order Charges"
      titleCount={charges.length}
      description="Order-level charges not tied to a single leg (e.g. customs brokerage). These roll into the total and are billed exactly once on the first grouped invoice."
      className="border-t border-border pt-4"
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

              if (editingChargeId === charge.id) {
                return (
                  <div
                    key={charge.id}
                    className="grid grid-cols-12 items-end gap-2 px-4 py-2 text-sm"
                  >
                    <div className="col-span-7">
                      <InputField control={editForm.control} name="description" label="" />
                    </div>
                    <div className="col-span-3">
                      <NumberField
                        control={editForm.control}
                        name="amount"
                        label=""
                        decimalScale={2}
                        thousandSeparator
                        sideText={currency}
                      />
                    </div>
                    <div className="col-span-2 flex justify-end gap-1">
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        disabled={isSaving || !editForm.formState.isValid}
                        onClick={() => handleSave(charge)}
                        aria-label="Save charge"
                      >
                        <CheckIcon className="size-3.5 text-green-600" />
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        disabled={isSaving}
                        onClick={() => setEditingChargeId(null)}
                        aria-label="Cancel editing"
                      >
                        <XIcon className="size-3.5" />
                      </Button>
                    </div>
                  </div>
                );
              }

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
                          onClick={() => startEditing(charge)}
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
        />
      )}

      {!chargesLocked && (
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
      )}

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
