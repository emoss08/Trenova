import {
  CustomerAutocompleteField,
  FormulaTemplateAutocompleteField,
  OrderAutocompleteField,
} from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { TextShimmer } from "@/components/ui/text-shimmer";
import { useShipmentTotalsPreview } from "@/hooks/use-shipment-totals-preview";
import { queries } from "@/lib/queries";
import { cn, formatCurrency } from "@/lib/utils";
import type { CreditStatus } from "@/types/customer";
import type { GetPreviousRatesRequest, Shipment } from "@/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangleIcon, ShieldAlertIcon, ShieldIcon } from "lucide-react";
import type React from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { FuelSurchargeChangeDialog } from "./additional-charges/fuel-surcharge-change-dialog";
import { PreviousRatesButton } from "./previous-rates-dialog";
import { ProfitabilitySummary } from "./profitability/profitability-summary";

function Inner({ children }: { children: React.ReactNode }) {
  const { control, getValues } = useFormContext<Shipment>();

  const serviceTypeId = useWatch({ control, name: "serviceTypeId" });
  const shipmentTypeId = useWatch({ control, name: "shipmentTypeId" });
  const customerId = useWatch({ control, name: "customerId" });
  const moves = useWatch({ control, name: "moves" });
  const originLocationId = moves?.[0]?.stops?.[0]?.locationId ?? "";
  const lastMove = moves?.[moves.length - 1];
  const destinationLocationId = lastMove?.stops?.[lastMove.stops.length - 1]?.locationId ?? "";
  const shipmentId = getValues("id");

  const previousRatesRequest: GetPreviousRatesRequest = {
    originLocationId,
    destinationLocationId,
    shipmentTypeId: shipmentTypeId ?? "",
    serviceTypeId: serviceTypeId ?? "",
    customerId: customerId ?? undefined,
    excludeShipmentId: shipmentId ?? undefined,
  };

  return (
    <FormSection
      title="Billing & Rating"
      description="Customer, rating method, and charge amounts"
      action={<PreviousRatesButton request={previousRatesRequest} />}
      className="border-t border-border pt-4"
    >
      {children}
    </FormSection>
  );
}
const CREDIT_STATUS_CONFIG: Record<
  string,
  { variant: "destructive" | "warning"; icon: typeof ShieldAlertIcon; label: string }
> = {
  Hold: {
    variant: "destructive",
    icon: ShieldAlertIcon,
    label: "Credit Hold",
  },
  Suspended: {
    variant: "destructive",
    icon: ShieldAlertIcon,
    label: "Credit Suspended",
  },
  Warning: {
    variant: "warning",
    icon: AlertTriangleIcon,
    label: "Credit Warning",
  },
  Review: {
    variant: "warning",
    icon: AlertTriangleIcon,
    label: "Under Credit Review",
  },
};

function CreditHoldAlert({ customerId }: { customerId: string }) {
  const { data: billingProfile } = useQuery({
    ...queries.customer.getBillingProfile(customerId),
    enabled: !!customerId,
  });

  if (!billingProfile) return null;

  const config = CREDIT_STATUS_CONFIG[billingProfile.creditStatus as CreditStatus];
  if (!config) return null;

  const Icon = config.icon;

  return (
    <Alert variant={config.variant} className="mb-3">
      <Icon className="size-4" />
      <AlertTitle>{config.label}</AlertTitle>
      <AlertDescription>
        {billingProfile.creditHoldReason ||
          (billingProfile.creditStatus === "Warning"
            ? "This customer is approaching their credit limit. Review before dispatching."
            : billingProfile.creditStatus === "Review"
              ? "This customer's credit is under review. Shipments may be delayed pending approval."
              : "This customer's account is restricted. New shipments may not be invoiced until the hold is resolved.")}
      </AlertDescription>
    </Alert>
  );
}

function ChargeSummaryRow({
  label,
  value,
  bold,
}: {
  label: string;
  value: number | null | undefined;
  bold?: boolean;
}) {
  return (
    <div className="flex items-center justify-between">
      <span
        className={cn("text-sm", bold ? "font-medium text-foreground" : "text-muted-foreground")}
      >
        {label}
      </span>
      <span
        className={cn(
          "tracking-tight tabular-nums",
          bold ? "text-base font-semibold text-foreground" : "text-sm text-muted-foreground",
        )}
      >
        {formatCurrency(value ?? 0)}
      </span>
    </div>
  );
}

function ChargeSummary({ isCalculating, error }: { isCalculating: boolean; error: string | null }) {
  const { control } = useFormContext<Shipment>();
  const otherChargeAmount = useWatch({ control, name: "otherChargeAmount" });
  const totalChargeAmount = useWatch({ control, name: "totalChargeAmount" });
  const freightChargeAmount = useWatch({ control, name: "freightChargeAmount" });

  return (
    <div className="relative mt-3 overflow-hidden rounded-lg border bg-muted/50 p-2">
      {isCalculating && (
        <div className="absolute inset-0 z-10 flex items-center justify-center rounded-lg bg-background/50 backdrop-blur-[2px]">
          <TextShimmer as="span" className="text-sm font-medium" duration={1.5}>
            Calculating...
          </TextShimmer>
        </div>
      )}
      {error && !isCalculating && (
        <div className="absolute inset-0 z-10 flex items-center justify-center rounded-lg bg-destructive/5 backdrop-blur-[2px]">
          <div className="flex items-center gap-2 text-destructive">
            <AlertTriangleIcon className="size-4" />
            <span className="text-sm font-medium">{error}</span>
          </div>
        </div>
      )}
      <div className="mb-3">
        <span className="text-xs font-medium">Charge Summary</span>
        <p className="mt-0.5 text-2xs text-muted-foreground">
          Automatically calculated based on the rating method, freight charges, and any additional
          accessorial charges.
        </p>
      </div>
      <div className="space-y-2">
        <ChargeSummaryRow label="Freight Charges" value={freightChargeAmount} />
        <ChargeSummaryRow label="Other Charges" value={otherChargeAmount} />
        <Separator className="my-2" />
        <ChargeSummaryRow label="Total" value={totalChargeAmount} bold />
      </div>
    </div>
  );
}

