import { ChargePayerControl } from "@/components/billing/charge-payer-control";
import { RatingReviewHold } from "@/components/billing/rating-review-hold";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  CustomerAutocompleteField,
  FormulaTemplateAutocompleteField,
  OrderAutocompleteField,
} from "@/components/autocomplete-fields";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { freightTermsChoices } from "@/lib/choices";
import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import { selectOptionMetaString } from "@/lib/select-option-meta";
import { useShipmentAutoRate } from "@/hooks/use-shipment-auto-rate";
import { useShipmentTotalsPreview } from "@/hooks/use-shipment-totals-preview";
import { customer as customerQueries } from "@/lib/queries/customer";
import { shipment as shipmentQueries } from "@/lib/queries/shipment";
import { useQuery, useQueryClient } from "@tanstack/react-query";
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
import { useRules } from "@trenova/shared/hooks/use-rules";
import { getProfile } from "@trenova/shared/lib/capability";
import { allocationRemainders } from "@trenova/shared/rules/allocation";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { CreditStatus } from "@trenova/shared/types/customer";
import type {
  ContractRate,
  GetPreviousRatesRequest,
  Shipment,
} from "@trenova/shared/types/shipment";
import { AlertTriangleIcon, ShieldAlertIcon } from "@trenova/shared/components/icons";
import type React from "react";
import { useEffect, useRef } from "react";
import { Link } from "react-router";
import { BillView } from "@/components/tariff-bill/bill-view";
import { ChoicesView } from "@/components/tariff-bill/choices-view";
import { readStoredTariffBill, readStoredTariffChoices } from "@trenova/shared/types/tariff-form";
import { formulaTemplateRoutes } from "@/lib/formula-template-routes";
import { useFormContext, useWatch } from "react-hook-form";
import { useShipmentDefaultPayer } from "./use-shipment-default-payer";
import { FuelSurchargeChangeDialog } from "./additional-charges/fuel-surcharge-change-dialog";
import { AutoRateDialog } from "./auto-rate-dialog";
import { BillingByPayerCard } from "./billing-by-payer-card";
import { PreviousRatesButton } from "./previous-rates-dialog";
import { ProfitabilitySummary } from "./profitability/profitability-summary";
import { ShipmentTariffInputs } from "./shipment-tariff-inputs";
import { WhyThisRate } from "./why-this-rate";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";

function Inner({ children }: { children: React.ReactNode }) {
  const t = useT();

  const { control } = useFormContext<Shipment>();

  const serviceTypeId = useWatch({ control, name: "serviceTypeId" });
  const shipmentTypeId = useWatch({ control, name: "shipmentTypeId" });
  const customerId = useWatch({ control, name: "customerId" });
  const moves = useWatch({ control, name: "moves" });
  const originLocationId = moves?.[0]?.stops?.[0]?.locationId ?? "";
  const lastMove = moves?.[moves.length - 1];
  const destinationLocationId = lastMove?.stops?.[lastMove.stops.length - 1]?.locationId ?? "";
  const shipmentId = useWatch({ control, name: "id" });

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
      title={t("Billing & rating")}
      description={t("Customer, rating method, and charge amounts")}
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
    label: "Credit hold",
  },
  Suspended: {
    variant: "destructive",
    icon: ShieldAlertIcon,
    label: "Credit suspended",
  },
  Warning: {
    variant: "warning",
    icon: AlertTriangleIcon,
    label: "Credit warning",
  },
  Review: {
    variant: "warning",
    icon: AlertTriangleIcon,
    label: "Under credit review",
  },
};

/** What the customer's credit standing stops, in the words dispatch reads. */
function creditConsequence(
  profile: {
    creditStatus?: string | null;
    enforceCreditLimit?: boolean | null;
    creditHoldAutomatic?: boolean | null;
  },
  t: ReturnType<typeof useT>,
): string {
  switch (profile.creditStatus) {
    case "Warning":
      return t("This customer is close to their credit limit.");
    case "Review":
      return t("This customer's credit is under review.");
    case "Hold":
    case "Suspended":
      if (!profile.enforceCreditLimit) {
        return t("Their credit limit isn't enforced, so nothing is blocked.");
      }
      return profile.creditHoldAutomatic
        ? t(
            "No new shipments can be booked for them until the balance is paid back under the limit. Shipments already booked are still billed.",
          )
        : t(
            "No new shipments can be booked for them, and their invoices wait until accounts receivable lifts the hold.",
          );
    default:
      return "";
  }
}

