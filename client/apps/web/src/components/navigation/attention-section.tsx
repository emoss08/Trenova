import { SidebarNavLink, SidebarSectionLabel } from "@/components/navigation/sidebar-primitives";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import type { AttentionSummaryQuery } from "@trenova/graphql/generated/graphql";
import { useAttentionSummary } from "@/hooks/use-attention";
import { useSidebarPreferences } from "@/hooks/use-sidebar-preferences";
import { isRouteActive } from "@/lib/route-utils";
import { cn } from "@trenova/shared/lib/utils";
import { useLocation } from "react-router";

type AttentionSummary = AttentionSummaryQuery["attentionSummary"];
type AttentionTone = "default" | "warning" | "destructive";

interface AttentionRowConfig {
  key: keyof AttentionSummary;
  label: string;
  module: string;
  path: string;
  tone: AttentionTone;
}

const ATTENTION_ROWS: AttentionRowConfig[] = [
  {
    key: "billingQueue",
    label: "Billing Queue",
    module: "Billing",
    path: "/billing/queue",
    tone: "default",
  },
  {
    key: "pendingApprovals",
    label: "Pending Approvals",
    module: "Billing",
    path: "/billing/pending-approvals",
    tone: "default",
  },
  {
    key: "reconciliationExceptions",
    label: "Reconciliation Exceptions",
    module: "Billing",
    path: "/billing/reconciliation-exceptions",
    tone: "destructive",
  },
  {
    key: "serviceFailures",
    label: "Service Failures",
    module: "Shipments",
    path: "/shipment-management/service-failures",
    tone: "warning",
  },
  {
    key: "ediAttention",
    label: "EDI Needs Attention",
    module: "EDI",
    path: "/edi/overview",
    tone: "warning",
  },
];

const ATTENTION_ROWS_BY_KEY = new Map(ATTENTION_ROWS.map((row) => [row.key as string, row]));

const TONE_DOT_CLASSES: Record<AttentionTone, string> = {
  default: "bg-info",
  warning: "bg-warning",
  destructive: "bg-destructive",
};

const TONE_BADGE_VARIANTS: Record<AttentionTone, BadgeVariant> = {
  default: "secondary",
  warning: "warning",
  destructive: "inactive",
};

function AttentionRow({
  row,
  count,
  currentPath,
}: {
  row: AttentionRowConfig;
  count: number;
  currentPath: string;
}) {
  const hasWork = count > 0;

  return (
    <SidebarNavLink to={row.path} active={isRouteActive(currentPath, row.path)} className="h-7">
      <span
        className={cn(
          "size-1.5 shrink-0 rounded-full",
          hasWork ? TONE_DOT_CLASSES[row.tone] : "bg-muted-foreground/40",
        )}
      />
      <span className="min-w-0 flex-1 truncate">{row.label}</span>
      <span className="text-2xs text-muted-foreground/70">{row.module}</span>
      <Badge
        variant={hasWork ? TONE_BADGE_VARIANTS[row.tone] : "secondary"}
        className="max-h-4 min-w-5 justify-center px-1.5 text-2xs tabular-nums"
      >
        {count > 99 ? "99+" : count}
      </Badge>
    </SidebarNavLink>
  );
}

export function AttentionSection() {
  const { pathname } = useLocation();
  const { data: summary, isLoading } = useAttentionSummary();
  const { data: preferences } = useSidebarPreferences();

  const rows = (preferences?.attentionMetrics ?? [])
    .map((key) => ATTENTION_ROWS_BY_KEY.get(key))
    .filter((row): row is AttentionRowConfig => row != null && summary?.[row.key] != null);

  if (!isLoading && rows.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-0.5">
      <SidebarSectionLabel>Needs Attention</SidebarSectionLabel>
      {isLoading
        ? Array.from({ length: 3 }, (_, index) => (
            <Skeleton key={index} className="h-7 w-full rounded-md" />
          ))
        : rows.map((row) => (
            <AttentionRow
              key={row.key}
              row={row}
              count={summary?.[row.key] ?? 0}
              currentPath={pathname}
            />
          ))}
    </div>
  );
}
