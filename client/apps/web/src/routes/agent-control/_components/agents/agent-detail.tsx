import { describeToolCall } from "@/components/assistant/tool-presentation";
import type { AgentControl } from "@/lib/graphql/agent-control";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AgentRosterStat } from "@/lib/graphql/ai-control";
import { queries } from "@/lib/queries";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { TierBar } from "../kit/controls";
import { Ic } from "../kit/ic";
import { Seg } from "../kit/layout";
import { agentMode, leadingStreak, type AgentMode } from "./roster-model";

/** The streak strip shows no more dots than this, however high the threshold. */
const STREAK_DOTS_MAX = 25;

export type AgentDetailProps = {
  agent: AgentDefinitionRow;
  stat: AgentRosterStat | undefined;
  split: readonly [number, number, number];
  control: AgentControl | undefined;
  eventLabel: (kind: string) => string;
  canUpdate: boolean;
  canUpdateControl: boolean;
  canRun: boolean;
  canDelete: boolean;
  onMode: (mode: AgentMode) => void;
  onEdit: () => void;
  onRun: () => void;
  onAsk: () => void;
  onActivity: () => void;
  onRemove: () => void;
  onEarnedAutonomy: () => void;
};

/** What one agent is set to and how it has done, under its read sheet's header. */
export function AgentDetail({
  agent,
  stat,
  split,
  control,
  eventLabel,
  canUpdate,
  canUpdateControl,
  canRun,
  canDelete,
  onMode,
  onEdit,
  onRun,
  onAsk,
  onActivity,
  onRemove,
  onEarnedAutonomy,
}: AgentDetailProps) {
  const t = useT();
  const mode = agentMode(agent);
  const tools = split[0] + split[1] + split[2];
  const trustQuery = useQuery(queries.assistant.agentTrust(agent.id));
  const streak = useMemo(
    () => leadingStreak(trustQuery.data?.results ?? [], agent.toolNames),
    [agent.toolNames, trustQuery.data?.results],
  );
  const threshold = control?.promotionThreshold ?? 0;
  const decided = stat ? stat.approved + stat.modified + stat.rejected + stat.failed : 0;
  const tierLabels = [t("Read"), t("Propose"), t("Act")];
  const modeNote: Record<AgentMode, string> = {
    live: t("Proposals are offered to a person for a decision."),
    shadow: t("Runs, but its proposals are recorded rather than offered."),
    sim: t("Its writes are previewed and recorded, never made."),
  };
  const toolName = streak ? describeToolCall(streak.toolName, null).title : "";

  return (
    <div className="ad">
      <div className="ad-g">
        <div className="ad-s">
          <h4>{t("Mode")}</h4>
          <Seg
            v={mode}
            className="sm"
            label={t("Mode")}
            opts={[
              ["live", t("Live")],
              ["shadow", t("Shadow")],
              ["sim", t("Simulation")],
            ]}
            onChange={(next) => canUpdate && next !== mode && onMode(next)}
          />
          <p className="ad-h">{modeNote[mode]}</p>
          <h4>{t("Who can use it")}</h4>
          <p className="ad-v">
            <Ic n="users" s={12} />
            {agent.accessMode === "Roles"
              ? agent.accessRoles.length
                ? agent.accessRoles.map((role) => role.name).join(", ")
                : t("Nobody yet: no role is granted it")
              : t("Everyone who can use the assistant")}
          </p>
          {agent.triggerMode === "Scheduled" && (
            <>
              <h4>{t("Schedule")}</h4>
              <p className="ad-v mono">
                {agent.cronExpression}
                {agent.cronTimezone && <span className="dim"> · {agent.cronTimezone}</span>}
              </p>
              {agent.nextRunAt && (
                <p className="ad-h">{t("Next run {0}", formatUnixDateTime(agent.nextRunAt))}</p>
              )}
            </>
          )}
          {agent.triggerMode === "Continuous" && (
            <>
              <h4>{t("Runs")}</h4>
              <p className="ad-v">
                {t("Every {0} seconds, at most {1} at once", agent.intervalSeconds, agent.maxConcurrentRuns)}
              </p>
            </>
          )}
          {agent.triggerMode === "Event" && (
            <>
              <h4>{t("Wakes on")}</h4>
              <div className="chips">
                {agent.eventKinds.map((kind) => (
                  <span key={kind} className="tg">
                    {eventLabel(kind)}
                  </span>
                ))}
              </div>
            </>
          )}
        </div>
        <div className="ad-s">
          <h4>
            {t("Tools")}
            <em className="mono">{tools || ""}</em>
          </h4>
          {tools ? (
            <>
              <TierBar counts={split} />
              <div className="tier-l">
                {tierLabels.map((label, index) => (
                  <span key={label}>
                    <i className={`t${index}`} />
                    <b className="mono">{split[index]}</b>
                    {label}
                  </span>
                ))}
              </div>
            </>
          ) : (
            <p className="ad-h">
              {t("No task tools. It explains how things work and can't read or change records.")}
            </p>
          )}
          {agent.delegates.length > 0 && (
            <>
              <h4>{t("Can hand work to")}</h4>
              <p className="ad-v">
                <Ic n="swap" s={12} />
                {agent.delegates.map((delegate) => delegate.name).join(", ")}
              </p>
            </>
          )}
        </div>
        <div className="ad-s">
          <h4>{t("Track record")}</h4>
          {stat && decided > 0 ? (
            <>
              <div className="rec">
                <span>
                  <b className="mono">{stat.approved}</b>
                  {t("approved")}
                </span>
                <span>
                  <b className="mono">{stat.modified}</b>
                  {t("changed")}
                </span>
                <span>
                  <b className="mono">{stat.rejected}</b>
                  {t("rejected")}
                </span>
                <span>
                  <b className={stat.failed ? "mono t-d" : "mono"}>{stat.failed}</b>
                  {t("failed")}
                </span>
              </div>
              {streak &&
                (control?.earnedAutonomy ? (
                  <div className="streak">
                    <div className="streak-d">
                      {Array.from({ length: Math.min(threshold, STREAK_DOTS_MAX) }, (_, index) => (
                        <i key={index} className={index < streak.streak ? "on" : undefined} />
                      ))}
                    </div>
                    <span>
                      {streak.streak >= threshold ? (
                        <>
                          <b>{toolName}</b> {t("earned its next tier")}
                        </>
                      ) : threshold - streak.streak === 1 ? (
                        <>
                          {t("1 more clean approval and")} <b>{toolName}</b>{" "}
                          {t("moves up a tier")}
                        </>
                      ) : (
                        <>
                          {t("{0} more clean approvals and", threshold - streak.streak)}{" "}
                          <b>{toolName}</b> {t("moves up a tier")}
                        </>
                      )}
                    </span>
                  </div>
                ) : (
                  <p className="ad-h">
                    {t(
                      "A streak of {0} clean approvals on {1}. Earned autonomy is off for the organization, so it changes nothing.",
                      streak.streak,
                      toolName,
                    )}{" "}
                    {canUpdateControl && (
                      <button type="button" className="lnk" onClick={onEarnedAutonomy}>
                        {t("Turn it on")}
                      </button>
                    )}
                  </p>
                ))}
            </>
          ) : (
            <p className="ad-h">
              {t("No decisions yet. The record starts with the first proposal someone decides on.")}
            </p>
          )}
        </div>
      </div>
      <div className="ad-bar">
        {canUpdate && (
          <button type="button" className="xa" onClick={onEdit}>
            <Ic n="edit" s={13} />
            {t("Edit agent")}
            <span className="kbd">E</span>
          </button>
        )}
        {agent.triggerMode !== "Chat" && (
          <button
            type="button"
            className="xa"
            disabled={!agent.enabled || !canRun}
            onClick={onRun}
          >
            <Ic n="play" s={12} />
            {t("Run now")}
            <span className="kbd">R</span>
          </button>
        )}
        <button type="button" className="xa" onClick={onActivity}>
          <Ic n="timeline" s={13} />
          {t("Runs and proposals")}
        </button>
        {agent.triggerMode === "Chat" && (
          <button type="button" className="xa" disabled={!agent.enabled} onClick={onAsk}>
            <Ic n="chat" s={13} />
            {t("Ask in Desk")}
          </button>
        )}
        <span className="sp" />
        <button
          type="button"
          className="xa d"
          disabled={Boolean(agent.systemKey) || !canDelete}
          title={agent.systemKey ? t("Started by Trenova itself; cannot be removed") : undefined}
          onClick={onRemove}
        >
          <Ic n={agent.systemKey ? "lock" : "trash"} s={13} />
          {agent.systemKey ? t("System agent") : t("Remove")}
        </button>
      </div>
    </div>
  );
}
