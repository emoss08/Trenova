import {
  agentLearningStatsQueryKey,
  fetchAgentLearningStats,
  type AgentLearningRow,
} from "@/lib/graphql/agent-memories";
import { useQuery } from "@tanstack/react-query";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";
import { Ic } from "../kit/ic";
import { Seg } from "../kit/layout";
import {
  LEARNING_PERIODS,
  acceptedShare,
  figure,
  learningRowLabel,
  type LearningPeriod,
} from "./memory-learning-model";

/**
 * Whether learning is working: for each way agents come to hold a memory, what
 * it cost, what it offered, and what people did with the offers. A trigger
 * that keeps finding nothing, or whose offers are mostly turned down, is
 * costing more than it teaches; one whose lessons agents are told again is
 * not being applied. Without these figures the triggers are tuned by feel.
 */
export function MemoryLearning() {
  const t = useT();
  const [days, setDays] = useState<LearningPeriod>(30);
  const statsQuery = useQuery({
    queryKey: agentLearningStatsQueryKey(days),
    queryFn: ({ signal }) => fetchAgentLearningStats(days, { signal }),
    placeholderData: (previous) => previous,
  });

  const rows = statsQuery.data?.rows ?? [];
  if (rows.length === 0 && days === 30) {
    return null;
  }

  return (
    <section className="sec" data-testid="memory-learning">
      <header className="sh2">
        <Ic n="gauge" s={14} />
        <h3>{t("How learning is going")}</h3>
        <span className="sp" />
        <Seg
          v={days}
          label={t("Stretch of days")}
          opts={LEARNING_PERIODS.map(
            (period) => [period, t("{0} days", formatNumber(period))] as const,
          )}
          onChange={setDays}
        />
      </header>
      {rows.length === 0 ? (
        <p className="sgm-c">{t("Nothing was learned in this stretch.")}</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("Learned from")}</TableHead>
              <TableHead className="text-right">{t("Look backs")}</TableHead>
              <TableHead className="text-right">{t("Found nothing")}</TableHead>
              <TableHead className="text-right">{t("Kept")}</TableHead>
              <TableHead className="text-right">{t("Waiting")}</TableHead>
              <TableHead className="text-right">{t("Turned down")}</TableHead>
              <TableHead className="text-right">{t("Lapsed")}</TableHead>
              <TableHead className="text-right">{t("Accepted")}</TableHead>
              <TableHead className="text-right">{t("Used since")}</TableHead>
              <TableHead className="text-right">{t("Told again")}</TableHead>
              <TableHead className="text-right">{t("Tokens")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row) => (
              <LearningRow key={`${row.source}:${row.trigger ?? ""}`} row={row} />
            ))}
          </TableBody>
        </Table>
      )}
    </section>
  );
}

function LearningRow({ row }: { row: AgentLearningRow }) {
  const t = useT();
  const looksBack = row.source === "Reflection";
  const share = acceptedShare(row);

  return (
    <TableRow>
      <TableCell>{learningRowLabel(row, t)}</TableCell>
      <TableCell className="text-right tabular-nums">{figure(row.lookBacks, looksBack)}</TableCell>
      <TableCell className="text-right tabular-nums">
        {figure(row.taughtNothing, looksBack)}
      </TableCell>
      <TableCell className="text-right tabular-nums">{figure(row.kept + row.retired)}</TableCell>
      <TableCell className="text-right tabular-nums">{figure(row.waiting)}</TableCell>
      <TableCell className="text-right tabular-nums">{figure(row.dismissed)}</TableCell>
      <TableCell className="text-right tabular-nums">{figure(row.lapsed)}</TableCell>
      <TableCell className="text-right tabular-nums">
        {share === null ? "—" : t("{0}%", formatNumber(share))}
      </TableCell>
      <TableCell className="text-right tabular-nums">{figure(row.used)}</TableCell>
      <TableCell className="text-right tabular-nums">{figure(row.toldAgain, looksBack)}</TableCell>
      <TableCell className="text-muted-foreground text-right tabular-nums">
        {figure(row.inputTokens + row.outputTokens, looksBack)}
      </TableCell>
    </TableRow>
  );
}
