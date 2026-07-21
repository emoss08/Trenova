import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { ARWorklistItem } from "@/lib/graphql/accounts-receivable";
import { queries } from "@/lib/queries";
import { cn, formatCurrency } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { CheckCircle2Icon } from "lucide-react";
import { m } from "motion/react";
import { Link } from "react-router";

const SEVERITY_STYLES: Record<string, string> = {
  Critical: "bg-red-500/15 text-red-700 dark:text-red-400",
  Warning: "bg-amber-500/15 text-amber-700 dark:text-amber-400",
  Watch: "bg-blue-500/15 text-blue-700 dark:text-blue-400",
};

export function CollectionsWorklistCard() {
  const { data: items, isLoading } = useQuery(queries.ar.collectionsWorklist(25));

  const rows = items ?? [];

  return (
    <Card className="gap-0 p-0">
      <CardHeader className="flex flex-row items-center justify-between border-b py-3">
        <CardTitle className="text-sm font-medium">
          Collections worklist
          {rows.length > 0 ? (
            <span className="ml-2 rounded-full bg-muted px-1.5 py-0.5 text-[11px] font-medium text-muted-foreground tabular-nums">
              {rows.length}
            </span>
          ) : null}
        </CardTitle>
        <Link
          to="/accounting/ar/open-items"
          className="text-xs text-muted-foreground hover:text-foreground hover:underline"
        >
          Open items
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
          <div className="flex h-56 flex-col items-center justify-center gap-2 text-sm text-muted-foreground">
            <CheckCircle2Icon className="size-5 text-emerald-500" />
            Nothing needs attention right now
          </div>
        ) : (
          <div className="max-h-80 divide-y overflow-y-auto">
            {rows.map((item, index) => (
              <WorklistRow key={item.invoiceId} item={item} index={index} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function WorklistRow({ item, index }: { item: ARWorklistItem; index: number }) {
  return (
    <m.div
      initial={{ opacity: 0, x: -6 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.25, delay: index * 0.03, ease: "easeOut" }}
    >
      <Link
        to={`/accounting/ar/customer-ledger?customerId=${item.customerId}`}
        className="flex items-center gap-3 rounded-md px-2 py-2 transition-colors hover:bg-muted/50"
      >
        <span
          className={cn(
            "inline-flex w-16 shrink-0 justify-center rounded-full px-1.5 py-0.5 text-[11px] font-medium",
            SEVERITY_STYLES[item.severity] ?? SEVERITY_STYLES.Watch,
          )}
        >
          {item.severity}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-baseline justify-between gap-2">
            <span className="truncate text-xs">
              <span className="font-mono font-medium">{item.invoiceNumber}</span>
              <span className="ml-1.5 text-muted-foreground">{item.customerName}</span>
            </span>
            <span className="shrink-0 text-xs font-semibold tabular-nums">
              {formatCurrency(item.openAmountMinor / 100)}
            </span>
          </div>
          <div className="mt-0.5 flex items-center gap-1.5">
            {item.daysPastDue > 0 ? (
              <span className="text-[11px] text-muted-foreground tabular-nums">
                {item.daysPastDue}d past due
              </span>
            ) : (
              <span className="text-[11px] text-muted-foreground">not yet due</span>
            )}
            {item.isDisputed ? <Badge variant="orange">Disputed</Badge> : null}
            {item.hasShortPay ? <Badge variant="inactive">Short-paid</Badge> : null}
          </div>
        </div>
      </Link>
    </m.div>
  );
}
