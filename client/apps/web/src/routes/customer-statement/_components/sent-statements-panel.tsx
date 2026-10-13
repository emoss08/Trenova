import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatMinor } from "@trenova/shared/lib/benefits";
import { formatUnixDateMedium, formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { queries } from "@/lib/queries";
import type { SentStatement } from "@/types/customer-statement";
import { useQuery } from "@tanstack/react-query";

/** The statements this customer has been emailed, newest first. */
export function SentStatementsPanel({ customerId }: { customerId: string }) {
  const t = useT();
  const { data, isLoading, isError } = useQuery(queries.customerStatement.sent(customerId));

  return (
    <SectionPanel title={t("Statements sent")} count={data?.length}>
      {isLoading ? (
        <div className="flex flex-col gap-2 p-3">
          <Skeleton className="h-4 w-2/3" />
          <Skeleton className="h-4 w-1/2" />
        </div>
      ) : isError ? (
        <SectionPanelQuiet>{t("Could not load the statements sent.")}</SectionPanelQuiet>
      ) : !data || data.length === 0 ? (
        <SectionPanelQuiet>
          {t(
            "No statement has been emailed to this customer yet. Statements you email, and the monthly ones billing control sends, appear here.",
          )}
        </SectionPanelQuiet>
      ) : (
        <ol className="divide-border divide-y">
          {data.map((statement) => (
            <SentStatementRow key={statement.id} statement={statement} />
          ))}
        </ol>
      )}
    </SectionPanel>
  );
}

function SentStatementRow({ statement }: { statement: SentStatement }) {
  const t = useT();
  const period =
    statement.periodStart > 0
      ? t(
          "{0} to {1}",
          formatUnixDateMedium(statement.periodStart),
          formatUnixDateMedium(statement.periodEnd),
        )
      : t("Through {0}", formatUnixDateMedium(statement.periodEnd));

  return (
    <li className="flex flex-col gap-1 px-3 py-2.5">
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <Badge variant={statement.monthly ? "info" : "neutral"}>
            {statement.monthly ? t("Monthly") : t("Sent by hand")}
          </Badge>
          <span className="truncate text-sm">{period}</span>
        </div>
        <span className="text-muted-foreground shrink-0 text-xs">
          {formatUnixDateTimeMedium(statement.sentAt)}
        </span>
      </div>
      <div className="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs">
        <span className="text-foreground tabular-nums">
          {t("{0} due", formatMinor(statement.endingBalanceMinor, statement.currencyCode))}
        </span>
        {statement.pastDueMinor > 0 ? (
          <span className="text-danger-foreground tabular-nums">
            {t("{0} past due", formatMinor(statement.pastDueMinor, statement.currencyCode))}
          </span>
        ) : null}
        <span className="truncate">
          {statement.sentByName
            ? t("To {0}, by {1}", statement.sentTo, statement.sentByName)
            : t("To {0}", statement.sentTo)}
        </span>
      </div>
    </li>
  );
}
