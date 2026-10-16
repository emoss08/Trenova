import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { BadgeVariant } from "@trenova/shared/types/badge";
import { SectionPanel, SectionPanelLink } from "@/components/section-panel";
import type { ARWorklistItem } from "@/lib/graphql/accounts-receivable";
import { ar } from "@/lib/queries/ar";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { CheckCircleIcon } from "@trenova/shared/components/icons";
import { Link } from "react-router";
import { PANEL_LIST_HEIGHT, PanelQuiet, RowsBodySkeleton } from "./dashboard-panel";

const WORKLIST_LIMIT = 25;

type WorklistSeverity = "Critical" | "Warning" | "Watch";

const SEVERITY_LABELS = defineLabels({
  Critical: "Critical",
  Warning: "Warning",
  Watch: "Watch",
} as const);

const SEVERITY_TONE: Record<WorklistSeverity, BadgeVariant> = {
  Critical: "danger",
  Warning: "warning",
  Watch: "info",
};

function severityOf(value: string): WorklistSeverity {
  return value === "Critical" || value === "Warning" ? value : "Watch";
}

export function CollectionsWorklistCard({ className }: { className?: string }) {
  const t = useT();

  const { data: items, isLoading } = useQuery(ar.collectionsWorklist(WORKLIST_LIMIT));

  const rows = items ?? [];

  return (
    <SectionPanel
      title={t("Collections worklist")}
      count={rows.length}
      help={t(
        "The open invoices most worth a call, ordered by how far past due they are and how much is at stake. Disputes and short payments are marked.",
      )}
      action={
        <SectionPanelLink to="/accounting/ar/collections">{t("Open collections")}</SectionPanelLink>
      }
      className={className}
    >
      {isLoading ? (
        <RowsBodySkeleton label={t("Loading the collections worklist")} />
      ) : rows.length === 0 ? (
        <PanelQuiet
          height={PANEL_LIST_HEIGHT}
          icon={<CheckCircleIcon className="text-success-foreground size-5" />}
        >
          {t("Nothing needs attention right now")}
        </PanelQuiet>
      ) : (
        <ul className={cn("divide-border-subtle divide-y overflow-y-auto", PANEL_LIST_HEIGHT)}>
          {rows.map((item) => (
            <li key={item.invoiceId}>
              <WorklistRow item={item} />
            </li>
          ))}
        </ul>
      )}
    </SectionPanel>
  );
}

function disputeLabel(item: ARWorklistItem, t: TranslateFn): string {
  const reason = item.openDisputeReasonCode;
  const amount = item.disputedAmountMinor > 0 ? formatCurrency(item.disputedAmountMinor / 100) : "";

  if (reason !== "" && amount !== "") return t("Disputed · {0} · {1}", reason, amount);
  if (reason !== "") return t("Disputed · {0}", reason);
  if (amount !== "") return t("Disputed · {0}", amount);
  return t("Disputed");
}

function WorklistRow({ item }: { item: ARWorklistItem }) {
  const t = useT();
  const severity = severityOf(item.severity);

  return (
    <Link
      to={`/accounting/ar/collections?customerId=${encodeURIComponent(item.customerId)}`}
      className="hover:bg-surface-hover ui-focus-ring flex items-center gap-3 px-3 py-2 transition-colors"
    >
      <Badge variant={SEVERITY_TONE[severity]} className="w-16">
        {SEVERITY_LABELS[severity]}
      </Badge>
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline justify-between gap-2">
          <span className="truncate text-xs">
            <span className="font-mono">{item.invoiceNumber}</span>
            <span className="text-muted-foreground ml-1.5">{item.customerName}</span>
          </span>
          <span className="shrink-0 font-mono text-xs tabular-nums">
            {formatCurrency(item.openAmountMinor / 100)}
          </span>
        </div>
        <div className="mt-0.5 flex items-center gap-1.5">
          <span className="text-muted-foreground text-xs tabular-nums">
            {item.daysPastDue > 0 ? t("{0}d past due", item.daysPastDue) : t("not yet due")}
          </span>
          {item.isDisputed ? <Badge variant="warning">{disputeLabel(item, t)}</Badge> : null}
          {item.hasShortPay ? <Badge variant="danger">{t("Short-paid")}</Badge> : null}
        </div>
      </div>
    </Link>
  );
}
