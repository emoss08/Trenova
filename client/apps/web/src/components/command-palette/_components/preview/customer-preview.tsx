import {
  findChoice,
  billingCycleChoices,
  creditStatusChoices,
  customerPaymentTermChoices,
  invoiceDeliveryChoices,
} from "@/lib/choices";
import { StatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { phaseTone, type StatusPhase } from "@trenova/shared/lib/status-phase";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { Progress } from "@trenova/shared/components/ui/progress";
import { useQuery } from "@tanstack/react-query";
import { PALETTE_ENTITIES } from "../../palette-entities";
import type { PaletteAction, PaletteIntent, PaletteRecord } from "../../palette-model";
import { PreviewError, PreviewFrame, PreviewSection, PreviewSkeleton } from "./preview-frame";
import { customerPreviewQuery, type CustomerPreviewData } from "./preview-queries";

type CreditStatus = NonNullable<NonNullable<CustomerPreviewData>["billingProfile"]>["creditStatus"];

const CREDIT_STATUS_PHASE: Record<CreditStatus, StatusPhase> = {
  Active: "complete",
  Warning: "attention",
  Hold: "failed",
  Suspended: "closed",
  Review: "awaiting",
};

function money(value: string | null | undefined): string | null {
  if (value == null || value === "") {
    return null;
  }
  const amount = Number(value);
  return Number.isFinite(amount) ? formatCurrency(amount) : null;
}

export function CustomerPreview({
  record,
  actions,
  onRun,
}: {
  record: PaletteRecord;
  actions: readonly PaletteAction[];
  onRun: (intent: PaletteIntent) => void;
}) {
  const t = useT();
  const { data: customer, isLoading, isError } = useQuery(customerPreviewQuery(record.id));
  const entity = PALETTE_ENTITIES.customer;

  if (isLoading) {
    return <PreviewSkeleton />;
  }
  if (isError || !customer) {
    return <PreviewError />;
  }

  const billing = customer.billingProfile;
  const place = [customer.city, customer.state?.abbreviation].filter(Boolean).join(", ");
  const limit = money(billing?.creditLimit);
  const balance = money(billing?.creditBalance);
  const creditLimit = billing?.creditLimit ? Number(billing.creditLimit) : 0;
  const used =
    billing && creditLimit > 0
      ? Math.min(100, Math.max(0, (Number(billing.creditBalance) / creditLimit) * 100))
      : null;

  return (
    <PreviewFrame
      icon={entity.icon}
      tileClass={entity.tileClass}
      title={customer.name}
      subtitle={[customer.code, place].filter(Boolean).join(" · ")}
      badge={<StatusBadge status={customer.status} />}
      actions={actions}
      onRun={onRun}
    >
      <PreviewSection title={t("Location")}>
        <DescriptionList columns={2}>
          <DescriptionItem label={t("Address")} span="full">
            {customer.addressLine1 ? (
              [customer.addressLine1, place, customer.postalCode].filter(Boolean).join(", ")
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
        </DescriptionList>
      </PreviewSection>
      {billing && (
        <PreviewSection title={t("Credit and billing")}>
          <DescriptionList columns={2}>
            <DescriptionItem label={t("Credit status")}>
              <Badge variant={phaseTone(CREDIT_STATUS_PHASE[billing.creditStatus])}>
                {t(
                  findChoice(creditStatusChoices, billing.creditStatus)?.label ??
                    billing.creditStatus,
                )}
              </Badge>
            </DescriptionItem>
            <DescriptionItem label={t("Payment terms")}>
              {t(
                findChoice(customerPaymentTermChoices, billing.paymentTerm)?.label ??
                  billing.paymentTerm,
              )}
            </DescriptionItem>
            <DescriptionItem label={t("Balance")} numeric>
              {balance ?? <DescriptionEmpty />}
            </DescriptionItem>
            <DescriptionItem label={t("Credit limit")} numeric>
              {limit ?? t("No limit")}
            </DescriptionItem>
            {used !== null && (
              <DescriptionItem label={t("Credit used")} span="full">
                <Progress
                  value={used}
                  size="sm"
                  showLabel
                  variant={used >= 90 ? "error" : used >= 75 ? "warning" : "success"}
                />
              </DescriptionItem>
            )}
            <DescriptionItem label={t("Invoices")}>
              {t(
                findChoice(invoiceDeliveryChoices, billing.invoiceDelivery)?.label ??
                  billing.invoiceDelivery,
              )}
            </DescriptionItem>
            <DescriptionItem label={t("Billing cycle")}>
              {t(
                findChoice(billingCycleChoices, billing.billingCycle)?.label ??
                  billing.billingCycle,
              )}
            </DescriptionItem>
          </DescriptionList>
        </PreviewSection>
      )}
      <p className="text-2xs text-foreground-subtle">
        {t("Updated {0}", formatUnixDateMedium(customer.updatedAt))}
      </p>
    </PreviewFrame>
  );
}
