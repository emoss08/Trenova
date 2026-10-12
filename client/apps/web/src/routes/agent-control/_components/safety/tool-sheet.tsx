import { usePermission } from "@/hooks/use-permission";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { fetchToolHolderAnswers, type AgentToolPolicy } from "@/lib/graphql/agent-safety";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { TOOL_RULE_EDIT_PARAM, toolRuleEditParser } from "../../ai-control-tabs";
import { useAddressedFlag } from "../../use-addressed-flag";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { Ic, type IcName } from "../kit/ic";
import { Tile } from "../kit/marks";
import { ReadSheet } from "../kit/read-sheet";
import { PolicyDetails } from "./policy-details";
import { AnswerBadge, EgressBadges } from "./safety-badges";
import { kindLabel, needsLabel, readsOutsideLabel, tierLabel } from "./safety-model";
import { ToolRuleEditor } from "./tool-rule-editor";
import { Button } from "@trenova/shared/components/ui/button";

const KIND_ICON: Record<AgentToolPolicy["kind"], IcName> = {
  Query: "search",
  Action: "tool",
  Runtime: "cog",
};

type ToolSheetProps = {
  policy: AgentToolPolicy | null;
  holders: readonly AgentDefinitionRow[];
  /** The agent the sheet was opened from, marked in the list of holders. */
  focus?: string | null;
  onClose: () => void;
};

/**
 * One tool's rule, read-only: who sees its work, the most it may do, what it needs and
 * whether it reads outside content, then every agent holding it with what that agent
 * makes of it before and after reading outside text.
 */
export function ToolSheet({ policy: opened, holders, focus, onClose }: ToolSheetProps) {
  const t = useT();
  const navigate = useAIControlNavigation();
  const { allowed: canChange } = usePermission(Resource.AgentControl, Operation.Update);
  const [saved, setSaved] = useState<AgentToolPolicy | null>(null);
  const [editing, setEditing] = useAddressedFlag(TOOL_RULE_EDIT_PARAM, toolRuleEditParser);
  // The editor opens over the sheet, so closing the sheet closes the editor with it.
  const close = () => {
    setEditing(false);
    onClose();
  };
  const policy = opened && saved?.name === opened.name ? saved : opened;

  return (
    <ReadSheet
      open={policy !== null}
      onClose={close}
      label={policy?.title ?? t("Tool rule")}
      head={
        policy && (
          <>
            <span className="src-i">
              <Ic n={KIND_ICON[policy.kind]} s={15} />
            </span>
            <div className="sh-t">
              <b>{policy.title}</b>
              <span className="mono">{policy.name}</span>
            </div>
          </>
        )
      }
    >
      {policy && (
        <>
          <div className="sh-m">
            <EgressBadges egress={policy.egress} />
            <span>{kindLabel(t, policy.kind)}</span>
          </div>
          <dl className="sfx">
            <div>
              <dt>{t("Most it may do")}</dt>
              <dd>{tierLabel(t, policy.promotableTier)}</dd>
            </div>
            <div>
              <dt>{t("Needs")}</dt>
              <dd>{needsLabel(t, policy)}</dd>
            </div>
            <div>
              <dt>{t("Reads outside content")}</dt>
              <dd>{readsOutsideLabel(t, policy) ?? t("Never")}</dd>
            </div>
            {policy.rule && (
              <div>
                <dt>{t("Your organization's rule")}</dt>
                <dd>
                  {policy.rule.reason || t("Held lower than the tool declares")}
                  <span className="dim">
                    {" · "}
                    {policy.rule.updatedBy
                      ? t(
                          "{0}, {1}",
                          policy.rule.updatedBy.name,
                          formatUnixDateMedium(policy.rule.updatedAt),
                        )
                      : formatUnixDateMedium(policy.rule.updatedAt)}
                  </span>
                </dd>
              </div>
            )}
            <div>
              <dt>{t("Who sees its work")}</dt>
              <dd>
                {policy.leavesOrganization
                  ? t("Leaves the organization — never past approval")
                  : policy.kind === "Query"
                    ? t("Nobody; it only reads")
                    : t("Stays inside the organization")}
              </dd>
            </div>
          </dl>
          <div className="sh-p">
            <h4 className="sh-k">
              {t("{0, plural, one {Held by # agent} other {Held by # agents}}", holders.length)}
            </h4>
            {holders.length > 0 ? (
              <HolderList policyName={policy.name} holders={holders} focus={focus ?? null} />
            ) : (
              <p className="ad-h">{t("No agent holds this tool.")}</p>
            )}
            <h4 className="sh-k">{t("The whole rule")}</h4>
            <PolicyDetails policy={policy} />
          </div>
          <div className="ad-bar sh-f">
            <span className="sp" />
            <button type="button" className="xa" onClick={() => navigate({ tab: "audit" })}>
              <Ic n="receipt" s={13} />
              {t("Audit trail")}
            </button>
            {canChange && (
              <Button type="button" variant="outline" onClick={() => setEditing(true)}>
                <Ic n="edit" s={13} />
                {t("Change tool rule")}
              </Button>
            )}
          </div>
          <ToolRuleEditor
            open={editing && canChange}
            policy={policy}
            holders={holders}
            onClose={() => setEditing(false)}
            onSaved={setSaved}
          />
        </>
      )}
    </ReadSheet>
  );
}

function HolderList({
  policyName,
  holders,
  focus,
}: {
  policyName: string;
  holders: readonly AgentDefinitionRow[];
  focus: string | null;
}) {
  const t = useT();
  const ids = holders.map((agent) => agent.id);
  const answers = useQuery({
    queryKey: ["agentSafety", "holderAnswers", policyName, ids.join(",")],
    queryFn: ({ signal }) => fetchToolHolderAnswers(policyName, ids, { signal }),
  });
  const byAgent = new Map((answers.data ?? []).map((row) => [row.agentId, row]));

  return (
    <div className="hl-l">
      {holders.map((agent) => {
        const row = byAgent.get(agent.id);
        return (
          <div
            key={agent.id}
            className={cn("hl-r", focus === agent.id && "on", !agent.enabled && "dim")}
          >
            <Tile agent={agent} s={24} />
            <div className="hl-n">
              <b>{agent.name}</b>
              <span>
                {agent.enabled
                  ? t("Ceiling {0}", tierLabel(t, agent.autonomyCeiling))
                  : t("Ceiling {0} · off", tierLabel(t, agent.autonomyCeiling))}
              </span>
            </div>
            {row && (
              <div className="ba">
                <AnswerBadge answer={row.clean.answer} />
                {row.tainted.answer !== row.clean.answer && (
                  <>
                    <Ic n="arrowR" s={11} />
                    <AnswerBadge answer={row.tainted.answer} />
                  </>
                )}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
