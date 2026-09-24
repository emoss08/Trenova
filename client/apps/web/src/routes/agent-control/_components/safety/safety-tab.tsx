import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { CircleAlertIcon } from "lucide-react";
import { useMemo } from "react";
import { ByAgentPanel } from "./by-agent-panel";
import { safetyFigures } from "./safety-model";
import { ToolRulesPanel } from "./tool-rules-panel";

/**
 * What the AI can do without a person, answered from the same policies the
 * runtime decides every call from: the rule each tool is held to, and what
 * each agent's settings, trust and reach make of those rules.
 */
export default function SafetyTab() {
  const t = useT();
  const policies = useQuery(queries.agentSafety.policies());
  const agents = useQuery(queries.agentSafety.agents());

  const figures = useMemo(
    () => (policies.data && agents.data ? safetyFigures(policies.data, agents.data) : undefined),
    [agents.data, policies.data],
  );

  if (policies.isError || agents.isError) {
    return (
      <Alert variant="destructive" size="sm">
        <CircleAlertIcon />
        <AlertDescription>
          {t("What agents can do without a person could not be loaded. Try again shortly.")}
        </AlertDescription>
      </Alert>
    );
  }

  if (!policies.data || !agents.data || !figures) {
    return (
      <div className="flex min-w-0 flex-col gap-4" aria-busy>
        <Skeleton className="h-16" />
        <Skeleton className="h-64" />
        <Skeleton className="h-48" />
      </div>
    );
  }

  return (
    <div className="flex min-w-0 flex-col gap-4">
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
      <ToolRulesPanel policies={policies.data} />
      <ByAgentPanel agents={agents.data} policies={policies.data} />
    </div>
  );
}
