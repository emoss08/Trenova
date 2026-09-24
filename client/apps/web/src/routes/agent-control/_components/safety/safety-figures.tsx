import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { CircleAlertIcon } from "lucide-react";

/** The figures move with agents' settings and trust; a minute old is still true enough. */
export const SAFETY_SUMMARY_STALE_MS = 60_000;

/** What the AI can do without a person, counted on the server over every tool and agent. */
export function SafetyFigures() {
  const t = useT();
  const summary = useQuery({
    ...queries.agentSafety.summary(),
    staleTime: SAFETY_SUMMARY_STALE_MS,
  });

  if (summary.isError) {
    return (
      <Alert variant="destructive" size="sm">
        <CircleAlertIcon />
        <AlertDescription>
          {t("What agents can do without a person could not be loaded. Try again shortly.")}
        </AlertDescription>
      </Alert>
    );
  }

  if (!summary.data) {
    return <Skeleton className="h-16" aria-busy />;
  }

  const figures = summary.data;
  return (
    <KpiStrip aria-label={t("AI safety figures")}>
      <KpiStripItem
        label={t("Tools that run without a person")}
        value={figures.runWithoutPerson}
        sub={t("on at least one agent")}
        tone={figures.runWithoutPerson > 0 ? "info" : undefined}
      />
      <KpiStripItem
        label={t("Tools that send outside the organization")}
        value={figures.leaveOrganization}
        sub={t("never past approval")}
      />
      <KpiStripItem
        label={t("Open agents with sensitive tools")}
        value={figures.openWithSensitive}
        sub={t("usable by everyone")}
        tone={figures.openWithSensitive > 0 ? "warning" : undefined}
      />
    </KpiStrip>
  );
}
