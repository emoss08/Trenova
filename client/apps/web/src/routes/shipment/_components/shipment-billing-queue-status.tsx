import { useT } from "@trenova/shared/i18n/use-t";
import {
  BillingQueueStatusBadge,
  billingQueueStatusBadges,
} from "@trenova/shared/components/status-badge";
import type { BillingQueueStatus } from "@trenova/shared/types/billing-queue";
import type { Shipment } from "@trenova/shared/types/shipment";
import { ExternalLinkIcon } from "lucide-react";
import { Link } from "react-router";

export function ShipmentBillingQueueBadge({ status }: { status?: BillingQueueStatus | null }) {
  const t = useT();

  if (!status) return null;

  return (
    <BillingQueueStatusBadge
      status={status}
      title={t("Billing queue: {0}", t(billingQueueStatusBadges[status].text))}
    />
  );
}

export function ShipmentBillingQueueStatus({ shipment }: { shipment: Shipment }) {
  const t = useT();

  if (!shipment.billingTransferStatus) return null;

  return (
    <div className="flex items-center gap-1.5">
      <ShipmentBillingQueueBadge status={shipment.billingTransferStatus} />
      <Link
        to={`/billing/queue?query=${encodeURIComponent(shipment.proNumber ?? "")}&includePosted=true`}
        className="text-2xs text-primary inline-flex items-center gap-0.5 underline-offset-4 hover:underline"
      >
        {t("View in billing queue")}
        <ExternalLinkIcon className="size-3" />
      </Link>
    </div>
  );
}
