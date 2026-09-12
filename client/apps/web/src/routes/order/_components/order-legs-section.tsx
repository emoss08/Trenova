import { useT } from "@trenova/shared/i18n/use-t";
import { EmptyState } from "@/components/empty-state";
import { ShipmentStatusBadge } from "@trenova/shared/components/status-badge";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { FormSection } from "@trenova/shared/components/ui/form";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import {
  createInvoiceFromShipments,
  detachOrderShipment,
  fetchOrderDetail,
} from "@/lib/graphql/order";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { Order } from "@trenova/shared/types/order";
import { useMutation, useQuery } from "@tanstack/react-query";
import { FileTextIcon, PackageIcon, PlusIcon, Trash2Icon, TruckIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { Link } from "react-router";
import { toast } from "sonner";
import {
  OffCycleInvoiceDialog,
  offCycleWarningFrom,
} from "@/components/billing/off-cycle-invoice-dialog";
import { AddLegDialog } from "./add-leg-dialog";
import { useOrderInvalidation, useOrderInvoiceInvalidation } from "./use-order-invalidation";

const INVOICEABLE_LEG_STATUSES = new Set(["ReadyToInvoice", "Completed"]);
const ORDER_STATUSES_LOCKED_FOR_LEGS = new Set(["Billed", "Closed", "Canceled"]);

export function OrderLegsSection() {
  const t = useT();

  const { control } = useFormContext<Order>();
  const orderId = useWatch({ control, name: "id" });
  const invalidateOrders = useOrderInvalidation();
  const invalidateInvoices = useOrderInvoiceInvalidation();
  const [addLegOpen, setAddLegOpen] = useState(false);
  const [checkedLegIds, setCheckedLegIds] = useState<ReadonlySet<string>>(new Set());
  const [confirmSingleLeg, setConfirmSingleLeg] = useState(false);
  const [offCycleWarning, setOffCycleWarning] = useState<string | null>(null);
  const [legPendingDetach, setLegPendingDetach] = useState<{
    id: string;
    proNumber: string;
  } | null>(null);

  const { data: order } = useQuery({
    queryKey: ["order-detail", orderId],
    queryFn: ({ signal }) => fetchOrderDetail(orderId!, { signal }),
    enabled: !!orderId,
  });

  const { mutate: detachLeg, isPending: isDetaching } = useMutation({
    mutationFn: (shipmentId: string) => detachOrderShipment(orderId!, shipmentId),
    onSuccess: () => {
      invalidateOrders();
      toast.success(t("Leg removed"), {
        description: t("The shipment has been moved onto its own order."),
      });
    },
    onError: (error) =>
      toast.error(t("Failed to remove leg"), {
        description: graphQLErrorMessage(error, "The shipment could not be detached."),
      }),
    onSettled: () => setLegPendingDetach(null),
  });

  const { mutate: createInvoice, isPending: isCreatingInvoice } = useMutation({
    mutationFn: ({ shipmentIds, reason }: { shipmentIds: string[]; reason?: string }) =>
      createInvoiceFromShipments(shipmentIds, reason),
    onSuccess: (invoice) => {
      setOffCycleWarning(null);
      invalidateInvoices();
      invalidateOrders();
      setCheckedLegIds(new Set());
      toast.success(t("Invoice created"), {
        description: `Invoice ${invoice.number} was created from this order.`,
      });
    },
    onError: (error) => {
      // Not a failure: the customer is on a statement and the server is asking
      // whether the biller really means to take this freight off it.
      const warning = offCycleWarningFrom(error);
      if (warning) {
        setOffCycleWarning(warning);
        return;
      }
      toast.error(t("Failed to create invoice"), {
        description: graphQLErrorMessage(error, "The invoice could not be created."),
      });
    },
    onSettled: () => setConfirmSingleLeg(false),
  });

  const legs = useMemo(() => order?.legs ?? [], [order?.legs]);
  const invoiceableLegs = useMemo(
    () => legs.filter((leg) => INVOICEABLE_LEG_STATUSES.has(leg.status)),
    [legs],
  );

  const selectedIds = useMemo(
    () => invoiceableLegs.filter((leg) => checkedLegIds.has(leg.id)).map((leg) => leg.id),
    [invoiceableLegs, checkedLegIds],
  );

  if (!orderId) {
    return null;
  }

  const currency = order?.currencyCode ?? "USD";
  const membershipLocked = ORDER_STATUSES_LOCKED_FOR_LEGS.has(order?.status ?? "");
  const activeLegs = legs.filter((leg) => leg.status !== "Canceled");
  const legsSubtotal = activeLegs.reduce((sum, leg) => sum + Number(leg.totalChargeAmount), 0);

  const allInvoiceableChecked =
    invoiceableLegs.length > 0 && invoiceableLegs.every((leg) => checkedLegIds.has(leg.id));

  function toggleAllInvoiceable() {
    setCheckedLegIds(
      allInvoiceableChecked ? new Set() : new Set(invoiceableLegs.map((leg) => leg.id)),
    );
  }

  function toggleLeg(legId: string) {
    setCheckedLegIds((previous) => {
      const next = new Set(previous);
      if (next.has(legId)) {
        next.delete(legId);
      } else {
        next.add(legId);
      }
      return next;
    });
  }

  // A lone leg takes the single-shipment path, which mints its own billing-queue
  // item rather than joining a grouped invoice, so it is worth confirming.
  function submitInvoice() {
    if (selectedIds.length === 1) {
      setConfirmSingleLeg(true);
      return;
    }
    createInvoice({ shipmentIds: selectedIds });
  }

  const legLabel = selectedIds.length === 1 ? "leg" : "legs";

  return (
    <>
      <FormSection
        title={t("Legs")}
        titleCount={legs.length}
        description={t("Shipments executing this order")}
        className="border-border border-t pt-4"
        action={
          legs.length > 0 &&
          !membershipLocked && (
            <Button type="button" variant="outline" size="xxs" onClick={() => setAddLegOpen(true)}>
              <PlusIcon className="size-3" />
              {t("Add Legs")}
            </Button>
          )
        }
      >
        {legs.length > 0 ? (
          <div className="rounded-lg border">
            <div className="border-border text-2xs text-muted-foreground grid grid-cols-12 gap-2 border-b px-4 py-2 uppercase">
              <span className="col-span-3 flex items-center gap-2">
                {!membershipLocked && invoiceableLegs.length > 0 && (
                  <Checkbox
                    checked={allInvoiceableChecked}
                    onCheckedChange={toggleAllInvoiceable}
                    aria-label={t("Select all invoiceable legs")}
                  />
                )}
                {t("Pro Number")}
              </span>
              <span className="col-span-3">{t("Status")}</span>
              <span className="col-span-2 text-right">{t("Freight")}</span>
              <span className="col-span-3 text-right">{t("Total")}</span>
              <span className="col-span-1" />
            </div>
            <div className="divide-y">
              {legs.map((leg) => {
                const invoiceable = INVOICEABLE_LEG_STATUSES.has(leg.status);
                return (
                  <div
                    key={leg.id}
                    className="grid grid-cols-12 items-center gap-2 px-4 py-2 text-sm"
                  >
                    <span className="col-span-3 flex items-center gap-2 font-mono">
                      {!membershipLocked &&
                        (invoiceable ? (
                          <Checkbox
                            checked={checkedLegIds.has(leg.id)}
                            onCheckedChange={() => toggleLeg(leg.id)}
                            aria-label={`Select leg ${leg.proNumber}`}
                          />
                        ) : (
                          <Tooltip>
                            <TooltipTrigger
                              render={
                                <span className="inline-flex" tabIndex={0}>
                                  <Checkbox
                                    disabled
                                    checked={false}
                                    aria-label={`Leg ${leg.proNumber} cannot be invoiced`}
                                  />
                                </span>
                              }
                            />
                            <TooltipContent>
                              {t("A leg that is {0} cannot be invoiced.", leg.status)}
                            </TooltipContent>
                          </Tooltip>
                        ))}
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
                          aria-label={t("Detach leg")}
                        >
                          <Trash2Icon className="text-destructive size-3.5" />
                        </Button>
                      )}
                    </span>
                  </div>
                );
              })}
            </div>
            <div className="border-border grid grid-cols-12 gap-2 border-t px-4 py-2 text-sm font-medium">
              <span className="col-span-8">{t("Legs subtotal")}</span>
              <span className="col-span-3 text-right tabular-nums">
                {formatCurrency(legsSubtotal, currency)}
              </span>
              <span className="col-span-1" />
            </div>
          </div>
        ) : (
          <EmptyState
            className="border-bg-sidebar-border max-h-[200px] rounded-lg border p-4"
            title={t("No Legs")}
            description={t("This order has no shipments attached yet")}
            icons={[PackageIcon, TruckIcon]}
            action={
              membershipLocked
                ? undefined
                : {
                    label: t("Add First Leg"),
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
              disabled={selectedIds.length === 0}
              isLoading={isCreatingInvoice}
              loadingText={t("Creating invoice...")}
              onClick={submitInvoice}
            >
              <FileTextIcon className="mr-1.5 size-3.5" />
              {selectedIds.length > 0
                ? t("Create invoice from {0} {1}", selectedIds.length, legLabel)
                : t("Create invoice")}
            </Button>
            {invoiceableLegs.length === 0 ? (
              <p className="text-2xs text-muted-foreground">
                {t(
                  "No leg is ready to invoice yet. A leg must be completed or ready to invoice before it can be billed.",
                )}
              </p>
            ) : selectedIds.length === 0 ? (
              <p className="text-2xs text-muted-foreground">
                {t(
                  "Select the legs to bill. Legs billed together become one invoice with a charge block per leg.",
                )}
              </p>
            ) : null}
          </div>
        )}
      </FormSection>

      <AlertDialog
        open={!!legPendingDetach}
        onOpenChange={(nextOpen) => !nextOpen && setLegPendingDetach(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("Detach leg {0}?", legPendingDetach?.proNumber)}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "The shipment moves onto its own new single-leg order and this order's status and total are recalculated. The only leg of an order cannot be detached.",
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Keep leg")}</AlertDialogCancel>
            <AlertDialogAction onClick={() => legPendingDetach && detachLeg(legPendingDetach.id)}>
              {t("Detach leg")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={confirmSingleLeg} onOpenChange={setConfirmSingleLeg}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("Bill this leg on its own invoice?")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "One leg produces a standalone invoice rather than a grouped one covering the order. The order's remaining legs will have to be billed separately.",
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
            <AlertDialogAction onClick={() => createInvoice({ shipmentIds: selectedIds })}>
              {t("Create invoice")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <OffCycleInvoiceDialog
        open={offCycleWarning !== null}
        warning={offCycleWarning}
        pending={isCreatingInvoice}
        onOpenChange={(next) => {
          if (!next) setOffCycleWarning(null);
        }}
        onConfirm={(reason) => createInvoice({ shipmentIds: selectedIds, reason })}
      />

      <AddLegDialog
        open={addLegOpen}
        onOpenChange={setAddLegOpen}
        orderId={orderId}
        customerId={order?.customerId}
      />
    </>
  );
}