function RatingBreakdownCard() {
  const { control } = useFormContext<Shipment>();
  const ratingDetail = useWatch({ control, name: "ratingDetail" });

  const breakdown = ratingDetail?.breakdown ?? [];
  const guardrail = ratingDetail?.guardrail;

  if (!ratingDetail || (breakdown.length === 0 && !guardrail?.applied)) {
    return null;
  }

  return (
    <div className="mt-3 rounded-lg border bg-muted/50 p-2">
      <div className="mb-3 flex items-center justify-between gap-2">
        <div>
          <span className="text-xs font-medium">Rating Breakdown</span>
          <p className="mt-0.5 text-2xs text-muted-foreground">
            Itemized amounts from {ratingDetail.formulaTemplateName || "the rating formula"}
          </p>
        </div>
        {ratingDetail.versionNumber ? (
          <Badge variant="outline" className="font-mono text-2xs">
            v{ratingDetail.versionNumber}
          </Badge>
        ) : null}
      </div>

      {breakdown.length > 0 && (
        <div className="space-y-2">
          {breakdown.map((item) => (
            <div key={item.name} className="flex items-center justify-between gap-3">
              <span className="text-sm text-muted-foreground">{item.label || item.name}</span>
              {item.error ? (
                <span className="flex items-center gap-1 text-xs text-destructive">
                  <AlertTriangleIcon className="size-3" />
                  {item.error}
                </span>
              ) : (
                <span className="text-sm tracking-tight text-muted-foreground tabular-nums">
                  {formatCurrency(item.amount)}
                </span>
              )}
            </div>
          ))}
        </div>
      )}

      {guardrail?.applied && (
        <div
          className={cn(
            "flex items-start gap-2 rounded-md border bg-background/50 px-2 py-1.5",
            breakdown.length > 0 && "mt-3",
          )}
        >
          <ShieldIcon className="mt-0.5 size-3.5 shrink-0 text-blue-500 dark:text-blue-400" />
          <p className="text-2xs text-muted-foreground">
            {guardrail.bound === "min" ? "Minimum" : "Maximum"} charge guardrail applied. The
            formula produced {formatCurrency(guardrail.rawResult)} and was clamped to{" "}
            {formatCurrency(
              (guardrail.bound === "min" ? guardrail.minCharge : guardrail.maxCharge) ?? 0,
            )}
            .
          </p>
        </div>
      )}
    </div>
  );
}

export default function ShipmentBillingDetails() {
  const { control, getValues } = useFormContext<Shipment>();
  const customerId = useWatch({ control, name: "customerId" });
  const shipmentId = getValues("id");
  const {
    isCalculating,
    error: totalsError,
    fuelSurchargeChange,
    resolveFuelSurchargeChange,
  } = useShipmentTotalsPreview();

  return (
    <Inner>
      <FuelSurchargeChangeDialog
        change={fuelSurchargeChange}
        onResolve={resolveFuelSurchargeChange}
      />
      {customerId && <CreditHoldAlert customerId={customerId} />}
      {shipmentId && <ProfitabilitySummary shipmentId={shipmentId} />}
      <FormGroup cols={2}>
        <FormControl>
          <CustomerAutocompleteField
            control={control}
            name="customerId"
            rules={{ required: true }}
            label="Customer"
            placeholder="Select Customer"
            description="Choose the customer who requested this shipment."
          />
        </FormControl>
        <FormControl>
          <OrderAutocompleteField
            control={control}
            name="orderId"
            label="Order"
            placeholder="Select Order"
            description="Optionally group this shipment under a commercial order for the same customer. Set on creation; use the order's Add Legs afterwards."
            disabled={!customerId}
            extraSearchParams={customerId ? { customerId, attachableOnly: "true" } : undefined}
          />
        </FormControl>
        <FormControl>
          <FormulaTemplateAutocompleteField
            control={control}
            name="formulaTemplateId"
            label="Rating Method"
            placeholder="Select Rating Method"
            description="Select how the shipment charges are calculated (e.g., per mile, per stop, flat rate)."
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl cols="full">
          <NumberField
            decimalScale={4}
            thousandSeparator
            control={control}
            rules={{ required: true }}
            name="baseRate"
            label="Base Rate"
            placeholder="Enter Base Rate"
            description="Per-unit rate used by the formula template to calculate freight charges."
            sideText="USD"
          />
        </FormControl>
      </FormGroup>

      <ChargeSummary isCalculating={isCalculating} error={totalsError} />
      <RatingBreakdownCard />
    </Inner>
  );
}
