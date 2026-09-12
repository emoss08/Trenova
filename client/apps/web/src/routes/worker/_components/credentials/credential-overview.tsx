import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import type { WorkerCredentialSummary } from "@/lib/graphql/worker-credential";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
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

/**
 * The roll-up. The one figure that matters is how many required credentials
 * are in good standing; the rest are counts a manager scans, so they are
 * plain numbers and the compliance badge is the only colour.
 */
export function CredentialOverview({ summary, canCreate, onAdd }: CredentialOverviewProps) {
  const t = useT();

  const healthy = useMemo(() => requiredHealthyCount(summary), [summary]);
  const chip = COMPLIANCE_CHIP[summary.complianceStatus] ?? COMPLIANCE_CHIP.Pending;

  return (
    <div data-testid="credential-overview" className="flex flex-col gap-4 rounded-lg border p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="text-sm font-semibold">{t("Qualification file")}</h3>
          <Badge variant={chip.variant}>{t(chip.label)}</Badge>
          <InfoPopover title={t("Qualification file")}>
            <p>
              {t("Each credential type holds one active credential per worker; renewing files a new one and archives the old. Health is graded from the expiry against the type's renewal window: Valid, Expiring soon while inside the window, Expired once past it, and Missing when nothing active is on file.")}
            </p>
            <p>
              {t("Expiring soon still counts as good standing. The file is Non-compliant while any required credential is expired or missing.")}
            </p>
          </InfoPopover>
        </div>
        {canCreate ? (
          <Button size="sm" onClick={onAdd}>
            <PlusIcon className="size-3.5" />
            {t("Add credential")}
          </Button>
        ) : null}
      </div>

      <div className="flex flex-wrap items-end justify-between gap-4">
        <div className="min-w-0">
          <div className="flex items-baseline gap-1">
            <span className="text-3xl leading-none font-semibold tracking-tight tabular-nums">
              {healthy}/{summary.requiredCount}
            </span>
            <span className="text-muted-foreground text-xs">required</span>
          </div>
          <p className="text-muted-foreground mt-1 text-xs">
            {summary.requiredCount === 0
              ? t("No credential types are required for this worker.")
              : t("{0} of {1} required credentials are in good standing.", healthy, summary.requiredCount)}
          </p>
        </div>
        <dl className="grid grid-cols-3 gap-x-6 text-xs">
          <Count label={t("Expiring")} value={summary.expiringCount} />
          <Count label={t("Expired")} value={summary.expiredCount} />
          <Count label={t("Missing")} value={summary.missingCount} />
        </dl>
      </div>
    </div>
  );
}

function Count({ label, value }: { label: string; value: number }) {
  return (
    <div className="flex flex-col">
      <dt className="text-2xs text-muted-foreground uppercase">{label}</dt>
      <dd className="font-medium tabular-nums">{value}</dd>
    </div>
  );
}
