import { OrderStatusBadge } from "@/components/status-badge";
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
import { Button } from "@/components/ui/button";
import { FormSection } from "@/components/ui/form";
import { Textarea } from "@/components/ui/textarea";
import { graphQLErrorMessage } from "@/lib/graphql";
import { cancelOrder, closeOrder, fetchOrderDetail } from "@/lib/graphql/order";
import { formatCurrency } from "@/lib/utils";
import type { Order } from "@/types/order";
import { useMutation, useQuery } from "@tanstack/react-query";
import { BanIcon, CheckCircle2Icon } from "lucide-react";
import { useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { useOrderInvalidation } from "./use-order-invalidation";

const CANCELABLE_STATUSES = new Set(["Draft", "Confirmed", "InProgress", "Completed"]);

function SummaryStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col rounded-lg border px-3 py-2">
      <span className="text-2xs text-muted-foreground uppercase">{label}</span>
      <span className="text-sm font-medium tabular-nums">{value}</span>
    </div>
  );
}

export function OrderSummarySection() {
  const { control } = useFormContext<Order>();
  const orderId = useWatch({ control, name: "id" });
  const invalidateOrders = useOrderInvalidation();
  const [cancelOpen, setCancelOpen] = useState(false);
  const [cancelReason, setCancelReason] = useState("");

  const { data: order } = useQuery({
    queryKey: ["order-detail", orderId],
    queryFn: () => fetchOrderDetail(orderId!),
    enabled: !!orderId,
  });

  const { mutate: close, isPending: isClosing } = useMutation({
    mutationFn: () => closeOrder(orderId!),
    onSuccess: () => {
      invalidateOrders();
      toast.success("Order closed", {
        description: "The order has been settled and closed.",
      });
    },
    onError: (error) =>
      toast.error("Failed to close order", {
        description: graphQLErrorMessage(error, "The order could not be closed."),
      }),
  });

  const { mutate: cancel, isPending: isCanceling } = useMutation({
    mutationFn: () => cancelOrder(orderId!, cancelReason.trim()),
    onSuccess: () => {
      invalidateOrders();
      setCancelOpen(false);
      setCancelReason("");
      toast.success("Order canceled", {
        description: "Every remaining leg has been canceled.",
      });
    },
    onError: (error) =>
      toast.error("Failed to cancel order", {
        description: graphQLErrorMessage(error, "The order could not be canceled."),
      }),
  });

  if (!orderId || !order) {
    return null;
  }

  const currency = order.currencyCode ?? "USD";
  const formatAmount = (value?: string | null) =>
    value != null ? formatCurrency(Number(value), currency) : "—";
  const legCount = order.legs.length;
  const activeLegCount = order.legs.filter((leg) => leg.status !== "Canceled").length;
  const canClose = order.status === "Billed";
  const canCancel = CANCELABLE_STATUSES.has(order.status);

  return (
    <FormSection
      title="Accounts Receivable"
      description="The commercial rollup across every leg and order-level charge"
      className="border-t border-border pt-4"
      action={
        <div className="flex gap-2">
          {canClose && (
            <Button
              type="button"
              variant="outline"
              size="xxs"
              isLoading={isClosing}
              loadingText="Closing..."
              onClick={() => close()}
            >
              <CheckCircle2Icon className="size-3" />
              Close Order
            </Button>
          )}
          {canCancel && (
            <Button
              type="button"
              variant="outline"
              size="xxs"
              onClick={() => setCancelOpen(true)}
            >
              <BanIcon className="size-3 text-destructive" />
              Cancel Order
            </Button>
          )}
        </div>
      }
    >
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <div className="flex flex-col rounded-lg border px-3 py-2">
          <span className="text-2xs text-muted-foreground uppercase">Status</span>
          <span className="pt-0.5">
            <OrderStatusBadge status={order.status} />
          </span>
        </div>
        <SummaryStat label="Quoted" value={formatAmount(order.quotedAmount)} />
        <SummaryStat
          label={`Total (${activeLegCount} of ${legCount} active leg${legCount === 1 ? "" : "s"})`}
          value={formatAmount(order.totalAmount)}
        />
        <SummaryStat
          label="Quote Variance"
          value={
            order.quotedAmount != null && order.totalAmount != null
              ? formatCurrency(Number(order.totalAmount) - Number(order.quotedAmount), currency)
              : "—"
          }
        />
      </div>

      <AlertDialog open={cancelOpen} onOpenChange={setCancelOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Cancel order {order.orderNumber}?</AlertDialogTitle>
            <AlertDialogDescription>
              Every remaining active leg will be canceled and the order will derive to Canceled.
              This cannot be undone. A reason is required.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <Textarea
            value={cancelReason}
            onChange={(event) => setCancelReason(event.target.value)}
            placeholder="Reason for cancellation"
            rows={3}
          />
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isCanceling}>Keep order</AlertDialogCancel>
            <AlertDialogAction
              disabled={!cancelReason.trim() || isCanceling}
              onClick={(event) => {
                event.preventDefault();
                cancel();
              }}
            >
              Cancel order
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </FormSection>
  );
}
