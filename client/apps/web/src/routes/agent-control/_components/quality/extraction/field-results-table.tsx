import type { AICorrectionFieldResult } from "@/lib/graphql/extraction-eval";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { OutcomeBadge } from "./extraction-badges";
import { fieldLabel } from "./extraction-model";

/** Each field side by side: what was read, what a person confirmed, and the outcome. */
export function FieldResultsTable({ results }: { results: AICorrectionFieldResult[] }) {
  const t = useT();

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t("Field")}</TableHead>
          <TableHead>{t("Read")}</TableHead>
          <TableHead>{t("Confirmed")}</TableHead>
          <TableHead>{t("Outcome")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {results.map((result) => (
          <TableRow key={result.key}>
            <TableCell className="whitespace-nowrap">{fieldLabel(result.key, t)}</TableCell>
            <TableCell className="max-w-64 truncate" title={result.predicted}>
              {result.predicted || <span className="text-muted-foreground">—</span>}
            </TableCell>
            <TableCell className="max-w-64 truncate" title={result.confirmed}>
              {result.confirmed || <span className="text-muted-foreground">—</span>}
            </TableCell>
            <TableCell>
              <OutcomeBadge value={result.outcome} t={t} />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
