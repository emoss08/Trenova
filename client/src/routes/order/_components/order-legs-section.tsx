import { EmptyState } from "@/components/empty-state";
import { ShipmentStatusBadge } from "@/components/status-badge";
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
import { graphQLErrorMessage } from "@/lib/graphql";
import { createInvoiceFromOrder, detachOrderShipment, fetchOrderDetail } from "@/lib/graphql/order";
import { formatCurrency } from "@/lib/utils";
import type { Order } from "@/types/order";
import { useMutation, useQuery } from "@tanstack/react-query";
import { FileTextIcon, PackageIcon, PlusIcon, Trash2Icon, TruckIcon } from "lucide-react";
import { useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { Link } from "react-router";
import { toast } from "sonner";
import { AddLegDialog } from "./add-leg-dialog";
import { useOrderInvalidation, useOrderInvoiceInvalidation } from "./use-order-invalidation";

const INVOICEABLE_LEG_STATUSES = new Set(["ReadyToInvoice", "Completed"]);
const ORDER_STATUSES_LOCKED_FOR_LEGS = new Set(["Billed", "Closed", "Canceled"]);

export function OrderLegsSection() {
  const { control } = useFormContext<Order>();
  const orderId = useWatch({ control, name: "id" });
  const invalidateOrders = useOrderInvalidation();
  const invalidateInvoices = useOrderInvoiceInvalidation();
  const [addLegOpen, setAddLegOpen] = useState(false);
  const [legPendingDetach, setLegPendingDetach] = useState<{
    id: string;
    proNumber: string;
  } | null>(null);

  const { data: order } = useQuery({
    queryKey: ["order-detail", orderId],
    queryFn: () => fetchOrderDetail(orderId!),
    enabled: !!orderId,
  });

  const { mutate: detachLeg, isPending: isDetaching } = useMutation({
    mutationFn: (shipmentId: string) => detachOrderShipment(orderId!, shipmentId),
    onSuccess: () => {
      invalidateOrders();
      toast.success("Leg removed", {
        description: "The shipment has been moved onto its own order.",
      });
    },
    onError: (error) =>
      toast.error("Failed to remove leg", {
        description: graphQLErrorMessage(error, "The shipment could not be detached."),
      }),
    onSettled: () => setLegPendingDetach(null),
  });

  const { mutate: createInvoice, isPending: isCreatingInvoice } = useMutation({
    mutationFn: () => createInvoiceFromOrder(orderId!),
    onSuccess: (invoice) => {
      invalidateInvoices();
      toast.success("Invoice created", {
        description: `Invoice ${invoice.number} was created from this order.`,
      });
    },
    onError: (error) =>
      toast.error("Failed to create invoice", {
        description: graphQLErrorMessage(error, "The grouped invoice could not be created."),
      }),
  });

  if (!orderId) {
    return null;
  }

  const legs = order?.legs ?? [];
  const currency = order?.currencyCode ?? "USD";
  const membershipLocked = ORDER_STATUSES_LOCKED_FOR_LEGS.has(order?.status ?? "");
  const activeLegs = legs.filter((leg) => leg.status !== "Canceled");
  const canCreateInvoice =
    activeLegs.length > 0 && activeLegs.every((leg) => INVOICEABLE_LEG_STATUSES.has(leg.status));
  const legsSubtotal = activeLegs.reduce((sum, leg) => sum + Number(leg.totalChargeAmount), 0);

  return (
    <>
      <FormSection
        title="Legs"
        titleCount={legs.length}
        description="Shipments executing this order"
        className="border-t border-border pt-4"
        action={
          legs.length > 0 &&
          !membershipLocked && (
            <Button type="button" variant="outline" size="xxs" onClick={() => setAddLegOpen(true)}>
              <PlusIcon className="size-3" />
              Add Legs
            </Button>
          )
        }
      >
        {legs.length > 0 ? (
          <div className="rounded-lg border">
            <div className="grid grid-cols-12 gap-2 border-b border-border px-4 py-2 text-2xs uppercase text-muted-foreground">
              <span className="col-span-3">Pro Number</span>
              <span className="col-span-3">Status</span>
              <span className="col-span-2 text-right">Freight</span>
              <span className="col-span-3 text-right">Total</span>
              <span className="col-span-1" />
            </div>
            <div className="divide-y">
              {legs.map((leg) => (
                <div key={leg.id} className="grid grid-cols-12 items-center gap-2 px-4 py-2 text-sm">
                  <span className="col-span-3 font-mono">
                    <Link
                      to={`/shipment-management/shipments?item=${leg.id}`}
                      className="hover:underline"
                    >
                      {leg.proNumber}
                    </Link>
                  </span>
                  <span className="col-span-3">
                    <ShipmentStatusBadge status={leg.status} />
                  </span>
                  <span className="col-span-2 text-right tabular-nums">
                    {formatCurrency(Number(leg.freightChargeAmount), currency)}
                  </span>
                  <span className="col-span-3 text-right tabular-nums">
                    {formatCurrency(Number(leg.totalChargeAmount), currency)}
                  </span>
                  <span className="col-span-1 flex justify-end">
                    {!membershipLocked && leg.status !== "Invoiced" && (
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        disabled={isDetaching}
                        onClick={() =>
                          setLegPendingDetach({ id: leg.id, proNumber: leg.proNumber })
                        }
                        aria-label="Detach leg"
                      >
                        <Trash2Icon className="size-3.5 text-destructive" />
                      </Button>
                    )}
                  </span>
                </div>
              ))}
            </div>
            <div className="grid grid-cols-12 gap-2 border-t border-border px-4 py-2 text-sm font-medium">
              <span className="col-span-8">Legs subtotal</span>
              <span className="col-span-3 text-right tabular-nums">
                {formatCurrency(legsSubtotal, currency)}
              </span>
              <span className="col-span-1" />
            </div>
          </div>
        ) : (
          <EmptyState
            className="border-bg-sidebar-border max-h-[200px] rounded-lg border p-4"
            title="No Legs"
            description="This order has no shipments attached yet"
            icons={[PackageIcon, TruckIcon]}
            action={
              membershipLocked
                ? undefined
                : {
                    label: "Add First Leg",
                    onClick: () => setAddLegOpen(true),
                    icon: PlusIcon,
                  }
            }
          />
        )}

        {legs.length > 0 && !membershipLocked && (
          <div className="flex flex-col gap-1">
            <Button
              type="button"
              className="w-fit"
              size="sm"
              disabled={!canCreateInvoice}
              isLoading={isCreatingInvoice}
              loadingText="Creating invoice..."
              onClick={() => createInvoice()}
            >
              <FileTextIcon className="mr-1.5 size-3.5" />
              Create grouped invoice
            </Button>
            {!canCreateInvoice && (
              <p className="text-2xs text-muted-foreground">
                Every active leg must be ready to invoice or completed before a grouped invoice
                can be created. Canceled legs are excluded.
              </p>
            )}
          </div>
        )}
      </FormSection>

      <AlertDialog
        open={!!legPendingDetach}
        onOpenChange={(nextOpen) => !nextOpen && setLegPendingDetach(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Detach leg {legPendingDetach?.proNumber}?</AlertDialogTitle>
            <AlertDialogDescription>
              The shipment moves onto its own new single-leg order and this order&apos;s status
              and total are recalculated. The only leg of an order cannot be detached.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep leg</AlertDialogCancel>
            <AlertDialogAction onClick={() => legPendingDetach && detachLeg(legPendingDetach.id)}>
              Detach leg
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AddLegDialog
        open={addLegOpen}
        onOpenChange={setAddLegOpen}
        orderId={orderId}
        customerId={order?.customerId}
      />
    </>
  );
}
