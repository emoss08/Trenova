import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type {
  AgentSafety,
  AgentToolAutonomy,
  AgentToolPolicy,
  AgentToolSafety,
} from "@/lib/graphql/agent-safety";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { TriangleAlertIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { AgentPicker } from "./agent-picker";
import { AnswerBadge, EgressBadges, HeldByChips } from "./safety-badges";
import { reachLabel, selectedAgents, sortToolsByExposure, tierLabel } from "./safety-model";

type ByAgentPanelProps = {
  agents: readonly AgentSafety[];
  policies: readonly AgentToolPolicy[];
};

/**
 * What each picked agent can do without a person, tool by tool: once for a
 * run that has read nothing from outside, once for a run that has. The
 * chips say which limit holds a call back, and the header says who can
 * reach the agent in the first place.
 */
export function ByAgentPanel({ agents, policies }: ByAgentPanelProps) {
  const t = useT();
  const [picked, setPicked] = useState<string[]>([]);
  const shown = useMemo(() => selectedAgents(agents, picked), [agents, picked]);
  const byName = useMemo(
    () => new Map(policies.map((policy) => [policy.name, policy])),
    [policies],
  );

  return (
    <SectionPanel
      title={t("By agent")}
      help={t(
        "Before outside text is a run that has read only the organization's own records. After outside text is a run that has read an email, a document or another message written outside the organization: anything it would send out waits for approval.",
      )}
      action={<AgentPicker agents={agents} picked={picked} onChange={setPicked} />}
    >
      {agents.length === 0 ? (
        <SectionPanelQuiet>{t("No agents yet.")}</SectionPanelQuiet>
      ) : (
        <div className="divide-border flex flex-col divide-y">
          {shown.map((safety) => (
            <AgentMatrix key={safety.agentId} safety={safety} policies={byName} />
          ))}
        </div>
      )}
    </SectionPanel>
  );
}

function AgentMatrix({
  safety,
  policies,
}: {
  safety: AgentSafety;
  policies: ReadonlyMap<string, AgentToolPolicy>;
}) {
  const t = useT();
  const tools = useMemo(() => sortToolsByExposure(safety.tools), [safety.tools]);

  return (
    <section aria-label={safety.agent.name} className="flex flex-col gap-3 py-3">
      <div className="flex flex-col gap-3 px-3">
        <div className="flex flex-wrap items-center gap-2">
          <h4 className="text-sm font-semibold">{safety.agent.name}</h4>
          {!safety.agent.enabled ? (
            <Badge variant="neutral" appearance="outline">
              {t("Off")}
            </Badge>
          ) : null}
        </div>
        <DescriptionList layout="inline">
          <DescriptionItem label={t("Who can use it")}>{reachLabel(t, safety)}</DescriptionItem>
          <DescriptionItem label={t("Ceiling")}>
            {tierLabel(t, safety.agent.autonomyCeiling)}
          </DescriptionItem>
        </DescriptionList>
        {safety.reach.warnings.map((warning) => (
          <Alert key={warning.kind} variant="warning" size="sm">
            <TriangleAlertIcon />
            <AlertDescription>
              {warning.kind === "OpenWithSensitiveTools"
                ? t(
                    "Everyone who can use the assistant can use this agent, and it holds tools that reach restricted data or leave the organization: {0}.",
                    warning.tools.map((name) => policies.get(name)?.title ?? name).join(", "),
                  )
                : t(
                    "This agent is restricted to roles and no role is granted it, so nobody can use it.",
                  )}
            </AlertDescription>
          </Alert>
        ))}
      </div>

      {tools.length === 0 ? (
        <SectionPanelQuiet>{t("This agent holds no tools.")}</SectionPanelQuiet>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("Tool")}</TableHead>
              <TableHead>{t("Who sees it")}</TableHead>
              <TableHead>{t("Before outside text")}</TableHead>
              <TableHead>{t("After outside text")}</TableHead>
              <TableHead>{t("Held by")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {tools.map((tool) => (
              <ToolRow key={tool.policyName} tool={tool} policy={policies.get(tool.policyName)} />
            ))}
          </TableBody>
        </Table>
      )}
    </section>
  );
}

function ToolRow({ tool, policy }: { tool: AgentToolSafety; policy?: AgentToolPolicy }) {
  const heldBy = [...new Set([...tool.clean.heldBy, ...tool.tainted.heldBy])];

  return (
    <TableRow>
      <TableCell className="align-top">
        <div className="flex min-w-0 flex-col">
          <span>{policy?.title ?? tool.policyName}</span>
          <span className="text-muted-foreground font-mono text-xs">{tool.policyName}</span>
        </div>
      </TableCell>
      <TableCell className="align-top">
        {policy ? <EgressBadges policy={policy} /> : <DescriptionEmpty />}
      </TableCell>
      <TableCell className="align-top">
        <AutonomyCell autonomy={tool.clean} />
      </TableCell>
      <TableCell className="align-top">
        <AutonomyCell autonomy={tool.tainted} />
      </TableCell>
      <TableCell className="align-top">
        <HeldByChips heldBy={heldBy} />
      </TableCell>
    </TableRow>
  );
}

function AutonomyCell({ autonomy }: { autonomy: AgentToolAutonomy }) {
  const t = useT();

  return (
    <div className="flex flex-col items-start gap-1">
      <AnswerBadge autonomy={autonomy} />
      {autonomy.earned ? (
        <span className="text-muted-foreground text-xs">{t("Tier earned")}</span>
      ) : null}
      {autonomy.approvalsToNext != null ? (
        <span className="text-muted-foreground text-xs">
          {t(
            "{0, plural, one {# clean approval to the next tier} other {# clean approvals to the next tier}}",
            autonomy.approvalsToNext,
          )}
        </span>
      ) : null}
    </div>
  );
}
