import { DocumentPanePreview } from "@/components/documents/document-pane-preview";
import { WorkPane, WorkPaneBody, WorkPaneFooter } from "@/components/work-pane";
import type { WorkerSafetyEventRow } from "@/lib/graphql/worker-safety";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  INSPECTION_RESULT_LABELS,
  SAFETY_EVENT_STATUS_LABELS,
  SAFETY_SEVERITY_LABELS,
  type InspectionResult,
  type SafetyEventStatus,
  type SafetySeverity,
} from "@trenova/shared/types/worker-safety";
import type { ReactNode } from "react";
import { decimalAmount, eventCarriesViolations, eventStatusTone } from "./safety-model";
import {
  formatDecimalMoney,
  safetyEventHeadline,
  type SafetyEventPermissions,
} from "./safety-event-row";
import { ViolationList } from "./violation-list";

export type SafetyEventPaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The event the pane reads; null when the link names one no longer on the record. */
  event: WorkerSafetyEventRow | null;
  permissions: SafetyEventPermissions;
  busy: boolean;
  onEdit: (event: WorkerSafetyEventRow) => void;
  onClose: (event: WorkerSafetyEventRow) => void;
  onReopen: (event: WorkerSafetyEventRow) => Promise<unknown>;
  onDelete: (event: WorkerSafetyEventRow) => void;
  onDiscipline: (event: WorkerSafetyEventRow) => void;
  onOpenDocuments: () => void;
  onViolationsChanged: () => Promise<unknown>;
};

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-1.5">
      <h4 className="text-xs font-medium text-foreground-subtle">{title}</h4>
      {children}
    </section>
  );
}

