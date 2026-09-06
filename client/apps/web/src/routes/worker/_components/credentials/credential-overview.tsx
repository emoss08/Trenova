import type { WorkerCredentialSummary } from "@/lib/graphql/worker-credential";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { RingGauge, type RingGaugeTone } from "@trenova/shared/components/ui/ring-gauge";
import { cn } from "@trenova/shared/lib/utils";
import { PlusIcon } from "lucide-react";
import { useMemo } from "react";

type CredentialOverviewProps = {
  summary: WorkerCredentialSummary;
  canCreate: boolean;
  onAdd: () => void;
};

const COMPLIANCE_CHIP: Record<
  string,
  { label: string; variant: "active" | "inactive" | "warning" }
> = {
  Compliant: { label: "Compliant", variant: "active" },
  NonCompliant: { label: "Non-compliant", variant: "inactive" },
  Pending: { label: "Pending review", variant: "warning" },
};

export function requiredHealthyCount(summary: WorkerCredentialSummary): number {
  return summary.items.filter(
    (item) => item.required && (item.health === "Valid" || item.health === "ExpiringSoon"),
  ).length;
}

export function CredentialOverview({ summary, canCreate, onAdd }: CredentialOverviewProps) {
  const healthy = useMemo(() => requiredHealthyCount(summary), [summary]);
  const ratio = summary.requiredCount === 0 ? 1 : healthy / summary.requiredCount;
  const tone: RingGaugeTone =
    summary.complianceStatus === "NonCompliant"
      ? "critical"
      : summary.expiringCount > 0
        ? "warning"
        : "success";
  const chip = COMPLIANCE_CHIP[summary.complianceStatus] ?? COMPLIANCE_CHIP.Pending;

  return (
    <div
      data-testid="credential-overview"
      className="bg-card border-border flex flex-wrap items-center gap-4 rounded-xl border p-4"
    >
      <RingGauge
        value={ratio}
        size={72}
        strokeWidth={7}
        tone={tone}
        aria-label="Required credentials"
      >
        <span className="text-sm font-semibold tabular-nums">
          {healthy}/{summary.requiredCount}
        </span>
      </RingGauge>

      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-semibold">Qualification file</h3>
          <Badge variant={chip.variant}>{chip.label}</Badge>
        </div>
        <p className="text-muted-foreground text-xs">
          {summary.requiredCount === 0
            ? "No credential types are required for this worker."
            : `${healthy} of ${summary.requiredCount} required credentials are in good standing.`}
        </p>
        <div className="mt-1 flex flex-wrap gap-2">
          <StatTile label="Expiring" value={summary.expiringCount} tone="warning" />
          <StatTile label="Expired" value={summary.expiredCount} tone="critical" />
          <StatTile label="Missing" value={summary.missingCount} tone="muted" />
        </div>
      </div>

      {canCreate ? (
        <Button size="sm" onClick={onAdd}>
          <PlusIcon className="size-3.5" />
          Add credential
        </Button>
      ) : null}
    </div>
  );
}

function StatTile({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone: "warning" | "critical" | "muted";
}) {
  const active = value > 0;
  return (
    <div
      className={cn(
        "flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs",
        !active && "text-muted-foreground border-dashed",
        active &&
          tone === "warning" &&
          "border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-400",
        active &&
          tone === "critical" &&
          "border-red-500/40 bg-red-500/10 text-red-700 dark:text-red-400",
        active && tone === "muted" && "border-border bg-muted/50",
      )}
    >
      <span className="font-semibold tabular-nums">{value}</span>
      <span>{label}</span>
    </div>
  );
}
