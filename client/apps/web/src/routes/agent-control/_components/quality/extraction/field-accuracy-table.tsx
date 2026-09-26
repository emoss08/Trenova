import type { ExtractionFieldAccuracy } from "@/lib/graphql/extraction-eval";
import { Progress } from "@trenova/shared/components/ui/progress";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatShare } from "../quality-model";
import { accuracyTone, fieldLabel } from "./extraction-model";

const TONE_VARIANT = {
  success: "success",
  warning: "warning",
  danger: "error",
} as const;

/** Accuracy per field, worst first, with the counts it rests on. */
export function FieldAccuracyTable({ fields }: { fields: ExtractionFieldAccuracy[] }) {
  const t = useT();

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t("Field")}</TableHead>
          <TableHead className="w-48">{t("Accuracy")}</TableHead>
          <TableHead className="text-right">{t("Correct")}</TableHead>
          <TableHead className="text-right">{t("Corrected")}</TableHead>
          <TableHead className="text-right">{t("Missed")}</TableHead>
          <TableHead className="text-right">{t("Unconfirmed")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {fields.map((field) => {
          const tone = accuracyTone(field.accuracy, field.scored);
          return (
            <TableRow key={field.key}>
              <TableCell>{fieldLabel(field.key, t)}</TableCell>
              <TableCell>
                <div className="flex items-center gap-2">
                  <Progress
                    className="w-24"
                    size="sm"
                    value={field.scored > 0 ? field.accuracy * 100 : 0}
                    variant={tone ? TONE_VARIANT[tone] : "default"}
                    aria-label={t("{0} accuracy", fieldLabel(field.key, t))}
                  />
                  <span className="tabular-nums">
                    {field.scored > 0 ? formatShare(field.accuracy) : "—"}
                  </span>
                </div>
              </TableCell>
              <TableCell className="text-right tabular-nums">{field.correct}</TableCell>
              <TableCell className="text-right tabular-nums">{field.corrected}</TableCell>
              <TableCell className="text-right tabular-nums">{field.missed}</TableCell>
              <TableCell className="text-muted-foreground text-right tabular-nums">
                {field.unconfirmed}
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
