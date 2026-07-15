import { EmptyState } from "@/components/empty-state";
import { ShipmentStatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import { FormSection } from "@/components/ui/form";
import { createInvoiceFromOrder, detachOrderShipment, fetchOrderDetail } from "@/lib/graphql/order";
import { formatCurrency } from "@/lib/utils";
import type { Order } from "@/types/order";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FileTextIcon, PackageIcon, PlusIcon, Trash2Icon, TruckIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { AddLegDialog } from "./add-leg-dialog";

const INVOICEABLE_LEG_STATUSES = new Set(["ReadyToInvoice", "Completed"]);

export function OrderLegsSection() {
  const { control } = useFormContext<Order>();
  const orderId = useWatch({ control, name: "id" });
  const queryClient = useQueryClient();
  const [addLegOpen, setAddLegOpen] = useState(false);

  const { data: order } = useQuery({
    queryKey: ["order-detail", orderId],
    queryFn: () => fetchOrderDetail(orderId!),
    enabled: !!orderId,
  });

  const invalidateDetail = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["order-detail", orderId] });
  }, [queryClient, orderId]);

  const { mutate: detachLeg, isPending: isDetaching } = useMutation({
    mutationFn: (shipmentId: string) => detachOrderShipment(orderId!, shipmentId),
    onSuccess: () => {
      invalidateDetail();
      toast.success("Leg removed", {
        description: "The shipment has been detached from this order.",
      });
    },
    onError: () => toast.error("Failed to remove leg"),
  });

  const { mutate: createInvoice, isPending: isCreatingInvoice } = useMutation({
    mutationFn: () => createInvoiceFromOrder(orderId!),
    onSuccess: (invoice) => {
      invalidateDetail();
      void queryClient.invalidateQueries({ queryKey: ["order-list"] });
      toast.success("Invoice created", {
        description: `Invoice ${invoice.number} was created from this order.`,
      });
    },
    onError: () => toast.error("Failed to create invoice"),
  });

  if (!orderId) {
    return null;
  }

  const legs = order?.legs ?? [];
  const currency = order?.currencyCode ?? "USD";
  const canCreateInvoice =
    legs.length > 0 && legs.every((leg) => INVOICEABLE_LEG_STATUSES.has(leg.status));

  return (
    <>
      <FormSection
        title="Legs"
        titleCount={legs.length}
        description="Shipments executing this order"
        className="border-t border-border pt-4"
        action={
          legs.length > 0 && (
            <Button type="button" variant="outline" size="xxs" onClick={() => setAddLegOpen(true)}>
              <PlusIcon className="size-3" />
              Add Leg
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
                  <span className="col-span-3 font-mono">{leg.proNumber}</span>
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
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      disabled={isDetaching}
                      onClick={() => detachLeg(leg.id)}
                      aria-label="Detach leg"
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
            className="border-bg-sidebar-border max-h-[200px] rounded-lg border p-4"
            title="No Legs"
            description="This order has no shipments attached yet"
            icons={[PackageIcon, TruckIcon]}
            action={{
              label: "Add First Leg",
              onClick: () => setAddLegOpen(true),
              icon: PlusIcon,
            }}
          />
        )}

        {legs.length > 0 && (
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
                All legs must be ready to invoice or completed before a grouped invoice can be
                created.
              </p>
            )}
          </div>
        )}
      </FormSection>

      <AddLegDialog
        open={addLegOpen}
        onOpenChange={setAddLegOpen}
        orderId={orderId}
        customerId={order?.customerId}
      />
    </>
  );
}
