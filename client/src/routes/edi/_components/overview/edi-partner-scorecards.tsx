import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { EdiPartnerScorecardsDocument } from "@/graphql/generated/graphql";
import { formatDurationFromSeconds } from "@/lib/date";
import { cn } from "@/lib/utils";
import type { ResultOf } from "@graphql-typed-document-node/core";
import { Link } from "react-router";

type EDIPartnerScorecard = ResultOf<
  typeof EdiPartnerScorecardsDocument
>["ediPartnerScorecards"][number];

function formatRate(rate?: number | null) {
  if (rate === null || rate === undefined) return "—";
  return `${Math.round(rate * 1000) / 10}%`;
}

function formatSeconds(seconds?: number | null) {
  if (seconds === null || seconds === undefined || seconds <= 0) return "—";
  return formatDurationFromSeconds(Math.round(seconds));
}

function attentionCellClass(count: number) {
  return cn("text-right tabular-nums", count > 0 && "font-semibold text-red-600 dark:text-red-400");
}

export function EDIPartnerScorecards({ scorecards }: { scorecards: EDIPartnerScorecard[] }) {
  if (scorecards.length === 0) {
    return (
      <div className="rounded-md border bg-background p-6 text-sm text-muted-foreground">
        No partner activity in the selected time range.
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-md border bg-background">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Partner</TableHead>
            <TableHead className="text-right">Sent</TableHead>
            <TableHead className="text-right">Failed</TableHead>
            <TableHead className="text-right">Dead-lettered</TableHead>
            <TableHead className="text-right">Received</TableHead>
            <TableHead className="text-right">Success rate</TableHead>
            <TableHead className="text-right">Ack avg</TableHead>
            <TableHead className="text-right">Ack p95</TableHead>
            <TableHead className="text-right">Overdue acks</TableHead>
            <TableHead className="text-right">&gt;4h</TableHead>
            <TableHead className="text-right">&gt;24h</TableHead>
            <TableHead className="text-right">Oldest pending</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {scorecards.map((card) => {
            const successRate = card.deliverySuccessRate;
            return (
              <TableRow key={card.partnerId}>
                <TableCell>
                  <Link
                    to={`/edi/messages?query=${encodeURIComponent(card.partnerCode)}`}
                    className="flex flex-col hover:underline"
                    title={`Open messages filtered to ${card.partnerName}`}
                  >
                    <span className="text-sm font-medium">{card.partnerName}</span>
                    <span className="text-xs text-muted-foreground">{card.partnerCode}</span>
                  </Link>
                </TableCell>
                <TableCell className="text-right tabular-nums">{card.sentCount}</TableCell>
                <TableCell className={attentionCellClass(card.failedCount)}>
                  {card.failedCount}
                </TableCell>
                <TableCell className={attentionCellClass(card.deadLetteredCount)}>
                  {card.deadLetteredCount}
                </TableCell>
                <TableCell className="text-right tabular-nums">{card.receivedCount}</TableCell>
                <TableCell
                  className={cn(
                    "text-right tabular-nums",
                    successRate !== null &&
                      successRate !== undefined &&
                      successRate < 0.95 &&
                      "font-semibold text-yellow-700 dark:text-yellow-400",
                  )}
                >
                  {formatRate(successRate)}
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {formatSeconds(card.avgAckSeconds)}
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {formatSeconds(card.p95AckSeconds)}
                </TableCell>
                <TableCell className={attentionCellClass(card.overdueAckCount)}>
                  {card.overdueAckCount}
                </TableCell>
                <TableCell className={attentionCellClass(card.pendingOver4hCount)}>
                  {card.pendingOver4hCount}
                </TableCell>
                <TableCell className={attentionCellClass(card.pendingOver24hCount)}>
                  {card.pendingOver24hCount}
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {formatSeconds(card.oldestPendingAgeSeconds)}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
