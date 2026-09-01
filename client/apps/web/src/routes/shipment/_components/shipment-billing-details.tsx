import {
  CustomerAutocompleteField,
  FormulaTemplateAutocompleteField,
  OrderAutocompleteField,
} from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useShipmentAutoRate } from "@/hooks/use-shipment-auto-rate";
import { useShipmentTotalsPreview } from "@/hooks/use-shipment-totals-preview";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import {
  CapabilityFields,
  type FieldDescriptor,
} from "@trenova/shared/components/capability-form-section";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { FormSection } from "@trenova/shared/components/ui/form";
import { Separator } from "@trenova/shared/components/ui/separator";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { getProfile } from "@trenova/shared/lib/capability";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { CreditStatus } from "@trenova/shared/types/customer";
import type {
  ContractRate,
  GetPreviousRatesRequest,
  Shipment,
} from "@trenova/shared/types/shipment";
import {
  AlertTriangleIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  ReceiptTextIcon,
  ShieldAlertIcon,
  ShieldIcon,
  SparklesIcon,
} from "lucide-react";
import type React from "react";
import { useState } from "react";
import { Link } from "react-router";
import { ReceiptView } from "@/components/formula-editor/receipt-view";
import { formulaTemplateRoutes } from "@/lib/formula-template-routes";
import { useFormContext, useWatch } from "react-hook-form";
import { FuelSurchargeChangeDialog } from "./additional-charges/fuel-surcharge-change-dialog";
import { AutoRateDialog } from "./auto-rate-dialog";
import { PreviousRatesButton } from "./previous-rates-dialog";
import { ProfitabilitySummary } from "./profitability/profitability-summary";
import { WhyThisRate } from "./why-this-rate";

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
      className="border-border border-t pt-4"
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
        className={cn("text-sm", bold ? "text-foreground font-medium" : "text-muted-foreground")}
      >
        {label}
      </span>
      <span
        className={cn(
          "tracking-tight tabular-nums",
          bold ? "text-foreground text-base font-semibold" : "text-muted-foreground text-sm",
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
    <div className="bg-muted/50 relative mt-3 overflow-hidden rounded-lg border p-2">
      {isCalculating && (
        <div className="bg-background/50 absolute inset-0 z-10 flex items-center justify-center rounded-lg backdrop-blur-[2px]">
          <TextShimmer as="span" className="text-sm font-medium" duration={1.5}>
            Calculating...
          </TextShimmer>
        </div>
      )}
      {error && !isCalculating && (
        <div className="bg-destructive/5 absolute inset-0 z-10 flex items-center justify-center rounded-lg backdrop-blur-[2px]">
          <div className="text-destructive flex items-center gap-2">
            <AlertTriangleIcon className="size-4" />
            <span className="text-sm font-medium">{error}</span>
          </div>
        </div>
      )}
      <div className="mb-3">
        <span className="text-xs font-medium">Charge Summary</span>
        <p className="text-2xs text-muted-foreground mt-0.5">
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

/**
 * Says which contract just filled in the rating fields.
 *
 * The numbers appear in the form as ordinary values, so without this the rater
 * has no way of telling a contract rate from something a colleague typed — and
 * they are free to change any of it, which is worth saying plainly.
 */
function ContractRateAppliedAlert({
  rate,
  onDismiss,
}: {
  rate: ContractRate;
  onDismiss: () => void;
}) {
  return (
    <Alert variant="info" className="mb-3">
      <SparklesIcon className="size-4" />
      <AlertTitle>
        Rated from {rate.agreementName || "a rate agreement"}
        {rate.ruleLabel ? ` — ${rate.ruleLabel}` : ""}
      </AlertTitle>
      <AlertDescription>
        <span>
          The rating method and base rate below came from the contract
          {rate.accessorials.length > 0
            ? `, along with ${rate.accessorials.length} automatic ${
                rate.accessorials.length === 1 ? "charge" : "charges"
              }`
            : ""}
          . Change any of them and this shipment is priced by hand instead.
        </span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="mt-1 h-6 px-1.5"
          onClick={onDismiss}
        >
          <span className="text-2xs">Dismiss</span>
        </Button>
      </AlertDescription>
    </Alert>
  );
}

/**
 * Asks why a shipment is priced at something other than its contract.
 *
 * It appears only once a shipment has actually departed, because that is the
 * only moment the question means anything. The answer is what the audit trail
 * explains the invoice with, and what the organization's billing policy can
 * require before the shipment is allowed to bill.
 */
function RateDepartureReason() {
  const { control } = useFormContext<Shipment>();
  const autoRated = useWatch({ control, name: "autoRated" });
  const agreementId = useWatch({ control, name: "rateAgreementId" });
  const overrideAmount = useWatch({ control, name: "rateOverrideAmount" });

  if (autoRated || !agreementId || !overrideAmount) {
    return null;
  }

  return (
    <div className="mt-3">
      <TextareaField
        control={control}
        name="rateOverrideReason"
        label="Reason for the rate change"
        placeholder="Why is this shipment priced differently from its contract?"
        description="This shipment no longer charges what its rate agreement says. The reason is kept with the rating history and shown on the rate leakage report."
      />
    </div>
  );
}

function RatingBreakdownCard() {
  const { control, getValues } = useFormContext<Shipment>();
  const [showReceipt, setShowReceipt] = useState(false);
  const ratingDetail = useWatch({ control, name: "ratingDetail" });
  const shipmentId = getValues("id");

  const breakdown = ratingDetail?.breakdown ?? [];
  const guardrail = ratingDetail?.guardrail;

  if (!ratingDetail || (breakdown.length === 0 && !guardrail?.applied)) {
    return null;
  }

  // The contract that priced the shipment leads, since that is what somebody
  // reading an invoice recognises. The formula name only appears when a formula
  // is what actually produced the number.
  const source = ratingDetail.agreementName || ratingDetail.formulaTemplateName;

  return (
    <div className="bg-muted/50 mt-3 rounded-lg border p-2">
      <div className="mb-3 flex items-center justify-between gap-2">
        <div className="flex flex-col gap-1 w-full">
          <div className="flex justify-between items-center">
            <div className="flex flex-row gap-1">
              <span className="text-xs font-medium">Rating Breakdown</span>
              <AutoRateDialog />
            </div>
            <WhyThisRate shipmentId={shipmentId} />
          </div>
          <p className="text-2xs text-muted-foreground mt-0.5">
            {ratingDetail.ruleLabel
              ? `${source} — ${ratingDetail.ruleLabel}`
              : `Itemized amounts from ${source || "the rating formula"}`}
          </p>
        </div>
        <div className="flex items-center gap-1">
          {ratingDetail.versionNumber ? (
            <Badge variant="outline" className="text-2xs font-mono">
              v{ratingDetail.versionNumber}
            </Badge>
          ) : null}
        </div>
      </div>

      {breakdown.length > 0 && (
        <div className="space-y-2">
          {breakdown.map((item) => (
            <div key={item.name} className="flex items-center justify-between gap-3">
              <span className="text-muted-foreground text-sm">{item.label || item.name}</span>
              {item.error ? (
                <span className="text-destructive flex items-center gap-1 text-xs">
                  <AlertTriangleIcon className="size-3" />
                  {item.error}
                </span>
              ) : (
                <span className="text-muted-foreground text-sm tracking-tight tabular-nums">
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
            "bg-background/50 flex items-start gap-2 rounded-md border px-2 py-1.5",
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

      {ratingDetail.receipt && (
        <div className="mt-3 border-t pt-3">
          <div className="mb-2 flex items-center justify-between gap-2">
            <button
              type="button"
              onClick={() => setShowReceipt((prev) => !prev)}
              aria-expanded={showReceipt}
              className="text-muted-foreground hover:text-foreground flex items-center gap-1 text-xs font-medium"
            >
              <ReceiptTextIcon className="size-3.5" />
              Calculation receipt
              {showReceipt ? (
                <ChevronDownIcon className="size-3" />
              ) : (
                <ChevronRightIcon className="size-3" />
              )}
            </button>
            {ratingDetail.formulaTemplateId && (
              <Link
                to={formulaTemplateRoutes.edit(ratingDetail.formulaTemplateId)}
                className="text-2xs text-primary hover:underline"
              >
                Open template
                {ratingDetail.versionNumber ? ` v${ratingDetail.versionNumber}` : ""}
              </Link>
            )}
          </div>
          {showReceipt && <ReceiptView receipt={ratingDetail.receipt} />}
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
  const { data: shipmentUIPolicy } = useQuery({ ...queries.shipment.uiPolicy() });

  // A contract prices a shipment once, while it is being typed. An existing
  // one already carries a rate somebody may have negotiated, and replacing it
  // on open is exactly what this design exists to stop.
  const { appliedRate, dismissAppliedRate } = useShipmentAutoRate({ enabled: !shipmentId });

  const profile = getProfile(shipmentUIPolicy);

  const descriptors: FieldDescriptor[] = [
    {
      name: "orderId",
      render: () => (
        <OrderAutocompleteField
          control={control}
          name="orderId"
          label="Order"
          placeholder="Select Order"
          description="Optionally group this shipment under a commercial order for the same customer. Set on creation; use the order's Add Legs afterwards."
          disabled={!customerId}
          extraSearchParams={customerId ? { customerId, attachableOnly: "true" } : undefined}
        />
      ),
    },
    {
      name: "customerId",
      render: () => (
        <CustomerAutocompleteField
          control={control}
          name="customerId"
          rules={{ required: true }}
          label="Customer"
          placeholder="Select Customer"
          description="Choose the customer who requested this shipment."
        />
      ),
    },
    {
      name: "formulaTemplateId",
      cols: "full",
      render: () => (
        <FormulaTemplateAutocompleteField
          control={control}
          name="formulaTemplateId"
          label="Rating Method"
          placeholder="Select Rating Method"
          description="Select how the shipment charges are calculated (e.g., per mile, per stop, flat rate)."
          rules={{ required: true }}
        />
      ),
    },
    {
      name: "baseRate",
      cols: "full",
      render: () => (
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
      ),
    },
  ];

  return (
    <Inner>
      <FuelSurchargeChangeDialog
        change={fuelSurchargeChange}
        onResolve={resolveFuelSurchargeChange}
      />
      {appliedRate && (
        <ContractRateAppliedAlert rate={appliedRate} onDismiss={dismissAppliedRate} />
      )}
      {customerId && <CreditHoldAlert customerId={customerId} />}
      {shipmentId && <ProfitabilitySummary shipmentId={shipmentId} />}
      <CapabilityFields descriptors={descriptors} profile={profile} />

      <ChargeSummary isCalculating={isCalculating} error={totalsError} />
      <RatingBreakdownCard />
      <RateDepartureReason />
    </Inner>
  );
}
