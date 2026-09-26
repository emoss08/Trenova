import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { ExtractionAccuracy } from "@/lib/graphql/extraction-eval";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatShare } from "../quality-model";
import { FieldAccuracyTable } from "./field-accuracy-table";
import { documentKindLabel, modelLabel } from "./extraction-model";

type Group = ExtractionAccuracy["byModel"][number];

function GroupTable({
  groups,
  heading,
  label,
}: {
  groups: Group[];
  heading: string;
  label: (key: string, t: TranslateFn) => string;
}) {
  const t = useT();

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{heading}</TableHead>
          <TableHead className="text-right">{t("Corrections")}</TableHead>
          <TableHead className="text-right">{t("Fields scored")}</TableHead>
          <TableHead className="text-right">{t("Accuracy")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {groups.map((group) => (
          <TableRow key={group.key || "none"}>
            <TableCell>{label(group.key, t)}</TableCell>
            <TableCell className="text-right tabular-nums">{group.corrections}</TableCell>
            <TableCell className="text-right tabular-nums">{group.scored}</TableCell>
            <TableCell className="text-right tabular-nums">
              {group.scored > 0 ? formatShare(group.accuracy) : "—"}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

/**
 * How production extraction did over the window, read from the corrections
 * people made when they created shipments from drafts. A field left as it was
 * drafted counts as correct, so these figures are a ceiling: the form is
 * pre-filled, and nobody is asked to confirm every value.
 */
export function AccuracyPanel({ accuracy }: { accuracy: ExtractionAccuracy }) {
  const t = useT();

  return (
    <div className="grid min-w-0 gap-4 xl:grid-cols-[2fr_1fr]">
      <SectionPanel
        title={t("Accuracy by field")}
        hint={t("worst first")}
        help={t(
          "Correct fields divided by fields a person confirmed: correct, corrected or missed. A field left as it was drafted counts as correct, so treat these as a ceiling.",
        )}
      >
        {accuracy.fields.length > 0 ? (
          <FieldAccuracyTable fields={accuracy.fields} />
        ) : (
          <SectionPanelQuiet>
            {t(
              "No corrections yet. They are recorded when someone creates a shipment from a document's draft.",
            )}
          </SectionPanelQuiet>
        )}
      </SectionPanel>
      <div className="flex min-w-0 flex-col gap-4">
        <SectionPanel title={t("By model")}>
          {accuracy.byModel.length > 0 ? (
            <GroupTable groups={accuracy.byModel} heading={t("Model")} label={modelLabel} />
          ) : (
            <SectionPanelQuiet>{t("Nothing to compare yet.")}</SectionPanelQuiet>
          )}
        </SectionPanel>
        <SectionPanel title={t("By document")}>
          {accuracy.byKind.length > 0 ? (
            <GroupTable
              groups={accuracy.byKind}
              heading={t("Document")}
              label={documentKindLabel}
            />
          ) : (
            <SectionPanelQuiet>{t("Nothing to compare yet.")}</SectionPanelQuiet>
          )}
        </SectionPanel>
      </div>
    </div>
  );
}
