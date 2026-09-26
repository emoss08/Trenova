import { useT } from "@trenova/shared/i18n/use-t";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { formatUnixDate, formatUnixDateTime } from "@trenova/shared/lib/date";
import { trainingExportHistoryQueryOptions } from "@/lib/graphql/agent-control";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { trainingExportTotals } from "./training-export-history-model";

/**
 * Every training export that has taken this organization's corrections, so an
 * administrator can see what was shared and under which grant of consent.
 */
export function TrainingExportHistory() {
  const t = useT();
  const { data, isPending, isError } = useQuery(trainingExportHistoryQueryOptions());
  const totals = useMemo(() => trainingExportTotals(data ?? []), [data]);

  if (isPending) {
    return <Skeleton className="h-4 w-56" />;
  }
  if (isError) {
    return (
      <p className="text-muted-foreground text-xs">
        {t("The training exports that included this organization could not be loaded")}
      </p>
    );
  }
  if (totals.exports === 0) {
    return (
      <p className="text-muted-foreground text-xs">
        {t("No training export has included this organization's corrections.")}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <p className="text-muted-foreground text-xs">
        {t(
          "Included in {0, plural, one {# training export} other {# training exports}}, {1, plural, one {# correction} other {# corrections}} in all",
          totals.exports,
          totals.corrections,
        )}
      </p>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t("Exported")}</TableHead>
            <TableHead className="text-right">{t("Corrections")}</TableHead>
            <TableHead className="text-right">{t("Held out for validation")}</TableHead>
            <TableHead>{t("Consent granted")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((entry) => (
            <TableRow key={entry.exportId}>
              <TableCell>{formatUnixDateTime(entry.exportedAt)}</TableCell>
              <TableCell className="text-right tabular-nums">{entry.examples}</TableCell>
              <TableCell className="text-right tabular-nums">{entry.validationExamples}</TableCell>
              <TableCell>{formatUnixDate(entry.consentGrantedAt)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
