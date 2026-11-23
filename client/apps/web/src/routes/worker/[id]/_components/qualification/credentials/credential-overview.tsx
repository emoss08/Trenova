import { InfoTooltip } from "@/components/info-tooltip";
import type { WorkerOverview } from "@/lib/graphql/worker-overview";
import type { WorkerCredentialSummary } from "@/lib/graphql/worker-credential";
import type { WorkerTarget } from "@/routes/worker/use-worker-state";
import { PlusIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ComplianceBadge } from "../../compliance-badge";
import { ClearanceRunway } from "../../overview/clearance-runway";
import { unverifiedCount } from "./credential-model";

const RUNWAY_SOURCES = ["Credential"] as const;

/**
 * The head of the credentials view: how many required credentials are in good standing, the
 * counts a manager scans, the compliance grade, and the one-lane runway of credential expiries
 * drawn from the page's overview read.
 */
export function CredentialOverview({
  summary,
  clearance,
  asOf,
  canCreate,
  addIntent,
  onAdd,
  onOpen,
}: {
  summary: WorkerCredentialSummary;
  clearance: WorkerOverview["clearance"] | null | undefined;
  asOf: number | null;
  canCreate: boolean;
  /** Reads the add form's code as the pointer or focus reaches Add credential. */
  addIntent: { onPointerEnter: () => void; onFocus: () => void };
  onAdd: () => void;
  onOpen: (target: WorkerTarget) => void;
}) {
  const t = useT();
  const onFile = summary.requiredCount - summary.missingCount;
  const unverified = unverifiedCount(summary);

  return (
    <section
      aria-label={t("Credentials standing")}
      data-testid="credential-overview"
      className="min-w-0 rounded-lg border bg-card"
    >
      <div className="flex min-h-13 flex-wrap items-center gap-x-4 gap-y-2 py-2.5 pr-3 pl-4">
        <div className="flex items-baseline gap-2">
          <span className="text-2xl leading-none font-semibold tracking-tight tabular-nums">
            {t("{0} of {1}", onFile, summary.requiredCount)}
          </span>
          <span className="text-sm text-muted-foreground">
            {summary.requiredCount === 0 ? t("nothing required") : t("required on file")}
          </span>
        </div>
        <p className="text-xs text-muted-foreground tabular-nums">
          <Count tone="warning" value={summary.expiringCount}>
            {t("{0} expiring", summary.expiringCount)}
          </Count>
          {" · "}
          <Count tone="danger" value={summary.expiredCount}>
            {t("{0} expired", summary.expiredCount)}
          </Count>
          {" · "}
          <Count tone="danger" value={summary.missingCount}>
            {t("{0} missing", summary.missingCount)}
          </Count>
          {" · "}
          <span>{t("{0} unverified", unverified)}</span>
        </p>
        <span className="inline-flex items-center gap-1">
          <ComplianceBadge status={summary.complianceStatus} />
          <InfoTooltip label={t("Compliance")}>
            {t(
              "Graded from the credentials on file. Non-compliant while a required credential is expired or missing; one expiring soon still counts as good standing.",
            )}
          </InfoTooltip>
        </span>
        {canCreate ? (
          <Button size="sm" className="ml-auto" {...addIntent} onClick={onAdd}>
            <PlusIcon className="size-3.5" />
            {t("Add credential")}
          </Button>
        ) : null}
      </div>
      {clearance && asOf ? (
        <div className="min-w-0 border-t border-border-subtle pt-1 pr-3 pb-2 pl-4">
          <ClearanceRunway
            clearance={clearance}
            asOf={asOf}
            variant="single"
            sources={RUNWAY_SOURCES}
            onOpen={onOpen}
          />
        </div>
      ) : null}
    </section>
  );
}

function Count({
  tone,
  value,
  children,
}: {
  tone: "warning" | "danger";
  value: number;
  children: string;
}) {
  return (
    <span
      className={cn(
        value > 0 && "font-medium",
        value > 0 && (tone === "warning" ? "text-warning-foreground" : "text-danger-foreground"),
      )}
    >
      {children}
    </span>
  );
}