/** One safety event read beside the list: its facts, what happened, what it cited and how it ended. */
export function SafetyEventPane({
  open,
  onOpenChange,
  event,
  permissions,
  busy,
  onEdit,
  onClose,
  onReopen,
  onDelete,
  onDiscipline,
  onOpenDocuments,
  onViolationsChanged,
}: SafetyEventPaneProps) {
  const t = useT();
  if (!event) {
    return (
      <WorkPane open={open} onOpenChange={onOpenChange} title={t("Safety event")}>
        <WorkPaneBody className="text-sm">
          <p className="text-sm text-muted-foreground">
            {t("This event is no longer on the record. It may have been deleted.")}
          </p>
        </WorkPaneBody>
      </WorkPane>
    );
  }

  const status = event.status as SafetyEventStatus;
  const closed = status === "Closed";
  const fine = decimalAmount(event.fineAmount);
  const cost = decimalAmount(event.costAmount);
  const facts: { label: string; value: string }[] = [
    { label: t("When"), value: formatUnixDateMedium(event.occurredAt) },
    { label: t("Where"), value: event.location || t("Not recorded") },
    {
      label: t("Severity"),
      value: t(SAFETY_SEVERITY_LABELS[event.severity as SafetySeverity] ?? event.severity),
    },
    { label: t("Reference"), value: event.referenceNumber || t("None") },
  ];
  if (event.kind === "Inspection") {
    facts.push(
      {
        label: t("Level"),
        value: event.inspectionLevel ? t("Level {0}", event.inspectionLevel) : t("Not recorded"),
      },
      {
        label: t("Outcome"),
        value: event.inspectionResult
          ? t(INSPECTION_RESULT_LABELS[event.inspectionResult as InspectionResult])
          : t("Not recorded"),
      },
    );
  }
  if (event.kind === "Accident" || event.kind === "Incident") {
    facts.push({ label: t("Preventable"), value: event.preventable ? t("Yes") : t("No") });
  }
  if (fine !== null) facts.push({ label: t("Fine"), value: formatDecimalMoney(fine) });
  if (cost !== null)
    facts.push({ label: t("Cost to the carrier"), value: formatDecimalMoney(cost) });
  facts.push({ label: t("Recorded by"), value: event.recordedBy?.name ?? t("Not recorded") });

  const points = event.pointsExpireAt
    ? event.activePoints > 0
      ? t(
          "{0, plural, one {# point} other {# points}} · roll off {1}",
          event.activePoints,
          formatUnixDateMedium(event.pointsExpireAt),
        )
      : t(
          "{0, plural, one {# point} other {# points}} · rolled off {1}",
          event.points,
          formatUnixDateMedium(event.pointsExpireAt),
        )
    : t("{0, plural, one {# point} other {# points}}", event.activePoints);

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={safetyEventHeadline(event)}
      contentKey={event.id}
    >
      <WorkPaneBody className="text-sm">
        <div className="flex items-center justify-between gap-2">
          <Badge variant={eventStatusTone(status)} dot={!closed}>
            {t(SAFETY_EVENT_STATUS_LABELS[status] ?? status)}
          </Badge>
          <span className="font-mono text-xs text-muted-foreground">{points}</span>
        </div>
        <DescriptionList layout="ruled" columns={2}>
          {facts.map((fact) => (
            <DescriptionItem key={fact.label} label={fact.label} numeric>
              {fact.value}
            </DescriptionItem>
          ))}
        </DescriptionList>
        <Section title={t("What happened")}>
          <p className="text-pretty">{event.description}</p>
        </Section>
        {eventCarriesViolations(event) ? (
          <Section title={t("Violations cited")}>
            <ViolationList event={event} onChanged={onViolationsChanged} />
          </Section>
        ) : null}
        {closed && event.resolution ? (
          <Section title={t("Resolution")}>
            <p className="text-pretty">{event.resolution}</p>
            {event.closedBy?.name ? (
              <p className="text-xs text-muted-foreground">
                {event.closedAt
                  ? t("{0}, {1}", event.closedBy.name, formatUnixDateMedium(event.closedAt))
                  : event.closedBy.name}
              </p>
            ) : null}
          </Section>
        ) : null}
        {event.document ? (
          <Section title={t("Document")}>
            <DocumentPanePreview
              document={{
                id: event.document.id,
                originalName: event.document.originalName || event.document.fileName,
                fileType: event.document.fileType,
                fileSize: event.document.fileSize ?? 0,
                previewStatus: event.document.previewStatus,
              }}
            >
              <p className="flex flex-wrap items-center justify-center gap-x-2 text-xs text-muted-foreground">
                <span className="tabular-nums">
                  {t("Added {0}", formatUnixDateMedium(event.document.createdAt))}
                </span>
                <button
                  type="button"
                  onClick={onOpenDocuments}
                  className="rounded-sm ui-focus-ring text-xs font-medium text-brand hover:underline"
                >
                  {t("Open in Documents")}
                </button>
              </p>
            </DocumentPanePreview>
          </Section>
        ) : null}
      </WorkPaneBody>
      <WorkPaneFooter
        leading={
          permissions.canDelete && !closed ? (
            <Button
              size="sm"
              variant="outline"
              className="text-danger-foreground"
              disabled={busy}
              onClick={() => onDelete(event)}
            >
              {t("Delete")}
            </Button>
          ) : null
        }
      >
        {permissions.canUpdate ? (
          <Button size="sm" variant="outline" disabled={busy} onClick={() => onEdit(event)}>
            {t("Edit")}
          </Button>
        ) : null}
        {permissions.canDiscipline ? (
          <Button size="sm" variant="outline" disabled={busy} onClick={() => onDiscipline(event)}>
            {t("Issue action")}
          </Button>
        ) : null}
        {permissions.canClose ? (
          closed ? (
            <Button
              size="sm"
              isLoading={busy}
              onClick={() => void onReopen(event).catch(() => undefined)}
            >
              {t("Reopen event")}
            </Button>
          ) : (
            <Button size="sm" disabled={busy} onClick={() => onClose(event)}>
              {t("Close event")}
            </Button>
          )
        ) : null}
      </WorkPaneFooter>
    </WorkPane>
  );
}
