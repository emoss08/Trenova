import { ExternalLinkIcon } from "lucide-react";
import { Link } from "react-router";

const SOURCE_ROUTE_MAP: Record<string, (id: string) => string> = {
  manual_journal: (id) => `/accounting/manual-journals/${id}`,
  journal_reversal: (id) => `/accounting/journal-reversals/${id}`,
  invoice: (id) => `/billing/invoices?item=${id}`,
  customer_payment: (id) => `/accounting/ar/customer-ledger?paymentId=${id}`,
  shipment: (id) => `/shipment-management/shipments?item=${id}`,
};

type SourceDrillDownLinkProps = {
  sourceType: string;
  sourceId: string;
  label?: string;
};

export function SourceDrillDownLink({ sourceType, sourceId, label }: SourceDrillDownLinkProps) {
  const routeBuilder = SOURCE_ROUTE_MAP[sourceType];

  if (!routeBuilder) {
    return (
      <span className="text-xs text-muted-foreground">
        {label ?? sourceType}: {sourceId}
      </span>
    );
  }

  return (
    <Link
      to={routeBuilder(sourceId)}
      className="inline-flex items-center gap-0.5 text-xs text-muted-foreground hover:text-foreground hover:underline"
    >
      {label ?? sourceType}
      <ExternalLinkIcon className="size-2.5" />
    </Link>
  );
}
