import { describeToolCall } from "@/components/assistant/tool-presentation";
import { queries } from "@/lib/queries";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { leadingStreak } from "../roster-model";
import { Blk } from "./block";

/** The streak strip shows this many dots. */
const STREAK_DOTS = 10;

type RecordBlockProps = {
  agentId: string;
  toolNames: readonly string[];
  earnedAutonomy: boolean;
};

/** How a saved agent has done over the last 30 days. */
export function RecordBlock({ agentId, toolNames, earnedAutonomy }: RecordBlockProps) {
  const t = useT();
  const scorecardQuery = useQuery(queries.agentScorecard.detail(agentId, "Last30Days"));
  const trustQuery = useQuery(queries.assistant.agentTrust(agentId));
  const card = scorecardQuery.data;
  const streak = useMemo(
    () => leadingStreak(trustQuery.data?.results ?? [], toolNames),
    [toolNames, trustQuery.data?.results],
  );

  const cost = card ? Number(card.costUsd) : null;
  const hours = card ? card.estimatedMinutesSaved / 60 : null;
  const figures: [string, string][] = [
    [t("Runs"), card ? card.runs.toLocaleString() : "—"],
    [
      t("Approved"),
      card?.approvalRate !== null && card?.approvalRate !== undefined
        ? `${Math.round(card.approvalRate * 100)}%`
        : "—",
    ],
    [t("Cost"), cost !== null && Number.isFinite(cost) ? `$${cost.toFixed(2)}` : "—"],
    [t("Time saved"), hours !== null && hours > 0 ? t("{0} h", Math.round(hours)) : "—"],
  ];

  return (
    <Blk id="record" title={t("How it's doing")} note={t("The last 30 days.")}>
      <div className="sc">
        {figures.map(([label, value]) => (
          <div key={label}>
            <span>{label}</span>
            <b className="mono">{value}</b>
          </div>
        ))}
      </div>
      {streak && (
        <div className="streak">
          <div className="streak-d">
            {Array.from({ length: STREAK_DOTS }, (_, index) => (
              <i key={index} className={index < streak.streak ? "on" : undefined} />
            ))}
          </div>
          <span>
            {t("{0} clean approvals in a row on", streak.streak)}{" "}
            <b>{describeToolCall(streak.toolName, null).title}</b>
            {earnedAutonomy ? "" : ` · ${t("earned autonomy is off for the organization")}`}
          </span>
        </div>
      )}
    </Blk>
  );
}
