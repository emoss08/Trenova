import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AgentRosterStat, WorkingRun } from "@/lib/graphql/ai-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime, formatUnixWeekday } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { KeyboardEvent } from "react";
import { runStatusLabel } from "../activity/agent-badges";
import { Ring, Runs, TierBar } from "../kit/controls";
import { Ic } from "../kit/ic";
import { Switch } from "../kit/layout";
import { Tile } from "../kit/marks";
import type { RowFact } from "./roster-model";

/** An approval rate under this reads as a warning. */
const APPROVAL_WARN_BELOW = 0.85;

export type AgentRowProps = {
  agent: AgentDefinitionRow;
  stat: AgentRosterStat | undefined;
  /** What it is doing right now, when a run of it is going. */
  working: WorkingRun | undefined;
  fact: RowFact;
  split: readonly [number, number, number];
  open: boolean;
  noProvider: boolean;
  canUpdate: boolean;
  canRun: boolean;
  onOpen: () => void;
  onToggle: (enabled: boolean) => void;
  onRun: () => void;
  onAsk: () => void;
  onReviewWaiting: () => void;
};

/** One agent on the roster: who it is, what it has done lately, and its controls. */
export function AgentRow({
  agent,
  stat,
  working,
  fact,
  split,
  open,
  noProvider,
  canUpdate,
  canRun,
  onOpen,
  onToggle,
  onRun,
  onAsk,
  onReviewWaiting,
}: AgentRowProps) {
  const t = useT();
  const tools = split[0] + split[1] + split[2];
  const runs = stat?.runs ?? 0;
  const rate = stat?.approvalRate ?? null;
  const chat = agent.triggerMode === "Chat";

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === "Enter" && event.target === event.currentTarget) {
      onOpen();
    }
  };

  return (
    <div
      className={cn("al", open && "sel", !agent.enabled && "dim")}
      role="button"
      tabIndex={0}
      aria-expanded={open}
      onClick={onOpen}
      onKeyDown={onKeyDown}
    >
      <span className="pc-mk">
        <Tile agent={agent} s={32} />
      </span>
      <span className="al-t">
        <span className="al-n">
          <b>{agent.name}</b>
          {agent.simulationMode ? (
            <span className="tg v">{t("Simulation")}</span>
          ) : agent.shadowMode ? (
            <span className="tg">
              <Ic n="eyeOff" s={10} />
              {t("Shadow")}
            </span>
          ) : null}
          {agent.systemKey && (
            <span className="tg">
              <Ic n="lock" s={10} />
              {t("System")}
            </span>
          )}
        </span>
        {working ? (
          <span className="al-d shm">
            {`${working.summary.trim() || runStatusLabel(working.status, t)}…`}
          </span>
        ) : (
          <span className="al-d">
            <FactLine fact={fact} />
            {agent.description}
          </span>
        )}
      </span>
      <span className="al-c al-act">
        {noProvider && agent.enabled ? (
          <span className="t-w al-s">{t("Waiting for a provider")}</span>
        ) : (
          <>
            <Runs
              days={stat?.runsByDay ?? EMPTY_DAYS}
              on={agent.enabled && !noProvider}
              label={t("Runs, last 14 days")}
            />
            <span className="mono al-s">
              {runs
                ? runs === 1
                  ? t("1 run")
                  : t("{0} runs", runs.toLocaleString())
                : agent.enabled
                  ? t("Not run yet")
                  : "—"}
            </span>
          </>
        )}
      </span>
      <span className="al-c al-ap">
        {agent.shadowMode && stat && stat.shadowRecorded > 0 ? (
          <span className="al-s" title={t("Proposals recorded in shadow")}>
            <b className="mono">{stat.shadowRecorded.toLocaleString()}</b> {t("recorded")}
          </span>
        ) : rate !== null ? (
          <>
            <Ring value={rate} size={18} className={rate < APPROVAL_WARN_BELOW ? "w" : undefined} />
            <span className="mono">{`${Math.round(rate * 100)}%`}</span>
          </>
        ) : (
          <span className="dim">—</span>
        )}
      </span>
      <span className="al-c al-w">
        {agent.enabled && agent.pendingProposals > 0 && !noProvider ? (
          <button
            type="button"
            className="pill w"
            title={t("Review in Desk")}
            onClick={(event) => {
              event.stopPropagation();
              onReviewWaiting();
            }}
          >
            <Ic n="inbox" s={11} />
            {agent.pendingProposals.toLocaleString()}
          </button>
        ) : (
          <span className="dim">—</span>
        )}
      </span>
      <span
        className="al-c al-tl"
        title={
          tools
            ? t("{0} read · {1} propose · {2} act", split[0], split[1], split[2])
            : t("No task tools")
        }
      >
        {tools ? (
          <>
            <TierBar counts={split} small />
            <span className="mono">{tools}</span>
          </>
        ) : (
          <span className="dim">—</span>
        )}
      </span>
      <span className="al-x" onClick={(event) => event.stopPropagation()}>
        {chat ? (
          <button
            type="button"
            className="ib"
            title={t("Ask in Desk")}
            aria-label={t("Ask {0} in Desk", agent.name)}
            disabled={!agent.enabled}
            onClick={onAsk}
          >
            <Ic n="chat" s={13} />
          </button>
        ) : (
          <button
            type="button"
            className="ib"
            title={t("Run now")}
            aria-label={t("Run {0} now", agent.name)}
            disabled={!agent.enabled || !canRun}
            onClick={onRun}
          >
            <Ic n="play" s={12} />
          </button>
        )}
        <Switch
          on={agent.enabled}
          label={agent.enabled ? t("Disable agent") : t("Enable agent")}
          disabled={!canUpdate}
          onChange={onToggle}
        />
      </span>
    </div>
  );
}

const EMPTY_DAYS: readonly number[] = Array.from({ length: 14 }, () => 0);

function FactLine({ fact }: { fact: RowFact }) {
  const t = useT();
  if (!fact) {
    return null;
  }
  if (fact.kind === "schedule") {
    return (
      <span className="al-m">
        <Ic n="clock" s={10} />
        {t("Next {0} {1}", formatUnixWeekday(fact.nextRunAt), formatUnixTime(fact.nextRunAt))}
      </span>
    );
  }
  if (fact.kind === "event") {
    return (
      <span className="al-m">
        <Ic n="bolt" s={10} />
        {fact.more > 0 ? `${fact.label} +${fact.more}` : fact.label}
      </span>
    );
  }
  return (
    <span className="al-m">
      <Ic n="users" s={10} />
      {fact.names.join(", ")}
    </span>
  );
}
