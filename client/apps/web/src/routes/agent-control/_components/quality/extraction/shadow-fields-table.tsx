import type { ExtractionShadowReport } from "@/lib/graphql/extraction-shadow";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { formatShare } from "../quality-model";
import { fieldLabel } from "./extraction-model";
import { accuracyGap } from "./shadow-model";

type FieldComparison = ExtractionShadowReport["fields"][number];

function Gap({ field }: { field: FieldComparison }) {
  const t = useT();
  const gap = accuracyGap(
    { accuracy: field.candidateAccuracy, scored: field.candidateScored },
    { accuracy: field.productionAccuracy, scored: field.productionScored },
  );
  if (gap === null) {
    return <span className="text-muted-foreground">—</span>;
  }

  return (
    <span className={cn(gap > 0 && "text-success", gap < 0 && "text-danger")}>
      {gap > 0 ? t("+{0} pts", gap) : t("{0} pts", gap)}
    </span>
  );
}

/** Each field's accuracy on both sides, the candidate's biggest shortfall first. */
export function ShadowFieldsTable({ fields }: { fields: FieldComparison[] }) {
  const t = useT();

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t("Field")}</TableHead>
          <TableHead className="text-right">{t("Candidate")}</TableHead>
          <TableHead className="text-right">{t("Production")}</TableHead>
          <TableHead className="text-right">{t("Difference")}</TableHead>
          <TableHead className="text-right">{t("Fields scored")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {fields.map((field) => (
          <TableRow key={field.key}>
            <TableCell className="whitespace-nowrap">{fieldLabel(field.key, t)}</TableCell>
            <TableCell className="text-right tabular-nums">
              {field.candidateScored > 0 ? formatShare(field.candidateAccuracy) : "—"}
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {field.productionScored > 0 ? formatShare(field.productionAccuracy) : "—"}
            </TableCell>
            <TableCell className="text-right tabular-nums">
              <Gap field={field} />
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {Math.max(field.candidateScored, field.productionScored)}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
