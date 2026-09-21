import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import type { ARWorklistItem } from "@/lib/graphql/accounts-receivable";
import { queries } from "@/lib/queries";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { CheckCircle2Icon } from "lucide-react";
import { Link } from "react-router";

const SEVERITY_STYLES: Record<string, string> = {
  Critical: "bg-danger-subtle text-danger-foreground",
  Warning: "bg-warning-subtle text-warning-foreground",
  Watch: "bg-info-subtle text-info-foreground",
};

export function CollectionsWorklistCard() {
  const t = useT();

  const { data: items, isLoading } = useQuery(queries.ar.collectionsWorklist(25));

  const rows = items ?? [];

  return (
    <Card className="gap-0 p-0">
      <CardHeader className="flex flex-row items-center justify-between border-b py-3">
        <CardTitle className="text-sm font-medium">
          {t("Collections worklist")}
          {rows.length > 0 ? (
            <span className="bg-muted text-muted-foreground ml-2 rounded-full px-1.5 py-0.5 text-xs font-medium tabular-nums">
              {rows.length}
            </span>
          ) : null}
        </CardTitle>
        <Link
          to="/accounting/ar/open-items"
          className="text-muted-foreground hover:text-foreground text-xs hover:underline"
        >
          {t("Open items")}
        </Link>
      </CardHeader>
      <CardContent className="p-2">
        {isLoading ? (
          <div className="space-y-2 p-2">
            {Array.from({ length: 6 }).map((_, index) => (
              <Skeleton key={index} className="h-10 w-full" />
            ))}
          </div>
        ) : rows.length === 0 ? (
          <div className="text-muted-foreground flex h-56 flex-col items-center justify-center gap-2 text-sm">
            <CheckCircle2Icon className="size-5 text-success-foreground" />
            {t("Nothing needs attention right now")}
          </div>
        ) : (
          <div className="max-h-80 divide-y overflow-y-auto">
            {rows.map((item) => (
              <WorklistRow key={item.invoiceId} item={item} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function WorklistRow({ item }: { item: ARWorklistItem }) {
  const t = useT();

  return (
    <Link
      to={`/accounting/ar/customer-ledger?customerId=${item.customerId}`}
      className="hover:bg-muted/50 flex items-center gap-3 rounded-md px-2 py-2 transition-colors"
    >
      <span
        className={cn(
          "inline-flex w-16 shrink-0 justify-center rounded-full px-1.5 py-0.5 text-xs font-medium",
          SEVERITY_STYLES[item.severity] ?? SEVERITY_STYLES.Watch,
        )}
      >
        {item.severity}
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline justify-between gap-2">
          <span className="truncate text-xs">
            <span className="font-mono font-medium">{item.invoiceNumber}</span>
            <span className="text-muted-foreground ml-1.5">{item.customerName}</span>
          </span>
          <span className="shrink-0 text-xs font-semibold tabular-nums">
            {formatCurrency(item.openAmountMinor / 100)}
          </span>
        </div>
        <div className="mt-0.5 flex items-center gap-1.5">
          {item.daysPastDue > 0 ? (
            <span className="text-muted-foreground text-xs tabular-nums">
              {t("{0}d past due", item.daysPastDue)}
            </span>
          ) : (
            <span className="text-muted-foreground text-xs">{t("not yet due")}</span>
          )}
          {item.isDisputed ? (
            <Badge variant="warning">
              {item.openDisputeReasonCode
                ? t("Disputed · {0}", item.openDisputeReasonCode)
                : t("Disputed")}
              {item.disputedAmountMinor > 0
                ? ` ${formatCurrency(item.disputedAmountMinor / 100)}`
                : ""}
            </Badge>
          ) : null}
          {item.hasShortPay ? <Badge variant="danger">{t("Short-paid")}</Badge> : null}
        </div>
      </div>
    </Link>
  );
}