function CreditHoldAlert({ customerId, payerName }: { customerId: string; payerName?: string }) {
  const t = useT();

  const { data: billingProfile } = useQuery({
    ...customerQueries.getBillingProfile(customerId),
    enabled: !!customerId,
  });

  if (!billingProfile) return null;

  const config = CREDIT_STATUS_CONFIG[billingProfile.creditStatus as CreditStatus];
  if (!config) return null;

  const Icon = config.icon;

  return (
    <Alert variant={config.variant} className="mb-3" data-testid={`credit-alert-${customerId}`}>
      <Icon className="size-4" />
      <AlertTitle>{payerName ? `${t(config.label)} — ${payerName}` : t(config.label)}</AlertTitle>
      <AlertDescription>
        {[billingProfile.creditHoldReason, creditConsequence(billingProfile, t)]
          .filter(Boolean)
          .join(". ")}
      </AlertDescription>
    </Alert>
  );
}

/**
 * Everyone a save would invoice: the shipment's payer plus every customer named
 * on a freight or accessorial split. Each gets their own credit check, since a
 * hold on any of them stops that payer's invoice.
 */
export function collectPayerIds(shipment: {
  customerId?: string | null;
  billToCustomerId?: string | null;
  freightAllocations?: readonly { billToCustomerId?: string | null }[] | null;
  additionalCharges?:
    | readonly ({ allocations?: readonly { billToCustomerId?: string | null }[] | null } | null)[]
    | null;
}): string[] {
  const ids: string[] = [];
  const push = (id: string | null | undefined) => {
    if (id && !ids.includes(id)) ids.push(id);
  };
  push(shipment.billToCustomerId || shipment.customerId);
  for (const row of shipment.freightAllocations ?? []) push(row?.billToCustomerId);
  for (const charge of shipment.additionalCharges ?? []) {
    for (const row of charge?.allocations ?? []) push(row?.billToCustomerId);
  }
  return ids;
}

function CreditHoldAlerts() {
  const { control } = useFormContext<Shipment>();
  const customerId = useWatch({ control, name: "customerId" });
  const billToCustomerId = useWatch({ control, name: "billToCustomerId" });
  const freightAllocations = useWatch({ control, name: "freightAllocations" });
  const additionalCharges = useWatch({ control, name: "additionalCharges" });
  const customer = useWatch({ control, name: "customer" });
  const billToCustomer = useWatch({ control, name: "billToCustomer" });

  const payerIds = collectPayerIds({
    customerId,
    billToCustomerId,
    freightAllocations,
    additionalCharges,
  });
  if (payerIds.length === 0) return null;

  const nameFor = (id: string) => {
    if (id === customer?.id) return customer.name;
    if (id === billToCustomer?.id) return billToCustomer.name;
    for (const row of freightAllocations ?? []) {
      if (row?.billToCustomer?.id === id) return row.billToCustomer.name;
    }
    for (const charge of additionalCharges ?? []) {
      for (const row of charge?.allocations ?? []) {
        if (row?.billToCustomer?.id === id) return row.billToCustomer.name;
      }
    }
    return undefined;
  };

  return (
    <>
      {payerIds.map((id) => (
        <CreditHoldAlert
          key={id}
          customerId={id}
          payerName={payerIds.length > 1 ? nameFor(id) : undefined}
        />
      ))}
    </>
  );
}

