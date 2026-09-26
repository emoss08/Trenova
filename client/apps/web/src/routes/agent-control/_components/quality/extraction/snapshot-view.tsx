import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { ExtractionSnapshot } from "@/lib/graphql/extraction-eval";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { fieldLabel } from "./extraction-model";

/** Field values and stops as a person confirmed them: what a run is scored against. */
export function SnapshotView({ snapshot, title }: { snapshot: ExtractionSnapshot; title: string }) {
  const t = useT();

  return (
    <div className="flex flex-col gap-4">
      <SectionPanel title={title} count={snapshot.fields.length}>
        {snapshot.fields.length > 0 ? (
          <DescriptionList columns={3} className="p-3">
            {snapshot.fields.map((field) => (
              <DescriptionItem key={field.key} label={fieldLabel(field.key, t)}>
                {field.value}
              </DescriptionItem>
            ))}
          </DescriptionList>
        ) : (
          <SectionPanelQuiet>{t("No field values.")}</SectionPanelQuiet>
        )}
      </SectionPanel>
      <SectionPanel title={t("Stops")} count={snapshot.stops.length}>
        {snapshot.stops.length > 0 ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("Stop")}</TableHead>
                <TableHead>{t("Name")}</TableHead>
                <TableHead>{t("Address")}</TableHead>
                <TableHead>{t("Date")}</TableHead>
                <TableHead>{t("Appointment")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {snapshot.stops.map((stop, index) => (
                <TableRow key={`${stop.role}-${stop.sequence}-${index}`}>
                  <TableCell className="whitespace-nowrap">
                    {stop.role === "pickup"
                      ? t("Pickup {0}", stop.sequence + 1)
                      : t("Delivery {0}", stop.sequence + 1)}
                  </TableCell>
                  <TableCell>{stop.name || "—"}</TableCell>
                  <TableCell>
                    {[stop.addressLine1, stop.city, stop.state, stop.postalCode]
                      .filter(Boolean)
                      .join(", ") || "—"}
                  </TableCell>
                  <TableCell className="whitespace-nowrap">{stop.date || "—"}</TableCell>
                  <TableCell>{stop.appointmentRequired ? t("Yes") : t("No")}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : (
          <SectionPanelQuiet>{t("No stops.")}</SectionPanelQuiet>
        )}
      </SectionPanel>
    </div>
  );
}