function ChargeSummaryRow({
  label,
  value,
  bold,
  action,
}: {
  label: string;
  value: number | null | undefined;
  bold?: boolean;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between">
      <span
        className={cn(
          "flex items-center gap-1.5 text-sm",
          bold ? "text-foreground font-medium" : "text-muted-foreground",
        )}
      >
        {label}
        {action}
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
  const t = useT();

  const { control } = useFormContext<Shipment>();
  const otherChargeAmount = useWatch({ control, name: "otherChargeAmount" });
  const totalChargeAmount = useWatch({ control, name: "totalChargeAmount" });
  const freightChargeAmount = useWatch({ control, name: "freightChargeAmount" });
  const freightAllocations = useWatch({ control, name: "freightAllocations" });
  const defaultPayer = useShipmentDefaultPayer();
  const rules = useRules();
  const freightTotal = freightChargeAmount ?? 0;
  const [freightSplit] = allocationRemainders(rules, {
    charges: [{ kind: "Freight", total: freightTotal, allocations: freightAllocations }],
  });

  return (
    <div className="bg-muted/50 relative mt-3 overflow-hidden rounded-lg border p-2">
      {isCalculating && (
        <div className="bg-background/50 absolute inset-0 z-10 flex items-center justify-center rounded-lg backdrop-blur-[2px]">
          <TextShimmer as="span" className="text-sm font-medium" duration={1.5}>
            {t("Calculating...")}
          </TextShimmer>
        </div>
      )}
      {error && !isCalculating && (
        <div className="bg-danger-subtle absolute inset-0 z-10 flex items-center justify-center rounded-lg backdrop-blur-[2px]">
          <div className="text-destructive flex items-center gap-2">
            <AlertTriangleIcon className="size-4" />
            <span className="text-sm font-medium">{error}</span>
          </div>
        </div>
      )}
      <div className="mb-3">
        <span className="text-xs font-medium">{t("Charge summary")}</span>
        <p className="text-2xs text-muted-foreground mt-0.5">
          {t(
            "Automatically calculated based on the rating method, freight charges, and any additional accessorial charges.",
          )}
        </p>
      </div>
      <div className="space-y-2">
        <ChargeSummaryRow
          label={t("Freight charges")}
          value={freightChargeAmount}
          action={
            <ChargePayerControl
              name="freightAllocations"
              chargeAmount={freightTotal}
              complete={freightSplit.complete}
              defaultPayer={defaultPayer}
              splitTitle={t("Split freight charge")}
              splitDescription={t(
                "Divide the {0} freight charge between the customers who pay for it. Each payer receives an invoice for their share.",
                formatCurrency(Number(freightChargeAmount ?? 0)),
              )}
            />
          }
        />
        <ChargeSummaryRow label={t("Other charges")} value={otherChargeAmount} />
        <Separator className="my-2" />
        <ChargeSummaryRow label={t("Total")} value={totalChargeAmount} bold />
      </div>
    </div>
  );
}

/**
 * Keeps the bill-to customer honest when the ordering customer changes. A
 * bill-to chosen for one customer means nothing for another, so it is cleared
 * along with the splits that named it; a bill-to seeded by a rate agreement
 * arrives with the agreement and is left alone.
 */
function useBillToFollowsCustomer() {
  const { control, setValue, getValues } = useFormContext<Shipment>();
  const customerId = useWatch({ control, name: "customerId" });
  const previous = useRef<string | null | undefined>(customerId);

  useEffect(() => {
    if (previous.current === customerId) return;
    const hadCustomer = Boolean(previous.current);
    previous.current = customerId;
    if (!hadCustomer) return;

    const opts = { shouldDirty: true, shouldValidate: false };
    if (getValues("billToCustomerId")) {
      setValue("billToCustomerId", null, opts);
      setValue("billToCustomer", null as never, opts);
    }
    if ((getValues("freightAllocations") ?? []).length > 0) {
      setValue("freightAllocations", [], opts);
    }
    const charges = getValues("additionalCharges") ?? [];
    if (charges.some((charge) => (charge?.allocations ?? []).length > 0)) {
      setValue(
        "additionalCharges",
        charges.map((charge) => ({ ...charge, allocations: [] })),
        opts,
      );
    }
  }, [customerId, getValues, setValue]);
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
  const t = useT();

  return (
    <Alert variant="info" className="mb-3">
      <AssistMark className="size-4" />
      <AlertTitle>
        {t(
          "Rated from {0}{1}",
          rate.agreementName || t("a rate agreement"),
          rate.ruleLabel ? ` — ${rate.ruleLabel}` : "",
        )}
      </AlertTitle>
      <AlertDescription>
        <span>
          {rate.charges.length > 0
            ? t(
                "The rating method and its input values below came from the contract, along with {0, plural, one {# automatic charge} other {# automatic charges}}. Change any of them and this shipment is priced by hand instead.",
                rate.charges.length,
              )
            : t(
                "The rating method and its input values below came from the contract. Change any of them and this shipment is priced by hand instead.",
              )}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="mt-1 h-6 px-1.5"
          onClick={onDismiss}
        >
          <span className="text-2xs">{t("Dismiss")}</span>
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
  const t = useT();

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
        label={t("Reason for the rate change")}
        placeholder={t("Why is this shipment priced differently from its contract?")}
        description={t(
          "This shipment no longer charges what its rate agreement says. The reason is kept with the rating history and shown on the rate leakage report.",
        )}
      />
    </div>
  );
}

/**
 * The bill the shipment was last rated to, line by line: each freight and accessorial charge
 * with its quantity and rate, the notes and any adjustment, and what each `best` or `worst`
 * chose with every option's subtotal, from the tariff version that ran.
 */
function RatingBillCard() {
  const t = useT();

  const { control } = useFormContext<Shipment>();
  const ratingDetail = useWatch({ control, name: "ratingDetail" });
  const shipmentId = useWatch({ control, name: "id" });
  // A shipment read over GraphQL reaches the form unparsed, so the stored bill is read here.
  const bill = readStoredTariffBill(ratingDetail?.bill);
  const choices = readStoredTariffChoices(ratingDetail?.choices);

  if (!ratingDetail || (!bill && !ratingDetail.refusal)) {
    return null;
  }

  // The contract that priced the shipment leads, since that is what somebody
  // reading an invoice recognises; the tariff's own name follows it.
  const source = ratingDetail.agreementName || ratingDetail.formulaTemplateName;

  return (
    <div className="bg-muted/50 mt-3 rounded-lg border p-2">
      <div className="mb-3 flex items-start justify-between gap-2">
        <div className="flex w-full flex-col gap-1">
          <div className="flex items-center justify-between">
            <div className="flex flex-row items-center gap-1">
              <span className="text-xs font-medium">{t("Bill")}</span>
              <AutoRateDialog />
            </div>
            <WhyThisRate shipmentId={shipmentId} />
          </div>
          <p className="text-2xs text-muted-foreground mt-0.5">
            {ratingDetail.ruleLabel
              ? t("{0} — {1}", source, ratingDetail.ruleLabel)
              : t("Billed by {0}", source || t("the tariff"))}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          {ratingDetail.versionNumber ? (
            <Badge variant="neutral" appearance="outline" className="text-2xs font-mono">
              {t("v{0}", ratingDetail.versionNumber)}
            </Badge>
          ) : null}
          {ratingDetail.formulaTemplateId && (
            <Link
              to={formulaTemplateRoutes.edit(ratingDetail.formulaTemplateId)}
              className="text-2xs text-primary hover:underline"
            >
              {t("Open tariff")}
            </Link>
          )}
        </div>
      </div>

      {ratingDetail.refusal ? (
        <Alert variant="destructive" size="sm">
          <AlertTriangleIcon className="size-4" />
          <AlertTitle>{t("The tariff refused to rate this shipment")}</AlertTitle>
          <AlertDescription>{ratingDetail.refusal}</AlertDescription>
        </Alert>
      ) : null}
      {bill && shipmentId ? (
        <ShipmentRatingReview
          shipmentId={shipmentId}
          reasons={bill.reviews.map((review) => review.text)}
          ratingDetail={ratingDetail}
        />
      ) : null}
      {bill ? <BillView bill={bill} currency={bill.currency || undefined} /> : null}
      {choices.length > 0 && (
        <ChoicesView
          choices={choices}
          currency={bill?.currency || undefined}
          className="mt-3 border-t pt-2"
        />
      )}
      {(ratingDetail.skipped ?? []).length > 0 && (
        <p className="text-2xs text-muted-foreground mt-2">
          {t(
            "{0, plural, one {Not added again, since the tariff billed it:} other {Not added again, since the tariff billed them:}} {1}",
            ratingDetail.skipped?.length ?? 0,
            (ratingDetail.skipped ?? []).map((skip) => skip.code).join(", "),
          )}
        </p>
      )}
    </div>
  );
}

/**
 * The reviews the tariff asked for on this rating, and the approval that releases the shipment to
 * be billed. Approving bumps the shipment's version, so the form takes the saved rating and
 * version back rather than refetching over unsaved edits.
 */
function ShipmentRatingReview({
  shipmentId,
  reasons,
  ratingDetail,
}: {
  shipmentId: string;
  reasons: string[];
  ratingDetail: NonNullable<Shipment["ratingDetail"]>;
}) {
  const { setValue } = useFormContext<Shipment>();
  const queryClient = useQueryClient();
  const approval = ratingDetail.reviewApproval;
  const awaiting = ratingDetail.awaitingReview ?? (reasons.length > 0 && !approval);

  return (
    <div className="mb-3">
      <RatingReviewHold
        shipmentId={shipmentId}
        reasons={reasons}
        awaiting={awaiting}
        approvedAt={approval?.approvedAt}
        approvedByName={approval?.approvedBy?.name}
        onApproved={(saved) => {
          setValue("ratingDetail", saved.ratingDetail, { shouldDirty: false });
          setValue("version", saved.version, { shouldDirty: false });
          void queryClient.invalidateQueries({ queryKey: ["shipment"] });
          void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
          void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
        }}
      />
    </div>
  );
}

export default function ShipmentBillingDetails() {
  const t = useT();

  const { control, setValue } = useFormContext<Shipment>();
  const customerId = useWatch({ control, name: "customerId" });
  const customer = useWatch({ control, name: "customer" });
  const shipmentId = useWatch({ control, name: "id" });
  const {
    isCalculating,
    error: totalsError,
    fuelSurchargeChange,
    resolveFuelSurchargeChange,
  } = useShipmentTotalsPreview();
  const { data: shipmentUIPolicy } = useQuery({ ...shipmentQueries.uiPolicy() });

  // A contract prices a shipment once, while it is being typed. An existing
  // one already carries a rate somebody may have negotiated, and replacing it
  // on open is exactly what this design exists to stop.
  const { appliedRate, dismissAppliedRate } = useShipmentAutoRate({ enabled: !shipmentId });
  useBillToFollowsCustomer();

  const profile = getProfile(shipmentUIPolicy);

  const handleBillToSelected = (option: GraphQLSelectOption | null) => {
    setValue(
      "billToCustomer",
      (option
        ? { id: option.id, name: option.label, code: selectOptionMetaString(option, "code") }
        : null) as never,
      { shouldDirty: true },
    );
  };

  const descriptors: FieldDescriptor[] = [
    {
      name: "orderId",
      render: () => (
        <OrderAutocompleteField
          control={control}
          name="orderId"
          label={t("Order")}
          placeholder={t("Select order")}
          description={t(
            "Optionally group this shipment under a commercial order for the same customer. Set on creation; use the order's Add Legs afterwards.",
          )}
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
          label={t("Customer")}
          placeholder={t("Select customer")}
          description={t("Choose the customer who requested this shipment.")}
        />
      ),
    },
    {
      name: "billToCustomerId",
      render: () => (
        <CustomerAutocompleteField
          control={control}
          name="billToCustomerId"
          label={t("Bill To")}
          clearable
          disabled={!customerId}
          placeholder={
            customer?.name ? t("Same as customer ({0})", customer.name) : t("Same as customer")
          }
          description={t(
            "The customer invoiced for this shipment when it is not the one who ordered it. Leave blank to bill the customer.",
          )}
          onOptionChange={handleBillToSelected}
        />
      ),
    },
    {
      name: "freightTerms",
      render: () => (
        <SelectField
          control={control}
          name="freightTerms"
          label={t("Freight terms")}
          options={freightTermsChoices}
          placeholder={t("Select freight terms")}
          description={t(
            "Who is responsible for the freight charges: prepaid by the shipper, collect from the consignee, or a third party.",
          )}
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
          label={t("Rating method")}
          placeholder={t("Select rating method")}
          description={t(
            "The tariff that bills this shipment. Its inputs, below, are the values it rates with.",
          )}
          rules={{ required: true }}
        />
      ),
    },
    {
      name: "tariffInputs",
      cols: "full",
      render: () => <ShipmentTariffInputs />,
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
      <CreditHoldAlerts />
      {shipmentId && <ProfitabilitySummary shipmentId={shipmentId} />}
      <CapabilityFields descriptors={descriptors} profile={profile} />

      <ChargeSummary isCalculating={isCalculating} error={totalsError} />
      <BillingByPayerCard />
      <RatingBillCard />
      <RateDepartureReason />
    </Inner>
  );
}
